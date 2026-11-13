package common

import (
	"cli-enonic/internal/app/commands/remote"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/urfave/cli"
)

func contextWithAuth(t *testing.T, auth string) *cli.Context {
	set := flag.NewFlagSet("test", flag.ContinueOnError)
	set.String("cred-file", "", "")
	set.String("auth", "", "")
	if auth != "" {
		set.Set("auth", auth)
	}
	return cli.NewContext(cli.NewApp(), set, nil)
}

func TestEnvTokenRequiresBeta(t *testing.T) {
	setupTokenTest(t, false)
	t.Setenv(CLI_TOKEN_ENV_VAR, "gh-oidc")
	exitCode := -1
	exit = func(code int) { exitCode = code }
	t.Cleanup(func() { exit = os.Exit })

	if req := CreateRequest(nil, "GET", "repo/list", nil); req != nil || exitCode != 1 {
		t.Fatalf("beta off: got %v, exit %d", req, exitCode)
	}
	// requests that would not carry it are not affected
	exitCode = -1
	if req := CreateRequest(nil, "GET", MARKET_URL, nil); req == nil || exitCode != -1 {
		t.Fatalf("beta off, market: got %v, exit %d", req, exitCode)
	}
}

func TestEnvTokenSentToTheManagementEndpointOnly(t *testing.T) {
	setupTokenTest(t, true)
	t.Setenv(CLI_TOKEN_ENV_VAR, " gh-oidc \n")
	t.Setenv(remote.CLI_REMOTE_USER, "su")
	t.Setenv(remote.CLI_REMOTE_PASS, "pw")
	// a stored login is not used while the variable is set
	SaveAccessToken(remote.LocalTarget(""), validToken("stored"))

	if got := CreateRequest(nil, "GET", "repo/list", nil).Header.Get("Authorization"); got != "Bearer gh-oidc" {
		t.Fatalf("got %q", got)
	}
	for _, other := range []string{MARKET_URL, SCOOP_MANIFEST_URL, "http://localhost:2609/server"} {
		if got := CreateRequest(nil, "GET", other, nil).Header.Get("Authorization"); got == "Bearer gh-oidc" {
			t.Fatalf("token sent to %s", other)
		}
	}
	// -a is more specific than the environment
	if _, pass, ok := CreateRequest(contextWithAuth(t, "me:secret"), "GET", "repo/list", nil).BasicAuth(); !ok || pass != "secret" {
		t.Fatal("-a did not win")
	}
}

func TestEnvTokenRefusedExitsWithoutPrompt(t *testing.T) {
	setupTokenTest(t, true)
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
		}))
		t.Setenv(remote.CLI_REMOTE_URL, server.URL)
		t.Setenv(CLI_TOKEN_ENV_VAR, "gh-oidc")
		// a stored login that could be refreshed must not be tried either
		SaveAccessToken(remote.GetActiveRemote(), &AccessToken{TokenEndpoint: server.URL + "/token", AccessToken: "x", RefreshToken: "rt", ExpiresAt: time.Now().Add(time.Hour).Unix()})

		exitCode, prompts := -1, 0
		exit = func(code int) { exitCode = code }
		promptAuth = func(string, bool) (string, string) { prompts++; return "su", "pw" }

		c := contextWithAuth(t, "")
		_, err := SendRequestCustom(c, CreateRequest(c, "GET", "repo/list", nil), "", 1)
		server.Close()
		exit, promptAuth = os.Exit, EnsureAuth
		if exitCode != 1 || prompts != 0 || err == nil {
			t.Fatalf("status %d: exit %d, prompts %d, err %v", status, exitCode, prompts, err)
		}
	}
}

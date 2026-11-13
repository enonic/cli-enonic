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

func TestEnvTokenSentToTheManagementEndpointOnly(t *testing.T) {
	for _, beta := range []bool{false, true} {
		setupTokenTest(t, beta)
		t.Setenv(CLI_TOKEN_ENV_VAR, " gh-oidc \n")
		t.Setenv(remote.CLI_REMOTE_USER, "su")
		t.Setenv(remote.CLI_REMOTE_PASS, "pw")
		if beta {
			// a stored login is not used while the variable is set
			SaveAccessToken(remote.LocalTarget(""), validToken("stored"))
		}

		if got := CreateRequest(nil, "GET", "repo/list", nil).Header.Get("Authorization"); got != "Bearer gh-oidc" {
			t.Fatalf("beta %v: got %q", beta, got)
		}
		for _, other := range []string{MARKET_URL, SCOOP_MANIFEST_URL, "http://localhost:2609/server"} {
			if got := CreateRequest(nil, "GET", other, nil).Header.Get("Authorization"); got == "Bearer gh-oidc" {
				t.Fatalf("beta %v: token sent to %s", beta, other)
			}
		}
		// -a is more specific than the environment
		if _, pass, ok := CreateRequest(contextWithAuth(t, "me:secret"), "GET", "repo/list", nil).BasicAuth(); !ok || pass != "secret" {
			t.Fatalf("beta %v: -a did not win", beta)
		}
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

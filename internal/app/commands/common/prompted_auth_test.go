package common

import (
	"cli-enonic/internal/app/commands/remote"
	"flag"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/urfave/cli"
)

func TestPromptedPasswordReusedWithinCommand(t *testing.T) {
	setupTokenTest(t, true)
	promptedAuth = map[string]string{}
	answers := []string{"su:wrong", "su:pw"}
	prompts := 0
	promptAuth = func(auth string, force bool) (string, string) {
		if auth != "" {
			user, pass, _ := strings.Cut(auth, ":")
			return user, pass
		}
		answer := answers[0]
		if len(answers) > 1 {
			answers = answers[1:]
		}
		prompts++
		user, pass, _ := strings.Cut(answer, ":")
		return user, pass
	}
	t.Cleanup(func() { promptAuth = EnsureAuth; promptedAuth = map[string]string{} })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user, pass, _ := r.BasicAuth(); user != "su" || pass != "pw" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	t.Setenv(remote.CLI_REMOTE_URL, server.URL)

	set := flag.NewFlagSet("test", flag.ContinueOnError)
	set.String("cred-file", "", "")
	c := cli.NewContext(cli.NewApp(), set, nil)

	// an operation and three task polls: the wrong password is asked again once, the right one is kept
	for i := 0; i < 4; i++ {
		res, err := SendRequestCustom(c, CreateRequest(c, "GET", "task/1", nil), "", 1)
		if err != nil || res.StatusCode != 200 {
			t.Fatalf("request %d: %v %v", i, res, err)
		}
		res.Body.Close()
	}
	if prompts != 2 {
		t.Fatalf("expected 2 prompts, got %d", prompts)
	}

	// another target does not get the password
	if got := rememberedAuth("https://other.example/repo"); got != "" {
		t.Fatalf("password remembered for another url: %q", got)
	}
}

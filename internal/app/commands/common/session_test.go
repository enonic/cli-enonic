package common

import (
	"cli-enonic/internal/app/commands/remote"
	"cli-enonic/internal/app/util"
	"testing"
)

func sessionCookie(t *testing.T) string {
	t.Helper()
	cookie, err := CreateRequest(nil, "GET", "repo/list", nil).Cookie(JSESSIONID)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func TestSessionOnlySentToItsUrl(t *testing.T) {
	setupTokenTest(t, true)
	t.Setenv(remote.CLI_REMOTE_USER, "su")
	t.Setenv(remote.CLI_REMOTE_PASS, "pw")
	WriteRuntimeData(RuntimeData{SessionID: "s1", SessionUrl: "http://localhost:4848"})
	prodUrl, _ := remote.ParseMarshalledUrl("https://prod.example")
	if err := remote.WriteEnvironments(&remote.EnvironmentsData{Environments: map[string]remote.Environment{"prod": {Url: prodUrl}}}); err != nil {
		t.Fatal(err)
	}

	if got := sessionCookie(t); got != "s1" {
		t.Fatalf("local: expected session cookie, got %q", got)
	}

	for _, other := range []string{MARKET_URL, SCOOP_MANIFEST_URL, "http://localhost:2609/server"} {
		if cookie, err := CreateRequest(nil, "GET", other, nil).Cookie(JSESSIONID); err == nil {
			t.Fatalf("session sent to %s: %v", other, cookie)
		}
	}

	remote.SelectEnv("prod")
	if got := sessionCookie(t); got != "" {
		t.Fatalf("prod: session sent to another instance: %q", got)
	}
	if rData := ReadRuntimeData(); rData.SessionID != "s1" {
		t.Fatalf("session of local was cleared by a password login to prod: %+v", rData)
	}

	remote.SelectEnv("")
	t.Setenv(util.BETA_ENV_VAR, "")
	t.Setenv(remote.CLI_REMOTE_URL, "https://prod.example")
	if got := sessionCookie(t); got != "s1" {
		t.Fatalf("beta off: expected session cookie as on master, got %q", got)
	}
}

package common

import (
	"cli-enonic/internal/app/commands/remote"
	"cli-enonic/internal/app/util"
	"net/http"
	"net/http/httptest"
	"testing"
)

func sessionCookie(t *testing.T, reqUrl string) string {
	t.Helper()
	cookie, err := CreateRequest(nil, "GET", reqUrl, nil).Cookie(JSESSIONID)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func TestNoSessionWithBeta(t *testing.T) {
	setupTokenTest(t, true)
	t.Setenv(remote.CLI_REMOTE_USER, "su")
	t.Setenv(remote.CLI_REMOTE_PASS, "pw")
	WriteRuntimeData(RuntimeData{SessionID: "s1"})

	for _, reqUrl := range []string{"repo/list", MARKET_URL, SCOOP_MANIFEST_URL} {
		if got := sessionCookie(t, reqUrl); got != "" {
			t.Fatalf("beta on: session sent to %s: %q", reqUrl, got)
		}
	}
	req := CreateRequest(nil, "GET", "repo/list", nil)
	if user, pass, ok := req.BasicAuth(); !ok || user != "su" || pass != "pw" {
		t.Fatalf("beta on: expected password authentication, got %q", req.Header.Get("Authorization"))
	}
	if rData := ReadRuntimeData(); rData.SessionID != "s1" {
		t.Fatalf("beta on: the session saved without beta was cleared: %+v", rData)
	}

	// a session cookie in a response is not saved
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: JSESSIONID, Value: "s2"})
	}))
	defer server.Close()
	t.Setenv(remote.CLI_REMOTE_URL, server.URL)
	res, err := SendRequestCustom(nil, CreateRequest(nil, "GET", "repo/list", nil), "", 1)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if rData := ReadRuntimeData(); rData.SessionID != "s1" {
		t.Fatalf("beta on: session from the response was saved: %+v", rData)
	}

	t.Setenv(util.BETA_ENV_VAR, "")
	if got := sessionCookie(t, "repo/list"); got != "s1" {
		t.Fatalf("beta off: expected session cookie as on master, got %q", got)
	}
}

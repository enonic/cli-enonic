package common

import (
	"cli-enonic/internal/app/commands/remote"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestResolveRequestUrl(t *testing.T) {
	setupTokenTest(t, true)
	cases := []struct{ base, req, want string }{
		{"http://localhost:4848/", "repo/list", "http://localhost:4848/repo/list"},
		{"http://localhost:4848", "/repo/list", "http://localhost:4848/repo/list"},
		{"https://host/mgmt", "repo/snapshot", "https://host/mgmt/repo/snapshot"},
		{"https://host/mgmt/", "/task/t1", "https://host/mgmt/task/t1"},
		{"https://host/mgmt", "app/list?installed=true", "https://host/mgmt/app/list?installed=true"},
		{"https://host/mgmt", "http://localhost:2609/server", "http://localhost:2609/server"},
	}
	for _, c := range cases {
		t.Setenv(remote.CLI_REMOTE_URL, c.base)
		got, err := resolveRequestUrl(c.req)
		if err != nil || got.String() != c.want {
			t.Errorf("%s + %s: got %v %v, want %s", c.base, c.req, got, err, c.want)
		}
		if req := doCreateSimpleRequest("GET", c.req, nil); req.URL.String() != c.want {
			t.Errorf("%s + %s: request to %s", c.base, c.req, req.URL)
		}
	}
}

func TestWithinManagement(t *testing.T) {
	setupTokenTest(t, true)
	t.Setenv(remote.CLI_REMOTE_URL, "https://host/mgmt")
	cases := map[string]bool{
		"https://host/mgmt":          true,
		"https://host/mgmt/task/t1":  true,
		"https://HOST:443/mgmt/x":    true,
		"https://host/task/t1":       false,
		"https://host/mgmtx/a":       false,
		"https://host/mgmt/../admin": false,
		"https://host:8443/mgmt/x":   false,
		"http://host/mgmt/x":         false,
		"https://sub.host/mgmt/x":    false,
	}
	for raw, want := range cases {
		u, _ := url.Parse(raw)
		if got := withinManagement(u); got != want {
			t.Errorf("%s: got %v, want %v", raw, got, want)
		}
	}
}

// With the endpoint under a path, the token goes to its operations and their task polls, nowhere else.
func TestTokenOnlyUnderTheManagementPath(t *testing.T) {
	setupTokenTest(t, true)
	t.Setenv(remote.CLI_REMOTE_URL, "https://host/mgmt")
	t.Setenv(remote.CLI_REMOTE_USER, "su")
	t.Setenv(remote.CLI_REMOTE_PASS, "pw")
	SaveAccessToken(remote.GetActiveRemote(), validToken("at"))

	for _, path := range []string{"repo/snapshot", "/task/t1"} {
		req := CreateRequest(nil, "GET", path, nil)
		if req.Header.Get("Authorization") != "Bearer at" || !strings.HasPrefix(req.URL.Path, "/mgmt/") {
			t.Errorf("%s: %s %q", path, req.URL, req.Header.Get("Authorization"))
		}
	}
	if got := CreateRequest(nil, "GET", "https://host/admin/x", nil).Header.Get("Authorization"); strings.HasPrefix(got, "Bearer") {
		t.Errorf("token sent outside the management path")
	}
}

func TestRedirectOutsideManagementRefused(t *testing.T) {
	setupTokenTest(t, true)
	var leaked string
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		leaked = r.Header.Get("Authorization")
	}))
	defer other.Close()
	mgmt := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repo/list":
			// another port of the same host: the default client would keep the Authorization header
			http.Redirect(w, r, strings.Replace(other.URL, "127.0.0.1", "localhost", 1)+"/steal", http.StatusFound)
		case "/moved":
			http.Redirect(w, r, "/repo/here", http.StatusFound)
		case "/repo/here":
			w.Write([]byte(r.Header.Get("Authorization")))
		}
	}))
	defer mgmt.Close()
	t.Setenv(remote.CLI_REMOTE_URL, strings.Replace(mgmt.URL, "127.0.0.1", "localhost", 1))

	for _, setup := range []func(){
		func() { SaveAccessToken(remote.GetActiveRemote(), validToken("stored")) },
		func() { t.Setenv(CLI_TOKEN_ENV_VAR, "gh-oidc") },
	} {
		setup()
		leaked = ""
		_, err := SendRequestCustom(nil, CreateRequest(nil, "GET", "repo/list", nil), "", 1)
		if err == nil || leaked != "" {
			t.Fatalf("redirect followed: err %v, leaked %q", err, leaked)
		}
	}

	// a redirect within the endpoint is followed
	res, err := SendRequestCustom(nil, CreateRequest(nil, "GET", "moved", nil), "", 1)
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("redirect within the endpoint: %v %v", res, err)
	}
	res.Body.Close()
}

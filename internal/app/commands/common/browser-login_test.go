package common

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// authorizeServer acts like XP's authorize and token endpoints: it checks the authorization request, sends
// the browser to the loopback redirect with a code, and gives tokens for the code with the PKCE verifier
func authorizeServer(t *testing.T, deny string) *httptest.Server {
	var challenge, redirectUri, resource string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		switch r.URL.Path {
		case "/authorize":
			q := r.URL.Query()
			redirectUri, challenge, resource = q.Get("redirect_uri"), q.Get("code_challenge"), q.Get("resource")
			if q.Get("response_type") != "code" || q.Get("client_id") != CLI_CLIENT_ID || q.Get("code_challenge_method") != "S256" ||
				!strings.HasPrefix(redirectUri, "http://127.0.0.1:") || challenge == "" || q.Get("state") == "" {
				http.Error(w, "bad authorization request "+q.Encode(), http.StatusBadRequest)
				return
			}
			back := url.Values{"state": {q.Get("state")}}
			if deny != "" {
				back.Set("error", deny)
				back.Set("error_description", "The client may not use authorization_code")
			} else {
				back.Set("code", "c1")
			}
			http.Redirect(w, r, redirectUri+"?"+back.Encode(), http.StatusFound)
		case "/token":
			f := r.PostForm
			if f.Get("grant_type") != AUTHORIZATION_CODE_GRANT_TYPE || f.Get("code") != "c1" || f.Get("redirect_uri") != redirectUri ||
				pkceChallenge(f.Get("code_verifier")) != challenge || f.Get("resource") != resource {
				jsonResponse(w, 400, map[string]string{"error": "invalid_grant"})
				return
			}
			jsonResponse(w, 200, map[string]interface{}{"access_token": "at", "refresh_token": "rt", "expires_in": 300})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func browse(t *testing.T, u string) *http.Response {
	t.Helper()
	res, err := http.Get(u)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	return res
}

func TestBrowserLogin(t *testing.T) {
	server := authorizeServer(t, "")
	endpoints := &OAuthEndpoints{Authorization: server.URL + "/authorize", Token: server.URL + "/token", Resource: "https://mgmt.example"}
	login, err := StartBrowserLogin(endpoints)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(login.AuthorizationUrl, "resource=https%3A%2F%2Fmgmt.example") {
		t.Fatalf("no resource in %s", login.AuthorizationUrl)
	}

	// a redirect with another state, e.g. from a stale tab, is not taken as the answer
	stale := browse(t, login.redirectUri+"?state=other&code=x")
	if stale.StatusCode != http.StatusBadRequest {
		t.Fatalf("stale redirect: %d", stale.StatusCode)
	}

	if res := browse(t, login.AuthorizationUrl); res.StatusCode != http.StatusOK {
		t.Fatalf("browser got %d", res.StatusCode)
	}
	token, err := login.Wait(http.DefaultClient)
	if err != nil {
		t.Fatal(err)
	}
	if token.AccessToken != "at" || token.RefreshToken != "rt" || token.Resource != "https://mgmt.example" || token.TokenEndpoint != endpoints.Token {
		t.Fatalf("unexpected token %+v", token)
	}
	if _, err := http.Get(login.redirectUri); err == nil {
		t.Fatal("still listening after login")
	}
}

func TestBrowserLoginRefused(t *testing.T) {
	server := authorizeServer(t, "unauthorized_client")
	login, err := StartBrowserLogin(&OAuthEndpoints{Authorization: server.URL + "/authorize", Token: server.URL + "/token"})
	if err != nil {
		t.Fatal(err)
	}
	browse(t, login.AuthorizationUrl)
	if _, err := login.Wait(http.DefaultClient); err == nil || err.Error() != "unauthorized_client: The client may not use authorization_code" {
		t.Fatalf("got %v", err)
	}
}

func TestBrowserLoginNotOffered(t *testing.T) {
	if _, err := StartBrowserLogin(&OAuthEndpoints{DeviceAuthorization: "http://x/device/code", Token: "http://x/token"}); err == nil {
		t.Fatal("expected error without an authorization endpoint")
	}
}

func TestDiscoverAuthorizationEndpoint(t *testing.T) {
	server := discoveryServer(t, discoveryOptions{})
	mgmtUrl, _ := url.Parse(server.URL)
	endpoints, err := DiscoverEndpoints(http.DefaultClient, mgmtUrl)
	if err != nil {
		t.Fatal(err)
	}
	if endpoints.Authorization != server.URL+"/_/idprovider/enonic-cloud/authorize" {
		t.Fatalf("unexpected endpoints %+v", endpoints)
	}
}

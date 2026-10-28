package common

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func jsonResponse(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func init() {
	defaultPollInterval = time.Millisecond
}

func TestDeviceLoginAndRefresh(t *testing.T) {
	t.Setenv("ENONIC_CLI_HOME_PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	polls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.Form.Get("client_id") != CLI_CLIENT_ID {
			jsonResponse(w, 400, map[string]string{"error": "invalid_request"})
			return
		}
		switch r.URL.Path {
		case "/_/idprovider/system/device/code":
			jsonResponse(w, 200, map[string]interface{}{
				"device_code": "dc", "user_code": "BCDF-GHJK", "verification_uri_complete": "http://x/device?user_code=BCDF-GHJK",
				"expires_in": 10, "interval": 0,
			})
		case "/_/idprovider/system/token":
			switch r.Form.Get("grant_type") {
			case DEVICE_CODE_GRANT_TYPE:
				polls++
				if polls < 2 {
					jsonResponse(w, 400, map[string]string{"error": "authorization_pending"})
					return
				}
				jsonResponse(w, 200, map[string]interface{}{"access_token": "at1", "refresh_token": "rt1", "expires_in": 0})
			case "refresh_token":
				if r.Form.Get("refresh_token") != "rt1" {
					jsonResponse(w, 400, map[string]string{"error": "invalid_grant"})
					return
				}
				jsonResponse(w, 200, map[string]interface{}{"access_token": "at2", "expires_in": 3600})
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	code, err := StartDeviceLogin(http.DefaultClient, server.URL, "system")
	if err != nil {
		t.Fatal(err)
	}
	if code.UserCode != "BCDF-GHJK" {
		t.Fatalf("unexpected user code %q", code.UserCode)
	}

	code.Interval = 0
	token, err := PollDeviceLogin(http.DefaultClient, server.URL, "system", code)
	if err != nil {
		t.Fatal(err)
	}
	if token.AccessToken != "at1" || polls != 2 {
		t.Fatalf("unexpected token %+v after %d polls", token, polls)
	}

	mgmtUrl, _ := url.Parse("http://localhost:4848/")
	if err := SaveAccessToken(mgmtUrl, token); err != nil {
		t.Fatal(err)
	}

	// token expires immediately, so it is refreshed and the refresh token is kept
	if got := getAccessToken(nil, mgmtUrl); got != "at2" {
		t.Fatalf("expected refreshed token, got %q", got)
	}
	stored := readTokens()[mgmtKey(mgmtUrl)]
	if stored.RefreshToken != "rt1" || stored.ExpiresAt < time.Now().Add(time.Hour-time.Minute).Unix() {
		t.Fatalf("unexpected stored token %+v", stored)
	}

	if removed, err := RemoveAccessToken(mgmtUrl); err != nil || !removed {
		t.Fatalf("expected token removed, got %v %v", removed, err)
	}
	if got := getAccessToken(nil, mgmtUrl); got != "" {
		t.Fatalf("expected no token after logout, got %q", got)
	}
}

func TestDeviceLoginDenied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 400, map[string]string{"error": "access_denied", "error_description": "The user denied the request"})
	}))
	defer server.Close()

	_, err := PollDeviceLogin(http.DefaultClient, server.URL, "system", &DeviceCode{DeviceCode: "dc", ExpiresIn: 10})
	if err == nil || err.Error() != "access_denied: The user denied the request" {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestRefreshRejectsErrorWithoutOAuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 503, map[string]string{"message": "Service Unavailable"})
	}))
	defer server.Close()

	token := &AccessToken{WebUrl: server.URL, IdProvider: "system", RefreshToken: "rt"}
	if _, err := refreshAccessToken(http.DefaultClient, token); err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("expected unexpected response error, got %v", err)
	}
}

func TestRefreshKeepsLoginReplacedMeanwhile(t *testing.T) {
	t.Setenv("ENONIC_CLI_HOME_PATH", t.TempDir())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 400, map[string]string{"error": "invalid_grant"})
	}))
	defer server.Close()

	mgmtUrl, _ := url.Parse("http://localhost:4848")
	stale := &AccessToken{WebUrl: server.URL, IdProvider: "system", AccessToken: "old", RefreshToken: "rt-old"}
	fresh := &AccessToken{WebUrl: server.URL, IdProvider: "system", AccessToken: "new", RefreshToken: "rt-new", ExpiresAt: time.Now().Add(time.Hour).Unix()}
	SaveAccessToken(mgmtUrl, fresh)

	// a refresh of the stale login fails, but must not remove the login stored meanwhile
	if got := renewAccessToken(nil, mgmtKey(mgmtUrl), stale); got != "" {
		t.Fatalf("expected failed refresh, got %q", got)
	}
	if stored := readTokens()[mgmtKey(mgmtUrl)]; stored == nil || stored.AccessToken != "new" {
		t.Fatalf("login stored meanwhile was lost: %+v", stored)
	}
}

func TestIsActiveRemoteUrl(t *testing.T) {
	t.Setenv("ENONIC_CLI_REMOTE_URL", "http://localhost:4848")
	tests := map[string]bool{
		"repo/list":                    true,
		"/repo/list":                   true,
		"http://localhost:4848/repo":   true,
		"HTTP://LOCALHOST:4848/repo":   true,
		"http://localhost:2609/server": false,
		"https://localhost:4848/repo":  false,
		"http://example.com:4848/repo": false,
		MARKET_URL:                     false,
	}
	for reqUrl, want := range tests {
		if got := isActiveRemoteUrl(reqUrl); got != want {
			t.Errorf("isActiveRemoteUrl(%q) = %v, want %v", reqUrl, got, want)
		}
	}
}

func TestUserLogin(t *testing.T) {
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"user:system:su"}`))
	if got := (&AccessToken{AccessToken: "h." + payload + ".s"}).UserLogin(); got != "su" {
		t.Fatalf("expected su, got %q", got)
	}
	if got := (&AccessToken{AccessToken: "opaque"}).UserLogin(); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

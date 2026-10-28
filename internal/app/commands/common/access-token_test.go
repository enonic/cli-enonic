package common

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
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

	code, err := StartDeviceLogin(server.URL, "system")
	if err != nil {
		t.Fatal(err)
	}
	if code.UserCode != "BCDF-GHJK" {
		t.Fatalf("unexpected user code %q", code.UserCode)
	}

	code.Interval = 0
	token, err := PollDeviceLogin(server.URL, "system", code)
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
	if got := getAccessToken(mgmtUrl); got != "at2" {
		t.Fatalf("expected refreshed token, got %q", got)
	}
	stored := readTokens()[mgmtKey(mgmtUrl)]
	if stored.RefreshToken != "rt1" || stored.ExpiresAt < time.Now().Add(time.Hour-time.Minute).Unix() {
		t.Fatalf("unexpected stored token %+v", stored)
	}

	if removed, err := RemoveAccessToken(mgmtUrl); err != nil || !removed {
		t.Fatalf("expected token removed, got %v %v", removed, err)
	}
	if got := getAccessToken(mgmtUrl); got != "" {
		t.Fatalf("expected no token after logout, got %q", got)
	}
}

func TestDeviceLoginDenied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 400, map[string]string{"error": "access_denied", "error_description": "The user denied the request"})
	}))
	defer server.Close()

	_, err := PollDeviceLogin(server.URL, "system", &DeviceCode{DeviceCode: "dc", ExpiresIn: 10})
	if err == nil || err.Error() != "access_denied: The user denied the request" {
		t.Fatalf("unexpected error %v", err)
	}
}

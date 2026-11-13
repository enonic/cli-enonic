package common

import (
	"cli-enonic/internal/app/commands/remote"
	"cli-enonic/internal/app/util"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/urfave/cli"
)

func jsonResponse(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func init() {
	defaultPollInterval = time.Millisecond
}

// setupTokenTest gives the test an empty enonic dir, with beta on unless told otherwise
func setupTokenTest(t *testing.T, beta bool) {
	home := t.TempDir()
	t.Setenv("ENONIC_CLI_HOME_PATH", home)
	t.Setenv("HOME", t.TempDir())
	os.Unsetenv(util.SNAP_ENV_VAR)
	t.Setenv(remote.CLI_ENV, "")
	t.Setenv(remote.CLI_REMOTE_URL, "")
	t.Setenv(remote.CLI_REMOTE_PROXY, "")
	t.Setenv(remote.CLI_REMOTE_USER, "")
	t.Setenv(remote.CLI_REMOTE_PASS, "")
	if beta {
		t.Setenv(util.BETA_ENV_VAR, "true")
	} else {
		t.Setenv(util.BETA_ENV_VAR, "")
	}
	remote.SelectEnv("")
	t.Cleanup(func() { remote.SelectEnv("") })
	if err := os.MkdirAll(util.GetInEnonicDir(), 0700); err != nil {
		t.Fatal(err)
	}
}

func envTarget(t *testing.T, name, rawUrl string) *remote.RemoteData {
	u, err := remote.ParseMarshalledUrl(rawUrl)
	if err != nil {
		t.Fatal(err)
	}
	return &remote.RemoteData{Name: name, Url: u}
}

func idpEndpoints(webUrl, idProvider string) *OAuthEndpoints {
	base := fmt.Sprintf("%s/_/idprovider/%s/", webUrl, idProvider)
	return &OAuthEndpoints{DeviceAuthorization: base + "device/code", Token: base + "token"}
}

func TestDeviceLoginAndRefresh(t *testing.T) {
	setupTokenTest(t, true)

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

	code, err := StartDeviceLogin(http.DefaultClient, idpEndpoints(server.URL, "system"))
	if err != nil {
		t.Fatal(err)
	}
	if code.UserCode != "BCDF-GHJK" {
		t.Fatalf("unexpected user code %q", code.UserCode)
	}

	code.Interval = 0
	token, err := PollDeviceLogin(http.DefaultClient, idpEndpoints(server.URL, "system"), code)
	if err != nil {
		t.Fatal(err)
	}
	if token.AccessToken != "at1" || polls != 2 {
		t.Fatalf("unexpected token %+v after %d polls", token, polls)
	}

	target := envTarget(t, "local", "http://localhost:4848/")
	if err := SaveAccessToken(target, token); err != nil {
		t.Fatal(err)
	}

	// token expires immediately, so it is refreshed and the refresh token is kept
	if got := getAccessToken(nil, target); got != "at2" {
		t.Fatalf("expected refreshed token, got %q", got)
	}
	_, stored := storedToken(target)
	if stored == nil || stored.RefreshToken != "rt1" || stored.ExpiresAt < time.Now().Add(time.Hour-time.Minute).Unix() {
		t.Fatalf("unexpected stored token %+v", stored)
	}

	if removed, err := RemoveAccessToken(target); err != nil || !removed {
		t.Fatalf("expected token removed, got %v %v", removed, err)
	}
	if got := getAccessToken(nil, target); got != "" {
		t.Fatalf("expected no token after logout, got %q", got)
	}
}

func TestDeviceLoginDenied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 400, map[string]string{"error": "access_denied", "error_description": "The user denied the request"})
	}))
	defer server.Close()

	_, err := PollDeviceLogin(http.DefaultClient, idpEndpoints(server.URL, "system"), &DeviceCode{DeviceCode: "dc", ExpiresIn: 10})
	if err == nil || err.Error() != "access_denied: The user denied the request" {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestRefreshRejectsErrorWithoutOAuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 503, map[string]string{"message": "Service Unavailable"})
	}))
	defer server.Close()

	token := &AccessToken{TokenEndpoint: server.URL + "/token", RefreshToken: "rt"}
	if _, err := refreshAccessToken(http.DefaultClient, token); err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("expected unexpected response error, got %v", err)
	}
}

func TestRefreshKeepsLoginReplacedMeanwhile(t *testing.T) {
	setupTokenTest(t, true)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 400, map[string]string{"error": "invalid_grant"})
	}))
	defer server.Close()

	target := envTarget(t, "local", "http://localhost:4848")
	stale := &AccessToken{TokenEndpoint: server.URL + "/token", AccessToken: "old", RefreshToken: "rt-old"}
	fresh := &AccessToken{TokenEndpoint: server.URL + "/token", AccessToken: "new", RefreshToken: "rt-new", ExpiresAt: time.Now().Add(time.Hour).Unix()}
	SaveAccessToken(target, fresh)
	path, _ := tokenFile(target)

	// a refresh of the stale login fails, but must not remove the login stored meanwhile
	if got := renewAccessToken(nil, path, stale); got != "" {
		t.Fatalf("expected failed refresh, got %q", got)
	}
	if _, stored := storedToken(target); stored == nil || stored.AccessToken != "new" {
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

func validToken(name string) *AccessToken {
	return &AccessToken{TokenEndpoint: "http://idp.example/token", AccessToken: name, RefreshToken: "rt-" + name, ExpiresAt: time.Now().Add(time.Hour).Unix()}
}

func TestTokenFilePerEnvironment(t *testing.T) {
	setupTokenTest(t, true)
	prod := envTarget(t, "prod", "https://prod.example")
	stage := envTarget(t, "stage", "https://stage.example")
	if err := SaveAccessToken(prod, validToken("p")); err != nil {
		t.Fatal(err)
	}
	if err := SaveAccessToken(stage, validToken("s")); err != nil {
		t.Fatal(err)
	}
	if removed, err := RemoveAccessToken(prod); !removed || err != nil {
		t.Fatalf("got %v %v", removed, err)
	}
	if got := getAccessToken(nil, stage); got != "s" {
		t.Fatalf("stage login lost, got %q", got)
	}
	if HasAccessToken(prod) || !HasAccessToken(stage) {
		t.Fatal("unexpected stored logins")
	}
	dir := util.GetInEnonicDir("tokens")
	if _, err := os.Stat(filepath.Join(dir, "prod.json")); !os.IsNotExist(err) {
		t.Fatalf("prod.json still there: %v", err)
	}
	if stat, err := os.Stat(filepath.Join(dir, "stage.json")); err != nil || stat.Mode().Perm() != 0600 {
		t.Fatalf("stage.json: %v %v", stat, err)
	}
	if stat, err := os.Stat(dir); err != nil || stat.Mode().Perm() != 0700 {
		t.Fatalf("tokens dir: %v %v", stat, err)
	}
}

func TestTokenNotSentToOtherUrl(t *testing.T) {
	setupTokenTest(t, true)
	if err := SaveAccessToken(envTarget(t, "prod", "https://a.example"), validToken("a")); err != nil {
		t.Fatal(err)
	}
	if got := getAccessToken(nil, envTarget(t, "prod", "https://b.example")); got != "" {
		t.Fatalf("token sent to another url: %q", got)
	}
	if got := getAccessToken(nil, envTarget(t, "prod", "https://a.example/")); got != "a" {
		t.Fatalf("expected token for the same url, got %q", got)
	}
}

func TestOneOffTokenFileName(t *testing.T) {
	setupTokenTest(t, true)
	a, err := tokenFile(envTarget(t, "", "http://Host.Example"))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := tokenFile(envTarget(t, "", "http://host.example:80/"))
	if a != b || !strings.HasSuffix(a, filepath.Join("tokens", "url-host.example-80.json")) {
		t.Fatalf("got %q and %q", a, b)
	}
	odd, err := tokenFile(&remote.RemoteData{Url: &remote.MarshalledUrl{URL: url.URL{Scheme: "http", Host: "..%2f..%2fx"}}})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(odd) != util.GetInEnonicDir("tokens") {
		t.Fatalf("escaped tokens dir: %q", odd)
	}
	if _, err := tokenFile(envTarget(t, "../x", "http://x.example")); err == nil {
		t.Fatal("expected error for invalid name")
	}
}

func TestBetaOffSendsNoToken(t *testing.T) {
	setupTokenTest(t, true)
	t.Setenv(remote.CLI_REMOTE_USER, "su")
	t.Setenv(remote.CLI_REMOTE_PASS, "pw")
	if err := SaveAccessToken(remote.LocalTarget(""), validToken("at")); err != nil {
		t.Fatal(err)
	}
	if got := CreateRequest(nil, "GET", "repo/list", nil).Header.Get("Authorization"); got != "Bearer at" {
		t.Fatalf("beta on: got %q", got)
	}

	t.Setenv(util.BETA_ENV_VAR, "")
	if got := CreateRequest(nil, "GET", "repo/list", nil).Header.Get("Authorization"); strings.HasPrefix(got, "Bearer") {
		t.Fatalf("beta off sent %q", got)
	}
}

func TestTransientRefreshFailureKeepsLogin(t *testing.T) {
	setupTokenTest(t, true)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 503, map[string]string{"message": "Service Unavailable"})
	}))
	defer server.Close()
	exitCode := -1
	exit = func(code int) { exitCode = code }
	t.Cleanup(func() { exit = os.Exit })

	target := envTarget(t, "local", "http://localhost:4848")
	SaveAccessToken(target, &AccessToken{TokenEndpoint: server.URL + "/token", AccessToken: "at", RefreshToken: "rt"})
	getAccessToken(nil, target)
	if exitCode != 1 {
		t.Fatalf("expected exit 1, got %d", exitCode)
	}
	if !HasAccessToken(target) {
		t.Fatal("login lost on a transient refresh failure")
	}
}

func TestRetryWithRefreshedTokenKeepsHeaders(t *testing.T) {
	setupTokenTest(t, true)
	var seen []string
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			jsonResponse(w, 200, map[string]interface{}{"access_token": "fresh", "expires_in": 3600})
			return
		}
		body, _ := io.ReadAll(r.Body)
		seen = append(seen, r.Header.Get("Authorization")+"|"+r.Header.Get("Accept")+"|"+string(body))
		if r.Header.Get("Authorization") != "Bearer fresh" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	t.Setenv(remote.CLI_REMOTE_URL, server.URL)
	target := remote.GetActiveRemote()
	SaveAccessToken(target, &AccessToken{TokenEndpoint: server.URL + "/token", AccessToken: "stale", RefreshToken: "rt", ExpiresAt: time.Now().Add(time.Hour).Unix()})

	set := flag.NewFlagSet("test", flag.ContinueOnError)
	set.String("cred-file", "", "")
	c := cli.NewContext(cli.NewApp(), set, nil)
	req := CreateRequest(c, "POST", "app/list", strings.NewReader("payload"))
	req.Header.Set("Accept", "text/event-stream")
	res, err := SendRequestCustom(c, req, "", 1)
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("got %v %v", res, err)
	}
	want := []string{"Bearer stale|text/event-stream|payload", "Bearer fresh|text/event-stream|payload"}
	if strings.Join(seen, "\n") != strings.Join(want, "\n") {
		t.Fatalf("requests:\n%s", strings.Join(seen, "\n"))
	}
}

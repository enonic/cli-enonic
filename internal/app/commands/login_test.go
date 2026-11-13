package commands

import (
	"cli-enonic/internal/app/commands/common"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"runtime"
	"testing"
)

// fakeAuthorizationServer redirects an authorization request to its redirect URI with a code, and gives a
// token for that code and the PKCE verifier
func fakeAuthorizationServer(t *testing.T) *httptest.Server {
	var challenge string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		switch r.URL.Path {
		case "/authorize":
			challenge = r.Form.Get("code_challenge")
			http.Redirect(w, r, r.Form.Get("redirect_uri")+"?"+url.Values{"code": {"c1"}, "state": {r.Form.Get("state")}}.Encode(), http.StatusFound)
		case "/token":
			sum := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
			w.Header().Set("Content-Type", "application/json")
			if r.Form.Get("code") != "c1" || base64.RawURLEncoding.EncodeToString(sum[:]) != challenge {
				w.WriteHeader(400)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"})
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "at", "expires_in": 60})
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// A browser on another device lands on an error page; the user pastes its address, after a wrong line.
func TestBrowserLoginCompletedByPaste(t *testing.T) {
	server := fakeAuthorizationServer(t)
	reader, writer := io.Pipe()
	defer writer.Close()
	browserAvailable = func() bool { return true }
	openBrowser = func(u string) error {
		go func() {
			noFollow := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
			res, err := noFollow.Get(u)
			if err != nil {
				writer.CloseWithError(err)
				return
			}
			res.Body.Close()
			io.WriteString(writer, "not the address\n"+res.Header.Get("Location")+"\n")
		}()
		return nil
	}
	pasteInput = reader
	t.Cleanup(func() {
		browserAvailable, openBrowser, pasteInput = detectBrowser, openDefaultBrowser, os.Stdin
	})

	token, err := browserLogin(http.DefaultClient, &common.OAuthEndpoints{Authorization: server.URL + "/authorize", Token: server.URL + "/token"})
	if err != nil || token.AccessToken != "at" {
		t.Fatalf("got %+v %v", token, err)
	}
}

func TestBrowserAvailable(t *testing.T) {
	for _, name := range []string{"SSH_CONNECTION", "SSH_CLIENT", "SSH_TTY", "DISPLAY", "WAYLAND_DISPLAY"} {
		t.Setenv(name, "")
	}
	desktop := runtime.GOOS == "windows" || runtime.GOOS == "darwin"
	if got := detectBrowser(); got != desktop {
		t.Fatalf("no display: got %v", got)
	}
	t.Setenv("DISPLAY", ":0")
	if !detectBrowser() {
		t.Fatal("with a display: no browser")
	}
	t.Setenv("SSH_CONNECTION", "10.0.0.1 22 10.0.0.2 22")
	if detectBrowser() {
		t.Fatal("over SSH: a browser")
	}
}

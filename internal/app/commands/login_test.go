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
	"strings"
	"testing"
	"time"
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

// An opener that runs the browser in the foreground must not hold up the login: the callback is taken and the
// code exchanged while the browser is still running.
func TestBrowserLoginWhileTheBrowserRuns(t *testing.T) {
	server := fakeAuthorizationServer(t)
	stop := make(chan struct{})
	defer close(stop)
	browserAvailable = func() bool { return true }
	browserStartWait = 10 * time.Millisecond
	openBrowser = func(u string) error {
		go http.Get(u) // the browser goes through authorization and back to the CLI
		<-stop         // and keeps running
		return nil
	}
	pasteInput = strings.NewReader("")
	t.Cleanup(func() {
		browserAvailable, openBrowser, pasteInput, browserStartWait = detectBrowser, openDefaultBrowser, os.Stdin, time.Second
	})

	done := make(chan error, 1)
	go func() {
		token, err := browserLogin(http.DefaultClient, &common.OAuthEndpoints{Authorization: server.URL + "/authorize", Token: server.URL + "/token"})
		if err == nil && token.AccessToken != "at" {
			err = io.ErrUnexpectedEOF
		}
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("login waited for the browser to exit")
	}
}

func TestStartBrowser(t *testing.T) {
	browserStartWait = 50 * time.Millisecond
	t.Cleanup(func() { openBrowser, browserStartWait = openDefaultBrowser, time.Second })

	openBrowser = func(string) error { return io.EOF }
	if startBrowser("http://x") {
		t.Fatal("an opener that fails at once counts as opened")
	}
	block := make(chan struct{})
	defer close(block)
	openBrowser = func(string) error { <-block; return nil }
	start := time.Now()
	if !startBrowser("http://x") || time.Since(start) > time.Second {
		t.Fatal("waited for a running browser")
	}
}

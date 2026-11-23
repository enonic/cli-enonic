package commands

import (
	"bufio"
	"cli-enonic/internal/app/commands/common"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/urfave/cli"
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
	loginInput = reader
	interactive = func() bool { return true }
	t.Cleanup(func() {
		browserAvailable, openBrowser, loginInput, interactive = detectBrowser, openDefaultBrowser, os.Stdin, isTerminal
	})
	go io.WriteString(writer, "\n")

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
	loginInput = strings.NewReader("\n")
	interactive = func() bool { return true }
	t.Cleanup(func() {
		browserAvailable, openBrowser, loginInput, interactive, browserStartWait = detectBrowser, openDefaultBrowser, os.Stdin, isTerminal, time.Second
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

func TestLoginTargetArguments(t *testing.T) {
	t.Setenv("ENONIC_CLI_HOME_PATH", t.TempDir())
	app := cli.NewApp()
	app.Commands = []cli.Command{Login}
	app.Writer = io.Discard
	app.ErrWriter = io.Discard
	app.ExitErrHandler = func(*cli.Context, error) {}

	err := app.Run([]string{"enonic", "login", "--env", "prod", "stage"})
	if err == nil || !strings.Contains(err.Error(), "not both") {
		t.Fatalf("argument with --env: %v", err)
	}
	if err := app.Run([]string{"enonic", "login", "prod", "stage"}); err == nil {
		t.Fatal("expected an error for two arguments")
	}
	if _, err := os.Stat(filepath.Join(os.Getenv("ENONIC_CLI_HOME_PATH"), ".enonic", "environments.toml")); !os.IsNotExist(err) {
		t.Fatalf("environments were written: %v", err)
	}
}

// fakeDeviceServer gives a device code and answers the token request once approved() is true
func fakeDeviceServer(t *testing.T, approved func() bool) *common.OAuthEndpoints {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/device/code":
			json.NewEncoder(w).Encode(map[string]interface{}{"device_code": "dc", "user_code": "BCDF-GHJK",
				"verification_uri": "https://idp.example/device", "verification_uri_complete": "https://idp.example/device?user_code=BCDF-GHJK",
				"expires_in": 60, "interval": 1})
		case "/token":
			if !approved() {
				w.WriteHeader(400)
				json.NewEncoder(w).Encode(map[string]string{"error": "authorization_pending"})
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "at", "expires_in": 60})
		}
	}))
	t.Cleanup(server.Close)
	return &common.OAuthEndpoints{DeviceAuthorization: server.URL + "/device/code", Token: server.URL + "/token"}
}

// openedUrls records what the browser was asked to open
type openedUrls struct {
	mu   sync.Mutex
	urls []string
}

func (o *openedUrls) open(u string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.urls = append(o.urls, u)
	return nil
}

func (o *openedUrls) list() []string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]string(nil), o.urls...)
}

func promptSetup(t *testing.T, input io.Reader, browser, terminal bool) *openedUrls {
	opened := &openedUrls{}
	browserAvailable = func() bool { return browser }
	interactive = func() bool { return terminal }
	openBrowser = opened.open
	loginInput = input
	t.Cleanup(func() {
		browserAvailable, openBrowser, loginInput, interactive = detectBrowser, openDefaultBrowser, os.Stdin, isTerminal
	})
	return opened
}

// Enter opens the browser at the complete verification URL, once, and the approval there ends the login.
func TestDeviceLoginEnterOpensBrowserOnce(t *testing.T) {
	opened := promptSetup(t, strings.NewReader("\n\n"), true, true)
	endpoints := fakeDeviceServer(t, func() bool { return len(opened.list()) > 0 })

	token, err := deviceLogin(http.DefaultClient, endpoints)
	if err != nil || token.AccessToken != "at" {
		t.Fatalf("got %+v %v", token, err)
	}
	if got := opened.list(); len(got) != 1 || got[0] != "https://idp.example/device?user_code=BCDF-GHJK" {
		t.Fatalf("opened %v", got)
	}
}

// Approval on another device ends the login without anyone pressing Enter.
func TestDeviceLoginApprovedBeforeEnter(t *testing.T) {
	reader, writer := io.Pipe()
	defer writer.Close()
	opened := promptSetup(t, reader, true, true)
	endpoints := fakeDeviceServer(t, func() bool { return true })

	token, err := deviceLogin(http.DefaultClient, endpoints)
	if err != nil || token.AccessToken != "at" {
		t.Fatalf("got %+v %v", token, err)
	}
	if got := opened.list(); len(got) != 0 {
		t.Fatalf("opened %v", got)
	}
}

// Without a browser here, or without a terminal, nothing is opened, whatever is read.
func TestDeviceLoginWithoutPrompt(t *testing.T) {
	for _, c := range []struct{ browser, terminal bool }{{false, true}, {true, false}} {
		opened := promptSetup(t, strings.NewReader("\n"), c.browser, c.terminal)
		endpoints := fakeDeviceServer(t, func() bool { return true })

		token, err := deviceLogin(http.DefaultClient, endpoints)
		if err != nil || token.AccessToken != "at" {
			t.Fatalf("%+v: got %+v %v", c, token, err)
		}
		if got := opened.list(); len(got) != 0 {
			t.Fatalf("%+v: opened %v", c, got)
		}
	}
}

// Without a terminal an empty line opens nothing, and a pasted address still logs in.
func TestBrowserLoginWithoutTerminalPasteOnly(t *testing.T) {
	server := fakeAuthorizationServer(t)
	reader, writer := io.Pipe()
	defer writer.Close()
	opened := promptSetup(t, reader, true, false)
	urls := printedUrls(t)

	done := make(chan error, 1)
	go func() {
		token, err := browserLogin(http.DefaultClient, &common.OAuthEndpoints{Authorization: server.URL + "/authorize", Token: server.URL + "/token"})
		if err == nil && token.AccessToken != "at" {
			err = io.ErrUnexpectedEOF
		}
		done <- err
	}()

	io.WriteString(writer, "\n")
	var authorizationUrl string
	select {
	case authorizationUrl = <-urls:
	case <-time.After(5 * time.Second):
		t.Fatal("no URL printed")
	}
	noFollow := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := noFollow.Get(authorizationUrl)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	io.WriteString(writer, res.Header.Get("Location")+"\n")

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("login did not finish")
	}
	if got := opened.list(); len(got) != 0 {
		t.Fatalf("opened %v", got)
	}
}

// printedUrls sends each indented URL written to stderr, the way the login shows the address to open
func printedUrls(t *testing.T) <-chan string {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stderr := os.Stderr
	os.Stderr = writer
	t.Cleanup(func() {
		os.Stderr = stderr
		writer.Close()
	})
	urls := make(chan string, 4)
	go func() {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			if line := strings.TrimSpace(scanner.Text()); strings.HasPrefix(line, "http") {
				urls <- line
			}
		}
	}()
	return urls
}

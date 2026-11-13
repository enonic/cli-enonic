package common

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"html"
	"net"
	"net/http"
	"net/url"
	"time"
)

const AUTHORIZATION_CODE_GRANT_TYPE = "authorization_code"

// how long browser login waits for the browser to come back
var browserLoginTimeout = 5 * time.Minute

// BrowserLogin is an authorization request with PKCE (RFC 7636) waiting for the browser to be redirected to
// a one-time server on the loopback interface (RFC 8252)
type BrowserLogin struct {
	// AuthorizationUrl is the page to open in the browser
	AuthorizationUrl string

	endpoints   *OAuthEndpoints
	redirectUri string
	verifier    string
	state       string
	listener    net.Listener
	server      *http.Server
	result      chan callbackResult
}

type callbackResult struct {
	code string
	err  error
}

func randomString() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// StartBrowserLogin listens on a free loopback port for the redirect and builds the authorization URL
func StartBrowserLogin(endpoints *OAuthEndpoints) (*BrowserLogin, error) {
	if endpoints.Authorization == "" {
		return nil, errors.New("the authorization server does not offer browser login")
	}
	verifier, err := randomString()
	if err != nil {
		return nil, err
	}
	state, err := randomString()
	if err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("could not listen for the browser on 127.0.0.1: %v", err)
	}
	login := &BrowserLogin{
		endpoints:   endpoints,
		redirectUri: fmt.Sprintf("http://127.0.0.1:%d/callback", listener.Addr().(*net.TCPAddr).Port),
		verifier:    verifier,
		state:       state,
		listener:    listener,
		result:      make(chan callbackResult, 1),
	}

	authorizationUrl, err := url.Parse(endpoints.Authorization)
	if err != nil {
		listener.Close()
		return nil, err
	}
	query := authorizationUrl.Query()
	query.Set("response_type", "code")
	query.Set("client_id", CLI_CLIENT_ID)
	query.Set("redirect_uri", login.redirectUri)
	query.Set("code_challenge", pkceChallenge(verifier))
	query.Set("code_challenge_method", "S256")
	query.Set("state", state)
	if endpoints.Resource != "" {
		query.Set("resource", endpoints.Resource)
	}
	authorizationUrl.RawQuery = query.Encode()
	login.AuthorizationUrl = authorizationUrl.String()

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", login.callback)
	login.server = &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go login.server.Serve(listener)
	return login, nil
}

func (l *BrowserLogin) callback(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	var result callbackResult
	switch {
	case query.Get("state") != l.state:
		// not the answer to this login, e.g. a stale tab; keep waiting
		writeCallbackPage(w, http.StatusBadRequest, "This page does not belong to the login in progress. Start again with 'enonic login --browser'.")
		return
	case query.Get("error") != "":
		result.err = (&oauthError{Error: query.Get("error"), ErrorDescription: query.Get("error_description")}).asError()
		writeCallbackPage(w, http.StatusOK, "Login failed: "+result.err.Error()+". You can close this window.")
	case query.Get("code") == "":
		result.err = errors.New("no code in the redirect")
		writeCallbackPage(w, http.StatusBadRequest, "Login failed: no code. You can close this window.")
	default:
		result.code = query.Get("code")
		writeCallbackPage(w, http.StatusOK, "You are logged in to Enonic CLI. You can close this window.")
	}
	select {
	case l.result <- result:
	default:
	}
}

func writeCallbackPage(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	fmt.Fprintf(w, "<!doctype html><html><head><meta charset=\"utf-8\"><title>Enonic CLI</title></head>"+
		"<body style=\"font-family:sans-serif;margin:3em\"><p>%s</p></body></html>", html.EscapeString(message))
}

// Wait waits for the browser to be redirected back, then exchanges the code for tokens
func (l *BrowserLogin) Wait(client *http.Client) (*AccessToken, error) {
	defer l.Close()
	var result callbackResult
	select {
	case result = <-l.result:
	case <-time.After(browserLoginTimeout):
		return nil, errors.New("timed out waiting for the browser")
	}
	if result.err != nil {
		return nil, result.err
	}

	form := url.Values{
		"grant_type":    {AUTHORIZATION_CODE_GRANT_TYPE},
		"code":          {result.code},
		"client_id":     {CLI_CLIENT_ID},
		"redirect_uri":  {l.redirectUri},
		"code_verifier": {l.verifier},
	}
	if l.endpoints.Resource != "" {
		form.Set("resource", l.endpoints.Resource)
	}
	var res tokenResponse
	if err := postForm(client, l.endpoints.Token, form, &res); err != nil {
		return nil, err
	}
	if res.Error != "" {
		return nil, res.asError()
	}
	return newAccessToken(l.endpoints, &res)
}

// Close stops listening for the browser
func (l *BrowserLogin) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	l.server.Shutdown(ctx)
}

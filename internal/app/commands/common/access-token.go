package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// XP registers the Enonic CLI under this client id with the system ID provider (XP 8.2+)
const CLI_CLIENT_ID = "enonic-cli"
const DEVICE_CODE_GRANT_TYPE = "urn:ietf:params:oauth:grant-type:device_code"

// Access token obtained through device login, kept per management endpoint
type AccessToken struct {
	WebUrl       string `json:"webUrl"`
	IdProvider   string `json:"idProvider"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken,omitempty"`
	ExpiresAt    int64  `json:"expiresAt"`
}

func (t *AccessToken) isExpired() bool {
	return time.Now().Add(time.Minute).Unix() > t.ExpiresAt
}

type DeviceCode struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationUri         string `json:"verification_uri"`
	VerificationUriComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type oauthError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func (e *oauthError) asError() error {
	if e.ErrorDescription != "" {
		return fmt.Errorf("%s: %s", e.Error, e.ErrorDescription)
	}
	return errors.New(e.Error)
}

type tokenResponse struct {
	oauthError
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

var oauthClient = &http.Client{Timeout: 30 * time.Second}

// RFC 8628 default when the server gives no interval
var defaultPollInterval = 5 * time.Second

// DefaultWebUrl guesses the web endpoint from the management endpoint: same host, web port
func DefaultWebUrl(mgmtUrl *url.URL) string {
	return fmt.Sprintf("%s://%s:%d", mgmtUrl.Scheme, mgmtUrl.Hostname(), HTTP_PORT)
}

func idProviderEndpoint(webUrl, idProvider, function string) string {
	return fmt.Sprintf("%s/_/idprovider/%s/%s", strings.TrimRight(webUrl, "/"), url.PathEscape(idProvider), function)
}

func postForm(endpoint string, form url.Values, target interface{}) (int, error) {
	res, err := oauthClient.PostForm(endpoint, form)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	if !strings.HasPrefix(res.Header.Get("Content-Type"), "application/json") {
		return res.StatusCode, fmt.Errorf("unexpected response from %s: %s", endpoint, res.Status)
	}
	return res.StatusCode, json.NewDecoder(res.Body).Decode(target)
}

func StartDeviceLogin(webUrl, idProvider string) (*DeviceCode, error) {
	var res struct {
		oauthError
		DeviceCode
	}
	form := url.Values{"client_id": {CLI_CLIENT_ID}}
	if _, err := postForm(idProviderEndpoint(webUrl, idProvider, "device/code"), form, &res); err != nil {
		return nil, err
	}
	if res.Error != "" {
		return nil, res.asError()
	}
	return &res.DeviceCode, nil
}

// PollDeviceLogin polls the token endpoint until the user approves or denies, or the device code expires
func PollDeviceLogin(webUrl, idProvider string, code *DeviceCode) (*AccessToken, error) {
	interval := time.Duration(code.Interval) * time.Second
	if interval <= 0 {
		interval = defaultPollInterval
	}
	deadline := time.Now().Add(time.Duration(code.ExpiresIn) * time.Second)
	form := url.Values{
		"grant_type":  {DEVICE_CODE_GRANT_TYPE},
		"device_code": {code.DeviceCode},
		"client_id":   {CLI_CLIENT_ID},
	}
	for time.Now().Before(deadline) {
		time.Sleep(interval)

		var res tokenResponse
		if _, err := postForm(idProviderEndpoint(webUrl, idProvider, "token"), form, &res); err != nil {
			return nil, err
		}
		switch res.Error {
		case "":
			return newAccessToken(webUrl, idProvider, &res), nil
		case "authorization_pending":
		case "slow_down":
			interval += defaultPollInterval
		default:
			return nil, res.asError()
		}
	}
	return nil, errors.New("device code expired")
}

func refreshAccessToken(token *AccessToken) (*AccessToken, error) {
	if token.RefreshToken == "" {
		return nil, errors.New("access token expired")
	}
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {token.RefreshToken},
		"client_id":     {CLI_CLIENT_ID},
	}
	var res tokenResponse
	if _, err := postForm(idProviderEndpoint(token.WebUrl, token.IdProvider, "token"), form, &res); err != nil {
		return nil, err
	}
	if res.Error != "" {
		return nil, res.asError()
	}
	refreshed := newAccessToken(token.WebUrl, token.IdProvider, &res)
	if refreshed.RefreshToken == "" {
		// the refresh token is returned unchanged for a while after it was issued
		refreshed.RefreshToken = token.RefreshToken
	}
	return refreshed, nil
}

func newAccessToken(webUrl, idProvider string, res *tokenResponse) *AccessToken {
	return &AccessToken{
		WebUrl:       webUrl,
		IdProvider:   idProvider,
		AccessToken:  res.AccessToken,
		RefreshToken: res.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(res.ExpiresIn) * time.Second).Unix(),
	}
}

// Tokens file maps management endpoint url to its access token

func tokensFile() string {
	return GetInEnonicDir("tokens.json")
}

func readTokens() map[string]*AccessToken {
	tokens := make(map[string]*AccessToken)
	if data, err := os.ReadFile(tokensFile()); err == nil {
		if err := json.Unmarshal(data, &tokens); err != nil {
			fmt.Fprintf(os.Stderr, "Ignoring unreadable %s: %v\n", tokensFile(), err)
		}
	}
	return tokens
}

func writeTokens(tokens map[string]*AccessToken) error {
	data, err := json.MarshalIndent(tokens, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(tokensFile()), 0700); err != nil {
		return err
	}
	return os.WriteFile(tokensFile(), data, 0600)
}

func mgmtKey(mgmtUrl *url.URL) string {
	return fmt.Sprintf("%s://%s%s", mgmtUrl.Scheme, mgmtUrl.Host, strings.TrimRight(mgmtUrl.Path, "/"))
}

func SaveAccessToken(mgmtUrl *url.URL, token *AccessToken) error {
	tokens := readTokens()
	tokens[mgmtKey(mgmtUrl)] = token
	return writeTokens(tokens)
}

// RemoveAccessToken returns false if there was no token for the management endpoint
func RemoveAccessToken(mgmtUrl *url.URL) (bool, error) {
	tokens := readTokens()
	key := mgmtKey(mgmtUrl)
	if _, ok := tokens[key]; !ok {
		return false, nil
	}
	delete(tokens, key)
	return true, writeTokens(tokens)
}

// getAccessToken returns a valid access token for the management endpoint, refreshing it if needed,
// or empty string if there is no login
func getAccessToken(mgmtUrl *url.URL) string {
	tokens := readTokens()
	key := mgmtKey(mgmtUrl)
	token, ok := tokens[key]
	if !ok {
		return ""
	}
	if token.isExpired() {
		return renewAccessToken(tokens, key, token)
	}
	return token.AccessToken
}

// forceRefreshAccessToken refreshes the access token even if it has not expired, for one that was refused,
// e.g. because XP restarted with a new in-memory signing key. Returns empty string if there is no login or
// the refresh fails
func forceRefreshAccessToken(mgmtUrl *url.URL) string {
	tokens := readTokens()
	key := mgmtKey(mgmtUrl)
	token, ok := tokens[key]
	if !ok {
		return ""
	}
	return renewAccessToken(tokens, key, token)
}

func renewAccessToken(tokens map[string]*AccessToken, key string, token *AccessToken) string {
	refreshed, err := refreshAccessToken(token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Login to %s has expired (%v). Run 'enonic login' to log in again.\n", key, err)
		delete(tokens, key)
		writeTokens(tokens)
		return ""
	}
	tokens[key] = refreshed
	if err := writeTokens(tokens); err != nil {
		fmt.Fprintf(os.Stderr, "Could not save refreshed access token: %v\n", err)
	}
	return refreshed.AccessToken
}

package common

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli"
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

// UserLogin is the login of the user the access token was issued for, read from its subject
// (user:<id provider>:<login>), or empty string if it cannot be read
func (t *AccessToken) UserLogin() string {
	parts := strings.Split(t.AccessToken, ".")
	if len(parts) != 3 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims struct {
		Sub string `json:"sub"`
	}
	if json.Unmarshal(payload, &claims) != nil {
		return ""
	}
	if subject := strings.SplitN(claims.Sub, ":", 3); len(subject) == 3 {
		return subject[2]
	}
	return claims.Sub
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

// RFC 8628 default when the server gives no interval
var defaultPollInterval = 5 * time.Second

// DefaultWebUrl guesses the web endpoint from the management endpoint: same host, web port
func DefaultWebUrl(mgmtUrl *url.URL) string {
	return fmt.Sprintf("%s://%s:%d", mgmtUrl.Scheme, mgmtUrl.Hostname(), HTTP_PORT)
}

func idProviderEndpoint(webUrl, idProvider, function string) string {
	return fmt.Sprintf("%s/_/idprovider/%s/%s", strings.TrimRight(webUrl, "/"), url.PathEscape(idProvider), function)
}

// OAuthClient is the client for the ID provider's OAuth endpoints, with the same proxy and client
// certificate settings as requests to the management endpoint
func OAuthClient(c *cli.Context) *http.Client {
	client := &http.Client{Timeout: 30 * time.Second}
	if transport := newHttpTransport(c); transport != nil {
		client.Transport = transport
	}
	return client
}

// postForm decodes a JSON response into target. A response other than 2xx is an error unless it
// carries an OAuth error, which the caller handles
func postForm(client *http.Client, endpoint string, form url.Values, target interface{}) error {
	res, err := client.PostForm(endpoint, form)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	unexpected := fmt.Errorf("unexpected response from %s: %s", endpoint, res.Status)
	if !strings.HasPrefix(res.Header.Get("Content-Type"), "application/json") {
		return unexpected
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		var oauthErr oauthError
		if json.Unmarshal(body, &oauthErr) != nil || oauthErr.Error == "" {
			return unexpected
		}
	}
	return json.NewDecoder(bytes.NewReader(body)).Decode(target)
}

func StartDeviceLogin(client *http.Client, webUrl, idProvider string) (*DeviceCode, error) {
	var res struct {
		oauthError
		DeviceCode
	}
	form := url.Values{"client_id": {CLI_CLIENT_ID}}
	if err := postForm(client, idProviderEndpoint(webUrl, idProvider, "device/code"), form, &res); err != nil {
		return nil, err
	}
	if res.Error != "" {
		return nil, res.asError()
	}
	if res.DeviceCode.DeviceCode == "" {
		return nil, errors.New("no device code in response")
	}
	return &res.DeviceCode, nil
}

// PollDeviceLogin polls the token endpoint until the user approves or denies, or the device code expires
func PollDeviceLogin(client *http.Client, webUrl, idProvider string, code *DeviceCode) (*AccessToken, error) {
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
		if err := postForm(client, idProviderEndpoint(webUrl, idProvider, "token"), form, &res); err != nil {
			return nil, err
		}
		switch res.Error {
		case "":
			return newAccessToken(webUrl, idProvider, &res)
		case "authorization_pending":
		case "slow_down":
			interval += defaultPollInterval
		default:
			return nil, res.asError()
		}
	}
	return nil, errors.New("device code expired")
}

func refreshAccessToken(client *http.Client, token *AccessToken) (*AccessToken, error) {
	if token.RefreshToken == "" {
		return nil, errors.New("access token expired")
	}
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {token.RefreshToken},
		"client_id":     {CLI_CLIENT_ID},
	}
	var res tokenResponse
	if err := postForm(client, idProviderEndpoint(token.WebUrl, token.IdProvider, "token"), form, &res); err != nil {
		return nil, err
	}
	if res.Error != "" {
		return nil, res.asError()
	}
	refreshed, err := newAccessToken(token.WebUrl, token.IdProvider, &res)
	if err != nil {
		return nil, err
	}
	if refreshed.RefreshToken == "" {
		// the refresh token is returned unchanged for a while after it was issued
		refreshed.RefreshToken = token.RefreshToken
	}
	return refreshed, nil
}

func newAccessToken(webUrl, idProvider string, res *tokenResponse) (*AccessToken, error) {
	if res.AccessToken == "" {
		return nil, errors.New("no access token in response")
	}
	return &AccessToken{
		WebUrl:       webUrl,
		IdProvider:   idProvider,
		AccessToken:  res.AccessToken,
		RefreshToken: res.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(res.ExpiresIn) * time.Second).Unix(),
	}, nil
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
	dir := filepath.Dir(tokensFile())
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	// replace the file atomically so a concurrent reader never sees partial JSON
	tmp, err := os.CreateTemp(dir, "tokens-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), tokensFile())
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
func getAccessToken(c *cli.Context, mgmtUrl *url.URL) string {
	key := mgmtKey(mgmtUrl)
	token, ok := readTokens()[key]
	if !ok {
		return ""
	}
	if token.isExpired() {
		return renewAccessToken(c, key, token)
	}
	return token.AccessToken
}

// forceRefreshAccessToken refreshes the access token even if it has not expired, for one that was refused,
// e.g. because XP restarted with a new in-memory signing key. Returns empty string if there is no login or
// the refresh fails
func forceRefreshAccessToken(c *cli.Context, mgmtUrl *url.URL) string {
	key := mgmtKey(mgmtUrl)
	token, ok := readTokens()[key]
	if !ok {
		return ""
	}
	return renewAccessToken(c, key, token)
}

// renewAccessToken refreshes the token and stores the result. The file is read again before writing, so
// only this endpoint's entry changes, and a login removed or replaced meanwhile by another command is kept
func renewAccessToken(c *cli.Context, key string, token *AccessToken) string {
	refreshed, err := refreshAccessToken(OAuthClient(c), token)

	tokens := readTokens()
	current, ok := tokens[key]
	unchanged := ok && current.RefreshToken == token.RefreshToken
	if err != nil {
		fmt.Fprintf(os.Stderr, "Login to %s has expired (%v). Run 'enonic login' to log in again.\n", key, err)
		if unchanged {
			delete(tokens, key)
			writeTokens(tokens)
		}
		return ""
	}
	if unchanged {
		tokens[key] = refreshed
		if err := writeTokens(tokens); err != nil {
			fmt.Fprintf(os.Stderr, "Could not save refreshed access token: %v\n", err)
		}
	}
	return refreshed.AccessToken
}

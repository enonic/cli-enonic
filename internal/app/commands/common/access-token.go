package common

import (
	"bytes"
	"cli-enonic/internal/app/commands/remote"
	"cli-enonic/internal/app/util"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/urfave/cli"
)

// XP registers the Enonic CLI under this client id with the system ID provider (XP 8.2+)
const CLI_CLIENT_ID = "enonic-cli"
const DEVICE_CODE_GRANT_TYPE = "urn:ietf:params:oauth:grant-type:device_code"

// OAuthEndpoints are where device login runs and where its token is refreshed
type OAuthEndpoints struct {
	// Authorization is the authorization endpoint for browser login (authorization code with PKCE), empty if
	// the server does not offer it
	Authorization string
	// DeviceAuthorization is the device authorization endpoint for device login, empty if the server does not
	// offer it
	DeviceAuthorization string
	Token               string
	// Resource is the management endpoint the token is asked for (RFC 8707), empty for the ID provider's
	// default audience
	Resource string
}

// Access token obtained through device login, kept per management endpoint
type AccessToken struct {
	TokenEndpoint string `json:"tokenEndpoint"`
	Resource      string `json:"resource,omitempty"`
	AccessToken   string `json:"accessToken"`
	RefreshToken  string `json:"refreshToken,omitempty"`
	ExpiresAt     int64  `json:"expiresAt"`
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

func StartDeviceLogin(client *http.Client, endpoints *OAuthEndpoints) (*DeviceCode, error) {
	var res struct {
		oauthError
		DeviceCode
	}
	form := url.Values{"client_id": {CLI_CLIENT_ID}}
	if endpoints.Resource != "" {
		form.Set("resource", endpoints.Resource)
	}
	if err := postForm(client, endpoints.DeviceAuthorization, form, &res); err != nil {
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
func PollDeviceLogin(client *http.Client, endpoints *OAuthEndpoints, code *DeviceCode) (*AccessToken, error) {
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
	if endpoints.Resource != "" {
		form.Set("resource", endpoints.Resource)
	}
	for time.Now().Before(deadline) {
		time.Sleep(interval)

		var res tokenResponse
		if err := postForm(client, endpoints.Token, form, &res); err != nil {
			return nil, err
		}
		switch res.Error {
		case "":
			return newAccessToken(endpoints, &res)
		case "authorization_pending":
		case "slow_down":
			interval += defaultPollInterval
		default:
			return nil, res.asError()
		}
	}
	return nil, errors.New("device code expired")
}

// loginRejectedError is a refresh the authorization server refused, so the login cannot be used any more,
// unlike a failure to reach it
type loginRejectedError struct {
	err error
}

func (e *loginRejectedError) Error() string {
	return e.err.Error()
}

func (e *loginRejectedError) Unwrap() error {
	return e.err
}

// exit ends the command, swapped in tests
var exit = os.Exit

func refreshAccessToken(client *http.Client, token *AccessToken) (*AccessToken, error) {
	if token.RefreshToken == "" {
		return nil, &loginRejectedError{errors.New("access token expired")}
	}
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {token.RefreshToken},
		"client_id":     {CLI_CLIENT_ID},
	}
	if token.Resource != "" {
		form.Set("resource", token.Resource)
	}
	var res tokenResponse
	if err := postForm(client, token.TokenEndpoint, form, &res); err != nil {
		return nil, err
	}
	if res.Error != "" {
		return nil, &loginRejectedError{res.asError()}
	}
	refreshed, err := newAccessToken(&OAuthEndpoints{Token: token.TokenEndpoint, Resource: token.Resource}, &res)
	if err != nil {
		return nil, err
	}
	if refreshed.RefreshToken == "" {
		// the refresh token is returned unchanged for a while after it was issued
		refreshed.RefreshToken = token.RefreshToken
	}
	return refreshed, nil
}

func newAccessToken(endpoints *OAuthEndpoints, res *tokenResponse) (*AccessToken, error) {
	if res.AccessToken == "" {
		return nil, errors.New("no access token in response")
	}
	return &AccessToken{
		TokenEndpoint: endpoints.Token,
		Resource:      endpoints.Resource,
		AccessToken:   res.AccessToken,
		RefreshToken:  res.RefreshToken,
		ExpiresAt:     time.Now().Add(time.Duration(res.ExpiresIn) * time.Second).Unix(),
	}, nil
}

// Each login is kept in its own file under tokens/: <environment>.json, or url-<host>-<port>.json for a
// one-off ENONIC_CLI_REMOTE_URL

var unsafeHostChars = regexp.MustCompile(`[^a-z0-9.-]`)

func tokenFile(target *remote.RemoteData) (string, error) {
	if target.Name != "" {
		if !remote.ValidEnvName(target.Name) {
			return "", fmt.Errorf("invalid environment name '%s'", target.Name)
		}
		return util.GetInEnonicDir("tokens", target.Name+".json"), nil
	}
	host := unsafeHostChars.ReplaceAllString(strings.ToLower(target.Url.Hostname()), "_")
	return util.GetInEnonicDir("tokens", fmt.Sprintf("%s%s-%s.json", remote.ONE_OFF_TOKEN_PREFIX, host, effectivePort(&target.Url.URL))), nil
}

// readToken returns the token stored in the file, or nil if there is none or it cannot be read
func readToken(path string) *AccessToken {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var token AccessToken
	if err := json.Unmarshal(data, &token); err != nil {
		fmt.Fprintf(os.Stderr, "Ignoring unreadable %s: %v\n", path, err)
		return nil
	}
	return &token
}

func writeToken(path string, token *AccessToken) error {
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	return util.WriteFileAtomic(path, data, 0600)
}

// mgmtKey is the management endpoint's resource identifier the way XP builds it: the origin without a default
// port, and the path without a trailing slash, empty for the root
func mgmtKey(mgmtUrl *url.URL) string {
	scheme := strings.ToLower(mgmtUrl.Scheme)
	host := mgmtUrl.Host
	if port := mgmtUrl.Port(); (scheme == "https" && port == "443") || (scheme == "http" && port == "80") {
		host = strings.TrimSuffix(host, ":"+port)
	}
	return fmt.Sprintf("%s://%s%s", scheme, host, strings.TrimRight(mgmtUrl.Path, "/"))
}

// TargetLabel names the target in messages: "<name> (<url>)", or the url for a one-off url
func TargetLabel(target *remote.RemoteData) string {
	if target.Name == "" {
		return target.Url.String()
	}
	return fmt.Sprintf("%s (%s)", target.Name, target.Url.String())
}

func SaveAccessToken(target *remote.RemoteData, token *AccessToken) error {
	path, err := tokenFile(target)
	if err != nil {
		return err
	}
	if token.Resource == "" {
		token.Resource = mgmtKey(&target.Url.URL)
	}
	return writeToken(path, token)
}

// RemoveAccessToken returns false if there was no login for the target
func RemoveAccessToken(target *remote.RemoteData) (bool, error) {
	path, err := tokenFile(target)
	if err != nil {
		return false, err
	}
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// HasAccessToken reports whether a login is stored for the target and its current url
func HasAccessToken(target *remote.RemoteData) bool {
	_, token := storedToken(target)
	return token != nil
}

// storedToken is the login stored for the target, or nil if there is none or it was made for another url,
// e.g. after the environment's url was changed
func storedToken(target *remote.RemoteData) (string, *AccessToken) {
	path, err := tokenFile(target)
	if err != nil {
		return "", nil
	}
	token := readToken(path)
	if token == nil || token.Resource != mgmtKey(&target.Url.URL) {
		return path, nil
	}
	return path, token
}

// getAccessToken returns a valid access token for the target, refreshing it if needed, or empty string if
// there is no login or beta is off
func getAccessToken(c *cli.Context, target *remote.RemoteData) string {
	if !util.IsBeta() {
		return ""
	}
	path, token := storedToken(target)
	if token == nil {
		return ""
	}
	if token.isExpired() {
		return renewAccessToken(c, path, token)
	}
	return token.AccessToken
}

// forceRefreshAccessToken refreshes the access token even if it has not expired, for one that was refused,
// e.g. because XP restarted with a new in-memory signing key. Returns empty string if there is no login or
// the refresh fails
func forceRefreshAccessToken(c *cli.Context, target *remote.RemoteData) string {
	if !util.IsBeta() {
		return ""
	}
	path, token := storedToken(target)
	if token == nil {
		return ""
	}
	return renewAccessToken(c, path, token)
}

// renewAccessToken refreshes the token and stores the result. The file is read again before writing, so a
// login removed or replaced meanwhile by another command is kept
func renewAccessToken(c *cli.Context, path string, token *AccessToken) string {
	refreshed, err := refreshAccessToken(OAuthClient(c), token)

	current := readToken(path)
	unchanged := current != nil && current.RefreshToken == token.RefreshToken
	var rejected *loginRejectedError
	if errors.As(err, &rejected) {
		fmt.Fprintf(os.Stderr, "Login to %s has expired (%v). Run 'enonic login' to log in again.\n", token.Resource, err)
		if unchanged {
			os.Remove(path)
		}
		return ""
	} else if err != nil {
		// the login may still be good once the authorization server is reachable, so keep it, and don't fall
		// back to password authentication
		fmt.Fprintf(os.Stderr, "Could not refresh the login to %s: %v\n", token.Resource, err)
		exit(1)
		return ""
	}
	if unchanged {
		if err := writeToken(path, refreshed); err != nil {
			fmt.Fprintf(os.Stderr, "Could not save refreshed access token: %v\n", err)
		}
	}
	return refreshed.AccessToken
}

package auth

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Flow is the object passed around while doing the Device Authorization Flow
type Flow struct {
	URI          string
	UserCode     string
	deviceCode   string
	pollInterval time.Duration
	expiresAt    time.Time
}

// The response from Auth0 when starting a Device Authorization Flow
type deviceCodeRequest struct {
	Error                   string `json:"error"`
	ErrorDescription        string `json:"error_description"`
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// The reponse from Auth0 when requesting tokens
type tokenRequest struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	IDToken          string `json:"id_token"`
	TokenType        string `json:"token_type"`
	ExpiresIn        int    `json:"expires_in"`
}

type tokenPayload struct {
	ExpiresAt int64 `json:"exp"`
}

// The final result after going through the Authorization flow
type tokens struct {
	AccessToken  string
	IDToken      string
	RefreshToken string
	ExpiresAt    int64
}

// Tells you if the access token is expired or not
func (t tokens) isExpired() bool {
	return time.Now().Add(time.Minute*5).Unix() > t.ExpiresAt
}

// Login to Auth0 using the Device Authorization Flow
func Login(instructions func(flow *Flow), afterTokenFetch func(int64)) error {
	flow, err := oAuthStartVerificationFlow()
	if err != nil {
		return err
	}

	instructions(flow)
	tokens, err := oAuthGetTokens(flow)
	afterTokenFetch(tokens.ExpiresAt)
	if err != nil {
		return err
	}

	err = saveTokens(tokens)
	if err != nil {
		return err
	}
	return nil
}

// Flow functions

func oAuthStartVerificationFlow() (*Flow, error) {
	// Do the request
	var res deviceCodeRequest
	payload := strings.NewReader("client_id=" + clientID + "&scope=" + urlEncode(scope) + "&audience=" + urlEncode(audience))
	if err := doRequest("/oauth/device/code", payload, &res); err != nil {
		return nil, err
	}

	// Check for errors
	if res.Error != "" {
		return nil, fmt.Errorf("device code request returned error: %s", res.ErrorDescription)
	}

	// Return authflow
	return &Flow{
		URI:          res.VerificationURIComplete,
		UserCode:     res.UserCode,
		deviceCode:   res.DeviceCode,
		pollInterval: time.Duration(res.Interval) * time.Second,
		expiresAt:    addSecondsToNow(res.ExpiresIn),
	}, nil
}

func oAuthGetTokens(flow *Flow) (*tokens, error) {
	// Poll for tokens
	for time.Now().Before(flow.expiresAt) {
		// Wait for "poll interval" amount of time
		time.Sleep(time.Duration(flow.pollInterval))

		// Do request
		var res tokenRequest
		payload := strings.NewReader("grant_type=" + urlEncode("urn:ietf:params:oauth:grant-type:device_code") + "&device_code=" + flow.deviceCode + "&client_id=" + clientID)
		if err := doRequest("/oauth/token", payload, &res); err != nil {
			return nil, err
		}
		if res.Error != "" {
			switch res.Error {
			case "authorization_pending":
				continue
			case "slow_down":
				flow.pollInterval = flow.pollInterval * 2
				continue
			default:
				return nil, fmt.Errorf("token request returned error: %s", res.ErrorDescription)
			}
		} else {
			exp, err := parseExpiredTime(res.AccessToken)

			if err != nil {
				return nil, err
			}

			return &tokens{
				AccessToken:  res.AccessToken,
				IDToken:      res.IDToken,
				RefreshToken: res.RefreshToken,
				ExpiresAt:    exp,
			}, nil
		}
	}

	return nil, fmt.Errorf("authentication flow expired")
}

// Util functions

func doRequest(endpoint string, payload io.Reader, response interface{}) error {
	// Create request
	req, err := http.NewRequest("POST", authURL+endpoint, payload)
	if err != nil {
		return fmt.Errorf("failed creating request: %s", err)
	}
	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Do request
	res, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("http request failed: %s", err)
	}
	defer res.Body.Close()

	// Unmarshal JSON
	return json.NewDecoder(res.Body).Decode(response)
}

func urlEncode(s string) string {
	return url.QueryEscape(s)
}

func addSecondsToNow(seconds int) time.Time {
	return time.Now().Add(time.Duration(seconds) * time.Second)
}

func parseExpiredTime(token string) (int64, error) {
	split := strings.Split(token, ".")
	if len(split) != 3 {
		return 0, fmt.Errorf("recieved invalid token from identity provider")
	}

	decoded, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(split[1])
	if err != nil {
		return 0, fmt.Errorf("failed to decode jwt token: %v", err)
	}

	var payload tokenPayload
	if err := json.NewDecoder(bytes.NewReader(decoded)).Decode(&payload); err != nil {
		return 0, fmt.Errorf("failed to decode token payload: %v", err)
	}

	return payload.ExpiresAt, nil
}

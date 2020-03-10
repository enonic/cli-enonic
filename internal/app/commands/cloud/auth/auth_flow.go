package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type AuthFlow struct {
	URI          string
	UserCode     string
	deviceCode   string
	pollInterval time.Duration
	expiresAt    time.Time
}

func endpoint(p string) string {
	return authURL + p
}

func urlEncode(s string) string {
	return url.QueryEscape(s)
}

func doRequest(req *http.Request) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	res, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("http request failed: %s", err)
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading body: %s", err)
	}

	jsonMap := make(map[string]interface{})
	err = json.Unmarshal(body, &jsonMap)
	if err != nil {
		return nil, fmt.Errorf("failed marshalling body: %s", err)
	}
	return jsonMap, nil
}

func oAuthStartVerificationFlow() (*AuthFlow, error) {
	url := endpoint("/oauth/device/code")

	payload := strings.NewReader("client_id=" + clientID + "&scope=" + urlEncode(scope))
	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		return nil, fmt.Errorf("failed creating auth flow request: %s", err)
	}
	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	res, err := doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed executing auth flow request: %s", err)
	}

	if desc, ok := res["error_description"]; ok {
		return nil, fmt.Errorf("recieved error from server while doing auth flow request: %s", desc.(string))
	}

	return &AuthFlow{
		URI:          res["verification_uri_complete"].(string),
		UserCode:     res["user_code"].(string),
		deviceCode:   res["device_code"].(string),
		pollInterval: time.Duration(res["interval"].(float64)) * time.Second,
		expiresAt:    time.Now().Add(time.Duration(res["expires_in"].(float64)) * time.Second),
	}, nil
}

func oAuthGetTokens(flow *AuthFlow) (*tokens, error) {
	for time.Now().Before(flow.expiresAt) {
		time.Sleep(time.Duration(flow.pollInterval))

		url := endpoint("/oauth/token")

		payload := strings.NewReader("grant_type=" + urlEncode("urn:ietf:params:oauth:grant-type:device_code") + "&device_code=" + flow.deviceCode + "&client_id=" + clientID)
		req, err := http.NewRequest("POST", url, payload)
		if err != nil {
			return nil, fmt.Errorf("failed creating token request: %s", err)
		}
		req.Header.Add("content-type", "application/x-www-form-urlencoded")

		res, err := doRequest(req)
		if err != nil {
			return nil, fmt.Errorf("failed executing token request: %s", err)
		}

		if desc, ok := res["error"]; ok {
			descString := desc.(string)
			switch descString {
			case "authorization_pending":
				continue
			case "slow_down":
				flow.pollInterval = flow.pollInterval * 2
				continue
			case "expired_token":
				return nil, fmt.Errorf("recieved error from server while token request: %s", res["error_description"].(string))
			case "access_denied":
				return nil, fmt.Errorf("recieved error from server while token request: %s", res["error_description"].(string))
			}
		} else {
			return &tokens{
				AccessToken:  res["id_token"].(string),
				RefreshToken: res["refresh_token"].(string),
				ExpiresAt:    addExpiryToNow(res["expires_in"].(float64)),
			}, nil
		}
	}

	return nil, fmt.Errorf("authentication flow expired")
}

func oAuthRefreshTokens(t *tokens) (*tokens, error) {
	url := endpoint("/oauth/token")

	payload := strings.NewReader("grant_type=refresh_token&client_id=" + clientID + "&client_secret=" + clientSecret + "&refresh_token=" + t.RefreshToken)
	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		return nil, fmt.Errorf("failed creating refresh request: %s", err)
	}
	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	res, err := doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed executing refresh request: %s", err)
	}

	if desc, ok := res["error_description"]; ok {
		return nil, fmt.Errorf("recieved error from server while doing refresh request: %s", desc.(string))
	}

	return &tokens{
		AccessToken:  res["id_token"].(string),
		RefreshToken: t.RefreshToken,
		ExpiresAt:    addExpiryToNow(res["expires_in"].(float64)),
	}, nil
}

func addExpiryToNow(s float64) int64 {
	return time.Now().Add(time.Duration(s) * time.Second).Unix()
}

type loginInstructions func(flow *AuthFlow)

func Login(instructions loginInstructions) error {
	flow, err := oAuthStartVerificationFlow()
	if err != nil {
		return err
	}
	instructions(flow)
	tokens, err := oAuthGetTokens(flow)
	if err != nil {
		return err
	}
	err = saveTokens(tokens)
	if err != nil {
		return err
	}
	return nil
}

package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// ErrNoDiscovery means the management endpoint does not say where a token for it is issued, e.g. XP
// older than 8.2, or a management endpoint without OAuth discovery
var ErrNoDiscovery = errors.New("the management endpoint does not serve OAuth discovery")

type protectedResourceMetadata struct {
	Resource             string   `json:"resource"`
	AuthorizationServers []string `json:"authorization_servers"`
}

type authorizationServerMetadata struct {
	Issuer                      string `json:"issuer"`
	DeviceAuthorizationEndpoint string `json:"device_authorization_endpoint"`
	TokenEndpoint               string `json:"token_endpoint"`
}

// DiscoverEndpoints finds where to log in for the management endpoint: its protected resource metadata
// (RFC 9728) names the authorization server, whose metadata (RFC 8414) gives the device authorization
// and token endpoints. The token is asked for the management endpoint itself (RFC 8707)
func DiscoverEndpoints(client *http.Client, mgmtUrl *url.URL) (*OAuthEndpoints, error) {
	resource := mgmtKey(mgmtUrl)

	var prm protectedResourceMetadata
	found, err := getJson(client, wellKnownUrl(mgmtUrl, "oauth-protected-resource"), &prm)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrNoDiscovery
	}
	if prm.Resource != resource {
		return nil, fmt.Errorf("protected resource metadata is for %q, not %q", prm.Resource, resource)
	}
	if len(prm.AuthorizationServers) == 0 {
		return nil, ErrNoDiscovery
	}

	issuer := prm.AuthorizationServers[0]
	issuerUrl, err := url.Parse(issuer)
	if err != nil || issuerUrl.Host == "" {
		return nil, fmt.Errorf("invalid authorization server %q", issuer)
	}
	var asm authorizationServerMetadata
	found, err = getJson(client, wellKnownUrl(issuerUrl, "oauth-authorization-server"), &asm)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("authorization server %s serves no metadata", issuer)
	}
	if asm.Issuer != issuer {
		return nil, fmt.Errorf("authorization server metadata is for %q, not %q", asm.Issuer, issuer)
	}
	if asm.DeviceAuthorizationEndpoint == "" || asm.TokenEndpoint == "" {
		return nil, fmt.Errorf("authorization server %s does not offer device login", issuer)
	}
	return &OAuthEndpoints{
		DeviceAuthorization: asm.DeviceAuthorizationEndpoint,
		Token:               asm.TokenEndpoint,
		Resource:            resource,
	}, nil
}

// wellKnownUrl puts the well-known suffix between the host and the path (RFC 8414, RFC 9728)
func wellKnownUrl(u *url.URL, suffix string) string {
	return fmt.Sprintf("%s://%s/.well-known/%s%s", u.Scheme, u.Host, suffix, strings.TrimRight(u.Path, "/"))
}

// getJson returns false for a 404, so the caller can tell a document that is not served from a failure
func getJson(client *http.Client, endpoint string, target interface{}) (bool, error) {
	res, err := client.Get(endpoint)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if res.StatusCode != http.StatusOK || !strings.HasPrefix(res.Header.Get("Content-Type"), "application/json") {
		return false, fmt.Errorf("unexpected response from %s: %s", endpoint, res.Status)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(body, target); err != nil {
		return false, fmt.Errorf("invalid response from %s: %v", endpoint, err)
	}
	return true, nil
}

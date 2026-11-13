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
	AuthorizationEndpoint       string `json:"authorization_endpoint"`
	DeviceAuthorizationEndpoint string `json:"device_authorization_endpoint"`
	TokenEndpoint               string `json:"token_endpoint"`
}

// DiscoverEndpoints finds where to log in for the management endpoint the way an MCP client does: its
// protected resource metadata (RFC 9728), from the challenge of an unauthenticated request or else the
// well-known location, names the authorization server, whose metadata (RFC 8414 or OpenID Connect
// Discovery) gives the device authorization and token endpoints. The token is asked for the management
// endpoint itself (RFC 8707)
func DiscoverEndpoints(client *http.Client, mgmtUrl *url.URL) (*OAuthEndpoints, error) {
	resource := mgmtKey(mgmtUrl)

	metadataUrl := resourceMetadataFromChallenge(client, resource)
	if metadataUrl == "" {
		metadataUrl = wellKnownUrl(mgmtUrl, "oauth-protected-resource")
	}
	var prm protectedResourceMetadata
	found, err := getJson(client, metadataUrl, &prm)
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

	return DiscoverIssuerEndpoints(client, prm.AuthorizationServers[0], mgmtUrl)
}

// DiscoverIssuerEndpoints reads the device authorization and token endpoints from the authorization
// server's metadata, for a token asked for the management endpoint
func DiscoverIssuerEndpoints(client *http.Client, issuer string, mgmtUrl *url.URL) (*OAuthEndpoints, error) {
	issuerUrl, err := url.Parse(issuer)
	if err != nil || issuerUrl.Host == "" {
		return nil, fmt.Errorf("invalid authorization server %q", issuer)
	}
	asm, err := authorizationServerMetadataOf(client, issuerUrl)
	if err != nil {
		return nil, err
	}
	if asm.Issuer != issuer {
		return nil, fmt.Errorf("authorization server metadata is for %q, not %q", asm.Issuer, issuer)
	}
	if (asm.DeviceAuthorizationEndpoint == "" && asm.AuthorizationEndpoint == "") || asm.TokenEndpoint == "" {
		return nil, fmt.Errorf("authorization server %s does not offer device or browser login", issuer)
	}
	return &OAuthEndpoints{
		Authorization:       asm.AuthorizationEndpoint,
		DeviceAuthorization: asm.DeviceAuthorizationEndpoint,
		Token:               asm.TokenEndpoint,
		Resource:            mgmtKey(mgmtUrl),
	}, nil
}

// authorizationServerMetadataOf tries the locations MCP clients try, in their order: OAuth metadata, then
// OpenID Connect Discovery with the well-known suffix inserted before the path and appended to it
func authorizationServerMetadataOf(client *http.Client, issuer *url.URL) (*authorizationServerMetadata, error) {
	locations := []string{wellKnownUrl(issuer, "oauth-authorization-server"), wellKnownUrl(issuer, "openid-configuration")}
	if path := strings.TrimRight(issuer.Path, "/"); path != "" {
		locations = append(locations, fmt.Sprintf("%s://%s%s/.well-known/openid-configuration", issuer.Scheme, issuer.Host, path))
	}
	for _, location := range locations {
		var asm authorizationServerMetadata
		found, err := getJson(client, location, &asm)
		if err != nil {
			return nil, err
		}
		if found {
			return &asm, nil
		}
	}
	return nil, fmt.Errorf("authorization server %s serves no metadata", issuer.String())
}

// resourceMetadataFromChallenge is the protected resource metadata URL in the Bearer challenge of an
// unauthenticated request to the resource (RFC 9728 section 5.1), or empty string if there is none
func resourceMetadataFromChallenge(client *http.Client, resource string) string {
	// XP names the resource after the path requested, so request the resource itself; the root as "/"
	probe := resource
	if u, err := url.Parse(resource); err == nil && u.Path == "" {
		probe += "/"
	}
	res, err := client.Get(probe)
	if err != nil {
		return ""
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		return ""
	}
	for _, header := range res.Header.Values("WWW-Authenticate") {
		if params, ok := bearerChallengeParams(header); ok {
			return params["resource_metadata"]
		}
	}
	return ""
}

// bearerChallengeParams reads the auth-params of the Bearer challenge in a WWW-Authenticate header value,
// which can hold several challenges
func bearerChallengeParams(header string) (map[string]string, bool) {
	var params map[string]string
	rest := header
	for rest != "" {
		rest = strings.TrimLeft(rest, " ,")
		name, value, next, isParam := nextAuthItem(rest)
		rest = next
		switch {
		case !isParam && params != nil:
			return params, true
		case !isParam:
			if strings.EqualFold(name, "Bearer") {
				params = make(map[string]string)
			}
		case params != nil:
			params[strings.ToLower(name)] = value
		}
	}
	return params, params != nil
}

// nextAuthItem reads an auth-scheme or an auth-param (name=token or name="quoted string") from s
func nextAuthItem(s string) (name, value, rest string, isParam bool) {
	i := strings.IndexAny(s, " =,")
	if i < 0 {
		return s, "", "", false
	}
	name = s[:i]
	afterName := strings.TrimLeft(s[i:], " ")
	if strings.HasPrefix(afterName, "=") {
		// a token68 ends with "=" padding, an auth-param has a single "=" followed by its value
		padding := len(afterName) - len(strings.TrimLeft(afterName, "="))
		afterPadding := afterName[padding:]
		if padding > 1 || afterPadding == "" || strings.HasPrefix(afterPadding, ",") {
			return name, "", afterPadding, false
		}
	}
	if !strings.HasPrefix(afterName, "=") {
		return name, "", afterName, false
	}
	v := strings.TrimLeft(afterName[1:], " ")
	if !strings.HasPrefix(v, "\"") {
		end := strings.IndexAny(v, " ,")
		if end < 0 {
			return name, v, "", true
		}
		return name, v[:end], v[end:], true
	}
	var b strings.Builder
	for j := 1; j < len(v); j++ {
		switch v[j] {
		case '\\':
			if j+1 < len(v) {
				j++
				b.WriteByte(v[j])
			}
		case '"':
			return name, b.String(), v[j+1:], true
		default:
			b.WriteByte(v[j])
		}
	}
	return name, b.String(), "", true
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

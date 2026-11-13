package common

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type discoveryOptions struct {
	resource      func(origin string) string
	challenge     bool
	metadataPath  string
	asMetadataUrl string
}

func discoveryServer(t *testing.T, opts discoveryOptions) *httptest.Server {
	if opts.resource == nil {
		opts.resource = func(origin string) string { return origin }
	}
	if opts.metadataPath == "" {
		opts.metadataPath = "/.well-known/oauth-protected-resource"
	}
	if opts.asMetadataUrl == "" {
		opts.asMetadataUrl = "/.well-known/oauth-authorization-server/_/idprovider/enonic-cloud"
	}
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		issuer := server.URL + "/_/idprovider/enonic-cloud"
		switch r.URL.Path {
		case "/":
			if opts.challenge {
				w.Header().Add("WWW-Authenticate", `Basic realm="xp"`)
				w.Header().Add("WWW-Authenticate", `Bearer resource_metadata="`+server.URL+opts.metadataPath+`"`)
			}
			w.WriteHeader(http.StatusUnauthorized)
		case opts.metadataPath:
			jsonResponse(w, 200, map[string]interface{}{
				"resource":              opts.resource(server.URL),
				"authorization_servers": []string{issuer},
			})
		case opts.asMetadataUrl:
			jsonResponse(w, 200, map[string]interface{}{
				"issuer":                        issuer,
				"authorization_endpoint":        issuer + "/authorize",
				"device_authorization_endpoint": issuer + "/device/code",
				"token_endpoint":                issuer + "/token",
			})
		case "/_/idprovider/enonic-cloud/device/code", "/_/idprovider/enonic-cloud/token":
			r.ParseForm()
			if r.Form.Get("resource") != server.URL {
				jsonResponse(w, 400, map[string]string{"error": "invalid_target"})
				return
			}
			if strings.HasSuffix(r.URL.Path, "/token") {
				jsonResponse(w, 200, map[string]interface{}{"access_token": "at-" + r.Form.Get("grant_type"), "refresh_token": "rt", "expires_in": 0})
				return
			}
			jsonResponse(w, 200, map[string]interface{}{"device_code": "dc", "user_code": "BCDF-GHJK", "expires_in": 10})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestDiscoverEndpointsAndLogin(t *testing.T) {
	server := discoveryServer(t, discoveryOptions{})
	mgmtUrl, _ := url.Parse(server.URL + "/")

	endpoints, err := DiscoverEndpoints(http.DefaultClient, mgmtUrl)
	if err != nil {
		t.Fatal(err)
	}
	issuer := server.URL + "/_/idprovider/enonic-cloud"
	if endpoints.DeviceAuthorization != issuer+"/device/code" || endpoints.Token != issuer+"/token" || endpoints.Resource != server.URL {
		t.Fatalf("unexpected endpoints %+v", endpoints)
	}

	code, err := StartDeviceLogin(http.DefaultClient, endpoints)
	if err != nil {
		t.Fatal(err)
	}
	token, err := PollDeviceLogin(http.DefaultClient, endpoints, code)
	if err != nil {
		t.Fatal(err)
	}
	if token.AccessToken != "at-"+DEVICE_CODE_GRANT_TYPE || token.Resource != server.URL {
		t.Fatalf("unexpected token %+v", token)
	}

	refreshed, err := refreshAccessToken(http.DefaultClient, token)
	if err != nil {
		t.Fatal(err)
	}
	if refreshed.AccessToken != "at-refresh_token" || refreshed.Resource != server.URL {
		t.Fatalf("unexpected refreshed token %+v", refreshed)
	}
}

func TestDiscoverEndpointsFromChallenge(t *testing.T) {
	server := discoveryServer(t, discoveryOptions{challenge: true, metadataPath: "/metadata/elsewhere"})
	mgmtUrl, _ := url.Parse(server.URL)

	endpoints, err := DiscoverEndpoints(http.DefaultClient, mgmtUrl)
	if err != nil {
		t.Fatal(err)
	}
	if endpoints.Resource != server.URL {
		t.Fatalf("unexpected endpoints %+v", endpoints)
	}
}

func TestDiscoverEndpointsFromOpenIdConfiguration(t *testing.T) {
	server := discoveryServer(t, discoveryOptions{asMetadataUrl: "/_/idprovider/enonic-cloud/.well-known/openid-configuration"})
	mgmtUrl, _ := url.Parse(server.URL)

	endpoints, err := DiscoverEndpoints(http.DefaultClient, mgmtUrl)
	if err != nil {
		t.Fatal(err)
	}
	if endpoints.Token != server.URL+"/_/idprovider/enonic-cloud/token" {
		t.Fatalf("unexpected endpoints %+v", endpoints)
	}
}

func TestDiscoverEndpointsRejectsOtherResource(t *testing.T) {
	server := discoveryServer(t, discoveryOptions{resource: func(string) string { return "https://other.example.com" }})
	mgmtUrl, _ := url.Parse(server.URL)

	_, err := DiscoverEndpoints(http.DefaultClient, mgmtUrl)
	if err == nil || !strings.Contains(err.Error(), "https://other.example.com") {
		t.Fatalf("expected resource mismatch, got %v", err)
	}
}

func TestDiscoverEndpointsNotServed(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	mgmtUrl, _ := url.Parse(server.URL)

	if _, err := DiscoverEndpoints(http.DefaultClient, mgmtUrl); !errors.Is(err, ErrNoDiscovery) {
		t.Fatalf("expected ErrNoDiscovery, got %v", err)
	}
}

func TestBearerChallengeParams(t *testing.T) {
	cases := map[string]string{
		`Bearer resource_metadata="https://x/.well-known/oauth-protected-resource"`:                      "https://x/.well-known/oauth-protected-resource",
		`Basic realm="a, b", Bearer error="invalid_token", resource_metadata="https://x/m", scope="a b"`: "https://x/m",
		`bearer resource_metadata=https://x/m, Basic realm="xp"`:                                         "https://x/m",
		`Bearer realm="x\"y", resource_metadata="https://x/m"`:                                           "https://x/m",
		`Negotiate YQ==, Bearer resource_metadata="https://x/m"`:                                         "https://x/m",
		`Negotiate YQ=, Bearer resource_metadata="https://x/m"`:                                          "https://x/m",
		`Negotiate YQ==` + "," + ` Bearer resource_metadata=https://x/m`:                                 "https://x/m",
	}
	for header, want := range cases {
		params, ok := bearerChallengeParams(header)
		if !ok || params["resource_metadata"] != want {
			t.Errorf("%s: got %v %v", header, params, ok)
		}
	}
	if _, ok := bearerChallengeParams(`Basic realm="xp"`); ok {
		t.Error("found a Bearer challenge in a Basic one")
	}
}

func TestWellKnownUrl(t *testing.T) {
	u, _ := url.Parse("https://admin.example.com/_/idprovider/system/")
	if got := wellKnownUrl(u, "oauth-authorization-server"); got != "https://admin.example.com/.well-known/oauth-authorization-server/_/idprovider/system" {
		t.Fatalf("unexpected url %q", got)
	}
}

func TestDiscoverIssuerEndpoints(t *testing.T) {
	server := discoveryServer(t, discoveryOptions{})
	mgmtUrl, _ := url.Parse(server.URL + "/")
	issuer := server.URL + "/_/idprovider/enonic-cloud"

	endpoints, err := DiscoverIssuerEndpoints(http.DefaultClient, issuer, mgmtUrl)
	if err != nil {
		t.Fatal(err)
	}
	if endpoints.DeviceAuthorization != issuer+"/device/code" || endpoints.Token != issuer+"/token" || endpoints.Resource != server.URL {
		t.Fatalf("unexpected endpoints %+v", endpoints)
	}
}

func TestDiscoverIssuerEndpointsRejectsOtherIssuer(t *testing.T) {
	server := discoveryServer(t, discoveryOptions{})
	mgmtUrl, _ := url.Parse(server.URL)

	// the metadata names the issuer without the trailing slash
	_, err := DiscoverIssuerEndpoints(http.DefaultClient, server.URL+"/_/idprovider/enonic-cloud/", mgmtUrl)
	if err == nil || !strings.Contains(err.Error(), "not") {
		t.Fatalf("expected issuer mismatch, got %v", err)
	}
}

// xpServer answers the way XP builds the resource identifier: the origin without a default port, and the path
// as requested, the root as the bare origin
func xpServer(t *testing.T) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := strings.TrimSuffix(r.Host, ":80")
		origin := "http://" + host
		pathOf := func(p string) string {
			if p == "/" {
				return ""
			}
			return p
		}
		const prm = "/.well-known/oauth-protected-resource"
		const asm = "/.well-known/oauth-authorization-server/_/idprovider/system"
		issuer := origin + "/_/idprovider/system"
		switch {
		case strings.HasPrefix(r.URL.Path, prm):
			jsonResponse(w, 200, map[string]interface{}{
				"resource":              origin + strings.TrimPrefix(r.URL.Path, prm),
				"authorization_servers": []string{issuer},
			})
		case r.URL.Path == asm:
			jsonResponse(w, 200, map[string]interface{}{
				"issuer":                        issuer,
				"device_authorization_endpoint": issuer + "/device/code",
				"token_endpoint":                issuer + "/token",
			})
		case strings.HasPrefix(r.URL.Path, "/.well-known/"):
			http.NotFound(w, r)
		default:
			w.Header().Add("WWW-Authenticate", `Bearer resource_metadata="`+origin+prm+pathOf(r.URL.Path)+`"`)
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestDiscoverEndpointsLikeXp(t *testing.T) {
	server := xpServer(t)
	cases := map[string]string{
		server.URL:             server.URL,
		server.URL + "/":       server.URL,
		server.URL + "/admin":  server.URL + "/admin",
		server.URL + "/admin/": server.URL + "/admin",
	}
	for mgmt, want := range cases {
		mgmtUrl, _ := url.Parse(mgmt)
		endpoints, err := DiscoverEndpoints(http.DefaultClient, mgmtUrl)
		if err != nil {
			t.Errorf("%s: %v", mgmt, err)
		} else if endpoints.Resource != want {
			t.Errorf("%s: resource %q, want %q", mgmt, endpoints.Resource, want)
		}
	}
}

func TestDiscoverEndpointsWithDefaultPort(t *testing.T) {
	server := xpServer(t)
	addr := strings.TrimPrefix(server.URL, "http://")
	dialer := &net.Dialer{}
	client := &http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, addr)
		},
	}}
	for _, mgmt := range []string{"http://xp.example:80", "http://xp.example:80/admin/"} {
		mgmtUrl, _ := url.Parse(mgmt)
		endpoints, err := DiscoverEndpoints(client, mgmtUrl)
		if err != nil {
			t.Errorf("%s: %v", mgmt, err)
			continue
		}
		if want := strings.TrimSuffix(strings.Replace(mgmt, ":80", "", 1), "/"); endpoints.Resource != want {
			t.Errorf("%s: resource %q, want %q", mgmt, endpoints.Resource, want)
		}
	}
}

func TestMgmtKey(t *testing.T) {
	cases := map[string]string{
		"https://host:443":        "https://host",
		"http://host:80/admin/":   "http://host/admin",
		"HTTPS://host:8443/admin": "https://host:8443/admin",
		"http://host:443":         "http://host:443",
		"https://host/":           "https://host",
	}
	for in, want := range cases {
		u, _ := url.Parse(in)
		if got := mgmtKey(u); got != want {
			t.Errorf("mgmtKey(%s) = %q, want %q", in, got, want)
		}
	}
}

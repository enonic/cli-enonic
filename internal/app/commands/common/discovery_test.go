package common

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func discoveryServer(t *testing.T, resource func(origin string) string) *httptest.Server {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		issuer := server.URL + "/_/idprovider/enonic-cloud"
		switch r.URL.Path {
		case "/.well-known/oauth-protected-resource":
			jsonResponse(w, 200, map[string]interface{}{
				"resource":              resource(server.URL),
				"authorization_servers": []string{issuer},
			})
		case "/.well-known/oauth-authorization-server/_/idprovider/enonic-cloud":
			jsonResponse(w, 200, map[string]interface{}{
				"issuer":                        issuer,
				"device_authorization_endpoint": issuer + "/device/code",
				"token_endpoint":                issuer + "/token",
			})
		case "/_/idprovider/enonic-cloud/device/code":
			r.ParseForm()
			if r.Form.Get("resource") != server.URL {
				jsonResponse(w, 400, map[string]string{"error": "invalid_target"})
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

func TestDiscoverEndpoints(t *testing.T) {
	server := discoveryServer(t, func(origin string) string { return origin })
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
	if code.UserCode != "BCDF-GHJK" {
		t.Fatalf("unexpected user code %q", code.UserCode)
	}
}

func TestDiscoverEndpointsRejectsOtherResource(t *testing.T) {
	server := discoveryServer(t, func(origin string) string { return "https://other.example.com" })
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

func TestWellKnownUrl(t *testing.T) {
	u, _ := url.Parse("https://admin.example.com/_/idprovider/system/")
	if got := wellKnownUrl(u, "oauth-authorization-server"); got != "https://admin.example.com/.well-known/oauth-authorization-server/_/idprovider/system" {
		t.Fatalf("unexpected url %q", got)
	}
}

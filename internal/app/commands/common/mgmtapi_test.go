package common

import (
	"cli-enonic/internal/app/commands/remote"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompatAtLeast(t *testing.T) {
	cases := map[string]bool{
		"7": false, "7.16": false, "8": false, "8.0": false, "8.1": false,
		"8.2": true, "8.10": true, "9": true, "10.0": true,
	}
	for compat, want := range cases {
		if got := compatAtLeast(compat, 8, 2); got != want {
			t.Errorf("%s: got %v", compat, got)
		}
	}
}

func TestServerApisFollowsCompatWithoutAsking(t *testing.T) {
	setupTokenTest(t, false)
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests++ }))
	defer server.Close()
	t.Setenv(remote.CLI_REMOTE_URL, server.URL)

	if ServerApis(newCompatCtx("8.1", true)) || !ServerApis(newCompatCtx("8.2", true)) || ServerApis(newCompatCtx("7", true)) {
		t.Fatal("--compat not followed")
	}
	if requests != 0 {
		t.Fatalf("asked the endpoint %d times", requests)
	}
}

func TestServerApisAsksOncePerEndpoint(t *testing.T) {
	setupTokenTest(t, false)
	t.Setenv(remote.CLI_REMOTE_USER, "su")
	t.Setenv(remote.CLI_REMOTE_PASS, "pw")

	probes := map[string]int{}
	endpoint := func(serves bool) *httptest.Server {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			probes[r.Host+r.URL.Path]++
			if !serves || r.URL.Path != SERVER_API_PROBE {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"repositories":[]}`)
		}))
		t.Cleanup(server.Close)
		return server
	}

	for _, tc := range []struct {
		server *httptest.Server
		want   bool
	}{{endpoint(true), true}, {endpoint(false), false}} {
		t.Setenv(remote.CLI_REMOTE_URL, tc.server.URL)
		for i := 0; i < 2; i++ {
			if got := ServerApis(newCompatCtx("", false)); got != tc.want {
				t.Fatalf("%s: got %v", tc.server.URL, got)
			}
		}
		if n := probes[strings.TrimPrefix(tc.server.URL, "http://")+SERVER_API_PROBE]; n != 1 {
			t.Fatalf("%s: asked %d times", tc.server.URL, n)
		}
	}
}

func TestParseResponseAcceptsAccepted(t *testing.T) {
	res := &http.Response{StatusCode: http.StatusAccepted, Body: io.NopCloser(strings.NewReader(`{"taskId":"t1"}`))}
	var task TaskResponse
	if enonicErr, err := ParseResponseCustom(res, &task); enonicErr != nil || err != nil || task.TaskId != "t1" {
		t.Fatalf("got %+v %v %v", task, enonicErr, err)
	}
}

func TestParseResponseReadsServerApiErrors(t *testing.T) {
	res := &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden",
		Body: io.NopCloser(strings.NewReader(`{"status":403,"message":"Verb [update] of API [server:index] is not exposed on this virtual host"}`))}
	var target map[string]interface{}
	enonicErr, _ := ParseResponseCustom(res, &target)
	if enonicErr == nil || enonicErr.Status != 403 || !strings.Contains(enonicErr.Message, "not exposed") {
		t.Fatalf("got %+v", enonicErr)
	}
}

func TestTaskPath(t *testing.T) {
	if got := taskPath(newCompatCtx("8.2", true), "t1"); got != "/server:task/t1" {
		t.Fatalf("got %s", got)
	}
	if got := taskPath(newCompatCtx("7", true), "t1"); got != "/task/t1" {
		t.Fatalf("got %s", got)
	}
}

package common

import (
	"cli-enonic/internal/app/commands/remote"
	"io"
	"net/http"
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

func TestServerApisFollowsCompat(t *testing.T) {
	cases := []struct {
		compat string
		set    bool
		want   bool
	}{{"", false, true}, {"8.2", true, true}, {"9", true, true}, {"8.1", true, false}, {"8", true, false}, {"7", true, false}}
	for _, tc := range cases {
		if got := ServerApis(newCompatCtx(tc.compat, tc.set)); got != tc.want {
			t.Errorf("--compat %q: got %v", tc.compat, got)
		}
	}
}

func TestServerApiPathEscapesSegments(t *testing.T) {
	cases := map[string]string{
		"release#1": "/server:export/release%231/load",
		"a%b":       "/server:export/a%25b/load",
		"a/b":       "/server:export/a%2Fb/load",
		"a?b c":     "/server:export/a%3Fb%20c/load",
	}
	for name, want := range cases {
		if got := ServerApiPath("export", name, "load"); got != want {
			t.Errorf("%s: got %s", name, got)
		}
	}
}

func TestRequestKeepsEscapedSegments(t *testing.T) {
	setupTokenTest(t, false)
	t.Setenv(remote.CLI_REMOTE_URL, "https://xp.example/mgmt")
	for name, want := range map[string]string{
		"release#1": "https://xp.example/mgmt/server:export/release%231/load",
		"a%b":       "https://xp.example/mgmt/server:export/a%25b/load",
		"a/b":       "https://xp.example/mgmt/server:export/a%2Fb/load",
	} {
		resolved, err := resolveRequestUrl(ServerApiPath("export", name, "load"))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got := resolved.String(); got != want {
			t.Errorf("%s: got %s", name, got)
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

package common

import (
	"cli-enonic/internal/app/commands/remote"
	"cli-enonic/internal/app/util"
	"github.com/urfave/cli"
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
	t.Setenv(util.BETA_ENV_VAR, "true")
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

// without beta the REST resources are used, as before, whatever --compat says
func TestServerApisNeedBeta(t *testing.T) {
	for _, beta := range []string{"", "false", "1"} {
		t.Setenv(util.BETA_ENV_VAR, beta)
		for _, compat := range []string{"", "8.2", "9", "8.1", "7"} {
			if ServerApis(newCompatCtx(compat, compat != "")) {
				t.Errorf("beta %q, --compat %q: got true", beta, compat)
			}
		}
		if got := taskPath(newCompatCtx("", false), "t1"); got != "/task/t1" {
			t.Errorf("beta %q: task path %s", beta, got)
		}
	}
}

func TestCompatUsageMentionsServerApisOnlyWithBeta(t *testing.T) {
	if strings.Contains(compatUsage(false), "8.2") || !strings.Contains(compatUsage(true), "8.2") {
		t.Fatal("8.2 must be described only with beta")
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
	t.Setenv(util.BETA_ENV_VAR, "true")
	res := &http.Response{StatusCode: http.StatusAccepted, Body: io.NopCloser(strings.NewReader(`{"taskId":"t1"}`))}
	var task TaskResponse
	if enonicErr, err := ParseResponseCustom(res, &task); enonicErr != nil || err != nil || task.TaskId != "t1" {
		t.Fatalf("got %+v %v %v", task, enonicErr, err)
	}
}

// without beta only 200 is a success, as before
func TestParseResponseWithoutBetaAcceptsOnlyOk(t *testing.T) {
	t.Setenv(util.BETA_ENV_VAR, "")
	res := &http.Response{StatusCode: http.StatusAccepted, Status: "202 Accepted", Body: io.NopCloser(strings.NewReader(`{"taskId":"t1"}`))}
	var task TaskResponse
	if _, err := ParseResponseCustom(res, &task); err == nil || err.Error() != "202 Accepted" {
		t.Fatalf("got %v", err)
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
	t.Setenv(util.BETA_ENV_VAR, "true")
	if got := taskPath(newCompatCtx("8.2", true), "t1"); got != "/server:task/t1" {
		t.Fatalf("got %s", got)
	}
	if got := taskPath(newCompatCtx("7", true), "t1"); got != "/task/t1" {
		t.Fatalf("got %s", got)
	}
}

func TestWithBetaCompat(t *testing.T) {
	flags := []cli.Flag{FORCE_FLAG}
	t.Setenv(util.BETA_ENV_VAR, "")
	if got := WithBetaCompat(flags); len(got) != 1 {
		t.Fatalf("beta off: got %v", got)
	}
	t.Setenv(util.BETA_ENV_VAR, "true")
	if got := WithBetaCompat(flags); len(got) != 2 || got[1].GetName() != "compat" {
		t.Fatalf("beta on: got %v", got)
	}
}

func TestCompatEnvVarOnlyWithBeta(t *testing.T) {
	if compatEnvVar(false) != "" || compatEnvVar(true) != CLI_COMPAT_ENV_VAR {
		t.Fatal("ENONIC_CLI_COMPAT must be read only with beta")
	}
}

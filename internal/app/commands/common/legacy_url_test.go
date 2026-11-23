package common

import (
	"cli-enonic/internal/app/commands/remote"
	"testing"
)

// without beta, request urls are built as before: a leading slash at the host's root, no slash normalisation
func TestRequestUrlWithoutBetaAsBefore(t *testing.T) {
	cases := []struct{ remoteUrl, path, want string }{
		{"https://xp.example/mgmt", "/task/t1", "/task/t1"},
		{"https://xp.example/mgmt", "repo/list", "/mgmt/repo/list"},
		{"http://localhost:4848/", "app/start", "//app/start"},
	}
	for _, tc := range cases {
		setupTokenTest(t, false)
		t.Setenv(remote.CLI_REMOTE_URL, tc.remoteUrl)
		if got := doCreateSimpleRequest("GET", tc.path, nil).URL.Path; got != tc.want {
			t.Errorf("%s + %s: got %s", tc.remoteUrl, tc.path, got)
		}
	}
}

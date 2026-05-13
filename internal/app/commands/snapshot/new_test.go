package snapshot

import (
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/urfave/cli"
)

// isolateEnonicHome points the CLI's enonic-home at a fresh tempdir, with a
// pre-populated `.enonic/.enonic` runtime-data file. This stops
// common.CreateRequest from failing on CI runners (no $HOME/.enonic dir) or
// dropping into the interactive auth prompt (empty SessionID).
func isolateEnonicHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	enonicDir := filepath.Join(home, ".enonic")
	if err := os.MkdirAll(enonicDir, 0755); err != nil {
		t.Fatalf("mkdir .enonic: %v", err)
	}
	runtime := []byte("SessionID = \"test-session\"\n")
	if err := os.WriteFile(filepath.Join(enonicDir, ".enonic"), runtime, 0640); err != nil {
		t.Fatalf("write runtime data: %v", err)
	}
	t.Setenv("ENONIC_CLI_HOME_PATH", home)
}

func newSnapCtx(compat, repo string) *cli.Context {
	fs := flag.NewFlagSet("test", 0)
	fs.String("compat", "", "")
	fs.String("repo", "", "")
	if compat != "" {
		fs.Set("compat", compat)
	}
	if repo != "" {
		fs.Set("repo", repo)
	}
	return cli.NewContext(nil, fs, nil)
}

func decodeJSONBody(t *testing.T, r io.Reader) map[string]interface{} {
	t.Helper()
	var params map[string]interface{}
	if err := json.NewDecoder(r).Decode(&params); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	return params
}

func TestCreateNewRequest_EmptyRepo(t *testing.T) {
	isolateEnonicHome(t)
	c := newSnapCtx("", "")
	req := createNewRequest(c)
	params := decodeJSONBody(t, req.Body)

	if _, ok := params["repositoryId"]; ok {
		t.Errorf("expected no repositoryId when --repo not set, got %v", params["repositoryId"])
	}
}

func TestCreateNewRequest_WithRepo(t *testing.T) {
	isolateEnonicHome(t)
	c := newSnapCtx("", "com.enonic.cms.default")
	req := createNewRequest(c)
	params := decodeJSONBody(t, req.Body)

	if params["repositoryId"] != "com.enonic.cms.default" {
		t.Errorf("expected repositoryId=com.enonic.cms.default, got %v", params["repositoryId"])
	}
}

func TestCreateCommand_HasCompatFlag(t *testing.T) {
	if !commandHasFlag(Create.Flags, "compat") {
		t.Error("snapshot create command must register --compat flag")
	}
}

func commandHasFlag(flags []cli.Flag, name string) bool {
	for _, f := range flags {
		if f.GetName() == name {
			return true
		}
	}
	return false
}

package snapshot

import (
	"flag"
	"testing"

	"github.com/urfave/cli"
)

func newRestoreCtx(compat, snap, repo string, latest, clean bool) *cli.Context {
	fs := flag.NewFlagSet("test", 0)
	fs.String("compat", "", "")
	fs.String("snapshot", "", "")
	fs.String("repo", "", "")
	fs.Bool("latest", false, "")
	fs.Bool("clean", false, "")
	if compat != "" {
		fs.Set("compat", compat)
	}
	if snap != "" {
		fs.Set("snapshot", snap)
	}
	if repo != "" {
		fs.Set("repo", repo)
	}
	if latest {
		fs.Set("latest", "true")
	}
	if clean {
		fs.Set("clean", "true")
	}
	return cli.NewContext(nil, fs, nil)
}

func TestCreateRestoreRequest_Latest(t *testing.T) {
	isolateEnonicHome(t)
	c := newRestoreCtx("", "", "", true, false)
	req := createRestoreRequest(c)
	params := decodeJSONBody(t, req.Body)

	if params["latest"] != true {
		t.Errorf("expected latest=true, got %v", params["latest"])
	}
	if _, ok := params["snapshotName"]; ok {
		t.Errorf("expected no snapshotName when --latest set, got %v", params["snapshotName"])
	}
}

func TestCreateRestoreRequest_BySnapshotName(t *testing.T) {
	isolateEnonicHome(t)
	c := newRestoreCtx("", "my-snap", "", false, false)
	req := createRestoreRequest(c)
	params := decodeJSONBody(t, req.Body)

	if params["snapshotName"] != "my-snap" {
		t.Errorf("expected snapshotName=my-snap, got %v", params["snapshotName"])
	}
	if _, ok := params["latest"]; ok {
		t.Errorf("expected no latest when --latest not set")
	}
}

func TestCreateRestoreRequest_Clean(t *testing.T) {
	isolateEnonicHome(t)
	c := newRestoreCtx("", "my-snap", "", false, true)
	req := createRestoreRequest(c)
	params := decodeJSONBody(t, req.Body)

	if params["force"] != true {
		t.Errorf("expected force=true when --clean set, got %v", params["force"])
	}
}

func TestCreateRestoreRequest_Repository(t *testing.T) {
	isolateEnonicHome(t)
	c := newRestoreCtx("", "my-snap", "my-repo", false, false)
	req := createRestoreRequest(c)
	params := decodeJSONBody(t, req.Body)

	if params["repository"] != "my-repo" {
		t.Errorf("expected repository=my-repo, got %v", params["repository"])
	}
}

func TestRestoreCommand_HasCompatFlag(t *testing.T) {
	if !commandHasFlag(Restore.Flags, "compat") {
		t.Error("snapshot restore command must register --compat flag")
	}
}

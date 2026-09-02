package export

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

func newCtx(t *testing.T, compat string, bools map[string]bool) *cli.Context {
	t.Helper()
	fs := flag.NewFlagSet("test", 0)
	fs.String("compat", "", "")
	fs.String("t", "", "")
	fs.String("path", "", "")
	fs.String("xsl-source", "", "")
	for _, name := range []string{"skip-ids", "skip-versions", "skip-permissions", "dry"} {
		fs.Bool(name, false, "")
	}
	fs.Set("t", "myExport")
	fs.Set("path", "com.enonic.cms.default:draft:/content")
	if compat != "" {
		fs.Set("compat", compat)
	}
	for name, val := range bools {
		if val {
			fs.Set(name, "true")
		}
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

func assertKeys(t *testing.T, params map[string]interface{}, want []string) {
	t.Helper()
	if len(params) != len(want) {
		t.Errorf("expected exactly %d keys %v, got %v", len(want), want, params)
	}
	for _, k := range want {
		if _, ok := params[k]; !ok {
			t.Errorf("expected key %q in payload, got %v", k, params)
		}
	}
}

// XP 8 ExportNodesRequestJson only accepts sourceRepoPath, exportName and
// batchSize. Any other field is rejected by Jackson (#690).
func TestCreateNewRequest_XP8OmitsLegacyFields(t *testing.T) {
	isolateEnonicHome(t)
	c := newCtx(t, "", map[string]bool{"skip-ids": true, "skip-versions": true, "dry": true})
	params := decodeJSONBody(t, createNewRequest(c).Body)

	assertKeys(t, params, []string{"exportName", "sourceRepoPath"})
	if params["exportName"] != "myExport" || params["sourceRepoPath"] != "com.enonic.cms.default:draft:/content" {
		t.Errorf("unexpected payload %v", params)
	}
}

func TestCreateNewRequest_CompatDefaults(t *testing.T) {
	isolateEnonicHome(t)
	c := newCtx(t, "7", nil)
	params := decodeJSONBody(t, createNewRequest(c).Body)

	assertKeys(t, params, []string{"exportName", "sourceRepoPath", "exportWithIds", "includeVersions", "dryRun"})
	if params["exportWithIds"] != true || params["includeVersions"] != true || params["dryRun"] != false {
		t.Errorf("unexpected compat defaults %v", params)
	}
}

func TestCreateNewRequest_CompatFlags(t *testing.T) {
	isolateEnonicHome(t)
	c := newCtx(t, "7.16", map[string]bool{"skip-ids": true, "skip-versions": true, "dry": true})
	params := decodeJSONBody(t, createNewRequest(c).Body)

	if params["exportWithIds"] != false || params["includeVersions"] != false || params["dryRun"] != true {
		t.Errorf("unexpected compat payload %v", params)
	}
}

// XP 8 ImportNodesRequestJson dropped dryRun but kept importWithIds,
// importWithPermissions, xslSource and xslParams.
func TestCreateLoadRequest_XP8OmitsDryRun(t *testing.T) {
	isolateEnonicHome(t)
	xslParams = nil
	c := newCtx(t, "", map[string]bool{"skip-ids": true, "dry": true})
	params := decodeJSONBody(t, createLoadRequest(c).Body)

	assertKeys(t, params, []string{"exportName", "targetRepoPath", "importWithIds", "importWithPermissions"})
	if params["importWithIds"] != false || params["importWithPermissions"] != true {
		t.Errorf("unexpected payload %v", params)
	}
}

func TestCreateLoadRequest_CompatIncludesDryRun(t *testing.T) {
	isolateEnonicHome(t)
	xslParams = nil
	c := newCtx(t, "7", map[string]bool{"dry": true})
	params := decodeJSONBody(t, createLoadRequest(c).Body)

	assertKeys(t, params, []string{"exportName", "targetRepoPath", "importWithIds", "importWithPermissions", "dryRun"})
	if params["dryRun"] != true {
		t.Errorf("expected dryRun=true in compat mode, got %v", params)
	}
}

// XP returns exportErrors as a list of strings, not objects.
func TestNewExportResponse_DecodesStringErrors(t *testing.T) {
	raw := `{"total":1,"exportedNodes":["/content"],"exportedBinaries":[],"exportErrors":["Failed to export /content/a"]}`
	var result NewExportResponse
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("failed to decode export response: %v", err)
	}
	if len(result.Errors) != 1 || result.Errors[0] != "Failed to export /content/a" {
		t.Errorf("unexpected errors %v", result.Errors)
	}
}

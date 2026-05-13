package dump

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func decodeBody(t *testing.T, r io.Reader) map[string]string {
	t.Helper()
	var params map[string]string
	if err := json.NewDecoder(r).Decode(&params); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	return params
}

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

func TestCreateUpgradeRequest_StripsZipSuffix(t *testing.T) {
	// When the user selects a dump file like "mydump.zip" from the prompt,
	// the upgrade request must send the bare name so the server does not
	// append a second .zip extension.
	isolateEnonicHome(t)
	req := createUpgradeRequest(nil, "mydump.zip")

	if req.URL.Path != "system/upgrade" && req.URL.Path != "/system/upgrade" {
		t.Errorf("unexpected path: %s", req.URL.Path)
	}

	params := decodeBody(t, req.Body)
	if params["name"] != "mydump" {
		t.Errorf("expected name=mydump (stripped), got %q", params["name"])
	}
}

func TestCreateUpgradeRequest_PassesBareNameThrough(t *testing.T) {
	isolateEnonicHome(t)
	req := createUpgradeRequest(nil, "mydump")
	params := decodeBody(t, req.Body)
	if params["name"] != "mydump" {
		t.Errorf("expected name=mydump, got %q", params["name"])
	}
}

package repo

import (
	"cli-enonic/internal/app/commands/remote"
	"cli-enonic/internal/app/util"
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/urfave/cli"
)

func isolateEnonicHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".enonic"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".enonic", ".enonic"), []byte("SessionID = \"test-session\"\n"), 0640); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ENONIC_CLI_HOME_PATH", home)
}

func TestCreateServerReindexRequest(t *testing.T) {
	isolateEnonicHome(t)
	fs := flag.NewFlagSet("test", 0)
	fs.String("r", "", "")
	fs.String("b", "", "")
	fs.Bool("i", false, "")
	fs.Set("r", "com.enonic.cms.default")
	fs.Set("b", "draft,master")
	req := createServerReindexRequest(cli.NewContext(nil, fs, nil))

	if req.Method != "POST" || req.URL.Path != "/server:index/com.enonic.cms.default/reindex" {
		t.Fatalf("got %s %s", req.Method, req.URL.Path)
	}
	var body struct {
		Branches   []string
		Initialize bool
		Repository *string
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Branches) != 2 || body.Branches[1] != "master" || body.Initialize || body.Repository != nil {
		t.Fatalf("unexpected body %+v", body)
	}
}

func TestCreateReplicasRequest(t *testing.T) {
	isolateEnonicHome(t)
	req := createReplicasRequest(cli.NewContext(nil, flag.NewFlagSet("test", 0), nil), "system.app", 2)

	if req.Method != "PUT" || req.URL.Path != "/server:index/system.app" {
		t.Fatalf("got %s %s", req.Method, req.URL.Path)
	}
	var body IndexReplicas
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Replicas["search"] != "2" || body.Replicas["storage"] != "2" || len(body.Replicas) != 2 {
		t.Fatalf("unexpected body %+v", body)
	}
}

// with beta and without --compat the list comes from /server:repo/
func TestListRepositoriesFromServerApi(t *testing.T) {
	listRepositoriesFrom(t, "true", "/server:repo/")
}

// without beta the list comes from the REST resource, as before
func TestListRepositoriesWithoutBetaFromRest(t *testing.T) {
	listRepositoriesFrom(t, "", "/repo/list")
}

func listRepositoriesFrom(t *testing.T, beta, wantPath string) {
	isolateEnonicHome(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv(remote.CLI_ENV, "")
	t.Setenv(remote.CLI_REMOTE_PROXY, "")
	t.Setenv(remote.CLI_REMOTE_USER, "su")
	t.Setenv(remote.CLI_REMOTE_PASS, "pw")
	t.Setenv(util.BETA_ENV_VAR, beta)
	remote.SelectEnv("")
	t.Cleanup(func() { remote.SelectEnv("") })

	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		if r.URL.Path != wantPath {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"repositories":[{"id":"system.app","branches":["master"]}]}`)
	}))
	defer server.Close()
	t.Setenv(remote.CLI_REMOTE_URL, server.URL)

	result := listRepositories(cli.NewContext(nil, flag.NewFlagSet("test", 0), nil), "")
	if len(result.Repositories) != 1 || result.Repositories[0].Id != "system.app" {
		t.Fatalf("unexpected result %+v", result)
	}
	if len(paths) != 1 || paths[0] != "GET "+wantPath {
		t.Fatalf("unexpected requests %v", paths)
	}
}

package env

import (
	"bytes"
	"cli-enonic/internal/app/commands/common"
	"cli-enonic/internal/app/commands/remote"
	"cli-enonic/internal/app/util"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/urfave/cli"
)

func setup(t *testing.T) *bytes.Buffer {
	t.Setenv("ENONIC_CLI_HOME_PATH", t.TempDir())
	os.Unsetenv(util.SNAP_ENV_VAR)
	t.Setenv(util.BETA_ENV_VAR, "true")
	t.Setenv(remote.CLI_ENV, "")
	t.Setenv(remote.CLI_REMOTE_URL, "")
	t.Setenv(remote.CLI_REMOTE_PROXY, "")
	remote.SelectEnv("")
	var buf bytes.Buffer
	out = &buf
	t.Cleanup(func() { out = os.Stdout })
	return &buf
}

func run(t *testing.T, args ...string) error {
	t.Helper()
	app := cli.NewApp()
	app.Commands = All()
	app.Writer = &bytes.Buffer{}
	app.ErrWriter = &bytes.Buffer{}
	app.ExitErrHandler = func(*cli.Context, error) {}
	return app.Run(append([]string{"enonic"}, args...))
}

func TestEnvLifecycle(t *testing.T) {
	buf := setup(t)
	for _, args := range [][]string{{"add", "prod", "https://prod.example"}, {"use", "prod"}} {
		if err := run(t, args...); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	if err := run(t, "current"); err != nil || buf.String() != "prod (https://prod.example), selected by enonic env use\n" {
		t.Fatalf("current: %q %v", buf.String(), err)
	}

	token := &common.AccessToken{TokenEndpoint: "http://idp/token", AccessToken: "at", ExpiresAt: time.Now().Add(time.Hour).Unix()}
	prod, _ := remote.EnvTarget("prod", "")
	if err := common.SaveAccessToken(prod, token); err != nil {
		t.Fatal(err)
	}

	buf.Reset()
	if err := run(t, "list"); err != nil {
		t.Fatal(err)
	}
	want := "  local  http://localhost:4848\n* prod   https://prod.example  logged in\n"
	if buf.String() != want {
		t.Fatalf("list:\n%q\nwant\n%q", buf.String(), want)
	}

	if err := run(t, "remove", "prod"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(util.GetInEnonicDir("tokens", "prod.json")); !os.IsNotExist(err) {
		t.Fatalf("token file left: %v", err)
	}
	buf.Reset()
	if err := run(t, "current"); err != nil || buf.String() != "local (http://localhost:4848), selected by default\n" {
		t.Fatalf("current after remove: %q %v", buf.String(), err)
	}
}

func TestEnvAddRejects(t *testing.T) {
	setup(t)
	if err := run(t, "add", "prod", "https://prod.example"); err != nil {
		t.Fatal(err)
	}
	cases := map[string][]string{
		"Invalid environment name 'Prod'. Use lowercase letters, digits, '-' and '_', not starting with 'url-'.":               {"add", "Prod", "https://x.example"},
		"Invalid environment name 'url-localhost-4848'. Use lowercase letters, digits, '-' and '_', not starting with 'url-'.": {"add", "url-localhost-4848", "https://x.example"},
		"Environment 'local' already exists.":                {"add", "local", "https://x.example"},
		"Environment 'prod' already exists.":                 {"add", "prod", "https://x.example"},
		"Invalid url 'ftp://x'.":                             {"add", "x", "ftp://x"},
		"Invalid url 'nohost'.":                              {"add", "x", "https://x.example", "--proxy", "nohost"},
		"The local environment cannot be removed.":           {"remove", "local"},
		"Unknown environment 'nope'. Run 'enonic env list'.": {"use", "nope"},
	}
	for want, args := range cases {
		if err := run(t, args...); err == nil || err.Error() != want {
			t.Errorf("%v: got %v, want %q", args, err, want)
		}
	}
	for _, args := range [][]string{{"add"}, {"add", "a", "https://a.example", "extra"}} {
		if err := run(t, args...); err == nil {
			t.Errorf("%v: expected usage error", args)
		}
	}
}

func TestEnvListWithUnknownActive(t *testing.T) {
	buf := setup(t)
	t.Setenv(remote.CLI_ENV, "nope")
	if err := run(t, "list"); err != nil || !strings.Contains(buf.String(), "local") {
		t.Fatalf("got %q %v", buf.String(), err)
	}
	if _, err := os.Stat(filepath.Join(util.GetInEnonicDir(), "environments.toml")); !os.IsNotExist(err) {
		t.Fatalf("list created the environments file: %v", err)
	}
}

func TestEnvAddCloudShortcut(t *testing.T) {
	buf := setup(t)
	t.Setenv(CLOUD_DOMAIN_VAR, "")
	if err := run(t, "add", "my-site"); err != nil {
		t.Fatal(err)
	}
	t.Setenv(CLOUD_DOMAIN_VAR, "enonic-dev.cloud")
	if err := run(t, "add", "sry-kk-test"); err != nil {
		t.Fatal(err)
	}
	if err := run(t, "list"); err != nil {
		t.Fatal(err)
	}
	want := "* local        http://localhost:4848\n" +
		"  my-site      https://my-site-mgmt.enonic.cloud\n" +
		"  sry-kk-test  https://sry-kk-test-mgmt.enonic-dev.cloud\n"
	if buf.String() != want {
		t.Fatalf("list:\n%q\nwant\n%q", buf.String(), want)
	}

	for _, name := range []string{"my_site", "trailing-"} {
		if err := run(t, "add", name); err == nil || !strings.Contains(err.Error(), "not an Enonic Cloud instance name") {
			t.Errorf("%s: got %v", name, err)
		}
	}
}

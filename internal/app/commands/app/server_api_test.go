package app

import (
	"encoding/json"
	"flag"
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

func TestApplicationInfosNameApplicationsByTitle(t *testing.T) {
	raw := `{"applications":[{"key":"com.enonic.app.superhero","title":"Superhero","version":"2.0.5","state":"started","local":false,"system":false},{"key":"com.enonic.app.local","version":"1.0.0","state":"stopped","local":true,"system":false}]}`
	var infos ApplicationInfos
	if err := json.Unmarshal([]byte(raw), &infos); err != nil {
		t.Fatal(err)
	}
	apps := infos.result().Applications
	if len(apps) != 2 || apps[0].DisplayName != "Superhero" || apps[1].DisplayName != "" || !apps[1].Local {
		t.Fatalf("unexpected applications %+v", apps)
	}
	if installed := infos.Applications[0].installed(); installed.DisplayName != "Superhero" || installed.Version != "2.0.5" {
		t.Fatalf("unexpected installed application %+v", installed)
	}
}

func TestAppRequests_ServerApi(t *testing.T) {
	isolateEnonicHome(t)
	c := cli.NewContext(nil, flag.NewFlagSet("test", 0), nil)

	jar := filepath.Join(t.TempDir(), "app.jar")
	if err := os.WriteFile(jar, []byte("jar"), 0644); err != nil {
		t.Fatal(err)
	}
	for want, path := range map[string]string{
		"/server:app/install": createInstallRequest(c, jar, "", true).URL.Path,
		"/server:app/pull":    createInstallRequest(c, "", "https://repo.example/app.jar", true).URL.Path,
		"/server:app/start":   createStartRequest(c, "com.enonic.app.superhero", true).URL.Path,
		"/server:app/stop":    createStopRequest(c, "com.enonic.app.superhero", true).URL.Path,
		"/app/start":          createStartRequest(c, "com.enonic.app.superhero", false).URL.Path,
	} {
		if path != want {
			t.Errorf("got %s, want %s", path, want)
		}
	}
}

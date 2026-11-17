package remote

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func setupEnvTest(t *testing.T, beta bool) {
	t.Setenv("ENONIC_CLI_HOME_PATH", t.TempDir())
	t.Setenv("SNAP_USER_COMMON", "")
	os.Unsetenv("SNAP_USER_COMMON")
	t.Setenv(CLI_ENV, "")
	t.Setenv(CLI_REMOTE_URL, "")
	t.Setenv(CLI_REMOTE_PROXY, "")
	if beta {
		t.Setenv("ENONIC_CLI_BETA", "true")
	} else {
		t.Setenv("ENONIC_CLI_BETA", "")
	}
	SelectEnv("")
	t.Cleanup(func() { SelectEnv("") })
}

func mustUrl(t *testing.T, s string) *MarshalledUrl {
	u, err := ParseMarshalledUrl(s)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func saveEnvs(t *testing.T, active string) {
	err := WriteEnvironments(&EnvironmentsData{
		Active: active,
		Environments: map[string]Environment{
			"prod":  {Url: mustUrl(t, "https://prod.example"), Proxy: mustUrl(t, "http://proxy.example:3128")},
			"stage": {Url: mustUrl(t, "https://stage.example")},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestResolveOrder(t *testing.T) {
	tests := []struct {
		name                    string
		flag, envVar, url, save string
		wantName, wantUrl       string
		wantSource              string
	}{
		{"default", "", "", "", "", "local", "http://localhost:4848", SOURCE_DEFAULT},
		{"saved", "", "", "", "stage", "stage", "https://stage.example", SOURCE_SAVED},
		{"url beats saved", "", "", "http://one.example:4848", "stage", "", "http://one.example:4848", SOURCE_URL},
		{"variable beats url", "", "prod", "http://one.example:4848", "stage", "prod", "https://prod.example", SOURCE_ENV_VAR},
		{"flag beats all", "stage", "prod", "http://one.example:4848", "prod", "stage", "https://stage.example", SOURCE_FLAG},
		{"flag local", "local", "prod", "", "prod", "local", "http://localhost:4848", SOURCE_FLAG},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupEnvTest(t, true)
			saveEnvs(t, tt.save)
			SelectEnv(tt.flag)
			t.Setenv(CLI_ENV, tt.envVar)
			t.Setenv(CLI_REMOTE_URL, tt.url)

			target, err := ResolveTarget()
			if err != nil {
				t.Fatal(err)
			}
			if target.Name != tt.wantName || target.Url.String() != tt.wantUrl || target.Source != tt.wantSource {
				t.Fatalf("got %q %q %q", target.Name, target.Url.String(), target.Source)
			}
			if active := GetActiveRemote(); active.Name != tt.wantName {
				t.Fatalf("GetActiveRemote picked %q", active.Name)
			}
		})
	}
}

func TestResolveUnknownEnvFails(t *testing.T) {
	want := "Unknown environment 'nope'. Run 'enonic env list'."
	cases := map[string]func(t *testing.T){
		"flag": func(t *testing.T) {
			t.Setenv(CLI_REMOTE_URL, "http://one.example:4848")
			SelectEnv("nope")
		},
		"envVar": func(t *testing.T) {
			t.Setenv(CLI_REMOTE_URL, "http://one.example:4848")
			t.Setenv(CLI_ENV, "nope")
		},
		"saved": func(t *testing.T) { saveEnvs(t, "nope") },
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			setupEnvTest(t, true)
			saveEnvs(t, "")
			setup(t)
			if _, err := ResolveTarget(); err == nil || err.Error() != want {
				t.Fatalf("got %v", err)
			}
		})
	}

	t.Run("valid flag ignores invalid variable", func(t *testing.T) {
		setupEnvTest(t, true)
		saveEnvs(t, "")
		SelectEnv("prod")
		t.Setenv(CLI_ENV, "nope")
		if target, err := ResolveTarget(); err != nil || target.Name != "prod" {
			t.Fatalf("got %+v %v", target, err)
		}
	})
}

func TestResolveDefaultsToLocalWithoutFile(t *testing.T) {
	setupEnvTest(t, true)
	target, err := ResolveTarget()
	if err != nil {
		t.Fatal(err)
	}
	if target.Name != LOCAL_ENV || target.Source != SOURCE_DEFAULT || target.Url.String() != DEFAULT_REMOTE_URL {
		t.Fatalf("got %+v", target)
	}
	if _, err := os.Stat(EnvironmentsFile()); !os.IsNotExist(err) {
		t.Fatalf("environments file was created: %v", err)
	}
}

func TestReadEnvironmentsRejectsCorruptFile(t *testing.T) {
	cases := map[string]string{
		"bad toml":               "not = [toml",
		"bad url":                "[environments.prod]\nurl = \"::\"\n",
		"no host":                "[environments.prod]\nurl = \"ftp:x\"\n",
		"bad name":               "[environments.\"Bad/Name\"]\nurl = \"https://x.example\"\n",
		"bad proxy":              "[environments.prod]\nurl = \"https://x.example\"\nproxy = \"not-a-url\"\n",
		"no hostname":            "[environments.prod]\nurl = \"http://:4848\"\n",
		"proxy without hostname": "[environments.prod]\nurl = \"https://x.example\"\nproxy = \"http://:3128\"\n",
		"url prefix":             "[environments.url-x]\nurl = \"https://x.example\"\n",
		"local":                  "[environments.local]\nurl = \"https://x.example\"\n",
		"bad active":             "active = \"../x\"\n",
		"shared login":           "[environments.\"sry/ee/test\"]\nurl = \"https://a.example\"\n[environments.sry-ee-test]\nurl = \"https://b.example\"\n",
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			setupEnvTest(t, true)
			if err := os.MkdirAll(strings.TrimSuffix(EnvironmentsFile(), "environments.toml"), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(EnvironmentsFile(), []byte(content), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := ReadEnvironments(); err == nil || !strings.Contains(err.Error(), EnvironmentsFile()) {
				t.Fatalf("got %v", err)
			}
			if _, err := ResolveTarget(); err == nil {
				t.Fatal("corrupt file fell back to a target")
			}
		})
	}
}

func TestWriteEnvironmentsRoundTrip(t *testing.T) {
	setupEnvTest(t, true)
	saveEnvs(t, "prod")
	data, err := ReadEnvironments()
	if err != nil {
		t.Fatal(err)
	}
	want := &EnvironmentsData{
		Active: "prod",
		Environments: map[string]Environment{
			"prod":  {Url: mustUrl(t, "https://prod.example"), Proxy: mustUrl(t, "http://proxy.example:3128")},
			"stage": {Url: mustUrl(t, "https://stage.example")},
		},
	}
	if !reflect.DeepEqual(data, want) {
		t.Fatalf("got %+v", data)
	}
	stat, err := os.Stat(EnvironmentsFile())
	if err != nil {
		t.Fatal(err)
	}
	if stat.Mode().Perm() != 0600 {
		t.Fatalf("mode %v", stat.Mode().Perm())
	}
}

func TestEnvProxyOverride(t *testing.T) {
	setupEnvTest(t, true)
	saveEnvs(t, "prod")
	if target, _ := ResolveTarget(); target.Proxy.String() != "http://proxy.example:3128" {
		t.Fatalf("got %v", target.Proxy)
	}
	t.Setenv(CLI_REMOTE_PROXY, "http://other.example:8888")
	if target, _ := ResolveTarget(); target.Proxy.String() != "http://other.example:8888" {
		t.Fatalf("got %v", target.Proxy)
	}
}

func TestBetaOffIgnoresEnvironments(t *testing.T) {
	setupEnvTest(t, false)
	saveEnvs(t, "prod")
	t.Setenv(CLI_ENV, "prod")
	SelectEnv("prod")

	active := GetActiveRemote()
	if active.Url.String() != "http://localhost:4848" || active.Name != "" || active.Source != "" {
		t.Fatalf("got %+v", active)
	}
	t.Setenv(CLI_REMOTE_URL, "http://one.example:4848")
	if got := GetActiveRemote().Url.String(); got != "http://one.example:4848" {
		t.Fatalf("got %s", got)
	}
}

func TestCloudEnvironmentNames(t *testing.T) {
	setupEnvTest(t, true)
	err := WriteEnvironments(&EnvironmentsData{
		Active: "sry/ee/test",
		Environments: map[string]Environment{
			"sry/ee/test": {Url: mustUrl(t, "https://sry-ee-test-mgmt.enonic-dev.cloud")},
			"prod":        {Url: mustUrl(t, "https://prod.example")},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(EnvironmentsFile())
	if !strings.Contains(string(content), `"sry/ee/test"`) {
		t.Fatalf("slash name not quoted:\n%s", content)
	}
	data, err := ReadEnvironments()
	if err != nil {
		t.Fatal(err)
	}
	if data.Active != "sry/ee/test" || data.Environments["sry/ee/test"].Url.String() != "https://sry-ee-test-mgmt.enonic-dev.cloud" {
		t.Fatalf("round trip lost the slash name: %+v", data)
	}

	t.Setenv(CLI_ENV, "sry/ee/test")
	if target, err := EnvTarget("sry/ee/test", SOURCE_ENV_VAR); err != nil || target.Name != "sry/ee/test" {
		t.Fatalf("got %+v %v", target, err)
	}

	for name, want := range map[string]bool{
		"sry/ee/test": true, "a1/b2/c3": true, "prod": true,
		"a/b": false, "a/b/c/d": false, "a-b/c/d": false, "A/b/c": false, "a//c": false, "/a/b": false, "url-x": false,
	} {
		if got := ValidEnvName(name); got != want {
			t.Errorf("ValidEnvName(%q) = %v, want %v", name, got, want)
		}
	}
}

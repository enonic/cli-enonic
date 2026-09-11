package sandbox

import (
	"cli-enonic/internal/app/commands/common"
	"flag"
	"fmt"
	"github.com/urfave/cli"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

// TestAppendRunModeArgs verifies that the run mode is always passed explicitly
// to the XP launcher. XP 8.1+ auto-enables dev mode when xp.runMode is unset
// and the SDK bundle is present, so `--prod` must set xp.runMode=prod rather
// than merely omitting the `dev` argument (#691).
func TestAppendRunModeArgs(t *testing.T) {
	tests := []struct {
		name    string
		devMode bool
		debug   bool
		want    []string
	}{
		{"dev", true, false, []string{"dev"}},
		{"prod", false, false, []string{"-Dxp.runMode=prod"}},
		{"dev with debug", true, true, []string{"debug", "dev"}},
		{"prod with debug", false, true, []string{"debug", "-Dxp.runMode=prod"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appendRunModeArgs(nil, tt.devMode, tt.debug)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("appendRunModeArgs(nil, %v, %v) = %v, want %v", tt.devMode, tt.debug, got, tt.want)
			}
		})
	}
}

func TestAppendRunModeArgsPreservesExisting(t *testing.T) {
	got := appendRunModeArgs([]string{"run", "image", "server.sh"}, false, true)
	want := []string{"run", "image", "server.sh", "debug", "-Dxp.runMode=prod"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("appendRunModeArgs() = %v, want %v", got, want)
	}
}

var prereleaseOnlyVersions = []string{"7.16.8", "8.0.4", "8.1.0-B1", "8.1.0-B2", "8.1.0-B3", "8.1.0-RC1", "8.1.0-SNAPSHOT"}

func TestFilterVersions(t *testing.T) {
	tests := []struct {
		name            string
		minDistro       string
		includeMinVer   bool
		includeUnstable bool
		want            []string
		wantLatest      string
	}{
		{
			name:          "stable only drops everything when min is a pre-release line",
			minDistro:     "8.1.0-B3",
			includeMinVer: true,
			want:          nil,
			wantLatest:    "",
		},
		{
			name:            "unstable keeps pre-releases at or above min, never SNAPSHOT",
			minDistro:       "8.1.0-B3",
			includeMinVer:   true,
			includeUnstable: true,
			want:            []string{"8.1.0-B3", "8.1.0-RC1"},
			wantLatest:      "8.1.0-RC1",
		},
		{
			name:          "stable releases above min are kept",
			minDistro:     "7.16.8",
			includeMinVer: false,
			want:          []string{"8.0.4"},
			wantLatest:    "8.0.4",
		},
		{
			name:       "no min returns all stable",
			want:       []string{"7.16.8", "8.0.4"},
			wantLatest: "8.0.4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, latest := filterVersions(prereleaseOnlyVersions, tt.minDistro, tt.includeMinVer, tt.includeUnstable)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("versions = %v, want %v", got, tt.want)
			}
			if latest != tt.wantLatest {
				t.Errorf("latest = %q, want %q", latest, tt.wantLatest)
			}
		})
	}
}

func TestFilterVersionsSkipsUnparsable(t *testing.T) {
	got, latest := filterVersions([]string{"garbage", "8.0.4"}, "", false, false)
	if !reflect.DeepEqual(got, []string{"8.0.4"}) || latest != "8.0.4" {
		t.Errorf("got %v / %q, want [8.0.4] / 8.0.4", got, latest)
	}
}

func serveMavenMetadata(t *testing.T, versions []string) func() {
	t.Helper()
	var sb strings.Builder
	sb.WriteString(`<metadata><groupId>com.enonic.xp</groupId><artifactId>enonic-xp-linux-sdk</artifactId><versioning><versions>`)
	for _, v := range versions {
		fmt.Fprintf(&sb, "<version>%s</version>", v)
	}
	sb.WriteString(`</versions></versioning></metadata>`)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprint(w, sb.String())
	}))
	original := remoteVersionUrl
	remoteVersionUrl = srv.URL + "/%s/maven-metadata.xml"
	return func() {
		remoteVersionUrl = original
		srv.Close()
	}
}

func newForceContext() *cli.Context {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.Bool("force", false, "")
	fs.String("cred-file", "", "")
	fs.String(common.CLIENT_KEY_FLAG.Name, "", "")
	fs.String(common.CLIENT_CERT_FLAG.Name, "", "")
	fs.Set("force", "true")
	return cli.NewContext(nil, fs, nil)
}

func TestResolveCreateVersionFallsBackToPrerelease(t *testing.T) {
	_, cleanupHome := setupTestSandbox(t, "unused", "7.16.8")
	defer cleanupHome()
	defer serveMavenMetadata(t, prereleaseOnlyVersions)()

	got := resolveCreateVersion(newForceContext(), "", "8.1.0-B3", false, true)
	if got != "8.1.0-RC1" {
		t.Errorf("resolveCreateVersion() = %q, want 8.1.0-RC1", got)
	}
}

func TestResolveCreateVersionPrefersStable(t *testing.T) {
	_, cleanupHome := setupTestSandbox(t, "unused", "7.16.8")
	defer cleanupHome()
	defer serveMavenMetadata(t, []string{"8.0.4", "8.1.0-RC1"})()

	got := resolveCreateVersion(newForceContext(), "", "8.0.0", false, true)
	if got != "8.0.4" {
		t.Errorf("resolveCreateVersion() = %q, want 8.0.4", got)
	}
}

func TestResolveCreateVersionNoFallbackForExplicitVersion(t *testing.T) {
	_, cleanupHome := setupTestSandbox(t, "unused", "7.16.8")
	defer cleanupHome()
	defer serveMavenMetadata(t, []string{"8.0.4", "8.1.0-RC1"})()

	version, total := ensureVersionCorrect(newForceContext(), "8.1.0", "8.1.0-B1", true, false, true)
	if total != 0 || version != "" {
		t.Fatalf("precondition: stable lookup = (%q, %d), want (\"\", 0)", version, total)
	}
	if os.Getenv("RESOLVE_EXPLICIT_CHILD") == "1" {
		resolveCreateVersion(newForceContext(), "8.1.0", "8.1.0-B1", false, true)
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestResolveCreateVersionNoFallbackForExplicitVersion")
	cmd.Env = append(os.Environ(), "RESOLVE_EXPLICIT_CHILD=1")
	out, err := cmd.CombinedOutput()
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != 1 {
		t.Fatalf("expected exit code 1 for explicit version, got %v:\n%s", err, out)
	}
	if !strings.Contains(string(out), "Enonic XP distribution '8.1.0' is not available") {
		t.Errorf("missing diagnostic for the requested version:\n%s", out)
	}
	if strings.Contains(string(out), "looking for pre-releases") {
		t.Errorf("explicit --version must not fall back to pre-releases:\n%s", out)
	}
}

func TestResolveCreateVersionWithAllDoesNotRetry(t *testing.T) {
	_, cleanupHome := setupTestSandbox(t, "unused", "7.16.8")
	defer cleanupHome()
	defer serveMavenMetadata(t, []string{"8.0.4"})()

	if os.Getenv("RESOLVE_ALL_CHILD") == "1" {
		resolveCreateVersion(newForceContext(), "", "8.1.0-B3", true, true)
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestResolveCreateVersionWithAllDoesNotRetry")
	cmd.Env = append(os.Environ(), "RESOLVE_ALL_CHILD=1")
	out, err := cmd.CombinedOutput()
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != 1 {
		t.Fatalf("expected exit code 1 when nothing matches, got %v:\n%s", err, out)
	}
	if !strings.Contains(string(out), "No Enonic XP distribution matching '8.1.0-B3' or higher") {
		t.Errorf("missing diagnostic for the project version:\n%s", out)
	}
	if strings.Contains(string(out), "looking for pre-releases") {
		t.Errorf("--all must not trigger the pre-release fallback:\n%s", out)
	}
}

package sandbox

import (
	"reflect"
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

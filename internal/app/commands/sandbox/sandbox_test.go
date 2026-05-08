package sandbox

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"cli-enonic/internal/app/util"
)

// setupTestSandbox creates a temporary enonic home directory with a sandbox for testing.
// Returns the temp dir path and a cleanup function.
// Note: ENONIC_CLI_HOME_PATH overrides the home directory but GetEnonicHome() still appends
// ".enonic" to it, so directories must be created under tmpDir/.enonic/.
func setupTestSandbox(t *testing.T, sandboxName, distroVersion string) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "enonic-cli-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// GetEnonicHome() returns filepath.Join(ENONIC_CLI_HOME_PATH, ".enonic")
	// so we need to create the structure under tmpDir/.enonic/
	enonicDir := filepath.Join(tmpDir, ".enonic")

	// Create required directory structure
	sandboxesDir := filepath.Join(enonicDir, "sandboxes")
	distrosDir := filepath.Join(enonicDir, "distributions")
	if err := os.MkdirAll(sandboxesDir, 0755); err != nil {
		t.Fatalf("Failed to create sandboxes dir: %v", err)
	}
	if err := os.MkdirAll(distrosDir, 0755); err != nil {
		t.Fatalf("Failed to create distributions dir: %v", err)
	}

	// Create sandbox directory with .enonic descriptor
	sandboxDir := filepath.Join(sandboxesDir, sandboxName)
	if err := os.MkdirAll(sandboxDir, 0755); err != nil {
		t.Fatalf("Failed to create sandbox dir: %v", err)
	}

	// Write sandbox .enonic file with distro info
	descriptorPath := filepath.Join(sandboxDir, ".enonic")
	distroName := fmt.Sprintf("enonic-xp-linux-sdk-%s", distroVersion)
	descriptorContent := fmt.Sprintf("distro = \"%s\"\n", distroName)
	if err := os.WriteFile(descriptorPath, []byte(descriptorContent), 0640); err != nil {
		t.Fatalf("Failed to write sandbox descriptor: %v", err)
	}

	// Point ENONIC_CLI_HOME_PATH to temp dir
	original := os.Getenv(util.ENONIC_CLI_HOME_ENV_VAR_NAME)
	os.Setenv(util.ENONIC_CLI_HOME_ENV_VAR_NAME, tmpDir)

	cleanup := func() {
		os.Setenv(util.ENONIC_CLI_HOME_ENV_VAR_NAME, original)
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// TestListSandboxesWithVersionFilter verifies that listSandboxes filters by minDistroVersion.
func TestListSandboxesWithVersionFilter(t *testing.T) {
	_, cleanup := setupTestSandbox(t, "xp8demo", "7.15.1")
	defer cleanup()

	// With a high min version, the sandbox should be filtered out
	filtered := listSandboxes("8.0.0")
	if len(filtered) != 0 {
		t.Errorf("Expected 0 sandboxes when filtering by version 8.0.0, got %d", len(filtered))
	}

	// With a low min version, the sandbox should be included
	all := listSandboxes("7.0.0")
	if len(all) != 1 {
		t.Errorf("Expected 1 sandbox when filtering by version 7.0.0, got %d", len(all))
	}

	// With no min version (empty string), all sandboxes should be returned
	allNoFilter := listSandboxes("")
	if len(allNoFilter) != 1 {
		t.Errorf("Expected 1 sandbox when no version filter, got %d", len(allNoFilter))
	}
}

// TestExistsFindsVersionFilteredSandbox verifies that Exists() finds a sandbox
// even when it would be filtered out by listSandboxes(minVersion).
func TestExistsFindsVersionFilteredSandbox(t *testing.T) {
	_, cleanup := setupTestSandbox(t, "xp8demo", "7.15.1")
	defer cleanup()

	// Verify the sandbox is filtered out by high min version
	filtered := listSandboxes("8.0.0")
	if len(filtered) != 0 {
		t.Errorf("Expected sandbox to be filtered by version 8.0.0, but got %d sandboxes", len(filtered))
	}

	// Verify the sandbox still exists via Exists()
	if !Exists("xp8demo") {
		t.Error("Expected Exists('xp8demo') to return true even when version-filtered")
	}

	// Verify listSandboxes("") finds it (the fallback used in the fix)
	allSandboxes := listSandboxes("")
	found := false
	for _, box := range allSandboxes {
		if box.Name == "xp8demo" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected listSandboxes('') to find 'xp8demo' sandbox")
	}
}

// TestListSandboxesUnfilteredContainsVersionFiltered verifies the fix logic:
// that calling listSandboxes("") as a fallback returns sandboxes that were
// filtered out by a specific MinDistroVersion.
func TestListSandboxesUnfilteredContainsVersionFiltered(t *testing.T) {
	_, cleanup := setupTestSandbox(t, "mybox", "7.15.1")
	defer cleanup()

	// Simulate the bug scenario: minDistroVersion is higher than sandbox version
	existingBoxes := listSandboxes("8.0.0")
	if len(existingBoxes) != 0 {
		t.Errorf("Pre-condition failed: expected empty list with minVersion 8.0.0, got %d", len(existingBoxes))
	}

	// The fix: when name is given, fall back to listSandboxes("") to find it
	lowerName := "mybox"
	var foundBox *Sandbox
	for _, box := range listSandboxes("") {
		if box.Name == lowerName {
			foundBox = box
			break
		}
	}

	if foundBox == nil {
		t.Error("Fix failed: sandbox 'mybox' should be found via listSandboxes('') fallback")
	} else if foundBox.Name != "mybox" {
		t.Errorf("Fix failed: expected sandbox name 'mybox', got '%s'", foundBox.Name)
	}
}

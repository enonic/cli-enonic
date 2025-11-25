package util

import (
    "testing"
	"os"
	"path/filepath"
)

func TestGetHomeDir(t *testing.T) {
	expected := os.Getenv("HOME")
	home := GetHomeDir()
	if home != expected {
		t.Errorf("GetHomeDir() failed: %s", home)
	}
}

func TestGetEnonicDir(t *testing.T) {
	expected := filepath.Join(os.Getenv("HOME"), ".enonic")
	home := GetEnonicHome()
	if home != expected {
		t.Errorf("GetHomeDir() failed: %s", home)
	}
}

func TestGetEnonicDirOverridden(t *testing.T) {
	expected := "/root/.enonic"
	os.Setenv("ENONIC_CLI_HOME_PATH", "/root")
	defer os.Unsetenv("ENONIC_CLI_HOME_PATH")
	home := GetEnonicHome()
	if (home != expected) {
		t.Errorf("GetHomeDir() failed: %s", home)
	}
}

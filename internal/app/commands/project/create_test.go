package project

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/src-d/go-git.v4"
)

func TestGitClone_EmptyRepository(t *testing.T) {
	tempDir := t.TempDir()
	remotePath := filepath.Join(tempDir, "remote.git")
	destPath := filepath.Join(tempDir, "dest")

	if _, err := git.PlainInit(remotePath, true); err != nil {
		t.Fatalf("failed to create empty remote repository: %v", err)
	}

	gitClone(remotePath, destPath, "", "", "", "")

	if _, err := os.Stat(filepath.Join(destPath, ".git")); err != nil {
		t.Fatalf("expected cloned repository metadata at %s: %v", destPath, err)
	}
}

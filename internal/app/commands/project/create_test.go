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

	repo, err := cloneRepository(remotePath, destPath, nil, true)
	if err != nil {
		t.Fatalf("expected empty repository clone to succeed: %v", err)
	}
	if repo == nil {
		t.Fatal("expected repository instance, got nil")
	}

	if _, err := os.Stat(filepath.Join(destPath, ".git")); err != nil {
		t.Fatalf("expected cloned repository metadata at %s: %v", destPath, err)
	}

	remote, err := repo.Remote(UPSTREAM_NAME)
	if err != nil {
		t.Fatalf("expected remote '%s' to be configured: %v", UPSTREAM_NAME, err)
	}
	if got := remote.Config().URLs; len(got) != 1 || got[0] != remotePath {
		t.Fatalf("expected remote URL %s, got %v", remotePath, got)
	}
}

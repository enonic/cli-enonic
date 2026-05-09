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

func TestGitClone_EmptyRepositoryIntoNonEmptyDestination(t *testing.T) {
	tempDir := t.TempDir()
	remotePath := filepath.Join(tempDir, "remote.git")
	destPath := filepath.Join(tempDir, "dest")

	if _, err := git.PlainInit(remotePath, true); err != nil {
		t.Fatalf("failed to create empty remote repository: %v", err)
	}

	if err := os.MkdirAll(destPath, 0755); err != nil {
		t.Fatalf("failed to create destination directory: %v", err)
	}
	sentinelPath := filepath.Join(destPath, "keep.txt")
	if err := os.WriteFile(sentinelPath, []byte("keep"), 0644); err != nil {
		t.Fatalf("failed to seed destination with sentinel file: %v", err)
	}

	repo, err := cloneRepository(remotePath, destPath, nil, true)
	if err == nil {
		t.Fatal("expected clone to fail for empty remote into non-empty destination")
	}
	if repo != nil {
		t.Fatal("expected repository instance to be nil on clone failure")
	}
	if _, err := os.Stat(sentinelPath); err != nil {
		t.Fatalf("expected destination contents to be preserved on clone failure: %v", err)
	}
}

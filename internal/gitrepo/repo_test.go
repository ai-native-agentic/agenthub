package gitrepo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsValidHash(t *testing.T) {
	tests := []struct {
		name string
		hash string
		want bool
	}{
		{"valid short", "abc123", true},
		{"valid full", "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2", true},
		{"invalid chars", "ghi123", false},
		{"too short", "ab", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidHash(tt.hash); got != tt.want {
				t.Errorf("IsValidHash(%q) = %v, want %v", tt.hash, got, tt.want)
			}
		})
	}
}

func TestInit(t *testing.T) {
	if _, err := os.LookPath("git"); err != nil {
		t.Skip("git not found on PATH")
	}
	dir := t.TempDir()
	repoPath := filepath.Join(dir, "test-repo.git")
	r, err := Init(repoPath)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if r.Path != repoPath {
		t.Errorf("Repo.Path = %q, want %q", r.Path, repoPath)
	}
	// Verify bare repo exists
	if _, err := os.Stat(filepath.Join(repoPath, "HEAD")); err != nil {
		t.Errorf("HEAD file not created: %v", err)
	}
}

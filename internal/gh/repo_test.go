package gh

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestHasGitRepo_True(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Skip("git not available")
	}

	if !HasGitRepo(dir) {
		t.Error("expected HasGitRepo to return true for initialized repo")
	}
}

func TestHasGitRepo_False(t *testing.T) {
	dir := t.TempDir()
	if HasGitRepo(dir) {
		t.Error("expected HasGitRepo to return false for empty dir")
	}
}

func TestHasGitRemote_False(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Skip("git not available")
	}

	if HasGitRemote(dir) {
		t.Error("expected HasGitRemote to return false for repo without remote")
	}
}

func TestHasGitRemote_True(t *testing.T) {
	dir := t.TempDir()
	cmds := [][]string{
		{"git", "init"},
		{"git", "remote", "add", "origin", "https://github.com/test/repo.git"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Skip("git not available")
		}
	}

	if !HasGitRemote(dir) {
		t.Error("expected HasGitRemote to return true for repo with remote")
	}
}

func TestInitAndCreateRepo_AlreadyHasRemote(t *testing.T) {
	dir := t.TempDir()
	cmds := [][]string{
		{"git", "init"},
		{"git", "remote", "add", "origin", "https://github.com/test/existing.git"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Skip("git not available")
		}
	}

	url, err := InitAndCreateRepo(CreateRepoOptions{
		Name: "test/existing",
		Dir:  dir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://github.com/test/existing.git" {
		t.Errorf("expected existing remote URL, got %s", url)
	}
}

func TestHasGitRepo_NestedSubdir(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Skip("git not available")
	}

	// Create a nested subdirectory within the git repo.
	subdir := filepath.Join(dir, "a", "b", "c")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}

	if !HasGitRepo(subdir) {
		t.Error("expected HasGitRepo to return true for a subdirectory within a git repo")
	}
}

func TestGHCLIAvailable(t *testing.T) {
	// We can't control whether gh is installed, but calling GHCLIAvailable
	// should not panic regardless.
	result := GHCLIAvailable()
	// Just verify it returns a bool (true or false) without panicking.
	if result {
		t.Log("gh CLI is available and authenticated")
	} else {
		t.Log("gh CLI is not available or not authenticated")
	}
}

func TestInitAndCreateRepo_NoGitNoGH(t *testing.T) {
	if _, err := exec.LookPath("gh"); err != nil {
		t.Skip("gh CLI not available, skipping")
	}

	// Use a temporary directory that is NOT a git repo.
	dir := t.TempDir()

	// InitAndCreateRepo with a bogus repo name should fail
	// (gh repo create will fail for a non-existent org/invalid name).
	_, err := InitAndCreateRepo(CreateRepoOptions{
		Name:    "this-org-does-not-exist-999/test-repo-xyz",
		Dir:     dir,
		Private: true,
	})
	if err == nil {
		t.Fatal("expected InitAndCreateRepo to fail with invalid repo name")
	}
}

func TestRunGit(t *testing.T) {
	dir := t.TempDir()
	if err := runGit(dir, "init"); err != nil {
		t.Skip("git not available")
	}

	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		t.Error("expected .git directory after git init")
	}
}

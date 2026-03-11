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

func TestRunGit(t *testing.T) {
	dir := t.TempDir()
	if err := runGit(dir, "init"); err != nil {
		t.Skip("git not available")
	}

	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		t.Error("expected .git directory after git init")
	}
}

package gh

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// HasGitRepo checks if the directory is inside a git repository.
func HasGitRepo(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = dir
	out, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

// HasGitRemote checks if the git repo has an "origin" remote.
func HasGitRemote(dir string) bool {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = dir
	return cmd.Run() == nil
}

// GHCLIAvailable checks if the gh CLI is installed and authenticated.
func GHCLIAvailable() bool {
	cmd := exec.Command("gh", "auth", "status")
	return cmd.Run() == nil
}

// CreateRepoOptions configures how the GitHub repo is created.
type CreateRepoOptions struct {
	Name        string // e.g. "my-project" or "org/my-project"
	Description string
	Private     bool
	Dir         string // working directory
}

// InitAndCreateRepo initializes a git repo, creates it on GitHub, and sets up the remote.
// It handles three scenarios:
// 1. No git repo at all → git init + gh repo create
// 2. Git repo but no remote → gh repo create + add remote
// 3. Git repo with remote → no-op (returns nil)
func InitAndCreateRepo(opts CreateRepoOptions) (string, error) {
	hasRepo := HasGitRepo(opts.Dir)
	hasRemote := hasRepo && HasGitRemote(opts.Dir)

	if hasRemote {
		// Already set up, return the existing remote URL
		cmd := exec.Command("git", "remote", "get-url", "origin")
		cmd.Dir = opts.Dir
		out, _ := cmd.Output()
		return strings.TrimSpace(string(out)), nil
	}

	if !hasRepo {
		// Initialize git repo
		if err := runGit(opts.Dir, "init"); err != nil {
			return "", fmt.Errorf("git init: %w", err)
		}
		if err := runGit(opts.Dir, "checkout", "-b", "main"); err != nil {
			return "", fmt.Errorf("git checkout -b main: %w", err)
		}
	}

	// Build gh repo create command
	args := []string{"repo", "create", opts.Name, "--source", opts.Dir, "--push"}
	if opts.Private {
		args = append(args, "--private")
	} else {
		args = append(args, "--public")
	}
	if opts.Description != "" {
		args = append(args, "--description", opts.Description)
	}

	cmd := exec.Command("gh", args...)
	cmd.Dir = opts.Dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gh repo create: %w", err)
	}

	// Get the remote URL that gh set up
	getURL := exec.Command("git", "remote", "get-url", "origin")
	getURL.Dir = opts.Dir
	out, err := getURL.Output()
	if err != nil {
		return "", fmt.Errorf("getting remote URL: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}

// GetAuthToken returns the current gh CLI auth token.
func GetAuthToken() (string, error) {
	cmd := exec.Command("gh", "auth", "token")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("gh auth token: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// SetRepoSecret sets a GitHub Actions secret on a repository.
func SetRepoSecret(repoFullName, secretName, secretValue string) error {
	cmd := exec.Command("gh", "secret", "set", secretName, "--repo", repoFullName)
	cmd.Stdin = strings.NewReader(secretValue)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh secret set %s: %s: %w", secretName, strings.TrimSpace(string(out)), err)
	}
	return nil
}

// HasRepoSecret checks whether a secret exists on a repository.
func HasRepoSecret(repoFullName, secretName string) bool {
	cmd := exec.Command("gh", "secret", "list", "--repo", repoFullName)
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, secretName+"\t") || strings.HasPrefix(line, secretName+" ") {
			return true
		}
	}
	return false
}

// GetRepoFullName returns "owner/repo" for the git repo in dir.
func GetRepoFullName(dir string) (string, error) {
	cmd := exec.Command("gh", "repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting repo name: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// CommitAndPush stages given files, commits, and pushes in the project repo.
func CommitAndPush(dir string, message string, files ...string) error {
	args := append([]string{"add"}, files...)
	if err := runGit(dir, args...); err != nil {
		return fmt.Errorf("git add: %w", err)
	}

	// Check if there are staged changes
	checkCmd := exec.Command("git", "diff", "--cached", "--quiet")
	checkCmd.Dir = dir
	if err := checkCmd.Run(); err == nil {
		return nil // nothing to commit
	}

	if err := runGit(dir, "commit", "-m", message); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}

	if err := runGit(dir, "push"); err != nil {
		return fmt.Errorf("git push: %w", err)
	}

	return nil
}

func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

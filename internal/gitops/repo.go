package gitops

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Repo represents a local gitops repository on disk.
type Repo struct {
	Path string
}

// NewRepo creates a new Repo pointing at the given directory.
func NewRepo(path string) *Repo {
	return &Repo{Path: path}
}

// AppPath returns the filesystem path where an application's manifests live.
// Preview and staging environments are stored under apps/previews/{name},
// while production goes under apps/{name}.
func (r *Repo) AppPath(name, environment string) string {
	env := strings.ToLower(environment)
	if env == "preview" || env == "staging" {
		return filepath.Join(r.Path, "apps", "previews", name)
	}
	return filepath.Join(r.Path, "apps", name)
}

// AppExists reports whether the application directory already exists.
func (r *Repo) AppExists(name, environment string) bool {
	p := r.AppPath(name, environment)
	info, err := os.Stat(p)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// WriteApp copies rendered manifest files from srcDir into the application
// directory, removing any existing content first.
func (r *Repo) WriteApp(name, environment, srcDir string) error {
	dst := r.AppPath(name, environment)

	// Remove existing app directory if present.
	if err := os.RemoveAll(dst); err != nil {
		return fmt.Errorf("removing existing app dir: %w", err)
	}

	return copyDir(srcDir, dst)
}

// DeleteApp removes the application directory entirely.
func (r *Repo) DeleteApp(name, environment string) error {
	dst := r.AppPath(name, environment)
	return os.RemoveAll(dst)
}

// Pull runs git pull --rebase in the repository.
func (r *Repo) Pull() error {
	return r.git("pull", "--rebase")
}

// CommitChanges stages all changes and creates a commit with the given message.
func (r *Repo) CommitChanges(message string) error {
	if err := r.git("add", "-A"); err != nil {
		return err
	}
	return r.git("commit", "-m", message)
}

// Push pushes the current branch to the remote.
func (r *Repo) Push() error {
	return r.git("push")
}

// CreateBranchAndPush creates a new branch, commits all changes, pushes the
// branch upstream, then switches back to main.
func (r *Repo) CreateBranchAndPush(branch, message string) error {
	if err := r.git("checkout", "-b", branch); err != nil {
		return fmt.Errorf("creating branch: %w", err)
	}

	if err := r.CommitChanges(message); err != nil {
		// Try to get back to main even on error.
		_ = r.git("checkout", "main")
		return fmt.Errorf("committing changes: %w", err)
	}

	if err := r.git("push", "-u", "origin", branch); err != nil {
		_ = r.git("checkout", "main")
		return fmt.Errorf("pushing branch: %w", err)
	}

	return r.git("checkout", "main")
}

// CreatePR creates a pull request using the gh CLI and returns the PR URL.
func (r *Repo) CreatePR(title, body, branch string) (string, error) {
	cmd := exec.Command("gh", "pr", "create", "--title", title, "--body", body, "--head", branch)
	cmd.Dir = r.Path

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("creating PR: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// git runs a git command inside the repository directory.
func (r *Repo) git(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Path

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), strings.TrimSpace(string(out)), err)
	}
	return nil
}

// copyDir recursively copies the directory tree rooted at src to dst.
func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		return copyFile(path, target)
	})
}

// copyFile copies a single file from src to dst, preserving permissions.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

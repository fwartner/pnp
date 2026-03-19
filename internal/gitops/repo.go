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
// Uses scope and environment to determine the correct subdirectory:
//   - preview/staging → apps/previews/{name}
//   - customer scope  → apps/customers/{name}
//   - agency scope    → apps/agency/{name}
//   - private scope   → apps/agency/{name}
func (r *Repo) AppPath(name, environment, scope string) string {
	env := strings.ToLower(environment)
	if env == "preview" || env == "staging" {
		return filepath.Join(r.Path, "apps", "previews", name)
	}
	switch strings.ToLower(scope) {
	case "customer":
		return filepath.Join(r.Path, "apps", "customers", name)
	default:
		return filepath.Join(r.Path, "apps", "agency", name)
	}
}

// AppExists reports whether the application directory already exists.
func (r *Repo) AppExists(name, environment, scope string) bool {
	p := r.AppPath(name, environment, scope)
	info, err := os.Stat(p)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// WriteApp copies rendered manifest files from srcDir into the application
// directory, removing any existing content first.
func (r *Repo) WriteApp(name, environment, scope, srcDir string) error {
	dst := r.AppPath(name, environment, scope)

	// Remove existing app directory if present.
	if err := os.RemoveAll(dst); err != nil {
		return fmt.Errorf("removing existing app dir: %w", err)
	}

	return copyDir(srcDir, dst)
}

// DeleteApp removes the application directory entirely.
func (r *Repo) DeleteApp(name, environment, scope string) error {
	dst := r.AppPath(name, environment, scope)
	return os.RemoveAll(dst)
}

// Pull runs git pull --rebase in the repository.
func (r *Repo) Pull() error {
	return r.git("pull", "--rebase")
}

// CommitChanges stages all changes and creates a commit with the given message.
// Returns nil without error if there are no changes to commit.
func (r *Repo) CommitChanges(message string) error {
	if err := r.git("add", "-A"); err != nil {
		return err
	}

	// Check if there are staged changes before committing.
	cmd := exec.Command("git", "diff", "--cached", "--quiet")
	cmd.Dir = r.Path
	if err := cmd.Run(); err == nil {
		// Exit code 0 means no staged changes — nothing to commit.
		return nil
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

// AppInfo holds metadata about a deployed application.
type AppInfo struct {
	Name        string
	Scope       string
	Environment string
	Domain      string
	Type        string
}

// Commit holds a git commit's hash and message.
type Commit struct {
	Hash    string
	Message string
}

// ListApps walks the gitops repo's apps directory and returns info about each app.
func (r *Repo) ListApps() ([]AppInfo, error) {
	var apps []AppInfo

	scopeDirs := []struct {
		dir   string
		scope string
		env   string
	}{
		{"apps/customers", "customer", "production"},
		{"apps/agency", "agency", "production"},
		{"apps/previews", "previews", "preview"},
	}

	for _, sd := range scopeDirs {
		appsDir := filepath.Join(r.Path, sd.dir)
		entries, err := os.ReadDir(appsDir)
		if err != nil {
			continue // directory may not exist
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			app := AppInfo{
				Name:        entry.Name(),
				Scope:       sd.scope,
				Environment: sd.env,
			}

			// Try to parse values.yaml for domain
			valuesPath := filepath.Join(appsDir, entry.Name(), "values.yaml")
			if data, err := os.ReadFile(valuesPath); err == nil {
				for _, line := range strings.Split(string(data), "\n") {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "subdomain:") {
						app.Domain = strings.TrimSpace(strings.TrimPrefix(line, "subdomain:"))
					}
				}
			}

			// Try to parse Chart.yaml for type
			chartPath := filepath.Join(appsDir, entry.Name(), "Chart.yaml")
			if data, err := os.ReadFile(chartPath); err == nil {
				for _, line := range strings.Split(string(data), "\n") {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "name:") {
						app.Type = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
					}
				}
			}

			apps = append(apps, app)
		}
	}

	return apps, nil
}

// GitLog returns the most recent commits affecting the given path.
func (r *Repo) GitLog(appPath string, limit int) ([]Commit, error) {
	relPath, err := filepath.Rel(r.Path, appPath)
	if err != nil {
		relPath = appPath
	}

	cmd := exec.Command("git", "log", "--oneline", fmt.Sprintf("-n%d", limit), "--", relPath)
	cmd.Dir = r.Path
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}

	var commits []Commit
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		c := Commit{Hash: parts[0]}
		if len(parts) > 1 {
			c.Message = parts[1]
		}
		commits = append(commits, c)
	}
	return commits, nil
}

// Revert creates a revert commit for the given commit hash.
func (r *Repo) Revert(hash string) error {
	cmd := exec.Command("git", "revert", "--no-edit", hash)
	cmd.Dir = r.Path
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git revert %s: %s: %w", hash, strings.TrimSpace(string(out)), err)
	}
	return nil
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

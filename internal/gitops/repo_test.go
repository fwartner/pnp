package gitops

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppPath(t *testing.T) {
	repo := NewRepo("/tmp/gitops-repo")

	tests := []struct {
		name     string
		env      string
		expected string
	}{
		{"myapp", "preview", "/tmp/gitops-repo/apps/previews/myapp"},
		{"myapp", "Preview", "/tmp/gitops-repo/apps/previews/myapp"},
		{"myapp", "staging", "/tmp/gitops-repo/apps/previews/myapp"},
		{"myapp", "Staging", "/tmp/gitops-repo/apps/previews/myapp"},
		{"myapp", "production", "/tmp/gitops-repo/apps/myapp"},
		{"myapp", "Production", "/tmp/gitops-repo/apps/myapp"},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			got := repo.AppPath(tt.name, tt.env)
			if got != tt.expected {
				t.Errorf("AppPath(%q, %q) = %q, want %q", tt.name, tt.env, got, tt.expected)
			}
		})
	}
}

func TestAppExists(t *testing.T) {
	tmp := t.TempDir()
	repo := NewRepo(tmp)

	if repo.AppExists("myapp", "production") {
		t.Fatal("expected AppExists to return false for non-existent app")
	}

	appDir := repo.AppPath("myapp", "production")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if !repo.AppExists("myapp", "production") {
		t.Fatal("expected AppExists to return true after creating directory")
	}
}

func TestWriteApp(t *testing.T) {
	tmp := t.TempDir()
	repo := NewRepo(tmp)

	// Create source directory with files.
	srcDir := filepath.Join(tmp, "src")
	if err := os.MkdirAll(filepath.Join(srcDir, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "deployment.yaml"), []byte("kind: Deployment"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "subdir", "config.yaml"), []byte("key: value"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write app.
	if err := repo.WriteApp("myapp", "production", srcDir); err != nil {
		t.Fatalf("WriteApp failed: %v", err)
	}

	// Verify files were copied.
	appDir := repo.AppPath("myapp", "production")

	data, err := os.ReadFile(filepath.Join(appDir, "deployment.yaml"))
	if err != nil {
		t.Fatalf("reading deployment.yaml: %v", err)
	}
	if string(data) != "kind: Deployment" {
		t.Errorf("unexpected content: %q", data)
	}

	data, err = os.ReadFile(filepath.Join(appDir, "subdir", "config.yaml"))
	if err != nil {
		t.Fatalf("reading subdir/config.yaml: %v", err)
	}
	if string(data) != "key: value" {
		t.Errorf("unexpected content: %q", data)
	}

	// Write again to verify it replaces existing content.
	srcDir2 := filepath.Join(tmp, "src2")
	if err := os.MkdirAll(srcDir2, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir2, "service.yaml"), []byte("kind: Service"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := repo.WriteApp("myapp", "production", srcDir2); err != nil {
		t.Fatalf("WriteApp (second call) failed: %v", err)
	}

	// Old file should be gone.
	if _, err := os.Stat(filepath.Join(appDir, "deployment.yaml")); !os.IsNotExist(err) {
		t.Error("expected deployment.yaml to be removed after second WriteApp")
	}

	// New file should exist.
	data, err = os.ReadFile(filepath.Join(appDir, "service.yaml"))
	if err != nil {
		t.Fatalf("reading service.yaml: %v", err)
	}
	if string(data) != "kind: Service" {
		t.Errorf("unexpected content: %q", data)
	}
}

func TestDeleteApp(t *testing.T) {
	tmp := t.TempDir()
	repo := NewRepo(tmp)

	appDir := repo.AppPath("myapp", "preview")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "app.yaml"), []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	if !repo.AppExists("myapp", "preview") {
		t.Fatal("expected app to exist before delete")
	}

	if err := repo.DeleteApp("myapp", "preview"); err != nil {
		t.Fatalf("DeleteApp failed: %v", err)
	}

	if repo.AppExists("myapp", "preview") {
		t.Fatal("expected app to not exist after delete")
	}
}

func TestCommitAndPush(t *testing.T) {
	// Check that git is available.
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found, skipping")
	}

	tmp := t.TempDir()

	// Initialise a test git repo with an initial commit.
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = tmp
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command %v failed: %s: %v", args, out, err)
		}
	}

	run("git", "init")
	run("git", "checkout", "-b", "main")
	run("git", "config", "user.email", "test@test.com")
	run("git", "config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(tmp, "README.md"), []byte("init"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("git", "add", "-A")
	run("git", "commit", "-m", "initial commit")

	repo := NewRepo(tmp)

	// Create a new file and commit it.
	if err := os.WriteFile(filepath.Join(tmp, "test.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := repo.CommitChanges("add test file"); err != nil {
		t.Fatalf("CommitChanges failed: %v", err)
	}

	// Verify the commit appears in the log.
	cmd := exec.Command("git", "log", "--oneline")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git log failed: %v", err)
	}

	if got := string(out); !contains(got, "add test file") {
		t.Errorf("expected commit message in log, got:\n%s", got)
	}
}

func TestCreateBranchAndPush(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found, skipping")
	}

	tmp := t.TempDir()

	// Create a bare remote repo to push to.
	bare := filepath.Join(tmp, "remote.git")
	if err := os.MkdirAll(bare, 0o755); err != nil {
		t.Fatal(err)
	}
	runCmd := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command %v in %s failed: %s: %v", args, dir, out, err)
		}
	}

	runCmd(bare, "git", "init", "--bare")

	// Create local repo with initial commit.
	local := filepath.Join(tmp, "local")
	if err := os.MkdirAll(local, 0o755); err != nil {
		t.Fatal(err)
	}
	runCmd(local, "git", "init")
	runCmd(local, "git", "checkout", "-b", "main")
	runCmd(local, "git", "config", "user.email", "test@test.com")
	runCmd(local, "git", "config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(local, "README.md"), []byte("init"), 0o644); err != nil {
		t.Fatal(err)
	}
	runCmd(local, "git", "add", "-A")
	runCmd(local, "git", "commit", "-m", "initial commit")
	runCmd(local, "git", "remote", "add", "origin", bare)

	// Push main so remote has it.
	runCmd(local, "git", "push", "-u", "origin", "main")

	repo := NewRepo(local)

	// Create a file to be committed on the new branch.
	if err := os.WriteFile(filepath.Join(local, "feature.txt"), []byte("feature"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := repo.CreateBranchAndPush("feature-branch", "add feature file")
	if err != nil {
		t.Fatalf("CreateBranchAndPush failed: %v", err)
	}

	// Verify the branch was created.
	cmd := exec.Command("git", "branch", "--list", "feature-branch")
	cmd.Dir = local
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git branch --list failed: %v", err)
	}
	if !contains(string(out), "feature-branch") {
		t.Errorf("expected feature-branch in branch list, got: %q", string(out))
	}

	// Verify the commit exists on that branch.
	cmd = exec.Command("git", "log", "feature-branch", "--oneline")
	cmd.Dir = local
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git log failed: %v", err)
	}
	if !contains(string(out), "add feature file") {
		t.Errorf("expected commit message on feature-branch, got: %q", string(out))
	}

	// Verify we're back on main.
	cmd = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = local
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git rev-parse failed: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "main" {
		t.Errorf("expected to be on main, got %q", got)
	}
}

func TestPull_NoRemote(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found, skipping")
	}

	tmp := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmp
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %s: %v", out, err)
	}

	repo := NewRepo(tmp)
	err := repo.Pull()
	if err == nil {
		t.Fatal("expected Pull to fail on repo without remote")
	}
}

func TestCommitChanges_NoChanges(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found, skipping")
	}

	tmp := t.TempDir()

	runCmd := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = tmp
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command %v failed: %s: %v", args, out, err)
		}
	}

	runCmd("git", "init")
	runCmd("git", "checkout", "-b", "main")
	runCmd("git", "config", "user.email", "test@test.com")
	runCmd("git", "config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(tmp, "README.md"), []byte("init"), 0o644); err != nil {
		t.Fatal(err)
	}
	runCmd("git", "add", "-A")
	runCmd("git", "commit", "-m", "initial commit")

	repo := NewRepo(tmp)
	err := repo.CommitChanges("empty commit")
	if err == nil {
		t.Fatal("expected CommitChanges to fail when nothing is staged")
	}
}

func TestWriteApp_NestedDirectories(t *testing.T) {
	tmp := t.TempDir()
	repo := NewRepo(tmp)

	// Create deeply nested source directory.
	srcDir := filepath.Join(tmp, "src")
	deepDir := filepath.Join(srcDir, "templates", "sub", "deep")
	if err := os.MkdirAll(deepDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "root.yaml"), []byte("root: true"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "templates", "mid.yaml"), []byte("mid: true"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deepDir, "file.yaml"), []byte("deep: true"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := repo.WriteApp("nested-app", "production", srcDir); err != nil {
		t.Fatalf("WriteApp failed: %v", err)
	}

	appDir := repo.AppPath("nested-app", "production")

	// Verify all files were copied at the right depth.
	checks := []struct {
		path    string
		content string
	}{
		{filepath.Join(appDir, "root.yaml"), "root: true"},
		{filepath.Join(appDir, "templates", "mid.yaml"), "mid: true"},
		{filepath.Join(appDir, "templates", "sub", "deep", "file.yaml"), "deep: true"},
	}

	for _, c := range checks {
		data, err := os.ReadFile(c.path)
		if err != nil {
			t.Fatalf("reading %s: %v", c.path, err)
		}
		if string(data) != c.content {
			t.Errorf("file %s: got %q, want %q", c.path, string(data), c.content)
		}
	}
}

func TestDeleteApp_NonExistent(t *testing.T) {
	tmp := t.TempDir()
	repo := NewRepo(tmp)

	// Deleting an app that doesn't exist should not return an error (RemoveAll is idempotent).
	err := repo.DeleteApp("does-not-exist", "production")
	if err != nil {
		t.Fatalf("DeleteApp on non-existent app returned error: %v", err)
	}
}

func TestAppPath_AllEnvironments(t *testing.T) {
	repo := NewRepo("/tmp/test-repo")

	tests := []struct {
		name     string
		appName  string
		env      string
		expected string
	}{
		{"preview lowercase", "myapp", "preview", "/tmp/test-repo/apps/previews/myapp"},
		{"preview mixed case", "myapp", "Preview", "/tmp/test-repo/apps/previews/myapp"},
		{"preview uppercase", "myapp", "PREVIEW", "/tmp/test-repo/apps/previews/myapp"},
		{"staging lowercase", "myapp", "staging", "/tmp/test-repo/apps/previews/myapp"},
		{"staging mixed case", "myapp", "Staging", "/tmp/test-repo/apps/previews/myapp"},
		{"staging uppercase", "myapp", "STAGING", "/tmp/test-repo/apps/previews/myapp"},
		{"production lowercase", "myapp", "production", "/tmp/test-repo/apps/myapp"},
		{"production mixed case", "myapp", "Production", "/tmp/test-repo/apps/myapp"},
		{"production uppercase", "myapp", "PRODUCTION", "/tmp/test-repo/apps/myapp"},
		{"empty env", "myapp", "", "/tmp/test-repo/apps/myapp"},
		{"other env", "myapp", "development", "/tmp/test-repo/apps/myapp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := repo.AppPath(tt.appName, tt.env)
			if got != tt.expected {
				t.Errorf("AppPath(%q, %q) = %q, want %q", tt.appName, tt.env, got, tt.expected)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

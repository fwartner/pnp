package gitops

import (
	"os"
	"os/exec"
	"path/filepath"
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

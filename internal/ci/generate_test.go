package ci

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fwartner/pnp/internal/config"
)

func TestGenerateLaravelWorkflow(t *testing.T) {
	dir := t.TempDir()
	image := "ghcr.io/example/laravel-app"
	gitopsRemote := "https://github.com/org/gitops.git"
	appName := "laravel-app"

	err := GenerateWorkflow("laravel-web", image, gitopsRemote, appName, dir)
	if err != nil {
		t.Fatalf("GenerateWorkflow returned error: %v", err)
	}

	outPath := filepath.Join(dir, ".github", "workflows", "deploy.yml")
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read generated workflow: %v", err)
	}

	content := string(data)

	for _, want := range []string{
		"Build & Deploy",
		"docker/build-push-action",
		"docker/login-action",
		"docker/metadata-action",
		"setup-buildx-action",
		"ghcr.io",
		image,
		"cache-from: type=gha",
		"org/gitops",
		"GITOPS_TOKEN",
		"apps/customer/laravel-app/values.yaml",
		"apps/agency/laravel-app/values.yaml",
		"apps/previews/laravel-app/values.yaml",
		"deploy(laravel-app)",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("workflow should contain %q", want)
		}
	}
}

func TestGenerateStrapiWorkflow(t *testing.T) {
	dir := t.TempDir()
	image := "ghcr.io/example/strapi-app"

	err := GenerateWorkflow("strapi", image, "https://github.com/org/gitops.git", "strapi-app", dir)
	if err != nil {
		t.Fatalf("GenerateWorkflow returned error: %v", err)
	}

	outPath := filepath.Join(dir, ".github", "workflows", "deploy.yml")
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read generated workflow: %v", err)
	}

	if !strings.Contains(string(data), image) {
		t.Errorf("strapi workflow should contain image reference %q", image)
	}
}

func TestGenerateWorkflow_UnknownType(t *testing.T) {
	dir := t.TempDir()
	err := GenerateWorkflow("unknown", "ghcr.io/example/app", "https://github.com/org/gitops.git", "app", dir)
	if err == nil {
		t.Fatal("expected error for unknown project type")
	}
	if !strings.Contains(err.Error(), "unsupported project type") {
		t.Errorf("expected 'unsupported project type' in error, got: %v", err)
	}
}

func TestGenerateWorkflow_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()

	workflowsDir := filepath.Join(dir, ".github", "workflows")
	if _, err := os.Stat(workflowsDir); !os.IsNotExist(err) {
		t.Fatal("expected .github/workflows to not exist before GenerateWorkflow")
	}

	err := GenerateWorkflow("laravel-api", "ghcr.io/example/api", "https://github.com/org/gitops.git", "api-app", dir)
	if err != nil {
		t.Fatalf("GenerateWorkflow returned error: %v", err)
	}

	info, err := os.Stat(workflowsDir)
	if err != nil {
		t.Fatalf("expected .github/workflows to exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected .github/workflows to be a directory")
	}
}

func TestExtractGitHubRepo(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://github.com/org/gitops.git", "org/gitops"},
		{"https://github.com/fwartner/pixelandprocess-gitops.git", "fwartner/pixelandprocess-gitops"},
		{"git@github.com:org/gitops.git", "org/gitops"},
		{"git@github.com:fwartner/pixelandprocess-gitops.git", "fwartner/pixelandprocess-gitops"},
		{"https://github.com/org/gitops", "org/gitops"},
		{"something-else", "something-else"},
	}
	for _, tt := range tests {
		got := extractGitHubRepo(tt.input)
		if got != tt.want {
			t.Errorf("extractGitHubRepo(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestGenerateDockerfile_Laravel(t *testing.T) {
	dir := t.TempDir()
	err := GenerateDockerfile("laravel-web", config.OctaneConfig{}, dir)
	if err != nil {
		t.Fatalf("GenerateDockerfile returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "Dockerfile"))
	if err != nil {
		t.Fatalf("failed to read Dockerfile: %v", err)
	}
	content := string(data)

	for _, want := range []string{
		"ghcr.io/fwartner/pnp/laravel-fpm:latest",
		"composer",
		"npm run build",
		"storage/framework",
		"EXPOSE 80",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("Laravel Dockerfile should contain %q", want)
		}
	}

	// Verify .dockerignore was created
	ignore, err := os.ReadFile(filepath.Join(dir, ".dockerignore"))
	if err != nil {
		t.Fatalf("failed to read .dockerignore: %v", err)
	}
	for _, want := range []string{"node_modules", "vendor", ".git", ".env"} {
		if !strings.Contains(string(ignore), want) {
			t.Errorf(".dockerignore should contain %q", want)
		}
	}
}

func TestGenerateDockerfile_LaravelOctaneFrankenPHP(t *testing.T) {
	dir := t.TempDir()
	err := GenerateDockerfile("laravel-web", config.OctaneConfig{Enabled: true, Server: "frankenphp"}, dir)
	if err != nil {
		t.Fatalf("GenerateDockerfile returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "Dockerfile"))
	if err != nil {
		t.Fatalf("failed to read Dockerfile: %v", err)
	}
	content := string(data)

	for _, want := range []string{
		"ghcr.io/fwartner/pnp/laravel-frankenphp:latest",
		"octane:start",
		"--server=frankenphp",
		"EXPOSE 8000",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("FrankenPHP Dockerfile should contain %q", want)
		}
	}
}

func TestGenerateDockerfile_LaravelOctaneSwoole(t *testing.T) {
	dir := t.TempDir()
	err := GenerateDockerfile("laravel-web", config.OctaneConfig{Enabled: true, Server: "swoole"}, dir)
	if err != nil {
		t.Fatalf("GenerateDockerfile returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "Dockerfile"))
	if err != nil {
		t.Fatalf("failed to read Dockerfile: %v", err)
	}

	if !strings.Contains(string(data), "--server=swoole") {
		t.Error("Swoole Dockerfile should contain --server=swoole")
	}
}

func TestGenerateDockerfile_LaravelOctaneRoadrunner(t *testing.T) {
	dir := t.TempDir()
	err := GenerateDockerfile("laravel-web", config.OctaneConfig{Enabled: true, Server: "roadrunner"}, dir)
	if err != nil {
		t.Fatalf("GenerateDockerfile returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "Dockerfile"))
	if err != nil {
		t.Fatalf("failed to read Dockerfile: %v", err)
	}
	content := string(data)

	for _, want := range []string{"--server=roadrunner", "ghcr.io/fwartner/pnp/laravel-roadrunner:latest"} {
		if !strings.Contains(content, want) {
			t.Errorf("RoadRunner Dockerfile should contain %q", want)
		}
	}
}

func TestGenerateDockerfile_Nextjs(t *testing.T) {
	dir := t.TempDir()
	err := GenerateDockerfile("nextjs-fullstack", config.OctaneConfig{}, dir)
	if err != nil {
		t.Fatalf("GenerateDockerfile returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "Dockerfile"))
	if err != nil {
		t.Fatalf("failed to read Dockerfile: %v", err)
	}
	content := string(data)

	for _, want := range []string{
		"ghcr.io/fwartner/pnp/nextjs:latest",
		".next/standalone",
		".next/static",
		"server.js",
		"nextjs", // non-root user
	} {
		if !strings.Contains(content, want) {
			t.Errorf("Next.js Dockerfile should contain %q", want)
		}
	}

	// Check .dockerignore
	ignore, err := os.ReadFile(filepath.Join(dir, ".dockerignore"))
	if err != nil {
		t.Fatalf("failed to read .dockerignore: %v", err)
	}
	if !strings.Contains(string(ignore), "node_modules") {
		t.Error(".dockerignore should contain node_modules")
	}
}

func TestGenerateDockerfile_Strapi(t *testing.T) {
	dir := t.TempDir()
	err := GenerateDockerfile("strapi", config.OctaneConfig{}, dir)
	if err != nil {
		t.Fatalf("GenerateDockerfile returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "Dockerfile"))
	if err != nil {
		t.Fatalf("failed to read Dockerfile: %v", err)
	}
	content := string(data)

	for _, want := range []string{
		"ghcr.io/fwartner/pnp/strapi:latest",
		"npm run build",
		"\"start\"",
		"strapi", // non-root user
	} {
		if !strings.Contains(content, want) {
			t.Errorf("Strapi Dockerfile should contain %q", want)
		}
	}
}

func TestGenerateDockerfile_UnknownType(t *testing.T) {
	dir := t.TempDir()
	err := GenerateDockerfile("unknown", config.OctaneConfig{}, dir)
	if err == nil {
		t.Fatal("expected error for unknown project type")
	}
}

func TestGenerateDockerfile_DockerignoreNotOverwritten(t *testing.T) {
	dir := t.TempDir()

	// Create existing .dockerignore
	existing := []byte("my-custom-ignore\n")
	os.WriteFile(filepath.Join(dir, ".dockerignore"), existing, 0o644)

	err := GenerateDockerfile("laravel-web", config.OctaneConfig{}, dir)
	if err != nil {
		t.Fatalf("GenerateDockerfile returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".dockerignore"))
	if err != nil {
		t.Fatalf("failed to read .dockerignore: %v", err)
	}

	if string(data) != "my-custom-ignore\n" {
		t.Error("existing .dockerignore should not be overwritten")
	}
}

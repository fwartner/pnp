package ci

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateLaravelWorkflow(t *testing.T) {
	dir := t.TempDir()
	image := "ghcr.io/example/laravel-app"

	err := GenerateWorkflow("laravel-web", image, dir)
	if err != nil {
		t.Fatalf("GenerateWorkflow returned error: %v", err)
	}

	outPath := filepath.Join(dir, ".github", "workflows", "deploy.yml")
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read generated workflow: %v", err)
	}

	content := string(data)

	if !strings.Contains(content, "docker") {
		t.Error("workflow should contain 'docker'")
	}

	if !strings.Contains(content, image) {
		t.Errorf("workflow should contain image reference %q", image)
	}
}

func TestGenerateStrapiWorkflow(t *testing.T) {
	dir := t.TempDir()
	image := "ghcr.io/example/strapi-app"

	err := GenerateWorkflow("strapi", image, dir)
	if err != nil {
		t.Fatalf("GenerateWorkflow returned error: %v", err)
	}

	outPath := filepath.Join(dir, ".github", "workflows", "deploy.yml")
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read generated workflow: %v", err)
	}

	content := string(data)

	if !strings.Contains(content, "node") {
		t.Error("strapi workflow should contain 'node'")
	}

	if !strings.Contains(content, image) {
		t.Errorf("strapi workflow should contain image reference %q", image)
	}
}

func TestGenerateWorkflow_UnknownType(t *testing.T) {
	dir := t.TempDir()
	err := GenerateWorkflow("unknown", "ghcr.io/example/app", dir)
	if err == nil {
		t.Fatal("expected error for unknown project type")
	}
	if !strings.Contains(err.Error(), "unsupported project type") {
		t.Errorf("expected 'unsupported project type' in error, got: %v", err)
	}
}

func TestGenerateWorkflow_FileContents(t *testing.T) {
	dir := t.TempDir()
	image := "ghcr.io/example/laravel-app"

	err := GenerateWorkflow("laravel-web", image, dir)
	if err != nil {
		t.Fatalf("GenerateWorkflow returned error: %v", err)
	}

	outPath := filepath.Join(dir, ".github", "workflows", "deploy.yml")
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read generated workflow: %v", err)
	}

	content := string(data)

	expectedSections := []string{
		"Setup PHP",
		"composer install",
		"Setup Node.js",
		"npm ci",
		"Build and push Docker image",
		"Log in to GHCR",
	}

	for _, section := range expectedSections {
		if !strings.Contains(content, section) {
			t.Errorf("laravel workflow should contain %q", section)
		}
	}
}

func TestGenerateWorkflow_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()

	// The .github/workflows directory does not exist yet.
	workflowsDir := filepath.Join(dir, ".github", "workflows")
	if _, err := os.Stat(workflowsDir); !os.IsNotExist(err) {
		t.Fatal("expected .github/workflows to not exist before GenerateWorkflow")
	}

	err := GenerateWorkflow("laravel-api", "ghcr.io/example/api", dir)
	if err != nil {
		t.Fatalf("GenerateWorkflow returned error: %v", err)
	}

	// Verify directory was created.
	info, err := os.Stat(workflowsDir)
	if err != nil {
		t.Fatalf("expected .github/workflows to exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected .github/workflows to be a directory")
	}

	// Verify the file exists.
	outPath := filepath.Join(workflowsDir, "deploy.yml")
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("expected deploy.yml to exist: %v", err)
	}
}

func TestGenerateNextjsWorkflow(t *testing.T) {
	dir := t.TempDir()
	image := "ghcr.io/example/nextjs-app"

	err := GenerateWorkflow("nextjs-fullstack", image, dir)
	if err != nil {
		t.Fatalf("GenerateWorkflow returned error: %v", err)
	}

	outPath := filepath.Join(dir, ".github", "workflows", "deploy.yml")
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read generated workflow: %v", err)
	}

	content := string(data)

	if !strings.Contains(content, "node") {
		t.Error("workflow should contain 'node'")
	}
}

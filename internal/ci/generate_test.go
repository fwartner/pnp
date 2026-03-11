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

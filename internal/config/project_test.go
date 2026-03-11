package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProjectConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".cluster.yaml")

	content := []byte(`name: acme-corp
type: laravel-web
environment: preview
domain: acme-corp.preview.pixelandprocess.de
image: ghcr.io/fwartner/acme-corp
database:
  enabled: true
  size: 5Gi
  name: acme
redis:
  enabled: true
infisical:
  projectSlug: customer-apps-f-jq3
  envSlug: prod
  secretsPath: /acme-corp/db
resources:
  cpu: 100m
  memory: 256Mi
`)
	os.WriteFile(path, content, 0644)

	cfg, err := LoadProjectConfigFrom(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Name != "acme-corp" {
		t.Errorf("expected acme-corp, got %s", cfg.Name)
	}
	if cfg.Type != "laravel-web" {
		t.Errorf("expected laravel-web, got %s", cfg.Type)
	}
	if cfg.Database.Size != "5Gi" {
		t.Errorf("expected 5Gi, got %s", cfg.Database.Size)
	}
}

func TestLoadProjectConfig_NotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".cluster.yaml")

	_, err := LoadProjectConfigFrom(path)
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestSaveProjectConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".cluster.yaml")

	cfg := ProjectConfig{
		Name:        "test-app",
		Type:        "nextjs-static",
		Environment: "preview",
		Domain:      "test.preview.pixelandprocess.de",
		Image:       "ghcr.io/test/app",
	}

	err := SaveProjectConfigTo(cfg, path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, err := LoadProjectConfigFrom(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded.Name != "test-app" {
		t.Errorf("expected test-app, got %s", loaded.Name)
	}
}

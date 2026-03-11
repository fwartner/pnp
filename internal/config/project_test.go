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

func TestLoadProjectConfig_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".cluster.yaml")

	invalidYAML := []byte(":\n  broken:\n\t- yaml {{{\n")
	os.WriteFile(path, invalidYAML, 0644)

	_, err := LoadProjectConfigFrom(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestSaveProjectConfig_AllFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".cluster.yaml")

	original := ProjectConfig{
		Name:        "full-app",
		Type:        "laravel-web",
		Environment: "staging",
		Domain:      "full-app.staging.example.com",
		Image:       "ghcr.io/org/full-app",
		Database: DatabaseConfig{
			Enabled: true,
			Size:    "10Gi",
			Name:    "full_db",
		},
		Redis: RedisConfig{Enabled: true},
		Infisical: ProjectInfisical{
			ProjectSlug: "proj-slug",
			EnvSlug:     "staging",
			SecretsPath: "/full-app/db",
		},
		Resources: ResourcesConfig{
			CPU:    "500m",
			Memory: "1Gi",
		},
		CI: CIConfig{Enabled: true},
	}

	if err := SaveProjectConfigTo(original, path); err != nil {
		t.Fatalf("save error: %v", err)
	}

	loaded, err := LoadProjectConfigFrom(path)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}

	if loaded.Name != original.Name {
		t.Errorf("Name: expected %s, got %s", original.Name, loaded.Name)
	}
	if loaded.Type != original.Type {
		t.Errorf("Type: expected %s, got %s", original.Type, loaded.Type)
	}
	if loaded.Environment != original.Environment {
		t.Errorf("Environment: expected %s, got %s", original.Environment, loaded.Environment)
	}
	if loaded.Domain != original.Domain {
		t.Errorf("Domain: expected %s, got %s", original.Domain, loaded.Domain)
	}
	if loaded.Image != original.Image {
		t.Errorf("Image: expected %s, got %s", original.Image, loaded.Image)
	}
	if loaded.Database.Enabled != original.Database.Enabled {
		t.Errorf("Database.Enabled: expected %v, got %v", original.Database.Enabled, loaded.Database.Enabled)
	}
	if loaded.Database.Size != original.Database.Size {
		t.Errorf("Database.Size: expected %s, got %s", original.Database.Size, loaded.Database.Size)
	}
	if loaded.Database.Name != original.Database.Name {
		t.Errorf("Database.Name: expected %s, got %s", original.Database.Name, loaded.Database.Name)
	}
	if loaded.Redis.Enabled != original.Redis.Enabled {
		t.Errorf("Redis.Enabled: expected %v, got %v", original.Redis.Enabled, loaded.Redis.Enabled)
	}
	if loaded.Infisical.ProjectSlug != original.Infisical.ProjectSlug {
		t.Errorf("Infisical.ProjectSlug: expected %s, got %s", original.Infisical.ProjectSlug, loaded.Infisical.ProjectSlug)
	}
	if loaded.Infisical.EnvSlug != original.Infisical.EnvSlug {
		t.Errorf("Infisical.EnvSlug: expected %s, got %s", original.Infisical.EnvSlug, loaded.Infisical.EnvSlug)
	}
	if loaded.Infisical.SecretsPath != original.Infisical.SecretsPath {
		t.Errorf("Infisical.SecretsPath: expected %s, got %s", original.Infisical.SecretsPath, loaded.Infisical.SecretsPath)
	}
	if loaded.Resources.CPU != original.Resources.CPU {
		t.Errorf("Resources.CPU: expected %s, got %s", original.Resources.CPU, loaded.Resources.CPU)
	}
	if loaded.Resources.Memory != original.Resources.Memory {
		t.Errorf("Resources.Memory: expected %s, got %s", original.Resources.Memory, loaded.Resources.Memory)
	}
	if loaded.CI.Enabled != original.CI.Enabled {
		t.Errorf("CI.Enabled: expected %v, got %v", original.CI.Enabled, loaded.CI.Enabled)
	}
}

func TestProjectConfig_EmptyOptionalFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".cluster.yaml")

	// Only required fields
	content := []byte(`name: minimal-app
type: nextjs-static
environment: production
`)
	os.WriteFile(path, content, 0644)

	cfg, err := LoadProjectConfigFrom(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Name != "minimal-app" {
		t.Errorf("Name: expected minimal-app, got %s", cfg.Name)
	}
	if cfg.Type != "nextjs-static" {
		t.Errorf("Type: expected nextjs-static, got %s", cfg.Type)
	}
	// Verify zero values for optional fields
	if cfg.Domain != "" {
		t.Errorf("Domain: expected empty, got %s", cfg.Domain)
	}
	if cfg.Image != "" {
		t.Errorf("Image: expected empty, got %s", cfg.Image)
	}
	if cfg.Database.Enabled != false {
		t.Error("Database.Enabled: expected false")
	}
	if cfg.Database.Size != "" {
		t.Errorf("Database.Size: expected empty, got %s", cfg.Database.Size)
	}
	if cfg.Database.Name != "" {
		t.Errorf("Database.Name: expected empty, got %s", cfg.Database.Name)
	}
	if cfg.Redis.Enabled != false {
		t.Error("Redis.Enabled: expected false")
	}
	if cfg.Resources.CPU != "" {
		t.Errorf("Resources.CPU: expected empty, got %s", cfg.Resources.CPU)
	}
	if cfg.Resources.Memory != "" {
		t.Errorf("Resources.Memory: expected empty, got %s", cfg.Resources.Memory)
	}
	if cfg.CI.Enabled != false {
		t.Error("CI.Enabled: expected false")
	}
}

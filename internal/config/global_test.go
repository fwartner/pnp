package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGlobalConfig_Default(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".pnp.yaml")

	cfg, err := LoadGlobalConfigFrom(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Defaults.Domain != "pixelandprocess.de" {
		t.Errorf("expected default domain pixelandprocess.de, got %s", cfg.Defaults.Domain)
	}
	if cfg.Defaults.ImageRegistry != "ghcr.io" {
		t.Errorf("expected default registry ghcr.io, got %s", cfg.Defaults.ImageRegistry)
	}
}

func TestLoadGlobalConfig_FromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".pnp.yaml")

	content := []byte(`gitopsRepo: /tmp/test-gitops
gitopsRemote: https://github.com/test/gitops.git
infisical:
  host: https://vault.test.de
  token: test-token
defaults:
  domain: test.de
  imageRegistry: ghcr.io
  githubOrg: testorg
`)
	os.WriteFile(path, content, 0644)

	cfg, err := LoadGlobalConfigFrom(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.GitopsRepo != "/tmp/test-gitops" {
		t.Errorf("expected /tmp/test-gitops, got %s", cfg.GitopsRepo)
	}
	if cfg.Infisical.Token != "test-token" {
		t.Errorf("expected test-token, got %s", cfg.Infisical.Token)
	}
	if cfg.Defaults.GithubOrg != "testorg" {
		t.Errorf("expected testorg, got %s", cfg.Defaults.GithubOrg)
	}
}

func TestSaveAndLoadGlobalConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".pnp.yaml")

	original := GlobalConfig{
		GitopsRepo:   "/tmp/my-gitops",
		GitopsRemote: "https://github.com/org/gitops.git",
		Infisical: InfisicalConfig{
			Host:            "https://vault.example.com",
			Token:           "secret-token-123",
			MailProjectSlug: "mail-slug-abc",
		},
		Defaults: DefaultsConfig{
			Domain:        "example.com",
			ImageRegistry: "docker.io",
			GithubOrg:     "myorg",
		},
	}

	if err := SaveGlobalConfigTo(original, path); err != nil {
		t.Fatalf("save error: %v", err)
	}

	loaded, err := LoadGlobalConfigFrom(path)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}

	if loaded.GitopsRepo != original.GitopsRepo {
		t.Errorf("GitopsRepo: expected %s, got %s", original.GitopsRepo, loaded.GitopsRepo)
	}
	if loaded.GitopsRemote != original.GitopsRemote {
		t.Errorf("GitopsRemote: expected %s, got %s", original.GitopsRemote, loaded.GitopsRemote)
	}
	if loaded.Infisical.Host != original.Infisical.Host {
		t.Errorf("Infisical.Host: expected %s, got %s", original.Infisical.Host, loaded.Infisical.Host)
	}
	if loaded.Infisical.Token != original.Infisical.Token {
		t.Errorf("Infisical.Token: expected %s, got %s", original.Infisical.Token, loaded.Infisical.Token)
	}
	if loaded.Infisical.MailProjectSlug != original.Infisical.MailProjectSlug {
		t.Errorf("Infisical.MailProjectSlug: expected %s, got %s", original.Infisical.MailProjectSlug, loaded.Infisical.MailProjectSlug)
	}
	if loaded.Defaults.Domain != original.Defaults.Domain {
		t.Errorf("Defaults.Domain: expected %s, got %s", original.Defaults.Domain, loaded.Defaults.Domain)
	}
	if loaded.Defaults.ImageRegistry != original.Defaults.ImageRegistry {
		t.Errorf("Defaults.ImageRegistry: expected %s, got %s", original.Defaults.ImageRegistry, loaded.Defaults.ImageRegistry)
	}
	if loaded.Defaults.GithubOrg != original.Defaults.GithubOrg {
		t.Errorf("Defaults.GithubOrg: expected %s, got %s", original.Defaults.GithubOrg, loaded.Defaults.GithubOrg)
	}
}

func TestLoadGlobalConfig_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".pnp.yaml")

	invalidYAML := []byte(":\n  invalid:\n\t- broken yaml {{{\n")
	os.WriteFile(path, invalidYAML, 0644)

	_, err := LoadGlobalConfigFrom(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestGlobalConfigPath(t *testing.T) {
	path, err := GlobalConfigPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Base(path) != ".pnp.yaml" {
		t.Errorf("expected path ending in .pnp.yaml, got %s", path)
	}
	if !filepath.IsAbs(path) {
		t.Errorf("expected absolute path, got %s", path)
	}
}

func TestDefaultGlobalConfig_InfisicalHost(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".pnp.yaml")

	// Load from non-existent file to get defaults
	cfg, err := LoadGlobalConfigFrom(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Infisical.Host != "https://vault.intern.pixelandprocess.de" {
		t.Errorf("expected default Infisical host, got %s", cfg.Infisical.Host)
	}
	if cfg.Infisical.MailProjectSlug != "cluster-shared-ys-zj" {
		t.Errorf("expected default MailProjectSlug, got %s", cfg.Infisical.MailProjectSlug)
	}
}

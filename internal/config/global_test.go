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

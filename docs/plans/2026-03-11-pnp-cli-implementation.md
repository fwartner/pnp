# PnP CLI Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a Go CLI tool (`pnp`) that automates Kubernetes deployments by detecting project type, running an interactive wizard, generating gitops manifests, and pushing to the gitops repo.

**Architecture:** Cobra CLI with Bubbletea/Huh for interactive wizard. Templates rendered via Go `text/template`. Gitops repo operations via shell exec (git). Infisical secrets via REST API. Config stored in `~/.pnp.yaml` (global) and `.cluster.yaml` (per-project).

**Tech Stack:** Go 1.22+, Cobra, Bubbletea, Huh, gopkg.in/yaml.v3, goreleaser for distribution.

**Reference gitops repo:** `/Users/fwartner/Projects/Development/pixelandprocess-gitops`

---

## Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `cmd/root.go`
- Create: `cmd/version.go`

**Step 1: Initialize Go module**

Run:
```bash
cd /Users/fwartner/Projects/Development/cluster-manager
go mod init github.com/fwartner/pnp
```

**Step 2: Install dependencies**

Run:
```bash
go get github.com/spf13/cobra@latest
go get github.com/charmbracelet/huh@latest
go get github.com/charmbracelet/lipgloss@latest
go get gopkg.in/yaml.v3
```

**Step 3: Create main.go**

```go
// main.go
package main

import "github.com/fwartner/pnp/cmd"

func main() {
	cmd.Execute()
}
```

**Step 4: Create cmd/root.go**

```go
// cmd/root.go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "pnp",
	Short: "Pixel & Process deployment manager",
	Long:  "A CLI tool to manage Kubernetes deployments for Pixel & Process projects.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

**Step 5: Create cmd/version.go**

```go
// cmd/version.go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("pnp %s (%s)\n", version, commit)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
```

**Step 6: Verify it compiles and runs**

Run:
```bash
go run main.go version
```
Expected: `pnp dev (none)`

**Step 7: Commit**

```bash
git add -A
git commit -m "feat: scaffold Go project with Cobra CLI"
```

---

## Task 2: Global Config (`~/.pnp.yaml`)

**Files:**
- Create: `internal/config/global.go`
- Create: `internal/config/global_test.go`

**Step 1: Write test for global config loading**

```go
// internal/config/global_test.go
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -v -run TestLoadGlobalConfig`
Expected: FAIL (package doesn't exist)

**Step 3: Implement global config**

```go
// internal/config/global.go
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type GlobalConfig struct {
	GitopsRepo   string          `yaml:"gitopsRepo"`
	GitopsRemote string          `yaml:"gitopsRemote"`
	Infisical    InfisicalConfig `yaml:"infisical"`
	Defaults     DefaultsConfig  `yaml:"defaults"`
}

type InfisicalConfig struct {
	Host  string `yaml:"host"`
	Token string `yaml:"token"`
}

type DefaultsConfig struct {
	Domain        string `yaml:"domain"`
	ImageRegistry string `yaml:"imageRegistry"`
	GithubOrg     string `yaml:"githubOrg"`
}

func defaultGlobalConfig() GlobalConfig {
	return GlobalConfig{
		Infisical: InfisicalConfig{
			Host: "https://vault.intern.pixelandprocess.de",
		},
		Defaults: DefaultsConfig{
			Domain:        "pixelandprocess.de",
			ImageRegistry: "ghcr.io",
		},
	}
}

func GlobalConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".pnp.yaml")
}

func LoadGlobalConfig() (GlobalConfig, error) {
	return LoadGlobalConfigFrom(GlobalConfigPath())
}

func LoadGlobalConfigFrom(path string) (GlobalConfig, error) {
	cfg := defaultGlobalConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func SaveGlobalConfig(cfg GlobalConfig) error {
	return SaveGlobalConfigTo(cfg, GlobalConfigPath())
}

func SaveGlobalConfigTo(cfg GlobalConfig, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
```

**Step 4: Run tests**

Run: `go test ./internal/config/ -v -run TestLoadGlobalConfig`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/config/global.go internal/config/global_test.go
git commit -m "feat: add global config loading (~/.pnp.yaml)"
```

---

## Task 3: Project Config (`.cluster.yaml`)

**Files:**
- Create: `internal/config/project.go`
- Create: `internal/config/project_test.go`

**Step 1: Write test**

```go
// internal/config/project_test.go
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -v -run TestLoadProjectConfig`
Expected: FAIL

**Step 3: Implement project config**

```go
// internal/config/project.go
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type ProjectConfig struct {
	Name        string           `yaml:"name"`
	Type        string           `yaml:"type"`
	Environment string           `yaml:"environment"`
	Domain      string           `yaml:"domain"`
	Image       string           `yaml:"image"`
	Database    DatabaseConfig   `yaml:"database"`
	Redis       RedisConfig      `yaml:"redis"`
	Infisical   ProjectInfisical `yaml:"infisical"`
	Resources   ResourcesConfig  `yaml:"resources"`
	CI          CIConfig         `yaml:"ci"`
}

type DatabaseConfig struct {
	Enabled bool   `yaml:"enabled"`
	Size    string `yaml:"size"`
	Name    string `yaml:"name"`
}

type RedisConfig struct {
	Enabled bool `yaml:"enabled"`
}

type ProjectInfisical struct {
	ProjectSlug string `yaml:"projectSlug"`
	EnvSlug     string `yaml:"envSlug"`
	SecretsPath string `yaml:"secretsPath"`
}

type ResourcesConfig struct {
	CPU    string `yaml:"cpu"`
	Memory string `yaml:"memory"`
}

type CIConfig struct {
	Enabled bool `yaml:"enabled"`
}

func LoadProjectConfig() (ProjectConfig, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return ProjectConfig{}, err
	}
	return LoadProjectConfigFrom(filepath.Join(cwd, ".cluster.yaml"))
}

func LoadProjectConfigFrom(path string) (ProjectConfig, error) {
	var cfg ProjectConfig
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func SaveProjectConfig(cfg ProjectConfig) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	return SaveProjectConfigTo(cfg, filepath.Join(cwd, ".cluster.yaml"))
}

func SaveProjectConfigTo(cfg ProjectConfig, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
```

**Step 4: Run tests**

Run: `go test ./internal/config/ -v`
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/config/project.go internal/config/project_test.go
git commit -m "feat: add project config (.cluster.yaml) support"
```

---

## Task 4: Project Type Detection

**Files:**
- Create: `internal/detect/detect.go`
- Create: `internal/detect/detect_test.go`

**Step 1: Write tests**

```go
// internal/detect/detect_test.go
package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectLaravelWeb(t *testing.T) {
	dir := t.TempDir()
	// Create composer.json and artisan file
	os.WriteFile(filepath.Join(dir, "composer.json"), []byte(`{}`), 0644)
	os.WriteFile(filepath.Join(dir, "artisan"), []byte(`#!/usr/bin/env php`), 0644)
	// Create app/Console/Kernel.php with schedule method (indicates scheduler)
	os.MkdirAll(filepath.Join(dir, "app", "Console"), 0755)
	os.WriteFile(filepath.Join(dir, "app", "Console", "Kernel.php"), []byte(`schedule`), 0644)
	// Create a job file (indicates queue usage)
	os.MkdirAll(filepath.Join(dir, "app", "Jobs"), 0755)
	os.WriteFile(filepath.Join(dir, "app", "Jobs", "TestJob.php"), []byte(`class TestJob`), 0644)

	result := DetectProjectType(dir)
	if result.Type != "laravel-web" {
		t.Errorf("expected laravel-web, got %s", result.Type)
	}
	if result.Confidence != "high" {
		t.Errorf("expected high confidence, got %s", result.Confidence)
	}
}

func TestDetectLaravelAPI(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "composer.json"), []byte(`{}`), 0644)
	os.WriteFile(filepath.Join(dir, "artisan"), []byte(`#!/usr/bin/env php`), 0644)
	// No Jobs directory, no scheduler → laravel-api

	result := DetectProjectType(dir)
	if result.Type != "laravel-api" {
		t.Errorf("expected laravel-api, got %s", result.Type)
	}
}

func TestDetectNextjsFullstack(t *testing.T) {
	dir := t.TempDir()
	pkg := map[string]interface{}{
		"dependencies": map[string]string{
			"next":    "14.0.0",
			"prisma":  "5.0.0",
		},
	}
	data, _ := json.Marshal(pkg)
	os.WriteFile(filepath.Join(dir, "package.json"), data, 0644)

	result := DetectProjectType(dir)
	if result.Type != "nextjs-fullstack" {
		t.Errorf("expected nextjs-fullstack, got %s", result.Type)
	}
}

func TestDetectNextjsStatic(t *testing.T) {
	dir := t.TempDir()
	pkg := map[string]interface{}{
		"dependencies": map[string]string{
			"next":  "14.0.0",
			"react": "18.0.0",
		},
	}
	data, _ := json.Marshal(pkg)
	os.WriteFile(filepath.Join(dir, "package.json"), data, 0644)

	result := DetectProjectType(dir)
	if result.Type != "nextjs-static" {
		t.Errorf("expected nextjs-static, got %s", result.Type)
	}
}

func TestDetectStrapi(t *testing.T) {
	dir := t.TempDir()
	pkg := map[string]interface{}{
		"dependencies": map[string]string{
			"@strapi/strapi": "5.0.0",
		},
	}
	data, _ := json.Marshal(pkg)
	os.WriteFile(filepath.Join(dir, "package.json"), data, 0644)

	result := DetectProjectType(dir)
	if result.Type != "strapi" {
		t.Errorf("expected strapi, got %s", result.Type)
	}
}

func TestDetectUnknown(t *testing.T) {
	dir := t.TempDir()
	result := DetectProjectType(dir)
	if result.Type != "unknown" {
		t.Errorf("expected unknown, got %s", result.Type)
	}
}

func TestDetectGitRemoteImage(t *testing.T) {
	dir := t.TempDir()
	// Create a fake git config
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)
	gitConfig := `[remote "origin"]
	url = git@github.com:fwartner/my-project.git
	fetch = +refs/heads/*:refs/remotes/origin/*
`
	os.WriteFile(filepath.Join(dir, ".git", "config"), []byte(gitConfig), 0644)

	image := InferImageFromGitRemote(dir, "ghcr.io")
	if image != "ghcr.io/fwartner/my-project" {
		t.Errorf("expected ghcr.io/fwartner/my-project, got %s", image)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/detect/ -v`
Expected: FAIL

**Step 3: Implement detection**

```go
// internal/detect/detect.go
package detect

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type DetectionResult struct {
	Type       string // laravel-web, laravel-api, nextjs-fullstack, nextjs-static, strapi, unknown
	Confidence string // high, medium, low
}

func DetectProjectType(dir string) DetectionResult {
	// Check for Laravel
	if fileExists(filepath.Join(dir, "composer.json")) && fileExists(filepath.Join(dir, "artisan")) {
		return detectLaravelType(dir)
	}

	// Check for Node.js projects
	pkgPath := filepath.Join(dir, "package.json")
	if fileExists(pkgPath) {
		return detectNodeType(dir, pkgPath)
	}

	return DetectionResult{Type: "unknown", Confidence: "low"}
}

func detectLaravelType(dir string) DetectionResult {
	hasJobs := dirExists(filepath.Join(dir, "app", "Jobs"))
	hasScheduler := false

	kernelPath := filepath.Join(dir, "app", "Console", "Kernel.php")
	if data, err := os.ReadFile(kernelPath); err == nil {
		hasScheduler = strings.Contains(string(data), "schedule")
	}

	// Also check routes/console.php for scheduled commands (Laravel 11+)
	consolePath := filepath.Join(dir, "routes", "console.php")
	if data, err := os.ReadFile(consolePath); err == nil {
		if strings.Contains(string(data), "Schedule") || strings.Contains(string(data), "schedule") {
			hasScheduler = true
		}
	}

	if hasJobs || hasScheduler {
		return DetectionResult{Type: "laravel-web", Confidence: "high"}
	}
	return DetectionResult{Type: "laravel-api", Confidence: "high"}
}

func detectNodeType(dir, pkgPath string) DetectionResult {
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return DetectionResult{Type: "unknown", Confidence: "low"}
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return DetectionResult{Type: "unknown", Confidence: "low"}
	}

	allDeps := make(map[string]string)
	for k, v := range pkg.Dependencies {
		allDeps[k] = v
	}
	for k, v := range pkg.DevDependencies {
		allDeps[k] = v
	}

	// Check Strapi
	if _, ok := allDeps["@strapi/strapi"]; ok {
		return DetectionResult{Type: "strapi", Confidence: "high"}
	}

	// Check Next.js
	if _, ok := allDeps["next"]; ok {
		if hasDBDependency(allDeps) {
			return DetectionResult{Type: "nextjs-fullstack", Confidence: "high"}
		}
		return DetectionResult{Type: "nextjs-static", Confidence: "high"}
	}

	return DetectionResult{Type: "unknown", Confidence: "low"}
}

func hasDBDependency(deps map[string]string) bool {
	dbPackages := []string{"prisma", "@prisma/client", "pg", "postgres", "typeorm", "drizzle-orm", "knex", "sequelize"}
	for _, pkg := range dbPackages {
		if _, ok := deps[pkg]; ok {
			return true
		}
	}
	return false
}

func InferImageFromGitRemote(dir string, registry string) string {
	// Try git command first
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		// Fallback: read .git/config
		return inferFromGitConfig(dir, registry)
	}
	return parseGitURL(strings.TrimSpace(string(out)), registry)
}

func inferFromGitConfig(dir string, registry string) string {
	data, err := os.ReadFile(filepath.Join(dir, ".git", "config"))
	if err != nil {
		return ""
	}
	re := regexp.MustCompile(`url\s*=\s*(.+)`)
	matches := re.FindStringSubmatch(string(data))
	if len(matches) < 2 {
		return ""
	}
	return parseGitURL(strings.TrimSpace(matches[1]), registry)
}

func parseGitURL(url string, registry string) string {
	// Handle SSH: git@github.com:org/repo.git
	if strings.HasPrefix(url, "git@") {
		url = strings.TrimPrefix(url, "git@github.com:")
		url = strings.TrimSuffix(url, ".git")
		return registry + "/" + url
	}
	// Handle HTTPS: https://github.com/org/repo.git
	if strings.Contains(url, "github.com") {
		re := regexp.MustCompile(`github\.com[/:](.+?)(?:\.git)?$`)
		matches := re.FindStringSubmatch(url)
		if len(matches) >= 2 {
			return registry + "/" + matches[1]
		}
	}
	return ""
}

func InferProjectName(dir string) string {
	return filepath.Base(dir)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
```

**Step 4: Run tests**

Run: `go test ./internal/detect/ -v`
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/detect/
git commit -m "feat: add project type detection (Laravel, Next.js, Strapi)"
```

---

## Task 5: Template Rendering Engine

**Files:**
- Create: `internal/templates/renderer.go`
- Create: `internal/templates/renderer_test.go`
- Create: `internal/templates/types.go`

This task generates the actual Helm chart files (Chart.yaml, values.yaml, and template YAMLs) that go into the gitops repo. It uses the same structure as the `_templates/` directory in the gitops repo but rendered from Go structs.

**Step 1: Write types**

```go
// internal/templates/types.go
package templates

// TemplateData holds all values needed to render gitops manifests.
type TemplateData struct {
	Name        string
	Namespace   string
	Subdomain   string
	Domain      string
	Image       string
	Tag         string
	AppKey      string // Laravel only
	DBName      string
	DBUsername  string
	DBSize      string
	RedisEnabled    bool
	QueueEnabled    bool
	SchedulerEnabled bool
	PersistenceEnabled bool
	PersistenceSize   string
	InfisicalProjectSlug string
	InfisicalEnvSlug     string
	InfisicalSecretsPath string
	InfisicalMailEnabled bool
	CPU    string
	Memory string
	ChartPath string // e.g., charts/laravel, charts/nextjs, charts/strapi
	RepoURL   string
}
```

**Step 2: Write test**

```go
// internal/templates/renderer_test.go
package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderLaravelWeb(t *testing.T) {
	data := TemplateData{
		Name:                 "acme-corp",
		Namespace:            "preview-acme-corp",
		Subdomain:            "acme-corp",
		Domain:               "preview.pixelandprocess.de",
		Image:                "ghcr.io/fwartner/acme-corp",
		Tag:                  "latest",
		AppKey:               "base64:testkey123",
		DBName:               "app",
		DBUsername:            "app",
		DBSize:               "5Gi",
		RedisEnabled:         true,
		QueueEnabled:         true,
		SchedulerEnabled:     true,
		PersistenceEnabled:   true,
		PersistenceSize:      "1Gi",
		InfisicalProjectSlug: "customer-apps-f-jq3",
		InfisicalEnvSlug:     "prod",
		InfisicalSecretsPath: "/acme-corp/db",
		InfisicalMailEnabled: true,
		CPU:                  "100m",
		Memory:               "256Mi",
		ChartPath:            "charts/laravel",
		RepoURL:              "https://github.com/fwartner/pixelandprocess-gitops.git",
	}

	outDir := t.TempDir()
	err := Render("laravel-web", data, outDir)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	// Check Chart.yaml exists
	chartYaml, err := os.ReadFile(filepath.Join(outDir, "Chart.yaml"))
	if err != nil {
		t.Fatalf("Chart.yaml not created: %v", err)
	}
	if !strings.Contains(string(chartYaml), "name: acme-corp") {
		t.Error("Chart.yaml missing app name")
	}

	// Check application.yaml exists and has correct content
	appYaml, err := os.ReadFile(filepath.Join(outDir, "templates", "application.yaml"))
	if err != nil {
		t.Fatalf("application.yaml not created: %v", err)
	}
	content := string(appYaml)
	if !strings.Contains(content, "helm-acme-corp") {
		t.Error("application.yaml missing helm release name")
	}
	if !strings.Contains(content, "queue") {
		t.Error("application.yaml missing queue config")
	}

	// Check cnpg-cluster.yaml
	cnpg, err := os.ReadFile(filepath.Join(outDir, "templates", "cnpg-cluster.yaml"))
	if err != nil {
		t.Fatalf("cnpg-cluster.yaml not created: %v", err)
	}
	if !strings.Contains(string(cnpg), "acme-corp-db") {
		t.Error("cnpg-cluster.yaml missing db name")
	}

	// Check infisical-secrets.yaml
	infisical, err := os.ReadFile(filepath.Join(outDir, "templates", "infisical-secrets.yaml"))
	if err != nil {
		t.Fatalf("infisical-secrets.yaml not created: %v", err)
	}
	if !strings.Contains(string(infisical), "customer-apps-f-jq3") {
		t.Error("infisical-secrets.yaml missing project slug")
	}
}

func TestRenderNextjsStatic(t *testing.T) {
	data := TemplateData{
		Name:      "my-site",
		Namespace: "preview-my-site",
		Subdomain: "my-site",
		Domain:    "preview.pixelandprocess.de",
		Image:     "ghcr.io/fwartner/my-site",
		Tag:       "latest",
		ChartPath: "charts/nextjs",
		RepoURL:   "https://github.com/fwartner/pixelandprocess-gitops.git",
	}

	outDir := t.TempDir()
	err := Render("nextjs-static", data, outDir)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	// Should NOT have cnpg-cluster.yaml
	_, err = os.ReadFile(filepath.Join(outDir, "templates", "cnpg-cluster.yaml"))
	if err == nil {
		t.Error("nextjs-static should not have cnpg-cluster.yaml")
	}

	// Should NOT have infisical-secrets.yaml
	_, err = os.ReadFile(filepath.Join(outDir, "templates", "infisical-secrets.yaml"))
	if err == nil {
		t.Error("nextjs-static should not have infisical-secrets.yaml")
	}
}
```

**Step 3: Run test to verify it fails**

Run: `go test ./internal/templates/ -v`
Expected: FAIL

**Step 4: Implement renderer**

```go
// internal/templates/renderer.go
package templates

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// Render generates all manifest files for the given project type into outDir.
func Render(projectType string, data TemplateData, outDir string) error {
	if err := os.MkdirAll(filepath.Join(outDir, "templates"), 0755); err != nil {
		return err
	}

	// Chart.yaml — same for all types
	if err := renderFile(chartYamlTpl, data, filepath.Join(outDir, "Chart.yaml")); err != nil {
		return fmt.Errorf("Chart.yaml: %w", err)
	}

	// values.yaml
	valuesTpl, err := valuesTemplate(projectType)
	if err != nil {
		return err
	}
	if err := renderFile(valuesTpl, data, filepath.Join(outDir, "values.yaml")); err != nil {
		return fmt.Errorf("values.yaml: %w", err)
	}

	// application.yaml
	appTpl, err := applicationTemplate(projectType)
	if err != nil {
		return err
	}
	if err := renderFile(appTpl, data, filepath.Join(outDir, "templates", "application.yaml")); err != nil {
		return fmt.Errorf("application.yaml: %w", err)
	}

	// Database resources (cnpg-cluster.yaml) — only for types with DB
	if needsDatabase(projectType) {
		if err := renderFile(cnpgClusterTpl, data, filepath.Join(outDir, "templates", "cnpg-cluster.yaml")); err != nil {
			return fmt.Errorf("cnpg-cluster.yaml: %w", err)
		}
	}

	// Infisical secrets — only for types that use secrets
	if needsInfisical(projectType) {
		tpl := infisicalTpl
		if projectType == "laravel-web" || projectType == "laravel-api" {
			tpl = infisicalWithMailTpl
		}
		if err := renderFile(tpl, data, filepath.Join(outDir, "templates", "infisical-secrets.yaml")); err != nil {
			return fmt.Errorf("infisical-secrets.yaml: %w", err)
		}
	}

	return nil
}

func needsDatabase(projectType string) bool {
	switch projectType {
	case "laravel-web", "laravel-api", "nextjs-fullstack", "strapi":
		return true
	}
	return false
}

func needsInfisical(projectType string) bool {
	return needsDatabase(projectType)
}

func renderFile(tplStr string, data TemplateData, outPath string) error {
	// Use << >> delimiters to avoid conflicts with Helm's {{ }}
	tpl, err := template.New("").Delims("<<", ">>").Parse(tplStr)
	if err != nil {
		return err
	}
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return tpl.Execute(f, data)
}

// --- Templates ---
// Using << >> delimiters so Helm {{ }} passes through literally.

var chartYamlTpl = `apiVersion: v2
name: << .Name >>
version: 0.1.0
`

func valuesTemplate(projectType string) (string, error) {
	switch projectType {
	case "laravel-web", "laravel-api":
		return laravelValuesTpl, nil
	case "nextjs-fullstack":
		return nextjsFullstackValuesTpl, nil
	case "nextjs-static":
		return nextjsStaticValuesTpl, nil
	case "strapi":
		return strapiValuesTpl, nil
	}
	return "", fmt.Errorf("unknown project type: %s", projectType)
}

func applicationTemplate(projectType string) (string, error) {
	switch projectType {
	case "laravel-web":
		return laravelWebApplicationTpl, nil
	case "laravel-api":
		return laravelAPIApplicationTpl, nil
	case "nextjs-fullstack":
		return nextjsFullstackApplicationTpl, nil
	case "nextjs-static":
		return nextjsStaticApplicationTpl, nil
	case "strapi":
		return strapiApplicationTpl, nil
	}
	return "", fmt.Errorf("unknown project type: %s", projectType)
}

// --- Laravel values ---
var laravelValuesTpl = `spec:
  source:
    repoURL: << .RepoURL >>
    targetRevision: main
    path: << .ChartPath >>
  destination:
    namespace: << .Namespace >>
domain: << .Domain >>
subdomain: << .Subdomain >>
image:
  repository: << .Image >>
  tag: << .Tag >>
app:
  key: << .AppKey >>
database:
  name: << .DBName >>
  username: << .DBUsername >>
mail:
  from: info@pixelandprocess.de
`

// --- Next.js fullstack values ---
var nextjsFullstackValuesTpl = `spec:
  source:
    repoURL: << .RepoURL >>
    targetRevision: main
    path: << .ChartPath >>
  destination:
    namespace: << .Namespace >>
domain: << .Domain >>
subdomain: << .Subdomain >>
image:
  repository: << .Image >>
  tag: << .Tag >>
database:
  name: << .DBName >>
  username: << .DBUsername >>
`

// --- Next.js static values ---
var nextjsStaticValuesTpl = `spec:
  source:
    repoURL: << .RepoURL >>
    targetRevision: main
    path: << .ChartPath >>
  destination:
    namespace: << .Namespace >>
domain: << .Domain >>
subdomain: << .Subdomain >>
image:
  repository: << .Image >>
  tag: << .Tag >>
`

// --- Strapi values ---
var strapiValuesTpl = `spec:
  source:
    repoURL: << .RepoURL >>
    targetRevision: main
    path: << .ChartPath >>
  destination:
    namespace: << .Namespace >>
domain: << .Domain >>
subdomain: << .Subdomain >>
image:
  repository: << .Image >>
  tag: << .Tag >>
database:
  name: << .DBName >>
  username: << .DBUsername >>
`

// --- Application templates ---
// These output Helm template syntax ({{ }}) literally because we use << >> delimiters.

var laravelWebApplicationTpl = `apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: helm-<< .Name >>
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: default
  source:
    repoURL: {{ .Values.spec.source.repoURL }}
    targetRevision: {{ .Values.spec.source.targetRevision }}
    path: {{ .Values.spec.source.path }}
    helm:
      values: |
        image:
          repository: {{ .Values.image.repository }}
          tag: {{ .Values.image.tag | quote }}
        imagePullSecrets:
          - name: ghcr-secret
        ingress:
          enabled: true
          className: traefik
          annotations:
            cert-manager.io/cluster-issuer: letsencrypt-prod
            traefik.ingress.kubernetes.io/router.entrypoints: websecure
            traefik.ingress.kubernetes.io/router.tls: "true"
          host: {{ .Values.subdomain }}.{{ .Values.domain }}
          tlsSecretName: {{ .Values.subdomain }}-tls
        app:
          key: {{ .Values.app.key | quote }}
          url: "https://{{ .Values.subdomain }}.{{ .Values.domain }}"
          env: production
          debug: "false"
        database:
          enabled: true
          size: << .DBSize >>
          name: {{ .Values.database.name }}
          username: {{ .Values.database.username }}
          existingSecret: {{ .Release.Name }}-db-credentials
          existingSecretPasswordKey: password
        redis:
          enabled: true
        queue:
          enabled: true
        scheduler:
          enabled: true
        mail:
          mailer: smtp
          host: smtp.postmarkapp.com
          port: "587"
          existingSecret: {{ .Release.Name }}-mail-credentials
          from: {{ .Values.mail.from }}
          fromName: {{ .Release.Name | quote }}
        persistence:
          enabled: true
          size: << .PersistenceSize >>
          storageClass: hcloud-volumes
        resources:
          web:
            requests:
              cpu: << .CPU >>
              memory: << .Memory >>
  destination:
    server: https://kubernetes.default.svc
    namespace: {{ .Values.spec.destination.namespace }}
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
`

var laravelAPIApplicationTpl = `apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: helm-<< .Name >>
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: default
  source:
    repoURL: {{ .Values.spec.source.repoURL }}
    targetRevision: {{ .Values.spec.source.targetRevision }}
    path: {{ .Values.spec.source.path }}
    helm:
      values: |
        image:
          repository: {{ .Values.image.repository }}
          tag: {{ .Values.image.tag | quote }}
        imagePullSecrets:
          - name: ghcr-secret
        ingress:
          enabled: true
          className: traefik
          annotations:
            cert-manager.io/cluster-issuer: letsencrypt-prod
            traefik.ingress.kubernetes.io/router.entrypoints: websecure
            traefik.ingress.kubernetes.io/router.tls: "true"
          host: {{ .Values.subdomain }}.{{ .Values.domain }}
          tlsSecretName: {{ .Values.subdomain }}-tls
        app:
          key: {{ .Values.app.key | quote }}
          url: "https://{{ .Values.subdomain }}.{{ .Values.domain }}"
          env: production
          debug: "false"
        database:
          enabled: true
          size: << .DBSize >>
          name: {{ .Values.database.name }}
          username: {{ .Values.database.username }}
          existingSecret: {{ .Release.Name }}-db-credentials
          existingSecretPasswordKey: password
        redis:
          enabled: true
        queue:
          enabled: false
        scheduler:
          enabled: false
        mail:
          mailer: smtp
          host: smtp.postmarkapp.com
          port: "587"
          existingSecret: {{ .Release.Name }}-mail-credentials
          from: {{ .Values.mail.from }}
          fromName: {{ .Release.Name | quote }}
        persistence:
          enabled: false
        resources:
          web:
            requests:
              cpu: << .CPU >>
              memory: << .Memory >>
  destination:
    server: https://kubernetes.default.svc
    namespace: {{ .Values.spec.destination.namespace }}
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
`

var nextjsFullstackApplicationTpl = `apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: helm-<< .Name >>
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: default
  source:
    repoURL: {{ .Values.spec.source.repoURL }}
    targetRevision: {{ .Values.spec.source.targetRevision }}
    path: {{ .Values.spec.source.path }}
    helm:
      values: |
        image:
          repository: {{ .Values.image.repository }}
          tag: {{ .Values.image.tag | quote }}
        imagePullSecrets:
          - name: ghcr-secret
        ingress:
          enabled: true
          className: traefik
          annotations:
            cert-manager.io/cluster-issuer: letsencrypt-prod
            traefik.ingress.kubernetes.io/router.entrypoints: websecure
            traefik.ingress.kubernetes.io/router.tls: "true"
          host: {{ .Values.subdomain }}.{{ .Values.domain }}
          tlsSecretName: {{ .Values.subdomain }}-tls
        database:
          enabled: false
        env:
          - name: DATABASE_URL
            value: "postgres://{{ .Values.database.username }}:$(DB_PASSWORD)@<< .Name >>-db-rw.<< .Namespace >>.svc.cluster.local:5432/{{ .Values.database.name }}"
        resources:
          requests:
            cpu: << .CPU >>
            memory: << .Memory >>
  destination:
    server: https://kubernetes.default.svc
    namespace: {{ .Values.spec.destination.namespace }}
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
`

var nextjsStaticApplicationTpl = `apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: helm-<< .Name >>
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: default
  source:
    repoURL: {{ .Values.spec.source.repoURL }}
    targetRevision: {{ .Values.spec.source.targetRevision }}
    path: {{ .Values.spec.source.path }}
    helm:
      values: |
        image:
          repository: {{ .Values.image.repository }}
          tag: {{ .Values.image.tag | quote }}
        imagePullSecrets:
          - name: ghcr-secret
        ingress:
          enabled: true
          className: traefik
          annotations:
            cert-manager.io/cluster-issuer: letsencrypt-prod
            traefik.ingress.kubernetes.io/router.entrypoints: websecure
            traefik.ingress.kubernetes.io/router.tls: "true"
          host: {{ .Values.subdomain }}.{{ .Values.domain }}
          tlsSecretName: {{ .Values.subdomain }}-tls
        database:
          enabled: false
        resources:
          requests:
            cpu: << .CPU >>
            memory: << .Memory >>
  destination:
    server: https://kubernetes.default.svc
    namespace: {{ .Values.spec.destination.namespace }}
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
`

var strapiApplicationTpl = `apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: helm-<< .Name >>
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: default
  source:
    repoURL: {{ .Values.spec.source.repoURL }}
    targetRevision: {{ .Values.spec.source.targetRevision }}
    path: {{ .Values.spec.source.path }}
    helm:
      values: |
        image:
          repository: {{ .Values.image.repository }}
          tag: {{ .Values.image.tag | quote }}
        imagePullSecrets:
          - name: ghcr-secret
        ingress:
          enabled: true
          className: traefik
          annotations:
            cert-manager.io/cluster-issuer: letsencrypt-prod
            traefik.ingress.kubernetes.io/router.entrypoints: websecure
            traefik.ingress.kubernetes.io/router.tls: "true"
          host: {{ .Values.subdomain }}.{{ .Values.domain }}
          tlsSecretName: {{ .Values.subdomain }}-tls
        persistence:
          enabled: true
          size: 5Gi
          storageClass: hcloud-volumes
        resources:
          requests:
            cpu: << .CPU >>
            memory: << .Memory >>
  destination:
    server: https://kubernetes.default.svc
    namespace: {{ .Values.spec.destination.namespace }}
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
`

// --- CNPG Cluster template ---
var cnpgClusterTpl = `apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: << .Name >>-db
  namespace: << .Namespace >>
spec:
  instances: 1
  storage:
    size: << .DBSize >>
  bootstrap:
    initdb:
      database: << .DBName >>
      owner: << .DBUsername >>
      secret:
        name: << .Name >>-db-credentials
`

// --- Infisical secrets (DB only) ---
var infisicalTpl = `apiVersion: secrets.infisical.com/v1alpha1
kind: InfisicalSecret
metadata:
  name: << .Name >>-db-infisical
  namespace: << .Namespace >>
spec:
  hostAPI: https://vault.intern.pixelandprocess.de
  resyncInterval: 60
  authentication:
    universalAuth:
      credentialsRef:
        secretName: infisical-machine-identity
        secretNamespace: infisical-operator-system
      secretsScope:
        projectSlug: << .InfisicalProjectSlug >>
        envSlug: << .InfisicalEnvSlug >>
        secretsPath: << .InfisicalSecretsPath >>
  managedSecretReference:
    secretName: << .Name >>-db-credentials
    secretNamespace: << .Namespace >>
    secretType: kubernetes.io/basic-auth
`

// --- Infisical secrets (DB + mail) for Laravel ---
var infisicalWithMailTpl = `apiVersion: secrets.infisical.com/v1alpha1
kind: InfisicalSecret
metadata:
  name: << .Name >>-db-infisical
  namespace: << .Namespace >>
spec:
  hostAPI: https://vault.intern.pixelandprocess.de
  resyncInterval: 60
  authentication:
    universalAuth:
      credentialsRef:
        secretName: infisical-machine-identity
        secretNamespace: infisical-operator-system
      secretsScope:
        projectSlug: << .InfisicalProjectSlug >>
        envSlug: << .InfisicalEnvSlug >>
        secretsPath: << .InfisicalSecretsPath >>
  managedSecretReference:
    secretName: << .Name >>-db-credentials
    secretNamespace: << .Namespace >>
    secretType: kubernetes.io/basic-auth
---
apiVersion: secrets.infisical.com/v1alpha1
kind: InfisicalSecret
metadata:
  name: << .Name >>-mail-infisical
  namespace: << .Namespace >>
spec:
  hostAPI: https://vault.intern.pixelandprocess.de
  resyncInterval: 60
  authentication:
    universalAuth:
      credentialsRef:
        secretName: infisical-machine-identity
        secretNamespace: infisical-operator-system
      secretsScope:
        projectSlug: cluster-shared-ys-zj
        envSlug: prod
        secretsPath: /smtp
  managedSecretReference:
    secretName: << .Name >>-mail-credentials
    secretNamespace: << .Namespace >>
    secretType: Opaque
`
```

**Step 5: Run tests**

Run: `go test ./internal/templates/ -v`
Expected: All PASS

**Step 6: Commit**

```bash
git add internal/templates/
git commit -m "feat: add template rendering engine for all project types"
```

---

## Task 6: Gitops Repository Operations

**Files:**
- Create: `internal/gitops/repo.go`
- Create: `internal/gitops/repo_test.go`

**Step 1: Write test**

```go
// internal/gitops/repo_test.go
package gitops

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func setupTestGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("setup git: %v", err)
		}
	}
	// Create initial commit
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test"), 0644)
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = dir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "init")
	cmd.Dir = dir
	cmd.Run()
	return dir
}

func TestAppPath(t *testing.T) {
	r := &Repo{Path: "/tmp/gitops"}

	path := r.AppPath("acme", "preview")
	if path != "/tmp/gitops/apps/previews/acme" {
		t.Errorf("expected previews path, got %s", path)
	}

	path = r.AppPath("acme", "staging")
	if path != "/tmp/gitops/apps/previews/acme" {
		t.Errorf("expected previews path for staging, got %s", path)
	}

	path = r.AppPath("acme", "production")
	if path != "/tmp/gitops/apps/acme" {
		t.Errorf("expected production path, got %s", path)
	}
}

func TestAppExists(t *testing.T) {
	dir := t.TempDir()
	r := &Repo{Path: dir}

	if r.AppExists("acme", "preview") {
		t.Error("app should not exist yet")
	}

	os.MkdirAll(filepath.Join(dir, "apps", "previews", "acme"), 0755)
	if !r.AppExists("acme", "preview") {
		t.Error("app should exist")
	}
}

func TestWriteApp(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "apps", "previews"), 0755)
	r := &Repo{Path: dir}

	srcDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "Chart.yaml"), []byte("name: test"), 0644)
	os.MkdirAll(filepath.Join(srcDir, "templates"), 0755)
	os.WriteFile(filepath.Join(srcDir, "templates", "app.yaml"), []byte("kind: App"), 0644)

	err := r.WriteApp("test-app", "preview", srcDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	appPath := filepath.Join(dir, "apps", "previews", "test-app")
	if _, err := os.Stat(filepath.Join(appPath, "Chart.yaml")); err != nil {
		t.Error("Chart.yaml not copied")
	}
	if _, err := os.Stat(filepath.Join(appPath, "templates", "app.yaml")); err != nil {
		t.Error("templates/app.yaml not copied")
	}
}

func TestDeleteApp(t *testing.T) {
	dir := t.TempDir()
	appDir := filepath.Join(dir, "apps", "previews", "test-app")
	os.MkdirAll(appDir, 0755)
	os.WriteFile(filepath.Join(appDir, "Chart.yaml"), []byte("test"), 0644)

	r := &Repo{Path: dir}
	err := r.DeleteApp("test-app", "preview")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(appDir); !os.IsNotExist(err) {
		t.Error("app directory should be deleted")
	}
}

func TestCommitAndPush(t *testing.T) {
	dir := setupTestGitRepo(t)
	r := &Repo{Path: dir}

	// Create a file to commit
	os.MkdirAll(filepath.Join(dir, "apps", "previews", "test"), 0755)
	os.WriteFile(filepath.Join(dir, "apps", "previews", "test", "Chart.yaml"), []byte("name: test"), 0644)

	err := r.CommitChanges("test: add test app")
	if err != nil {
		t.Fatalf("commit failed: %v", err)
	}

	// Verify commit exists
	cmd := exec.Command("git", "log", "--oneline", "-1")
	cmd.Dir = dir
	out, _ := cmd.Output()
	if len(out) == 0 {
		t.Error("no commit found")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/gitops/ -v`
Expected: FAIL

**Step 3: Implement gitops repo operations**

```go
// internal/gitops/repo.go
package gitops

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Repo struct {
	Path string
}

func NewRepo(path string) *Repo {
	return &Repo{Path: path}
}

// AppPath returns the directory where an app's manifests should live.
func (r *Repo) AppPath(name string, environment string) string {
	switch environment {
	case "production":
		return filepath.Join(r.Path, "apps", name)
	default: // preview, staging
		return filepath.Join(r.Path, "apps", "previews", name)
	}
}

// AppExists checks whether an app directory already exists.
func (r *Repo) AppExists(name string, environment string) bool {
	_, err := os.Stat(r.AppPath(name, environment))
	return err == nil
}

// WriteApp copies rendered manifests from srcDir into the correct app location.
func (r *Repo) WriteApp(name string, environment string, srcDir string) error {
	destDir := r.AppPath(name, environment)

	// Remove existing if present (for updates)
	os.RemoveAll(destDir)

	return copyDir(srcDir, destDir)
}

// DeleteApp removes an app directory.
func (r *Repo) DeleteApp(name string, environment string) error {
	return os.RemoveAll(r.AppPath(name, environment))
}

// Pull runs git pull on the repo.
func (r *Repo) Pull() error {
	return r.git("pull", "--rebase")
}

// CommitChanges stages all changes and commits.
func (r *Repo) CommitChanges(message string) error {
	if err := r.git("add", "-A"); err != nil {
		return err
	}
	return r.git("commit", "-m", message)
}

// Push pushes to origin main.
func (r *Repo) Push() error {
	return r.git("push")
}

// CreateBranchAndPush creates a new branch, commits, and pushes.
func (r *Repo) CreateBranchAndPush(branch string, message string) error {
	if err := r.git("checkout", "-b", branch); err != nil {
		return err
	}
	if err := r.CommitChanges(message); err != nil {
		return err
	}
	if err := r.git("push", "-u", "origin", branch); err != nil {
		return err
	}
	// Switch back to main
	return r.git("checkout", "main")
}

// CreatePR creates a GitHub pull request using gh CLI.
func (r *Repo) CreatePR(title string, body string, branch string) (string, error) {
	cmd := exec.Command("gh", "pr", "create", "--title", title, "--body", body, "--head", branch)
	cmd.Dir = r.Path
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh pr create: %s: %w", string(out), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (r *Repo) git(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Path
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}
```

**Step 4: Run tests**

Run: `go test ./internal/gitops/ -v`
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/gitops/
git commit -m "feat: add gitops repo operations (write, delete, commit, PR)"
```

---

## Task 7: Infisical API Client

**Files:**
- Create: `internal/infisical/client.go`
- Create: `internal/infisical/client_test.go`

**Step 1: Write test**

```go
// internal/infisical/client_test.go
package infisical

import (
	"crypto/rand"
	"encoding/base64"
	"testing"
)

func TestGeneratePassword(t *testing.T) {
	pw := GeneratePassword()
	if len(pw) < 24 {
		t.Errorf("password too short: %d chars", len(pw))
	}
}

func TestGenerateLaravelKey(t *testing.T) {
	key := GenerateLaravelKey()
	if len(key) < 10 {
		t.Errorf("key too short: %s", key)
	}
	if key[:7] != "base64:" {
		t.Errorf("expected base64: prefix, got %s", key[:7])
	}
}

func TestSecretPayload(t *testing.T) {
	secrets := map[string]string{
		"username": "app",
		"password": "secret123",
	}

	payload := buildCreateSecretsPayload(secrets, "my-project", "prod", "/app/db")
	if len(payload.Secrets) != 2 {
		t.Errorf("expected 2 secrets, got %d", len(payload.Secrets))
	}
}

// Note: actual API calls are tested via integration tests only.
// The client methods that call Infisical API are not unit-tested
// to avoid requiring a running Infisical instance.
var _ = rand.Reader
var _ = base64.StdEncoding
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/infisical/ -v`
Expected: FAIL

**Step 3: Implement Infisical client**

```go
// internal/infisical/client.go
package infisical

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	Host  string
	Token string
}

type secretPayload struct {
	Secrets     []secretEntry `json:"secrets"`
	ProjectSlug string        `json:"workspaceSlug"`
	Environment string        `json:"environment"`
	SecretPath  string        `json:"secretPath"`
}

type secretEntry struct {
	SecretKey   string `json:"secretKey"`
	SecretValue string `json:"secretValue"`
}

func NewClient(host, token string) *Client {
	return &Client{Host: host, Token: token}
}

func GeneratePassword() string {
	b := make([]byte, 32)
	rand.Read(b)
	encoded := base64.URLEncoding.EncodeToString(b)
	// Trim to 32 chars, remove problematic characters
	result := ""
	for _, c := range encoded {
		if c != '/' && c != '+' && c != '=' && c != '-' && c != '_' {
			result += string(c)
		}
		if len(result) >= 32 {
			break
		}
	}
	if len(result) < 24 {
		return encoded[:32]
	}
	return result
}

func GenerateLaravelKey() string {
	b := make([]byte, 32)
	rand.Read(b)
	return "base64:" + base64.StdEncoding.EncodeToString(b)
}

func buildCreateSecretsPayload(secrets map[string]string, projectSlug, envSlug, secretsPath string) secretPayload {
	entries := make([]secretEntry, 0, len(secrets))
	for k, v := range secrets {
		entries = append(entries, secretEntry{SecretKey: k, SecretValue: v})
	}
	return secretPayload{
		Secrets:     entries,
		ProjectSlug: projectSlug,
		Environment: envSlug,
		SecretPath:  secretsPath,
	}
}

// CreateSecrets pushes secrets to Infisical via the batch create API.
func (c *Client) CreateSecrets(secrets map[string]string, projectSlug, envSlug, secretsPath string) error {
	payload := buildCreateSecretsPayload(secrets, projectSlug, envSlug, secretsPath)
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v3/secrets/batch/raw", c.Host)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("infisical API call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("infisical API error %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}
```

**Step 4: Run tests**

Run: `go test ./internal/infisical/ -v`
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/infisical/
git commit -m "feat: add Infisical API client for secret management"
```

---

## Task 8: Interactive Wizard

**Files:**
- Create: `internal/wizard/wizard.go`

This task uses the `huh` library for interactive forms. It cannot be easily unit-tested since it's TUI-based, but we test the logic that feeds into it.

**Step 1: Implement wizard**

```go
// internal/wizard/wizard.go
package wizard

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/fwartner/pnp/internal/config"
	"github.com/fwartner/pnp/internal/detect"
)

var projectTypes = []string{"laravel-web", "laravel-api", "nextjs-fullstack", "nextjs-static", "strapi"}
var environments = []string{"preview", "staging", "production"}

// Run executes the interactive wizard and returns a filled ProjectConfig.
func Run(detected detect.DetectionResult, inferredImage string, projectName string, globalCfg config.GlobalConfig) (config.ProjectConfig, error) {
	cfg := config.ProjectConfig{
		Name:        projectName,
		Type:        detected.Type,
		Environment: "preview",
		Image:       inferredImage,
		Database: config.DatabaseConfig{
			Enabled: true,
			Size:    "5Gi",
			Name:    "app",
		},
		Redis: config.RedisConfig{
			Enabled: true,
		},
		Infisical: config.ProjectInfisical{
			ProjectSlug: "customer-apps-f-jq3",
			EnvSlug:     "prod",
		},
		Resources: config.ResourcesConfig{
			CPU:    "100m",
			Memory: "256Mi",
		},
	}

	// --- Step 1: Basic info ---
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Project name").
				Value(&cfg.Name),

			huh.NewSelect[string]().
				Title("Project type").
				Description(fmt.Sprintf("Detected: %s (%s confidence)", detected.Type, detected.Confidence)).
				Options(huh.NewOptions(projectTypes...)...).
				Value(&cfg.Type),

			huh.NewSelect[string]().
				Title("Environment").
				Options(huh.NewOptions(environments...)...).
				Value(&cfg.Environment),
		),
	).Run()
	if err != nil {
		return cfg, err
	}

	// --- Step 2: Domain ---
	cfg.Domain = defaultDomain(cfg.Name, cfg.Environment, globalCfg.Defaults.Domain)
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Domain").
				Value(&cfg.Domain),

			huh.NewInput().
				Title("Container image").
				Value(&cfg.Image),
		),
	).Run()
	if err != nil {
		return cfg, err
	}

	// --- Step 3: Database & Redis (skip for nextjs-static) ---
	if cfg.Type != "nextjs-static" {
		err = huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Enable PostgreSQL database?").
					Value(&cfg.Database.Enabled),

				huh.NewInput().
					Title("Database name").
					Value(&cfg.Database.Name),

				huh.NewInput().
					Title("Database storage size").
					Value(&cfg.Database.Size),
			),
		).Run()
		if err != nil {
			return cfg, err
		}

		if cfg.Type == "laravel-web" || cfg.Type == "laravel-api" {
			err = huh.NewForm(
				huh.NewGroup(
					huh.NewConfirm().
						Title("Enable Redis?").
						Value(&cfg.Redis.Enabled),
				),
			).Run()
			if err != nil {
				return cfg, err
			}
		}
	} else {
		cfg.Database.Enabled = false
		cfg.Redis.Enabled = false
	}

	// --- Step 4: Infisical (if DB enabled) ---
	if cfg.Database.Enabled {
		cfg.Infisical.SecretsPath = "/" + cfg.Name + "/db"
		err = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Infisical project slug").
					Value(&cfg.Infisical.ProjectSlug),

				huh.NewInput().
					Title("Infisical environment").
					Value(&cfg.Infisical.EnvSlug),

				huh.NewInput().
					Title("Infisical secrets path").
					Value(&cfg.Infisical.SecretsPath),
			),
		).Run()
		if err != nil {
			return cfg, err
		}
	}

	// --- Step 5: Resources ---
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("CPU request").
				Value(&cfg.Resources.CPU),

			huh.NewInput().
				Title("Memory request").
				Value(&cfg.Resources.Memory),
		),
	).Run()
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}

func defaultDomain(name, environment, baseDomain string) string {
	switch environment {
	case "preview":
		return name + ".preview." + baseDomain
	case "staging":
		return name + ".staging." + baseDomain
	default:
		return name + "." + baseDomain
	}
}

// Subdomain extracts the subdomain part from a full domain.
func Subdomain(domain, baseDomain string) string {
	// For custom domains (production), use the full domain
	if baseDomain == "" {
		return domain
	}
	// Strip the base domain suffix
	suffix := "." + baseDomain
	if len(domain) > len(suffix) && domain[len(domain)-len(suffix):] == suffix {
		return domain[:len(domain)-len(suffix)]
	}
	return domain
}
```

**Step 2: Verify it compiles**

Run: `go build ./internal/wizard/`
Expected: Success

**Step 3: Commit**

```bash
git add internal/wizard/
git commit -m "feat: add interactive wizard with huh forms"
```

---

## Task 9: GitHub Actions CI Generator

**Files:**
- Create: `internal/ci/generate.go`
- Create: `internal/ci/generate_test.go`

**Step 1: Write test**

```go
// internal/ci/generate_test.go
package ci

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateLaravelWorkflow(t *testing.T) {
	dir := t.TempDir()
	err := GenerateWorkflow("laravel-web", "ghcr.io/fwartner/my-app", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	path := filepath.Join(dir, ".github", "workflows", "deploy.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("workflow not created: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "docker") {
		t.Error("missing docker build step")
	}
	if !strings.Contains(content, "ghcr.io/fwartner/my-app") {
		t.Error("missing image reference")
	}
}

func TestGenerateNextjsWorkflow(t *testing.T) {
	dir := t.TempDir()
	err := GenerateWorkflow("nextjs-static", "ghcr.io/fwartner/site", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	path := filepath.Join(dir, ".github", "workflows", "deploy.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("workflow not created: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "node") {
		t.Error("missing node setup step")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/ci/ -v`
Expected: FAIL

**Step 3: Implement CI generator**

```go
// internal/ci/generate.go
package ci

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

func GenerateWorkflow(projectType string, image string, projectDir string) error {
	outDir := filepath.Join(projectDir, ".github", "workflows")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	var tplStr string
	switch projectType {
	case "laravel-web", "laravel-api":
		tplStr = laravelWorkflowTpl
	case "nextjs-fullstack", "nextjs-static":
		tplStr = nextjsWorkflowTpl
	case "strapi":
		tplStr = strapiWorkflowTpl
	default:
		return fmt.Errorf("unsupported project type: %s", projectType)
	}

	tpl, err := template.New("workflow").Parse(tplStr)
	if err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(outDir, "deploy.yml"))
	if err != nil {
		return err
	}
	defer f.Close()

	return tpl.Execute(f, map[string]string{"Image": image})
}

var laravelWorkflowTpl = `name: Build & Push

on:
  push:
    branches: [main]

env:
  IMAGE: {{ .Image }}

jobs:
  build:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write

    steps:
      - uses: actions/checkout@v4

      - name: Set up PHP
        uses: shivammathur/setup-php@v2
        with:
          php-version: "8.3"

      - name: Install Composer dependencies
        run: composer install --no-dev --optimize-autoloader

      - name: Install Node.js
        uses: actions/setup-node@v4
        with:
          node-version: "20"

      - name: Install npm dependencies & build
        run: |
          npm ci
          npm run build

      - name: Log in to GHCR
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ "{{" }} github.actor {{ "}}" }}
          password: ${{ "{{" }} secrets.GITHUB_TOKEN {{ "}}" }}

      - name: Build and push docker image
        uses: docker/build-push-action@v5
        with:
          context: .
          push: true
          tags: |
            ${{ "{{" }} env.IMAGE {{ "}}" }}:${{ "{{" }} github.sha {{ "}}" }}
            ${{ "{{" }} env.IMAGE {{ "}}" }}:latest
`

var nextjsWorkflowTpl = `name: Build & Push

on:
  push:
    branches: [main]

env:
  IMAGE: {{ .Image }}

jobs:
  build:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write

    steps:
      - uses: actions/checkout@v4

      - name: Set up node
        uses: actions/setup-node@v4
        with:
          node-version: "20"

      - name: Install dependencies
        run: npm ci

      - name: Build
        run: npm run build

      - name: Log in to GHCR
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ "{{" }} github.actor {{ "}}" }}
          password: ${{ "{{" }} secrets.GITHUB_TOKEN {{ "}}" }}

      - name: Build and push docker image
        uses: docker/build-push-action@v5
        with:
          context: .
          push: true
          tags: |
            ${{ "{{" }} env.IMAGE {{ "}}" }}:${{ "{{" }} github.sha {{ "}}" }}
            ${{ "{{" }} env.IMAGE {{ "}}" }}:latest
`

var strapiWorkflowTpl = `name: Build & Push

on:
  push:
    branches: [main]

env:
  IMAGE: {{ .Image }}

jobs:
  build:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write

    steps:
      - uses: actions/checkout@v4

      - name: Set up node
        uses: actions/setup-node@v4
        with:
          node-version: "20"

      - name: Install dependencies
        run: npm ci

      - name: Build
        run: npm run build

      - name: Log in to GHCR
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ "{{" }} github.actor {{ "}}" }}
          password: ${{ "{{" }} secrets.GITHUB_TOKEN {{ "}}" }}

      - name: Build and push docker image
        uses: docker/build-push-action@v5
        with:
          context: .
          push: true
          tags: |
            ${{ "{{" }} env.IMAGE {{ "}}" }}:${{ "{{" }} github.sha {{ "}}" }}
            ${{ "{{" }} env.IMAGE {{ "}}" }}:latest
`
```

**Step 4: Run tests**

Run: `go test ./internal/ci/ -v`
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/ci/
git commit -m "feat: add GitHub Actions workflow generator"
```

---

## Task 10: Deploy Command

**Files:**
- Create: `cmd/deploy.go`

This ties everything together: detection → wizard → template rendering → gitops write → optional secrets → commit/push or PR.

**Step 1: Implement deploy command**

```go
// cmd/deploy.go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/fwartner/pnp/internal/ci"
	"github.com/fwartner/pnp/internal/config"
	"github.com/fwartner/pnp/internal/detect"
	"github.com/fwartner/pnp/internal/gitops"
	"github.com/fwartner/pnp/internal/infisical"
	"github.com/fwartner/pnp/internal/templates"
	"github.com/fwartner/pnp/internal/wizard"
	"github.com/spf13/cobra"
)

var (
	flagPR          bool
	flagSkipSecrets bool
	flagWithCI      bool
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy project to Kubernetes cluster",
	Long:  "Detect project type, run wizard, generate manifests, and push to gitops repo.",
	RunE:  runDeploy,
}

func init() {
	deployCmd.Flags().BoolVar(&flagPR, "pr", false, "Create a PR instead of pushing directly to main")
	deployCmd.Flags().BoolVar(&flagSkipSecrets, "skip-secrets", false, "Skip creating secrets in Infisical")
	deployCmd.Flags().BoolVar(&flagWithCI, "with-ci", false, "Generate GitHub Actions workflow in project")
	rootCmd.AddCommand(deployCmd)
}

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	successStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	errorStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9"))
)

func runDeploy(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	fmt.Println(titleStyle.Render("🚀 PnP Deploy"))
	fmt.Println()

	// Load global config
	globalCfg, err := config.LoadGlobalConfig()
	if err != nil {
		return fmt.Errorf("loading global config: %w", err)
	}

	if globalCfg.GitopsRepo == "" {
		return fmt.Errorf("gitopsRepo not set in ~/.pnp.yaml — run: pnp init")
	}

	// Check for existing .cluster.yaml
	var projectCfg config.ProjectConfig
	existingConfig := false
	if cfg, err := config.LoadProjectConfig(); err == nil {
		projectCfg = cfg
		existingConfig = true
		fmt.Printf("Found existing .cluster.yaml for %s\n", projectCfg.Name)
	}

	if !existingConfig {
		// Detect project type
		detected := detect.DetectProjectType(cwd)
		fmt.Printf("Detected: %s (%s confidence)\n", detected.Type, detected.Confidence)

		// Infer image from git remote
		inferredImage := detect.InferImageFromGitRemote(cwd, globalCfg.Defaults.ImageRegistry)
		projectName := detect.InferProjectName(cwd)

		// Run wizard
		projectCfg, err = wizard.Run(detected, inferredImage, projectName, globalCfg)
		if err != nil {
			return err
		}
	}

	// Determine namespace
	namespace := namespaceFor(projectCfg.Name, projectCfg.Environment)

	// Determine subdomain and domain parts
	baseDomain := globalCfg.Defaults.Domain
	subdomain := wizard.Subdomain(projectCfg.Domain, baseDomain)
	domain := baseDomain
	if projectCfg.Environment == "preview" {
		domain = "preview." + baseDomain
	} else if projectCfg.Environment == "staging" {
		domain = "staging." + baseDomain
	}

	// Determine chart path
	chartPath := chartPathFor(projectCfg.Type)

	// Build template data
	tplData := templates.TemplateData{
		Name:                   projectCfg.Name,
		Namespace:              namespace,
		Subdomain:              subdomain,
		Domain:                 domain,
		Image:                  projectCfg.Image,
		Tag:                    "latest",
		AppKey:                 "", // Set below for Laravel
		DBName:                 projectCfg.Database.Name,
		DBUsername:              "app",
		DBSize:                 projectCfg.Database.Size,
		RedisEnabled:           projectCfg.Redis.Enabled,
		QueueEnabled:           projectCfg.Type == "laravel-web",
		SchedulerEnabled:       projectCfg.Type == "laravel-web",
		PersistenceEnabled:     projectCfg.Type == "laravel-web" || projectCfg.Type == "strapi",
		PersistenceSize:        "1Gi",
		InfisicalProjectSlug:   projectCfg.Infisical.ProjectSlug,
		InfisicalEnvSlug:       projectCfg.Infisical.EnvSlug,
		InfisicalSecretsPath:   projectCfg.Infisical.SecretsPath,
		InfisicalMailEnabled:   projectCfg.Type == "laravel-web" || projectCfg.Type == "laravel-api",
		CPU:                    projectCfg.Resources.CPU,
		Memory:                 projectCfg.Resources.Memory,
		ChartPath:              chartPath,
		RepoURL:                globalCfg.GitopsRemote,
	}

	if tplData.RepoURL == "" {
		tplData.RepoURL = "https://github.com/fwartner/pixelandprocess-gitops.git"
	}
	if tplData.CPU == "" {
		tplData.CPU = "100m"
	}
	if tplData.Memory == "" {
		tplData.Memory = "256Mi"
	}

	// Generate Laravel APP_KEY if needed
	if strings.HasPrefix(projectCfg.Type, "laravel") {
		tplData.AppKey = infisical.GenerateLaravelKey()
	}

	// --- Render templates to temp dir ---
	fmt.Println("\nRendering manifests...")
	tmpDir, err := os.MkdirTemp("", "pnp-render-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	if err := templates.Render(projectCfg.Type, tplData, tmpDir); err != nil {
		return fmt.Errorf("rendering templates: %w", err)
	}

	// --- Write to gitops repo ---
	repo := gitops.NewRepo(globalCfg.GitopsRepo)

	fmt.Println("Updating gitops repo...")
	repo.Pull()

	if err := repo.WriteApp(projectCfg.Name, projectCfg.Environment, tmpDir); err != nil {
		return fmt.Errorf("writing to gitops repo: %w", err)
	}

	// --- Create secrets in Infisical ---
	if !flagSkipSecrets && projectCfg.Database.Enabled && globalCfg.Infisical.Token != "" {
		fmt.Println("Creating secrets in Infisical...")
		client := infisical.NewClient(globalCfg.Infisical.Host, globalCfg.Infisical.Token)
		secrets := map[string]string{
			"username": "app",
			"password": infisical.GeneratePassword(),
		}
		if err := client.CreateSecrets(secrets, projectCfg.Infisical.ProjectSlug, projectCfg.Infisical.EnvSlug, projectCfg.Infisical.SecretsPath); err != nil {
			fmt.Println(errorStyle.Render("Warning: failed to create Infisical secrets: " + err.Error()))
			fmt.Println("You may need to create them manually.")
		}
	}

	// --- Commit and push ---
	commitMsg := fmt.Sprintf("deploy: %s (%s/%s)", projectCfg.Name, projectCfg.Type, projectCfg.Environment)

	if flagPR {
		branch := fmt.Sprintf("deploy/%s", projectCfg.Name)
		fmt.Printf("Creating PR on branch %s...\n", branch)
		if err := repo.CreateBranchAndPush(branch, commitMsg); err != nil {
			return fmt.Errorf("creating branch: %w", err)
		}
		prURL, err := repo.CreatePR(
			fmt.Sprintf("Deploy %s", projectCfg.Name),
			fmt.Sprintf("Automated deployment of %s (%s) to %s environment.", projectCfg.Name, projectCfg.Type, projectCfg.Environment),
			branch,
		)
		if err != nil {
			return fmt.Errorf("creating PR: %w", err)
		}
		fmt.Println(successStyle.Render("PR created: " + prURL))
	} else {
		if err := repo.CommitChanges(commitMsg); err != nil {
			return fmt.Errorf("committing: %w", err)
		}
		if err := repo.Push(); err != nil {
			return fmt.Errorf("pushing: %w", err)
		}
		fmt.Println(successStyle.Render("Pushed to main — ArgoCD will sync automatically."))
	}

	// --- Generate CI workflow ---
	if flagWithCI {
		fmt.Println("Generating GitHub Actions workflow...")
		if err := ci.GenerateWorkflow(projectCfg.Type, projectCfg.Image, cwd); err != nil {
			return fmt.Errorf("generating CI workflow: %w", err)
		}
		fmt.Println(successStyle.Render("Created .github/workflows/deploy.yml"))
	}

	// --- Save .cluster.yaml ---
	if !existingConfig {
		if err := config.SaveProjectConfig(projectCfg); err != nil {
			return fmt.Errorf("saving .cluster.yaml: %w", err)
		}
		fmt.Println(successStyle.Render("Saved .cluster.yaml"))
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("✅ %s deployed to https://%s", projectCfg.Name, projectCfg.Domain)))
	return nil
}

func namespaceFor(name, environment string) string {
	switch environment {
	case "production":
		return name
	default:
		return "preview-" + name
	}
}

func chartPathFor(projectType string) string {
	switch projectType {
	case "laravel-web", "laravel-api":
		return "charts/laravel"
	case "nextjs-fullstack", "nextjs-static":
		return "charts/nextjs"
	case "strapi":
		return "charts/strapi"
	}
	return "charts/nextjs"
}
```

**Step 2: Verify it compiles**

Run: `go build .`
Expected: Success

**Step 3: Commit**

```bash
git add cmd/deploy.go
git commit -m "feat: add deploy command (wizard → render → gitops push)"
```

---

## Task 11: Update Command

**Files:**
- Create: `cmd/update.go`

**Step 1: Implement update command**

```go
// cmd/update.go
package cmd

import (
	"fmt"
	"os"

	"github.com/fwartner/pnp/internal/config"
	"github.com/fwartner/pnp/internal/gitops"
	"github.com/fwartner/pnp/internal/infisical"
	"github.com/fwartner/pnp/internal/templates"
	"github.com/fwartner/pnp/internal/wizard"
	"github.com/spf13/cobra"
	"strings"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an existing deployment from .cluster.yaml",
	RunE:  runUpdate,
}

func init() {
	updateCmd.Flags().BoolVar(&flagPR, "pr", false, "Create a PR instead of pushing directly")
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	fmt.Println(titleStyle.Render("🔄 PnP Update"))
	fmt.Println()

	projectCfg, err := config.LoadProjectConfig()
	if err != nil {
		return fmt.Errorf("no .cluster.yaml found — run 'pnp deploy' first")
	}

	globalCfg, err := config.LoadGlobalConfig()
	if err != nil {
		return fmt.Errorf("loading global config: %w", err)
	}

	namespace := namespaceFor(projectCfg.Name, projectCfg.Environment)
	baseDomain := globalCfg.Defaults.Domain
	subdomain := wizard.Subdomain(projectCfg.Domain, baseDomain)
	domain := baseDomain
	if projectCfg.Environment == "preview" {
		domain = "preview." + baseDomain
	} else if projectCfg.Environment == "staging" {
		domain = "staging." + baseDomain
	}

	tplData := templates.TemplateData{
		Name:                 projectCfg.Name,
		Namespace:            namespace,
		Subdomain:            subdomain,
		Domain:               domain,
		Image:                projectCfg.Image,
		Tag:                  "latest",
		DBName:               projectCfg.Database.Name,
		DBUsername:            "app",
		DBSize:               projectCfg.Database.Size,
		RedisEnabled:         projectCfg.Redis.Enabled,
		QueueEnabled:         projectCfg.Type == "laravel-web",
		SchedulerEnabled:     projectCfg.Type == "laravel-web",
		PersistenceEnabled:   projectCfg.Type == "laravel-web" || projectCfg.Type == "strapi",
		PersistenceSize:      "1Gi",
		InfisicalProjectSlug: projectCfg.Infisical.ProjectSlug,
		InfisicalEnvSlug:     projectCfg.Infisical.EnvSlug,
		InfisicalSecretsPath: projectCfg.Infisical.SecretsPath,
		InfisicalMailEnabled: projectCfg.Type == "laravel-web" || projectCfg.Type == "laravel-api",
		CPU:                  projectCfg.Resources.CPU,
		Memory:               projectCfg.Resources.Memory,
		ChartPath:            chartPathFor(projectCfg.Type),
		RepoURL:              globalCfg.GitopsRemote,
	}

	if tplData.RepoURL == "" {
		tplData.RepoURL = "https://github.com/fwartner/pixelandprocess-gitops.git"
	}
	if tplData.CPU == "" {
		tplData.CPU = "100m"
	}
	if tplData.Memory == "" {
		tplData.Memory = "256Mi"
	}
	if strings.HasPrefix(projectCfg.Type, "laravel") {
		tplData.AppKey = infisical.GenerateLaravelKey()
	}

	tmpDir, err := os.MkdirTemp("", "pnp-update-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	fmt.Println("Re-rendering manifests...")
	if err := templates.Render(projectCfg.Type, tplData, tmpDir); err != nil {
		return fmt.Errorf("rendering: %w", err)
	}

	repo := gitops.NewRepo(globalCfg.GitopsRepo)
	repo.Pull()

	if err := repo.WriteApp(projectCfg.Name, projectCfg.Environment, tmpDir); err != nil {
		return fmt.Errorf("writing: %w", err)
	}

	commitMsg := fmt.Sprintf("update: %s (%s)", projectCfg.Name, projectCfg.Type)

	if flagPR {
		branch := fmt.Sprintf("update/%s", projectCfg.Name)
		if err := repo.CreateBranchAndPush(branch, commitMsg); err != nil {
			return err
		}
		prURL, err := repo.CreatePR("Update "+projectCfg.Name, "Update deployment for "+projectCfg.Name, branch)
		if err != nil {
			return err
		}
		fmt.Println(successStyle.Render("PR created: " + prURL))
	} else {
		if err := repo.CommitChanges(commitMsg); err != nil {
			return err
		}
		if err := repo.Push(); err != nil {
			return err
		}
		fmt.Println(successStyle.Render("Updated and pushed — ArgoCD will sync."))
	}

	return nil
}
```

**Step 2: Verify it compiles**

Run: `go build .`
Expected: Success

**Step 3: Commit**

```bash
git add cmd/update.go
git commit -m "feat: add update command"
```

---

## Task 12: Destroy Command

**Files:**
- Create: `cmd/destroy.go`

**Step 1: Implement destroy command**

```go
// cmd/destroy.go
package cmd

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/fwartner/pnp/internal/config"
	"github.com/fwartner/pnp/internal/gitops"
	"github.com/spf13/cobra"
)

var flagCleanSecrets bool

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Remove a deployment from the cluster",
	RunE:  runDestroy,
}

func init() {
	destroyCmd.Flags().BoolVar(&flagPR, "pr", false, "Create a PR instead of pushing directly")
	destroyCmd.Flags().BoolVar(&flagCleanSecrets, "clean-secrets", false, "Also delete secrets from Infisical")
	rootCmd.AddCommand(destroyCmd)
}

func runDestroy(cmd *cobra.Command, args []string) error {
	fmt.Println(titleStyle.Render("💥 PnP Destroy"))
	fmt.Println()

	projectCfg, err := config.LoadProjectConfig()
	if err != nil {
		return fmt.Errorf("no .cluster.yaml found")
	}

	globalCfg, err := config.LoadGlobalConfig()
	if err != nil {
		return fmt.Errorf("loading global config: %w", err)
	}

	// Confirm destruction
	var confirmed bool
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Destroy %s (%s)?", projectCfg.Name, projectCfg.Environment)).
				Description("This will remove the deployment from the gitops repo. ArgoCD will delete all resources.").
				Value(&confirmed),
		),
	).Run()
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println("Aborted.")
		return nil
	}

	repo := gitops.NewRepo(globalCfg.GitopsRepo)
	repo.Pull()

	fmt.Printf("Removing %s from gitops repo...\n", projectCfg.Name)
	if err := repo.DeleteApp(projectCfg.Name, projectCfg.Environment); err != nil {
		return fmt.Errorf("deleting app: %w", err)
	}

	commitMsg := fmt.Sprintf("destroy: %s", projectCfg.Name)

	if flagPR {
		branch := fmt.Sprintf("destroy/%s", projectCfg.Name)
		if err := repo.CreateBranchAndPush(branch, commitMsg); err != nil {
			return err
		}
		prURL, err := repo.CreatePR("Destroy "+projectCfg.Name, "Remove deployment for "+projectCfg.Name, branch)
		if err != nil {
			return err
		}
		fmt.Println(successStyle.Render("PR created: " + prURL))
	} else {
		if err := repo.CommitChanges(commitMsg); err != nil {
			return err
		}
		if err := repo.Push(); err != nil {
			return err
		}
		fmt.Println(successStyle.Render("Removed and pushed — ArgoCD will clean up resources."))
	}

	return nil
}
```

**Step 2: Verify it compiles**

Run: `go build .`
Expected: Success

**Step 3: Commit**

```bash
git add cmd/destroy.go
git commit -m "feat: add destroy command"
```

---

## Task 13: Status Command

**Files:**
- Create: `cmd/status.go`

**Step 1: Implement status command**

```go
// cmd/status.go
package cmd

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/fwartner/pnp/internal/config"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show deployment status from ArgoCD",
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	fmt.Println(titleStyle.Render("📊 PnP Status"))
	fmt.Println()

	projectCfg, err := config.LoadProjectConfig()
	if err != nil {
		return fmt.Errorf("no .cluster.yaml found")
	}

	appName := "helm-" + projectCfg.Name

	// Try argocd CLI first
	out, err := exec.Command("argocd", "app", "get", appName, "-o", "json").Output()
	if err != nil {
		// Fallback: try kubectl
		out, err = exec.Command("kubectl", "get", "application", appName, "-n", "argocd", "-o", "json").Output()
		if err != nil {
			return fmt.Errorf("could not get app status (install argocd CLI or kubectl): %w", err)
		}
	}

	var app struct {
		Status struct {
			Sync struct {
				Status string `json:"status"`
			} `json:"sync"`
			Health struct {
				Status string `json:"status"`
			} `json:"health"`
		} `json:"status"`
	}

	if err := json.Unmarshal(out, &app); err != nil {
		return fmt.Errorf("parsing status: %w", err)
	}

	syncStatus := app.Status.Sync.Status
	healthStatus := app.Status.Health.Status

	fmt.Printf("App:         %s\n", appName)
	fmt.Printf("Environment: %s\n", projectCfg.Environment)
	fmt.Printf("Domain:      https://%s\n", projectCfg.Domain)
	fmt.Printf("Sync:        %s\n", colorizeStatus(syncStatus))
	fmt.Printf("Health:      %s\n", colorizeStatus(healthStatus))

	return nil
}

func colorizeStatus(status string) string {
	switch status {
	case "Synced", "Healthy":
		return successStyle.Render(status)
	case "OutOfSync", "Degraded":
		return errorStyle.Render(status)
	default:
		return status
	}
}
```

**Step 2: Verify it compiles**

Run: `go build .`
Expected: Success

**Step 3: Commit**

```bash
git add cmd/status.go
git commit -m "feat: add status command (ArgoCD sync/health)"
```

---

## Task 14: Init Command

**Files:**
- Create: `cmd/init.go`

The init command creates `~/.pnp.yaml` interactively.

**Step 1: Implement init command**

```go
// cmd/init.go
package cmd

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/fwartner/pnp/internal/config"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize PnP global configuration",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	fmt.Println(titleStyle.Render("⚙️  PnP Init"))
	fmt.Println()

	cfg, _ := config.LoadGlobalConfig()

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Path to gitops repo (local clone)").
				Value(&cfg.GitopsRepo),

			huh.NewInput().
				Title("Gitops repo remote URL").
				Placeholder("https://github.com/org/gitops.git").
				Value(&cfg.GitopsRemote),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Infisical vault URL").
				Value(&cfg.Infisical.Host),

			huh.NewInput().
				Title("Infisical machine identity token").
				Value(&cfg.Infisical.Token),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Default domain").
				Value(&cfg.Defaults.Domain),

			huh.NewInput().
				Title("Default image registry").
				Value(&cfg.Defaults.ImageRegistry),

			huh.NewInput().
				Title("Default GitHub org/user").
				Value(&cfg.Defaults.GithubOrg),
		),
	).Run()
	if err != nil {
		return err
	}

	if err := config.SaveGlobalConfig(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Println(successStyle.Render(fmt.Sprintf("Config saved to %s", config.GlobalConfigPath())))
	return nil
}
```

**Step 2: Verify it compiles**

Run: `go build .`
Expected: Success

**Step 3: Commit**

```bash
git add cmd/init.go
git commit -m "feat: add init command for global config setup"
```

---

## Task 15: Goreleaser & Homebrew Distribution

**Files:**
- Create: `.goreleaser.yaml`

**Step 1: Create goreleaser config**

```yaml
# .goreleaser.yaml
version: 2

builds:
  - env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X github.com/fwartner/pnp/cmd.version={{.Version}}
      - -X github.com/fwartner/pnp/cmd.commit={{.ShortCommit}}
    binary: pnp

brews:
  - repository:
      owner: fwartner
      name: homebrew-tap
    name: pnp
    homepage: https://github.com/fwartner/pnp
    description: "Pixel & Process Kubernetes deployment manager"
    install: |
      bin.install "pnp"

archives:
  - format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
```

**Step 2: Commit**

```bash
git add .goreleaser.yaml
git commit -m "feat: add goreleaser config for distribution"
```

---

## Task 16: End-to-End Smoke Test

**Files:**
- Create: `e2e_test.go`

**Step 1: Write an e2e test that exercises the template pipeline**

```go
// e2e_test.go
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fwartner/pnp/internal/config"
	"github.com/fwartner/pnp/internal/detect"
	"github.com/fwartner/pnp/internal/gitops"
	"github.com/fwartner/pnp/internal/templates"
)

func TestE2E_LaravelDeployPipeline(t *testing.T) {
	// 1. Set up a fake Laravel project
	projectDir := t.TempDir()
	os.WriteFile(filepath.Join(projectDir, "composer.json"), []byte(`{}`), 0644)
	os.WriteFile(filepath.Join(projectDir, "artisan"), []byte(`#!/usr/bin/env php`), 0644)
	os.MkdirAll(filepath.Join(projectDir, "app", "Jobs"), 0755)
	os.WriteFile(filepath.Join(projectDir, "app", "Jobs", "Test.php"), []byte(`class Test`), 0644)

	// 2. Detect type
	result := detect.DetectProjectType(projectDir)
	if result.Type != "laravel-web" {
		t.Fatalf("expected laravel-web, got %s", result.Type)
	}

	// 3. Build config (simulating wizard output)
	projectCfg := config.ProjectConfig{
		Name:        "test-customer",
		Type:        result.Type,
		Environment: "preview",
		Domain:      "test-customer.preview.pixelandprocess.de",
		Image:       "ghcr.io/fwartner/test-customer",
		Database:    config.DatabaseConfig{Enabled: true, Size: "5Gi", Name: "app"},
		Redis:       config.RedisConfig{Enabled: true},
		Infisical:   config.ProjectInfisical{ProjectSlug: "customer-apps-f-jq3", EnvSlug: "prod", SecretsPath: "/test-customer/db"},
		Resources:   config.ResourcesConfig{CPU: "100m", Memory: "256Mi"},
	}

	// 4. Render templates
	tplData := templates.TemplateData{
		Name:                 projectCfg.Name,
		Namespace:            "preview-test-customer",
		Subdomain:            "test-customer",
		Domain:               "preview.pixelandprocess.de",
		Image:                projectCfg.Image,
		Tag:                  "latest",
		AppKey:               "base64:testkey",
		DBName:               "app",
		DBUsername:            "app",
		DBSize:               "5Gi",
		RedisEnabled:         true,
		QueueEnabled:         true,
		SchedulerEnabled:     true,
		PersistenceEnabled:   true,
		PersistenceSize:      "1Gi",
		InfisicalProjectSlug: "customer-apps-f-jq3",
		InfisicalEnvSlug:     "prod",
		InfisicalSecretsPath: "/test-customer/db",
		CPU:                  "100m",
		Memory:               "256Mi",
		ChartPath:            "charts/laravel",
		RepoURL:              "https://github.com/fwartner/pixelandprocess-gitops.git",
	}

	renderDir := t.TempDir()
	if err := templates.Render(projectCfg.Type, tplData, renderDir); err != nil {
		t.Fatalf("render failed: %v", err)
	}

	// 5. Write to fake gitops repo
	gitopsDir := t.TempDir()
	os.MkdirAll(filepath.Join(gitopsDir, "apps", "previews"), 0755)
	repo := gitops.NewRepo(gitopsDir)

	if err := repo.WriteApp("test-customer", "preview", renderDir); err != nil {
		t.Fatalf("write app failed: %v", err)
	}

	// 6. Verify output
	appDir := filepath.Join(gitopsDir, "apps", "previews", "test-customer")

	// Check Chart.yaml
	chart, err := os.ReadFile(filepath.Join(appDir, "Chart.yaml"))
	if err != nil {
		t.Fatalf("missing Chart.yaml: %v", err)
	}
	if !strings.Contains(string(chart), "name: test-customer") {
		t.Error("Chart.yaml missing name")
	}

	// Check application.yaml has ArgoCD Application
	appYaml, err := os.ReadFile(filepath.Join(appDir, "templates", "application.yaml"))
	if err != nil {
		t.Fatalf("missing application.yaml: %v", err)
	}
	appContent := string(appYaml)
	if !strings.Contains(appContent, "helm-test-customer") {
		t.Error("missing ArgoCD app name")
	}
	if !strings.Contains(appContent, "charts/laravel") {
		t.Error("missing chart path")
	}
	if !strings.Contains(appContent, "queue") {
		t.Error("missing queue config")
	}

	// Check CNPG cluster
	cnpg, err := os.ReadFile(filepath.Join(appDir, "templates", "cnpg-cluster.yaml"))
	if err != nil {
		t.Fatalf("missing cnpg-cluster.yaml: %v", err)
	}
	if !strings.Contains(string(cnpg), "test-customer-db") {
		t.Error("missing CNPG cluster name")
	}

	// Check Infisical secrets
	inf, err := os.ReadFile(filepath.Join(appDir, "templates", "infisical-secrets.yaml"))
	if err != nil {
		t.Fatalf("missing infisical-secrets.yaml: %v", err)
	}
	infContent := string(inf)
	if !strings.Contains(infContent, "customer-apps-f-jq3") {
		t.Error("missing Infisical project slug")
	}
	if !strings.Contains(infContent, "mail-credentials") {
		t.Error("missing mail credentials (should be present for laravel-web)")
	}
}

func TestE2E_NextjsStaticDeployPipeline(t *testing.T) {
	// Simpler: no DB, no Infisical, no Redis
	tplData := templates.TemplateData{
		Name:      "landing-page",
		Namespace: "preview-landing-page",
		Subdomain: "landing-page",
		Domain:    "preview.pixelandprocess.de",
		Image:     "ghcr.io/fwartner/landing-page",
		Tag:       "latest",
		CPU:       "100m",
		Memory:    "128Mi",
		ChartPath: "charts/nextjs",
		RepoURL:   "https://github.com/fwartner/pixelandprocess-gitops.git",
	}

	renderDir := t.TempDir()
	if err := templates.Render("nextjs-static", tplData, renderDir); err != nil {
		t.Fatalf("render failed: %v", err)
	}

	// Should NOT have cnpg or infisical templates
	if _, err := os.Stat(filepath.Join(renderDir, "templates", "cnpg-cluster.yaml")); err == nil {
		t.Error("nextjs-static should not have cnpg-cluster.yaml")
	}
	if _, err := os.Stat(filepath.Join(renderDir, "templates", "infisical-secrets.yaml")); err == nil {
		t.Error("nextjs-static should not have infisical-secrets.yaml")
	}

	// Should have application.yaml
	appYaml, err := os.ReadFile(filepath.Join(renderDir, "templates", "application.yaml"))
	if err != nil {
		t.Fatalf("missing application.yaml: %v", err)
	}
	if !strings.Contains(string(appYaml), "helm-landing-page") {
		t.Error("missing app name")
	}
}
```

**Step 2: Run all tests**

Run: `go test ./... -v`
Expected: All PASS

**Step 3: Commit**

```bash
git add e2e_test.go
git commit -m "test: add e2e smoke tests for deploy pipeline"
```

---

## Summary

| Task | Component | Description |
|------|-----------|-------------|
| 1 | Scaffolding | Go module, Cobra CLI, main.go |
| 2 | Config | Global config (~/.pnp.yaml) |
| 3 | Config | Project config (.cluster.yaml) |
| 4 | Detection | Auto-detect project type from files |
| 5 | Templates | Render Helm manifests for all 5 types |
| 6 | Gitops | Clone, write, commit, push, PR operations |
| 7 | Infisical | API client for secret management |
| 8 | Wizard | Interactive TUI with huh forms |
| 9 | CI | GitHub Actions workflow generator |
| 10 | Deploy | Main deploy command |
| 11 | Update | Update existing deployment |
| 12 | Destroy | Remove deployment |
| 13 | Status | Show ArgoCD sync status |
| 14 | Init | Global config setup wizard |
| 15 | Release | Goreleaser + Homebrew tap |
| 16 | E2E | End-to-end smoke tests |

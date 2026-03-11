package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fwartner/pnp/internal/detect"
	"github.com/fwartner/pnp/internal/gitops"
	"github.com/fwartner/pnp/internal/templates"
)

func TestE2E_LaravelDeployPipeline(t *testing.T) {
	// 1. Create fake Laravel project in temp dir
	projectDir := t.TempDir()

	// composer.json
	if err := os.WriteFile(filepath.Join(projectDir, "composer.json"), []byte(`{"require":{"laravel/framework":"^11.0"}}`), 0644); err != nil {
		t.Fatalf("writing composer.json: %v", err)
	}

	// artisan
	if err := os.WriteFile(filepath.Join(projectDir, "artisan"), []byte("#!/usr/bin/env php\n"), 0755); err != nil {
		t.Fatalf("writing artisan: %v", err)
	}

	// app/Jobs/Test.php
	jobsDir := filepath.Join(projectDir, "app", "Jobs")
	if err := os.MkdirAll(jobsDir, 0755); err != nil {
		t.Fatalf("creating Jobs dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(jobsDir, "Test.php"), []byte("<?php\nnamespace App\\Jobs;\nclass Test {}\n"), 0644); err != nil {
		t.Fatalf("writing Test.php: %v", err)
	}

	// 2. Detect type
	result := detect.DetectProjectType(projectDir)
	if result.Type != "laravel-web" {
		t.Fatalf("expected type laravel-web, got %s", result.Type)
	}
	if result.Confidence != "high" {
		t.Fatalf("expected confidence high, got %s", result.Confidence)
	}

	// 3. Build TemplateData (simulating wizard output)
	data := templates.TemplateData{
		Name:                 "test-customer",
		Namespace:            "customer-apps-f-jq3",
		Subdomain:            "test-customer",
		Domain:               "pixelandprocess.de",
		Image:                "ghcr.io/fwartner/test-customer",
		Tag:                  "latest",
		AppKey:               "base64:FAKEKEYHERE=",
		DBName:               "test_customer",
		DBUsername:            "test_customer",
		DBSize:               "1Gi",
		RedisEnabled:         true,
		QueueEnabled:         true,
		SchedulerEnabled:     true,
		PersistenceEnabled:   true,
		PersistenceSize:      "5Gi",
		InfisicalProjectSlug: "customer-apps-f-jq3",
		InfisicalEnvSlug:     "prod",
		InfisicalSecretsPath: "/test-customer",
		InfisicalMailEnabled: true,
		CPU:                  "100m",
		Memory:               "256Mi",
		ChartPath:            "charts/laravel",
		RepoURL:              "https://github.com/fwartner/gitops-repo.git",
	}

	// 4. Render templates
	renderDir := t.TempDir()
	if err := templates.Render("laravel-web", data, renderDir); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// 5. Write to fake gitops dir using gitops.NewRepo + WriteApp
	gitopsDir := t.TempDir()
	repo := gitops.NewRepo(gitopsDir)
	if err := repo.WriteApp("test-customer", "production", renderDir); err != nil {
		t.Fatalf("WriteApp failed: %v", err)
	}

	appDir := repo.AppPath("test-customer", "production")

	// 6. Verify Chart.yaml
	chartData, err := os.ReadFile(filepath.Join(appDir, "Chart.yaml"))
	if err != nil {
		t.Fatalf("reading Chart.yaml: %v", err)
	}
	chartContent := string(chartData)
	if !strings.Contains(chartContent, "name: test-customer") {
		t.Errorf("Chart.yaml should contain 'name: test-customer', got:\n%s", chartContent)
	}

	// Verify application.yaml
	appData, err := os.ReadFile(filepath.Join(appDir, "templates", "application.yaml"))
	if err != nil {
		t.Fatalf("reading application.yaml: %v", err)
	}
	appContent := string(appData)
	for _, expected := range []string{"helm-test-customer", "queue"} {
		if !strings.Contains(appContent, expected) {
			t.Errorf("application.yaml should contain %q, got:\n%s", expected, appContent)
		}
	}

	// charts/laravel is rendered into values.yaml (the Helm values file), not application.yaml
	valuesData, err := os.ReadFile(filepath.Join(appDir, "values.yaml"))
	if err != nil {
		t.Fatalf("reading values.yaml: %v", err)
	}
	valuesContent := string(valuesData)
	if !strings.Contains(valuesContent, "charts/laravel") {
		t.Errorf("values.yaml should contain 'charts/laravel', got:\n%s", valuesContent)
	}

	// Verify cnpg-cluster.yaml
	cnpgData, err := os.ReadFile(filepath.Join(appDir, "templates", "cnpg-cluster.yaml"))
	if err != nil {
		t.Fatalf("reading cnpg-cluster.yaml: %v", err)
	}
	cnpgContent := string(cnpgData)
	if !strings.Contains(cnpgContent, "test-customer-db") {
		t.Errorf("cnpg-cluster.yaml should contain 'test-customer-db', got:\n%s", cnpgContent)
	}

	// Verify infisical-secrets.yaml
	infData, err := os.ReadFile(filepath.Join(appDir, "templates", "infisical-secrets.yaml"))
	if err != nil {
		t.Fatalf("reading infisical-secrets.yaml: %v", err)
	}
	infContent := string(infData)
	if !strings.Contains(infContent, "customer-apps-f-jq3") {
		t.Errorf("infisical-secrets.yaml should contain 'customer-apps-f-jq3', got:\n%s", infContent)
	}
	if !strings.Contains(infContent, "mail-credentials") {
		t.Errorf("infisical-secrets.yaml should contain 'mail-credentials', got:\n%s", infContent)
	}
}

func TestE2E_NextjsStaticDeployPipeline(t *testing.T) {
	// 1. Build TemplateData for nextjs-static
	data := templates.TemplateData{
		Name:      "landing-page",
		Namespace: "landing-ns",
		Subdomain: "landing",
		Domain:    "example.com",
		Image:     "ghcr.io/fwartner/landing-page",
		Tag:       "latest",
		CPU:       "50m",
		Memory:    "128Mi",
		ChartPath: "charts/nextjs-static",
		RepoURL:   "https://github.com/fwartner/gitops-repo.git",
	}

	// 2. Render to temp dir
	renderDir := t.TempDir()
	if err := templates.Render("nextjs-static", data, renderDir); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// 3. Verify NO cnpg-cluster.yaml and NO infisical-secrets.yaml
	if _, err := os.Stat(filepath.Join(renderDir, "templates", "cnpg-cluster.yaml")); err == nil {
		t.Errorf("cnpg-cluster.yaml should NOT exist for nextjs-static")
	}
	if _, err := os.Stat(filepath.Join(renderDir, "templates", "infisical-secrets.yaml")); err == nil {
		t.Errorf("infisical-secrets.yaml should NOT exist for nextjs-static")
	}

	// 4. Verify application.yaml contains "helm-landing-page"
	appData, err := os.ReadFile(filepath.Join(renderDir, "templates", "application.yaml"))
	if err != nil {
		t.Fatalf("reading application.yaml: %v", err)
	}
	appContent := string(appData)
	if !strings.Contains(appContent, "helm-landing-page") {
		t.Errorf("application.yaml should contain 'helm-landing-page', got:\n%s", appContent)
	}
}

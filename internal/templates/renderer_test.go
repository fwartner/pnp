package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testData() TemplateData {
	return TemplateData{
		Name:                 "my-app",
		Namespace:            "my-namespace",
		Subdomain:            "app",
		Domain:               "example.com",
		Image:                "ghcr.io/org/my-app",
		Tag:                  "latest",
		AppKey:               "base64:testkey123",
		DBName:               "my_app_db",
		DBUsername:            "my_app_user",
		DBSize:               "5Gi",
		RedisEnabled:         true,
		QueueEnabled:         true,
		SchedulerEnabled:     true,
		PersistenceEnabled:   true,
		PersistenceSize:      "10Gi",
		InfisicalProjectSlug: "my-project-slug",
		InfisicalEnvSlug:     "production",
		InfisicalSecretsPath: "/db",
		InfisicalMailEnabled: true,
		CPU:                  "250m",
		Memory:               "256Mi",
		ChartPath:            "charts/my-app",
		RepoURL:              "https://github.com/org/gitops.git",
	}
}

func TestRenderLaravelWeb(t *testing.T) {
	outDir := t.TempDir()
	data := testData()

	err := Render("laravel-web", data, outDir)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Chart.yaml has name
	chartContent, err := os.ReadFile(filepath.Join(outDir, "Chart.yaml"))
	if err != nil {
		t.Fatalf("reading Chart.yaml: %v", err)
	}
	if !strings.Contains(string(chartContent), "name: my-app") {
		t.Errorf("Chart.yaml missing name, got:\n%s", chartContent)
	}

	// application.yaml has helm release name and queue config
	appContent, err := os.ReadFile(filepath.Join(outDir, "templates", "application.yaml"))
	if err != nil {
		t.Fatalf("reading application.yaml: %v", err)
	}
	if !strings.Contains(string(appContent), "name: helm-my-app") {
		t.Errorf("application.yaml missing helm release name, got:\n%s", appContent)
	}
	if !strings.Contains(string(appContent), "queue:") {
		t.Errorf("application.yaml missing queue config, got:\n%s", appContent)
	}
	if !strings.Contains(string(appContent), "enabled: true") {
		t.Errorf("application.yaml missing queue enabled: true")
	}

	// cnpg-cluster.yaml has db name
	cnpgContent, err := os.ReadFile(filepath.Join(outDir, "templates", "cnpg-cluster.yaml"))
	if err != nil {
		t.Fatalf("reading cnpg-cluster.yaml: %v", err)
	}
	if !strings.Contains(string(cnpgContent), "database: my_app_db") {
		t.Errorf("cnpg-cluster.yaml missing db name, got:\n%s", cnpgContent)
	}

	// infisical-secrets.yaml has project slug
	infisicalContent, err := os.ReadFile(filepath.Join(outDir, "templates", "infisical-secrets.yaml"))
	if err != nil {
		t.Fatalf("reading infisical-secrets.yaml: %v", err)
	}
	if !strings.Contains(string(infisicalContent), "projectSlug: my-project-slug") {
		t.Errorf("infisical-secrets.yaml missing project slug, got:\n%s", infisicalContent)
	}
	// Laravel types should have mail infisical secret too
	if !strings.Contains(string(infisicalContent), "mail-infisical") {
		t.Errorf("infisical-secrets.yaml missing mail infisical secret for laravel type")
	}
}

func TestRenderNextjsStatic(t *testing.T) {
	outDir := t.TempDir()
	data := testData()

	err := Render("nextjs-static", data, outDir)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Chart.yaml should exist
	if _, err := os.Stat(filepath.Join(outDir, "Chart.yaml")); err != nil {
		t.Errorf("Chart.yaml should exist: %v", err)
	}

	// application.yaml should exist
	if _, err := os.Stat(filepath.Join(outDir, "templates", "application.yaml")); err != nil {
		t.Errorf("application.yaml should exist: %v", err)
	}

	// cnpg-cluster.yaml should NOT exist
	if _, err := os.Stat(filepath.Join(outDir, "templates", "cnpg-cluster.yaml")); !os.IsNotExist(err) {
		t.Errorf("cnpg-cluster.yaml should not exist for nextjs-static")
	}

	// infisical-secrets.yaml should NOT exist
	if _, err := os.Stat(filepath.Join(outDir, "templates", "infisical-secrets.yaml")); !os.IsNotExist(err) {
		t.Errorf("infisical-secrets.yaml should not exist for nextjs-static")
	}
}

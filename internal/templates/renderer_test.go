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
		DBPassword:           "supersecretpassword123",
		DBSize:               "5Gi",
		RedisEnabled:         true,
		QueueEnabled:         true,
		SchedulerEnabled:     true,
		PersistenceEnabled:   true,
		PersistenceSize:      "10Gi",
		InfisicalProjectSlug:     "my-project-slug",
		InfisicalEnvSlug:         "production",
		InfisicalSecretsPath:     "/db",
		InfisicalMailEnabled:     true,
		InfisicalHost:            "https://vault.example.com",
		InfisicalMailProjectSlug: "cluster-shared-test",
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

	// db-credentials.yaml should exist with generated password
	dbCredContent, err := os.ReadFile(filepath.Join(outDir, "templates", "db-credentials.yaml"))
	if err != nil {
		t.Fatalf("reading db-credentials.yaml: %v", err)
	}
	dbCredStr := string(dbCredContent)
	if !strings.Contains(dbCredStr, "my-app-db-credentials") {
		t.Errorf("db-credentials.yaml missing secret name")
	}
	if !strings.Contains(dbCredStr, "supersecretpassword123") {
		t.Errorf("db-credentials.yaml missing password")
	}
	if !strings.Contains(dbCredStr, "kubernetes.io/basic-auth") {
		t.Errorf("db-credentials.yaml missing secret type")
	}

	// app-secret.yaml should exist with APP_KEY for Laravel
	appSecretContent, err := os.ReadFile(filepath.Join(outDir, "templates", "app-secret.yaml"))
	if err != nil {
		t.Fatalf("reading app-secret.yaml: %v", err)
	}
	appSecretStr := string(appSecretContent)
	if !strings.Contains(appSecretStr, "base64:testkey123") {
		t.Errorf("app-secret.yaml missing APP_KEY")
	}
	if !strings.Contains(appSecretStr, "my-app-env") {
		t.Errorf("app-secret.yaml missing secret name")
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

	// db-credentials.yaml should NOT exist
	if _, err := os.Stat(filepath.Join(outDir, "templates", "db-credentials.yaml")); !os.IsNotExist(err) {
		t.Errorf("db-credentials.yaml should not exist for nextjs-static")
	}

	// app-secret.yaml should NOT exist
	if _, err := os.Stat(filepath.Join(outDir, "templates", "app-secret.yaml")); !os.IsNotExist(err) {
		t.Errorf("app-secret.yaml should not exist for nextjs-static")
	}
}

func TestRenderLaravelAPI(t *testing.T) {
	outDir := t.TempDir()
	data := TemplateData{
		Name:                     "api-service",
		Namespace:                "preview-api-service",
		Subdomain:                "api-service",
		Domain:                   "preview.pixelandprocess.de",
		Image:                    "ghcr.io/test/api",
		Tag:                      "v1.2.0",
		AppKey:                   "base64:apikey456",
		DBName:                   "api_db",
		DBUsername:                "api_user",
		DBSize:                   "10Gi",
		PersistenceSize:          "1Gi",
		InfisicalProjectSlug:     "api-slug",
		InfisicalEnvSlug:         "prod",
		InfisicalSecretsPath:     "/api/db",
		InfisicalHost:            "https://vault.pnp.de",
		InfisicalMailProjectSlug: "mail-slug",
		CPU:                      "100m",
		Memory:                   "256Mi",
		ChartPath:                "charts/laravel",
		RepoURL:                  "https://github.com/test/gitops.git",
	}

	err := Render("laravel-api", data, outDir)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Chart.yaml has name
	chartContent, err := os.ReadFile(filepath.Join(outDir, "Chart.yaml"))
	if err != nil {
		t.Fatalf("reading Chart.yaml: %v", err)
	}
	if !strings.Contains(string(chartContent), "name: api-service") {
		t.Errorf("Chart.yaml missing name, got:\n%s", chartContent)
	}

	// application.yaml has helm release name, queue disabled, scheduler disabled
	appContent, err := os.ReadFile(filepath.Join(outDir, "templates", "application.yaml"))
	if err != nil {
		t.Fatalf("reading application.yaml: %v", err)
	}
	appStr := string(appContent)
	if !strings.Contains(appStr, "name: helm-api-service") {
		t.Errorf("application.yaml missing helm release name")
	}
	// Verify queue and scheduler are disabled for laravel-api
	if !strings.Contains(appStr, "queue:") {
		t.Errorf("application.yaml missing queue section")
	}
	if !strings.Contains(appStr, "scheduler:") {
		t.Errorf("application.yaml missing scheduler section")
	}
	// Check that queue/scheduler are followed by enabled: false
	queueIdx := strings.Index(appStr, "queue:")
	schedulerIdx := strings.Index(appStr, "scheduler:")
	if queueIdx == -1 || !strings.Contains(appStr[queueIdx:queueIdx+50], "enabled: false") {
		t.Errorf("application.yaml queue should have enabled: false")
	}
	if schedulerIdx == -1 || !strings.Contains(appStr[schedulerIdx:schedulerIdx+50], "enabled: false") {
		t.Errorf("application.yaml scheduler should have enabled: false")
	}

	// cnpg-cluster.yaml exists with correct DB name
	cnpgContent, err := os.ReadFile(filepath.Join(outDir, "templates", "cnpg-cluster.yaml"))
	if err != nil {
		t.Fatalf("reading cnpg-cluster.yaml: %v", err)
	}
	if !strings.Contains(string(cnpgContent), "database: api_db") {
		t.Errorf("cnpg-cluster.yaml missing db name, got:\n%s", cnpgContent)
	}

	// infisical-secrets.yaml exists with both DB and mail secrets
	infisicalContent, err := os.ReadFile(filepath.Join(outDir, "templates", "infisical-secrets.yaml"))
	if err != nil {
		t.Fatalf("reading infisical-secrets.yaml: %v", err)
	}
	infStr := string(infisicalContent)
	if !strings.Contains(infStr, "api-service-db-credentials") {
		t.Errorf("infisical-secrets.yaml missing db credentials secret")
	}
	if !strings.Contains(infStr, "api-service-mail-credentials") {
		t.Errorf("infisical-secrets.yaml missing mail credentials secret for laravel-api")
	}
}

func TestRenderNextjsFullstack(t *testing.T) {
	outDir := t.TempDir()
	data := TemplateData{
		Name:                 "frontend-app",
		Namespace:            "preview-frontend-app",
		Subdomain:            "frontend-app",
		Domain:               "preview.pixelandprocess.de",
		Image:                "ghcr.io/test/frontend",
		Tag:                  "latest",
		DBName:               "frontend_db",
		DBUsername:            "frontend_user",
		DBSize:               "5Gi",
		InfisicalProjectSlug: "frontend-slug",
		InfisicalEnvSlug:     "prod",
		InfisicalSecretsPath: "/frontend/db",
		InfisicalHost:        "https://vault.pnp.de",
		CPU:                  "200m",
		Memory:               "512Mi",
		ChartPath:            "charts/nextjs",
		RepoURL:              "https://github.com/test/gitops.git",
	}

	err := Render("nextjs-fullstack", data, outDir)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Chart.yaml has name
	chartContent, err := os.ReadFile(filepath.Join(outDir, "Chart.yaml"))
	if err != nil {
		t.Fatalf("reading Chart.yaml: %v", err)
	}
	if !strings.Contains(string(chartContent), "name: frontend-app") {
		t.Errorf("Chart.yaml missing name, got:\n%s", chartContent)
	}

	// application.yaml has helm release name and DATABASE_URL
	appContent, err := os.ReadFile(filepath.Join(outDir, "templates", "application.yaml"))
	if err != nil {
		t.Fatalf("reading application.yaml: %v", err)
	}
	appStr := string(appContent)
	if !strings.Contains(appStr, "name: helm-frontend-app") {
		t.Errorf("application.yaml missing helm release name")
	}
	if !strings.Contains(appStr, "DATABASE_URL") {
		t.Errorf("application.yaml missing DATABASE_URL")
	}

	// cnpg-cluster.yaml exists
	if _, err := os.Stat(filepath.Join(outDir, "templates", "cnpg-cluster.yaml")); err != nil {
		t.Errorf("cnpg-cluster.yaml should exist for nextjs-fullstack: %v", err)
	}

	// infisical-secrets.yaml exists (DB only, no mail)
	infisicalContent, err := os.ReadFile(filepath.Join(outDir, "templates", "infisical-secrets.yaml"))
	if err != nil {
		t.Fatalf("reading infisical-secrets.yaml: %v", err)
	}
	infStr := string(infisicalContent)
	if !strings.Contains(infStr, "frontend-app-db-credentials") {
		t.Errorf("infisical-secrets.yaml missing db credentials secret")
	}
	// nextjs-fullstack should NOT have mail credentials
	if strings.Contains(infStr, "mail-credentials") {
		t.Errorf("infisical-secrets.yaml should NOT contain mail-credentials for nextjs-fullstack")
	}
}

func TestRenderStrapi(t *testing.T) {
	outDir := t.TempDir()
	data := TemplateData{
		Name:                 "cms-app",
		Namespace:            "preview-cms-app",
		Subdomain:            "cms-app",
		Domain:               "preview.pixelandprocess.de",
		Image:                "ghcr.io/test/cms",
		Tag:                  "v2.0.0",
		DBName:               "cms_db",
		DBUsername:            "cms_user",
		DBSize:               "5Gi",
		PersistenceSize:      "2Gi",
		InfisicalProjectSlug: "cms-slug",
		InfisicalEnvSlug:     "prod",
		InfisicalSecretsPath: "/cms/db",
		InfisicalHost:        "https://vault.pnp.de",
		CPU:                  "300m",
		Memory:               "768Mi",
		ChartPath:            "charts/strapi",
		RepoURL:              "https://github.com/test/gitops.git",
	}

	err := Render("strapi", data, outDir)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Chart.yaml has name
	chartContent, err := os.ReadFile(filepath.Join(outDir, "Chart.yaml"))
	if err != nil {
		t.Fatalf("reading Chart.yaml: %v", err)
	}
	if !strings.Contains(string(chartContent), "name: cms-app") {
		t.Errorf("Chart.yaml missing name, got:\n%s", chartContent)
	}

	// application.yaml has helm release name and persistence with PersistenceSize
	appContent, err := os.ReadFile(filepath.Join(outDir, "templates", "application.yaml"))
	if err != nil {
		t.Fatalf("reading application.yaml: %v", err)
	}
	appStr := string(appContent)
	if !strings.Contains(appStr, "name: helm-cms-app") {
		t.Errorf("application.yaml missing helm release name")
	}
	if !strings.Contains(appStr, "persistence:") {
		t.Errorf("application.yaml missing persistence section")
	}
	if !strings.Contains(appStr, "size: 2Gi") {
		t.Errorf("application.yaml missing persistence size 2Gi, got:\n%s", appStr)
	}

	// cnpg-cluster.yaml exists
	if _, err := os.Stat(filepath.Join(outDir, "templates", "cnpg-cluster.yaml")); err != nil {
		t.Errorf("cnpg-cluster.yaml should exist for strapi: %v", err)
	}

	// infisical-secrets.yaml exists (DB only, no mail)
	infisicalContent, err := os.ReadFile(filepath.Join(outDir, "templates", "infisical-secrets.yaml"))
	if err != nil {
		t.Fatalf("reading infisical-secrets.yaml: %v", err)
	}
	infStr := string(infisicalContent)
	if !strings.Contains(infStr, "cms-app-db-credentials") {
		t.Errorf("infisical-secrets.yaml missing db credentials secret")
	}
	// strapi should NOT have mail credentials
	if strings.Contains(infStr, "mail-credentials") {
		t.Errorf("infisical-secrets.yaml should NOT contain mail-credentials for strapi")
	}
}

func TestRender_UnknownType(t *testing.T) {
	outDir := t.TempDir()
	data := testData()

	err := Render("unknown-type", data, outDir)
	if err == nil {
		t.Fatal("expected error for unknown project type, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported project type") {
		t.Errorf("expected 'unsupported project type' in error, got: %v", err)
	}
}

func TestRender_TemplateDataFields(t *testing.T) {
	outDir := t.TempDir()
	data := TemplateData{
		Name:                 "resource-app",
		Namespace:            "preview-resource-app",
		Subdomain:            "resource-app",
		Domain:               "preview.pixelandprocess.de",
		Image:                "ghcr.io/test/resource",
		Tag:                  "latest",
		AppKey:               "base64:reskey",
		DBName:               "res_db",
		DBUsername:            "res_user",
		DBSize:               "5Gi",
		PersistenceSize:      "1Gi",
		InfisicalProjectSlug: "res-slug",
		InfisicalEnvSlug:     "prod",
		InfisicalSecretsPath: "/res/db",
		InfisicalHost:        "https://vault.pnp.de",
		InfisicalMailProjectSlug: "mail-res",
		CPU:                  "500m",
		Memory:               "1024Mi",
		ChartPath:            "charts/laravel",
		RepoURL:              "https://github.com/test/gitops.git",
	}

	err := Render("laravel-web", data, outDir)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Verify CPU and Memory values appear in application.yaml
	appContent, err := os.ReadFile(filepath.Join(outDir, "templates", "application.yaml"))
	if err != nil {
		t.Fatalf("reading application.yaml: %v", err)
	}
	appStr := string(appContent)
	if !strings.Contains(appStr, "cpu: 500m") {
		t.Errorf("application.yaml missing cpu: 500m")
	}
	if !strings.Contains(appStr, "memory: 1024Mi") {
		t.Errorf("application.yaml missing memory: 1024Mi")
	}
}

func TestRender_InfisicalHostConfigurable(t *testing.T) {
	outDir := t.TempDir()
	data := TemplateData{
		Name:                 "host-app",
		Namespace:            "preview-host-app",
		Subdomain:            "host-app",
		Domain:               "preview.pixelandprocess.de",
		Image:                "ghcr.io/test/host",
		Tag:                  "latest",
		AppKey:               "base64:hostkey",
		DBName:               "host_db",
		DBUsername:            "host_user",
		DBSize:               "5Gi",
		PersistenceSize:      "1Gi",
		InfisicalProjectSlug: "host-slug",
		InfisicalEnvSlug:     "prod",
		InfisicalSecretsPath: "/host/db",
		InfisicalHost:        "https://custom-vault.mycompany.io",
		InfisicalMailProjectSlug: "mail-host",
		CPU:                  "250m",
		Memory:               "256Mi",
		ChartPath:            "charts/laravel",
		RepoURL:              "https://github.com/test/gitops.git",
	}

	err := Render("laravel-web", data, outDir)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Verify custom InfisicalHost appears in infisical-secrets.yaml
	infisicalContent, err := os.ReadFile(filepath.Join(outDir, "templates", "infisical-secrets.yaml"))
	if err != nil {
		t.Fatalf("reading infisical-secrets.yaml: %v", err)
	}
	infStr := string(infisicalContent)
	if !strings.Contains(infStr, "https://custom-vault.mycompany.io") {
		t.Errorf("infisical-secrets.yaml should contain custom host, got:\n%s", infStr)
	}
}

func TestRender_InfisicalMailProjectSlugConfigurable(t *testing.T) {
	outDir := t.TempDir()
	data := TemplateData{
		Name:                     "mail-app",
		Namespace:                "preview-mail-app",
		Subdomain:                "mail-app",
		Domain:                   "preview.pixelandprocess.de",
		Image:                    "ghcr.io/test/mail",
		Tag:                      "latest",
		AppKey:                   "base64:mailkey",
		DBName:                   "mail_db",
		DBUsername:                "mail_user",
		DBSize:                   "5Gi",
		PersistenceSize:          "1Gi",
		InfisicalProjectSlug:     "mail-db-slug",
		InfisicalEnvSlug:         "prod",
		InfisicalSecretsPath:     "/mail/db",
		InfisicalHost:            "https://vault.pnp.de",
		InfisicalMailProjectSlug: "custom-mail-project-slug",
		CPU:                      "250m",
		Memory:                   "256Mi",
		ChartPath:                "charts/laravel",
		RepoURL:                  "https://github.com/test/gitops.git",
	}

	err := Render("laravel-web", data, outDir)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Verify custom InfisicalMailProjectSlug appears in the mail section of infisical-secrets.yaml
	infisicalContent, err := os.ReadFile(filepath.Join(outDir, "templates", "infisical-secrets.yaml"))
	if err != nil {
		t.Fatalf("reading infisical-secrets.yaml: %v", err)
	}
	infStr := string(infisicalContent)
	if !strings.Contains(infStr, "custom-mail-project-slug") {
		t.Errorf("infisical-secrets.yaml should contain custom mail project slug, got:\n%s", infStr)
	}
	// Also verify it appears in the mail section (second document), not the DB section
	// The mail section is after the "---" separator
	parts := strings.Split(infStr, "---")
	if len(parts) < 2 {
		t.Fatalf("infisical-secrets.yaml should have two YAML documents separated by ---")
	}
	mailSection := parts[1]
	if !strings.Contains(mailSection, "custom-mail-project-slug") {
		t.Errorf("custom-mail-project-slug should be in the mail section, not found there")
	}
	if !strings.Contains(mailSection, "mail-app-mail-credentials") {
		t.Errorf("mail section should reference mail-app-mail-credentials")
	}
}

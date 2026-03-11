package cmd

import (
	"testing"

	"github.com/fwartner/pnp/internal/config"
)

func TestNamespaceFromConfig_Preview(t *testing.T) {
	cfg := config.ProjectConfig{Name: "myapp", Environment: "preview"}
	ns := namespaceFromConfig(cfg)
	if ns != "preview-myapp" {
		t.Errorf("expected preview-myapp, got %s", ns)
	}
}

func TestNamespaceFromConfig_Staging(t *testing.T) {
	cfg := config.ProjectConfig{Name: "myapp", Environment: "staging"}
	ns := namespaceFromConfig(cfg)
	if ns != "preview-myapp" {
		t.Errorf("expected preview-myapp, got %s", ns)
	}
}

func TestNamespaceFromConfig_Production(t *testing.T) {
	cfg := config.ProjectConfig{Name: "myapp", Environment: "production"}
	ns := namespaceFromConfig(cfg)
	if ns != "myapp" {
		t.Errorf("expected myapp, got %s", ns)
	}
}

func TestNamespaceFromConfig_CaseInsensitive(t *testing.T) {
	cases := []struct {
		env  string
		want string
	}{
		{"Preview", "preview-myapp"},
		{"PREVIEW", "preview-myapp"},
		{"Staging", "preview-myapp"},
		{"STAGING", "preview-myapp"},
		{"Production", "myapp"},
		{"PRODUCTION", "myapp"},
	}
	for _, tc := range cases {
		t.Run(tc.env, func(t *testing.T) {
			cfg := config.ProjectConfig{Name: "myapp", Environment: tc.env}
			ns := namespaceFromConfig(cfg)
			if ns != tc.want {
				t.Errorf("env=%s: expected %s, got %s", tc.env, tc.want, ns)
			}
		})
	}
}

func TestBuildTemplateData_LaravelWeb(t *testing.T) {
	projCfg := config.ProjectConfig{
		Name:        "acme",
		Type:        "laravel-web",
		Environment: "preview",
		Domain:      "acme.preview.pixelandprocess.de",
		Image:       "ghcr.io/fwartner/acme",
		Database: config.DatabaseConfig{
			Enabled: true,
			Size:    "5Gi",
			Name:    "acme_db",
		},
		Redis:       config.RedisConfig{Enabled: true},
		Queue:       config.QueueConfig{Enabled: true, Replicas: 1},
		Scheduler:   config.SchedulerConfig{Enabled: true},
		Persistence: config.PersistenceConfig{Enabled: true, Size: "5Gi"},
		Infisical: config.ProjectInfisical{
			ProjectSlug: "customer-apps-f-jq3",
			EnvSlug:     "prod",
			SecretsPath: "/acme/db",
		},
		Resources: config.ResourcesConfig{
			CPU:    "100m",
			Memory: "256Mi",
		},
	}
	globalCfg := config.GlobalConfig{
		GitopsRemote: "https://github.com/test/gitops.git",
		Infisical: config.InfisicalConfig{
			Host:            "https://vault.intern.pixelandprocess.de",
			MailProjectSlug: "cluster-shared-ys-zj",
		},
		Defaults: config.DefaultsConfig{
			Domain: "pixelandprocess.de",
		},
	}

	td := buildTemplateData(projCfg, globalCfg)

	if td.Name != "acme" {
		t.Errorf("Name: expected acme, got %s", td.Name)
	}
	if td.Namespace != "preview-acme" {
		t.Errorf("Namespace: expected preview-acme, got %s", td.Namespace)
	}
	if td.Subdomain != "acme.preview" {
		t.Errorf("Subdomain: expected acme.preview, got %s", td.Subdomain)
	}
	if td.Domain != "pixelandprocess.de" {
		t.Errorf("Domain: expected pixelandprocess.de, got %s", td.Domain)
	}
	if td.Image != "ghcr.io/fwartner/acme" {
		t.Errorf("Image: expected ghcr.io/fwartner/acme, got %s", td.Image)
	}
	if td.Tag != "latest" {
		t.Errorf("Tag: expected latest, got %s", td.Tag)
	}
	if td.DBName != "acme_db" {
		t.Errorf("DBName: expected acme_db, got %s", td.DBName)
	}
	if td.DBSize != "5Gi" {
		t.Errorf("DBSize: expected 5Gi, got %s", td.DBSize)
	}
	if !td.RedisEnabled {
		t.Error("RedisEnabled: expected true")
	}
	if !td.QueueEnabled {
		t.Error("QueueEnabled: expected true for laravel-web")
	}
	if !td.SchedulerEnabled {
		t.Error("SchedulerEnabled: expected true for laravel-web")
	}
	if !td.PersistenceEnabled {
		t.Error("PersistenceEnabled: expected true for laravel-web")
	}
	if td.InfisicalHost != "https://vault.intern.pixelandprocess.de" {
		t.Errorf("InfisicalHost: expected https://vault.intern.pixelandprocess.de, got %s", td.InfisicalHost)
	}
	if td.InfisicalMailProjectSlug != "cluster-shared-ys-zj" {
		t.Errorf("InfisicalMailProjectSlug: expected cluster-shared-ys-zj, got %s", td.InfisicalMailProjectSlug)
	}
	if td.ChartPath != "apps/acme" {
		t.Errorf("ChartPath: expected apps/acme, got %s", td.ChartPath)
	}
	if td.RepoURL != "https://github.com/test/gitops.git" {
		t.Errorf("RepoURL: expected https://github.com/test/gitops.git, got %s", td.RepoURL)
	}
	if td.CPU != "100m" {
		t.Errorf("CPU: expected 100m, got %s", td.CPU)
	}
	if td.Memory != "256Mi" {
		t.Errorf("Memory: expected 256Mi, got %s", td.Memory)
	}
	if !td.InfisicalMailEnabled {
		t.Error("InfisicalMailEnabled: expected true for laravel-web")
	}
}

func TestBuildTemplateData_LaravelAPI(t *testing.T) {
	projCfg := config.ProjectConfig{
		Name:        "api-app",
		Type:        "laravel-api",
		Environment: "production",
		Domain:      "api.pixelandprocess.de",
		Image:       "ghcr.io/fwartner/api-app",
		Redis:       config.RedisConfig{Enabled: true},
		Resources: config.ResourcesConfig{
			CPU:    "200m",
			Memory: "512Mi",
		},
	}
	globalCfg := config.GlobalConfig{
		Defaults: config.DefaultsConfig{Domain: "pixelandprocess.de"},
	}

	td := buildTemplateData(projCfg, globalCfg)

	if td.QueueEnabled {
		t.Error("QueueEnabled: expected false for laravel-api")
	}
	if td.SchedulerEnabled {
		t.Error("SchedulerEnabled: expected false for laravel-api")
	}
	if td.PersistenceEnabled {
		t.Error("PersistenceEnabled: expected false for laravel-api")
	}
	if !td.InfisicalMailEnabled {
		t.Error("InfisicalMailEnabled: expected true for laravel-api")
	}
}

func TestBuildTemplateData_NextjsStatic(t *testing.T) {
	projCfg := config.ProjectConfig{
		Name:        "static-site",
		Type:        "nextjs-static",
		Environment: "production",
		Domain:      "static.pixelandprocess.de",
		Image:       "ghcr.io/fwartner/static-site",
		Redis:       config.RedisConfig{Enabled: false},
		Resources: config.ResourcesConfig{
			CPU:    "50m",
			Memory: "128Mi",
		},
	}
	globalCfg := config.GlobalConfig{
		Defaults: config.DefaultsConfig{Domain: "pixelandprocess.de"},
	}

	td := buildTemplateData(projCfg, globalCfg)

	if td.RedisEnabled {
		t.Error("RedisEnabled: expected false for nextjs-static")
	}
	if td.QueueEnabled {
		t.Error("QueueEnabled: expected false for nextjs-static")
	}
	if td.SchedulerEnabled {
		t.Error("SchedulerEnabled: expected false for nextjs-static")
	}
	if td.PersistenceEnabled {
		t.Error("PersistenceEnabled: expected false for nextjs-static")
	}
	if td.DBName != "" {
		t.Errorf("DBName: expected empty, got %s", td.DBName)
	}
	if td.DBSize != "" {
		t.Errorf("DBSize: expected empty, got %s", td.DBSize)
	}
	if td.InfisicalMailEnabled {
		t.Error("InfisicalMailEnabled: expected false for nextjs-static")
	}
}

func TestBuildTemplateData_Strapi(t *testing.T) {
	projCfg := config.ProjectConfig{
		Name:        "cms",
		Type:        "strapi",
		Environment: "preview",
		Domain:      "cms.preview.pixelandprocess.de",
		Image:       "ghcr.io/fwartner/cms",
		Database: config.DatabaseConfig{
			Enabled: true,
			Size:    "10Gi",
			Name:    "cms_db",
		},
		Redis:       config.RedisConfig{Enabled: false},
		Persistence: config.PersistenceConfig{Enabled: true, Size: "5Gi"},
		Resources: config.ResourcesConfig{
			CPU:    "150m",
			Memory: "384Mi",
		},
	}
	globalCfg := config.GlobalConfig{
		Defaults: config.DefaultsConfig{Domain: "pixelandprocess.de"},
	}

	td := buildTemplateData(projCfg, globalCfg)

	if !td.PersistenceEnabled {
		t.Error("PersistenceEnabled: expected true for strapi")
	}
	if td.InfisicalMailEnabled {
		t.Error("InfisicalMailEnabled: expected false for strapi")
	}
	if td.QueueEnabled {
		t.Error("QueueEnabled: expected false for strapi")
	}
	if td.SchedulerEnabled {
		t.Error("SchedulerEnabled: expected false for strapi")
	}
}

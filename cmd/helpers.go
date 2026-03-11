package cmd

import (
	"strings"

	"github.com/fwartner/pnp/internal/config"
	"github.com/fwartner/pnp/internal/templates"
	"github.com/fwartner/pnp/internal/wizard"
)

// buildTemplateData constructs a TemplateData from the project and global
// configuration. This is shared between deploy and update commands.
func buildTemplateData(projCfg config.ProjectConfig, globalCfg config.GlobalConfig) templates.TemplateData {
	namespace := projCfg.Name
	env := strings.ToLower(projCfg.Environment)
	if env == "preview" || env == "staging" {
		namespace = "preview-" + projCfg.Name
	}

	isLaravelWeb := projCfg.Type == "laravel-web"

	return templates.TemplateData{
		Name:      projCfg.Name,
		Namespace: namespace,
		Subdomain: wizard.Subdomain(projCfg.Domain, globalCfg.Defaults.Domain),
		Domain:    globalCfg.Defaults.Domain,
		Image:     projCfg.Image,
		Tag:       "latest",
		DBName:    projCfg.Database.Name,
		DBUsername: projCfg.Name,
		DBSize:    projCfg.Database.Size,

		RedisEnabled:     projCfg.Redis.Enabled,
		QueueEnabled:     isLaravelWeb,
		SchedulerEnabled: isLaravelWeb,

		PersistenceEnabled: isLaravelWeb || projCfg.Type == "strapi",
		PersistenceSize:    "5Gi",

		InfisicalProjectSlug:     projCfg.Infisical.ProjectSlug,
		InfisicalEnvSlug:         projCfg.Infisical.EnvSlug,
		InfisicalSecretsPath:     projCfg.Infisical.SecretsPath,
		InfisicalMailEnabled:     isLaravelWeb || projCfg.Type == "laravel-api",
		InfisicalHost:            globalCfg.Infisical.Host,
		InfisicalMailProjectSlug: globalCfg.Infisical.MailProjectSlug,

		CPU:    projCfg.Resources.CPU,
		Memory: projCfg.Resources.Memory,

		ChartPath: "apps/" + projCfg.Name,
		RepoURL:   globalCfg.GitopsRemote,
	}
}

// namespaceFromConfig returns the Kubernetes namespace for a project.
func namespaceFromConfig(projCfg config.ProjectConfig) string {
	namespace := projCfg.Name
	env := strings.ToLower(projCfg.Environment)
	if env == "preview" || env == "staging" {
		namespace = "preview-" + projCfg.Name
	}
	return namespace
}

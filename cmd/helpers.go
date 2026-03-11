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
	namespace := namespaceFromConfig(projCfg)
	isLaravel := projCfg.Type == "laravel-web" || projCfg.Type == "laravel-api"

	queueReplicas := projCfg.Queue.Replicas
	if queueReplicas < 1 {
		queueReplicas = 1
	}

	reverbPort := projCfg.Reverb.Port
	if reverbPort == 0 {
		reverbPort = 8080
	}

	octaneServer := projCfg.Octane.Server
	if octaneServer == "" {
		octaneServer = "frankenphp"
	}

	persistenceSize := projCfg.Persistence.Size
	if persistenceSize == "" {
		persistenceSize = "5Gi"
	}

	scopeDomain := globalCfg.EffectiveDomain(projCfg.Scope)

	return templates.TemplateData{
		Name:      projCfg.Name,
		Namespace: namespace,
		Subdomain: wizard.Subdomain(projCfg.Domain, scopeDomain),
		Domain:    scopeDomain,
		Image:     projCfg.Image,
		Tag:       "latest",
		DBName:    projCfg.Database.Name,
		DBUsername: projCfg.Name,
		DBSize:    projCfg.Database.Size,

		RedisEnabled:     projCfg.Redis.Enabled,
		QueueEnabled:     projCfg.Queue.Enabled,
		QueueReplicas:    queueReplicas,
		SchedulerEnabled: projCfg.Scheduler.Enabled,

		HorizonEnabled: projCfg.Horizon.Enabled,
		ReverbEnabled:  projCfg.Reverb.Enabled,
		ReverbPort:     reverbPort,
		OctaneEnabled:  projCfg.Octane.Enabled,
		OctaneServer:   octaneServer,

		PersistenceEnabled: projCfg.Persistence.Enabled,
		PersistenceSize:    persistenceSize,

		InfisicalProjectSlug:     projCfg.Infisical.ProjectSlug,
		InfisicalEnvSlug:         projCfg.Infisical.EnvSlug,
		InfisicalSecretsPath:     projCfg.Infisical.SecretsPath,
		InfisicalMailEnabled:     isLaravel,
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

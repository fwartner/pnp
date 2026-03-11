package wizard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/fwartner/pnp/internal/config"
	"github.com/fwartner/pnp/internal/detect"
)

var projectTypes = []string{"laravel-web", "laravel-api", "nextjs-fullstack", "nextjs-static", "strapi"}
var environments = []string{"preview", "staging", "production"}
var scopes = []string{"customer", "private", "agency"}
var octaneServers = []string{"frankenphp", "swoole", "roadrunner"}

// Run is an alias for RunAdvanced for backwards compatibility.
func Run(detected detect.DetectionResult, inferredImage string, projectName string, globalCfg config.GlobalConfig) (config.ProjectConfig, error) {
	return RunAdvanced(detected, inferredImage, projectName, globalCfg)
}

// RunBasic executes a simplified 4-question wizard with smart defaults.
func RunBasic(detected detect.DetectionResult, inferredImage string, projectName string, globalCfg config.GlobalConfig) (config.ProjectConfig, error) {
	cfg := newDefaultConfig(detected, inferredImage, projectName)

	// Only 4 questions: name, type, scope, environment
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
				Title("Project scope").
				Description("Determines default org, domain, visibility, and Infisical project").
				Options(
					huh.NewOption("Customer project", "customer"),
					huh.NewOption("Private / internal project", "private"),
					huh.NewOption("Agency project (Pixel & Process)", "agency"),
				).
				Value(&cfg.Scope),
			huh.NewSelect[string]().
				Title("Environment").
				Options(huh.NewOptions(environments...)...).
				Value(&cfg.Environment),
		),
	).Run()
	if err != nil {
		return cfg, err
	}

	// Apply all smart defaults based on the 4 answers
	ApplyDefaults(&cfg, globalCfg)

	return cfg, nil
}

// RunAdvanced executes the full 7-step interactive wizard with all options.
func RunAdvanced(detected detect.DetectionResult, inferredImage string, projectName string, globalCfg config.GlobalConfig) (config.ProjectConfig, error) {
	cfg := newDefaultConfig(detected, inferredImage, projectName)

	// Step 1: Scope and basic info
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Project scope").
				Description("Determines default org, domain, visibility, and Infisical project").
				Options(
					huh.NewOption("Customer project", "customer"),
					huh.NewOption("Private / internal project", "private"),
					huh.NewOption("Agency project (Pixel & Process)", "agency"),
				).
				Value(&cfg.Scope),
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

	// Apply scope-based defaults.
	scopeDomain := globalCfg.EffectiveDomain(cfg.Scope)
	scopeRegistry := globalCfg.EffectiveImageRegistry(cfg.Scope)
	scopeOrg := globalCfg.EffectiveGithubOrg(cfg.Scope)
	scopeInfisicalSlug := globalCfg.EffectiveInfisicalProjectSlug(cfg.Scope)

	cfg.Domain = defaultDomain(cfg.Name, cfg.Environment, scopeDomain)
	cfg.Infisical.ProjectSlug = scopeInfisicalSlug
	cfg.Infisical.EnvSlug = "prod"

	// Build image from scope-specific org/registry if not already inferred.
	if cfg.Image == "" && scopeOrg != "" {
		cfg.Image = scopeRegistry + "/" + scopeOrg + "/" + cfg.Name
	}

	// Step 2: Domain and image
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

	// Step 3: Database & Redis (skip for nextjs-static)
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

		if isLaravel(cfg.Type) {
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
		cfg.Queue.Enabled = false
		cfg.Scheduler.Enabled = false
		cfg.Persistence.Enabled = false
	}

	// Step 4: Queue, Scheduler, Persistence (Laravel only)
	if isLaravel(cfg.Type) {
		err = huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Enable queue worker?").
					Value(&cfg.Queue.Enabled),
				huh.NewConfirm().
					Title("Enable task scheduler?").
					Value(&cfg.Scheduler.Enabled),
				huh.NewConfirm().
					Title("Enable persistent storage?").
					Value(&cfg.Persistence.Enabled),
			),
		).Run()
		if err != nil {
			return cfg, err
		}

		if cfg.Persistence.Enabled {
			err = huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Persistent storage size").
						Value(&cfg.Persistence.Size),
				),
			).Run()
			if err != nil {
				return cfg, err
			}
		}
	}

	// Step 5: Laravel features — Horizon, Reverb, Octane
	if isLaravel(cfg.Type) {
		cwd := "."
		features := detect.DetectLaravelFeatures(cwd)
		if features.Horizon {
			cfg.Horizon.Enabled = true
		}
		if features.Reverb {
			cfg.Reverb.Enabled = true
		}
		if features.Octane {
			cfg.Octane.Enabled = true
		}

		featureDesc := ""
		if features.Horizon || features.Reverb || features.Octane {
			var detected []string
			if features.Horizon {
				detected = append(detected, "Horizon")
			}
			if features.Reverb {
				detected = append(detected, "Reverb")
			}
			if features.Octane {
				detected = append(detected, "Octane")
			}
			featureDesc = fmt.Sprintf(" (Detected: %s)", strings.Join(detected, ", "))
		}

		err = huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Enable Laravel Horizon?" + featureDesc).
					Value(&cfg.Horizon.Enabled),
				huh.NewConfirm().
					Title("Enable Laravel Reverb (WebSockets)?").
					Value(&cfg.Reverb.Enabled),
				huh.NewConfirm().
					Title("Enable Laravel Octane?").
					Value(&cfg.Octane.Enabled),
			),
		).Run()
		if err != nil {
			return cfg, err
		}

		if cfg.Horizon.Enabled {
			cfg.Queue.Enabled = false
		}

		if cfg.Octane.Enabled {
			err = huh.NewForm(
				huh.NewGroup(
					huh.NewSelect[string]().
						Title("Octane server").
						Options(huh.NewOptions(octaneServers...)...).
						Value(&cfg.Octane.Server),
				),
			).Run()
			if err != nil {
				return cfg, err
			}
		}
	}

	// Step 6: Infisical (if DB enabled)
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

	// Step 7: Resources
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

// ApplyDefaults fills in smart defaults based on the project type, scope, and environment.
func ApplyDefaults(cfg *config.ProjectConfig, globalCfg config.GlobalConfig) {
	scopeDomain := globalCfg.EffectiveDomain(cfg.Scope)
	scopeRegistry := globalCfg.EffectiveImageRegistry(cfg.Scope)
	scopeOrg := globalCfg.EffectiveGithubOrg(cfg.Scope)
	scopeInfisicalSlug := globalCfg.EffectiveInfisicalProjectSlug(cfg.Scope)

	// Domain
	cfg.Domain = defaultDomain(cfg.Name, cfg.Environment, scopeDomain)

	// Image
	if cfg.Image == "" && scopeOrg != "" {
		cfg.Image = scopeRegistry + "/" + scopeOrg + "/" + cfg.Name
	}

	// Infisical
	cfg.Infisical.ProjectSlug = scopeInfisicalSlug
	cfg.Infisical.EnvSlug = "prod"
	cfg.Infisical.SecretsPath = "/" + cfg.Name + "/db"

	// Resources — slightly higher than old defaults for production readiness
	cfg.Resources.CPU = "200m"
	cfg.Resources.Memory = "512Mi"

	// Type-specific defaults
	switch cfg.Type {
	case "nextjs-static":
		cfg.Database.Enabled = false
		cfg.Redis.Enabled = false
		cfg.Queue.Enabled = false
		cfg.Scheduler.Enabled = false
		cfg.Persistence.Enabled = false
	case "nextjs-fullstack":
		cfg.Database.Enabled = true
		cfg.Redis.Enabled = false
		cfg.Queue.Enabled = false
		cfg.Scheduler.Enabled = false
		cfg.Persistence.Enabled = false
	case "strapi":
		cfg.Database.Enabled = true
		cfg.Redis.Enabled = false
		cfg.Queue.Enabled = false
		cfg.Scheduler.Enabled = false
		cfg.Persistence.Enabled = true
	case "laravel-web":
		cfg.Database.Enabled = true
		cfg.Redis.Enabled = true
		cfg.Queue.Enabled = true
		cfg.Scheduler.Enabled = true
		cfg.Persistence.Enabled = true
	case "laravel-api":
		cfg.Database.Enabled = true
		cfg.Redis.Enabled = true
		cfg.Queue.Enabled = true
		cfg.Scheduler.Enabled = true
		cfg.Persistence.Enabled = false
	}
}

// newDefaultConfig creates a ProjectConfig with sensible initial values.
func newDefaultConfig(detected detect.DetectionResult, inferredImage string, projectName string) config.ProjectConfig {
	return config.ProjectConfig{
		Name:        projectName,
		Scope:       "customer",
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
		Queue: config.QueueConfig{
			Enabled:  true,
			Replicas: 1,
		},
		Scheduler: config.SchedulerConfig{
			Enabled: true,
		},
		Horizon: config.HorizonConfig{
			Enabled: false,
		},
		Reverb: config.ReverbConfig{
			Enabled: false,
			Port:    8080,
		},
		Octane: config.OctaneConfig{
			Enabled: false,
			Server:  "frankenphp",
		},
		Persistence: config.PersistenceConfig{
			Enabled: true,
			Size:    "5Gi",
		},
		Resources: config.ResourcesConfig{
			CPU:    "100m",
			Memory: "256Mi",
		},
	}
}

func isLaravel(projectType string) bool {
	return projectType == "laravel-web" || projectType == "laravel-api"
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
	if baseDomain == "" {
		return domain
	}
	suffix := "." + baseDomain
	if len(domain) > len(suffix) && domain[len(domain)-len(suffix):] == suffix {
		return domain[:len(domain)-len(suffix)]
	}
	return domain
}

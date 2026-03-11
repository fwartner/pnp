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

	// Step 1: Basic info
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

	// Step 2: Domain and image
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

	// Step 4: Infisical (if DB enabled)
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

	// Step 5: Resources
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
	if baseDomain == "" {
		return domain
	}
	suffix := "." + baseDomain
	if len(domain) > len(suffix) && domain[len(domain)-len(suffix):] == suffix {
		return domain[:len(domain)-len(suffix)]
	}
	return domain
}

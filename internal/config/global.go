package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type GlobalConfig struct {
	GitopsRepo   string                    `yaml:"gitopsRepo"`
	GitopsRemote string                    `yaml:"gitopsRemote"`
	Infisical    InfisicalConfig           `yaml:"infisical"`
	Defaults     DefaultsConfig            `yaml:"defaults"`
	Profiles     map[string]ProfileConfig  `yaml:"profiles,omitempty"`
}

type InfisicalConfig struct {
	Host             string `yaml:"host"`
	Token            string `yaml:"token"`
	MailProjectSlug  string `yaml:"mailProjectSlug"`
}

type DefaultsConfig struct {
	Domain        string `yaml:"domain"`
	ImageRegistry string `yaml:"imageRegistry"`
	GithubOrg     string `yaml:"githubOrg"`
}

// ProfileConfig holds per-scope overrides (customer, private, agency).
type ProfileConfig struct {
	GithubOrg            string `yaml:"githubOrg,omitempty"`
	Domain               string `yaml:"domain,omitempty"`
	ImageRegistry        string `yaml:"imageRegistry,omitempty"`
	RepoVisibility       string `yaml:"repoVisibility,omitempty"` // public, private
	InfisicalProjectSlug string `yaml:"infisicalProjectSlug,omitempty"`
}

// ProfileFor returns the profile for the given scope, falling back to empty defaults.
func (g GlobalConfig) ProfileFor(scope string) ProfileConfig {
	if g.Profiles != nil {
		if p, ok := g.Profiles[scope]; ok {
			return p
		}
	}
	return ProfileConfig{}
}

// EffectiveGithubOrg returns the GitHub org for a scope, falling back to the global default.
func (g GlobalConfig) EffectiveGithubOrg(scope string) string {
	if p := g.ProfileFor(scope); p.GithubOrg != "" {
		return p.GithubOrg
	}
	return g.Defaults.GithubOrg
}

// EffectiveDomain returns the domain for a scope, falling back to the global default.
func (g GlobalConfig) EffectiveDomain(scope string) string {
	if p := g.ProfileFor(scope); p.Domain != "" {
		return p.Domain
	}
	return g.Defaults.Domain
}

// EffectiveImageRegistry returns the image registry for a scope, falling back to the global default.
func (g GlobalConfig) EffectiveImageRegistry(scope string) string {
	if p := g.ProfileFor(scope); p.ImageRegistry != "" {
		return p.ImageRegistry
	}
	return g.Defaults.ImageRegistry
}

// EffectiveRepoVisibility returns the repo visibility for a scope, defaulting to "private".
func (g GlobalConfig) EffectiveRepoVisibility(scope string) string {
	if p := g.ProfileFor(scope); p.RepoVisibility != "" {
		return p.RepoVisibility
	}
	return "private"
}

// EffectiveInfisicalProjectSlug returns the Infisical project slug for a scope.
func (g GlobalConfig) EffectiveInfisicalProjectSlug(scope string) string {
	if p := g.ProfileFor(scope); p.InfisicalProjectSlug != "" {
		return p.InfisicalProjectSlug
	}
	return "customer-apps-f-jq3"
}

func defaultGlobalConfig() GlobalConfig {
	return GlobalConfig{
		Infisical: InfisicalConfig{
			Host:            "https://vault.intern.pixelandprocess.de",
			MailProjectSlug: "cluster-shared-ys-zj",
		},
		Defaults: DefaultsConfig{
			Domain:        "pixelandprocess.de",
			ImageRegistry: "ghcr.io",
		},
	}
}

func GlobalConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}
	return filepath.Join(home, ".pnp.yaml"), nil
}

func LoadGlobalConfig() (GlobalConfig, error) {
	path, err := GlobalConfigPath()
	if err != nil {
		return defaultGlobalConfig(), err
	}
	return LoadGlobalConfigFrom(path)
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
	path, err := GlobalConfigPath()
	if err != nil {
		return err
	}
	return SaveGlobalConfigTo(cfg, path)
}

func SaveGlobalConfigTo(cfg GlobalConfig, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

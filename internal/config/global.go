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

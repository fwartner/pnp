package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type ProjectConfig struct {
	Name        string           `yaml:"name"`
	Type        string           `yaml:"type"`
	Environment string           `yaml:"environment"`
	Domain      string           `yaml:"domain"`
	Image       string           `yaml:"image"`
	Database    DatabaseConfig   `yaml:"database"`
	Redis       RedisConfig      `yaml:"redis"`
	Infisical   ProjectInfisical `yaml:"infisical"`
	Resources   ResourcesConfig  `yaml:"resources"`
	CI          CIConfig         `yaml:"ci"`
}

type DatabaseConfig struct {
	Enabled bool   `yaml:"enabled"`
	Size    string `yaml:"size"`
	Name    string `yaml:"name"`
}

type RedisConfig struct {
	Enabled bool `yaml:"enabled"`
}

type ProjectInfisical struct {
	ProjectSlug string `yaml:"projectSlug"`
	EnvSlug     string `yaml:"envSlug"`
	SecretsPath string `yaml:"secretsPath"`
}

type ResourcesConfig struct {
	CPU    string `yaml:"cpu"`
	Memory string `yaml:"memory"`
}

type CIConfig struct {
	Enabled bool `yaml:"enabled"`
}

func LoadProjectConfig() (ProjectConfig, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return ProjectConfig{}, err
	}
	return LoadProjectConfigFrom(filepath.Join(cwd, ".cluster.yaml"))
}

func LoadProjectConfigFrom(path string) (ProjectConfig, error) {
	var cfg ProjectConfig
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func SaveProjectConfig(cfg ProjectConfig) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	return SaveProjectConfigTo(cfg, filepath.Join(cwd, ".cluster.yaml"))
}

func SaveProjectConfigTo(cfg ProjectConfig, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

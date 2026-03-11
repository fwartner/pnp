package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type ProjectConfig struct {
	Name        string            `yaml:"name"`
	Scope       string            `yaml:"scope"` // customer, private, agency
	Type        string            `yaml:"type"`
	Environment string            `yaml:"environment"`
	Domain      string            `yaml:"domain"`
	Image       string            `yaml:"image"`
	Database    DatabaseConfig    `yaml:"database"`
	Redis       RedisConfig       `yaml:"redis"`
	Queue       QueueConfig       `yaml:"queue"`
	Scheduler   SchedulerConfig   `yaml:"scheduler"`
	Horizon     HorizonConfig     `yaml:"horizon"`
	Reverb      ReverbConfig      `yaml:"reverb"`
	Octane      OctaneConfig      `yaml:"octane"`
	Persistence PersistenceConfig `yaml:"persistence"`
	Infisical   ProjectInfisical  `yaml:"infisical"`
	Resources   ResourcesConfig   `yaml:"resources"`
	CI          CIConfig          `yaml:"ci"`
	Secrets     SecretsConfig     `yaml:"secrets,omitempty"`
}

type SecretsConfig struct {
	AppKey     string `yaml:"appKey,omitempty"`
	DBPassword string `yaml:"dbPassword,omitempty"`
}

type DatabaseConfig struct {
	Enabled bool   `yaml:"enabled"`
	Size    string `yaml:"size"`
	Name    string `yaml:"name"`
}

type RedisConfig struct {
	Enabled bool `yaml:"enabled"`
}

type QueueConfig struct {
	Enabled  bool `yaml:"enabled"`
	Replicas int  `yaml:"replicas"`
}

type SchedulerConfig struct {
	Enabled bool `yaml:"enabled"`
}

type HorizonConfig struct {
	Enabled bool `yaml:"enabled"`
}

type ReverbConfig struct {
	Enabled bool `yaml:"enabled"`
	Port    int  `yaml:"port"`
}

type OctaneConfig struct {
	Enabled bool   `yaml:"enabled"`
	Server  string `yaml:"server"` // frankenphp, swoole, roadrunner
}

type PersistenceConfig struct {
	Enabled bool   `yaml:"enabled"`
	Size    string `yaml:"size"`
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

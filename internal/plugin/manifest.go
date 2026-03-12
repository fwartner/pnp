package plugin

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Manifest represents a plugin.yaml file.
type Manifest struct {
	Name     string   `yaml:"name"`
	Version  string   `yaml:"version"`
	Provides Provides `yaml:"provides"`
}

// Provides describes what the plugin provides.
type Provides struct {
	Types    []TypeEntry    `yaml:"types,omitempty"`
	Commands []CommandEntry `yaml:"commands,omitempty"`
	Hooks    []HookEntry    `yaml:"hooks,omitempty"`
}

// TypeEntry describes a project type provided by a plugin.
type TypeEntry struct {
	Name   string `yaml:"name"`
	Binary string `yaml:"binary"`
}

// CommandEntry describes a CLI command provided by a plugin.
type CommandEntry struct {
	Name        string `yaml:"name"`
	Binary      string `yaml:"binary"`
	Description string `yaml:"description"`
}

// HookEntry describes a hook provided by a plugin.
type HookEntry struct {
	Event  string `yaml:"event"`
	Binary string `yaml:"binary"`
}

// LoadManifest parses a plugin.yaml file.
func LoadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}

	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

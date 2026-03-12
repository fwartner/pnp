package types

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// FileExists checks if a file exists at the given path.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ReadPackageDeps reads package.json and returns a merged map of dependencies and devDependencies.
func ReadPackageDeps(path string) map[string]string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}

	merged := make(map[string]string)
	for k, v := range pkg.Dependencies {
		merged[k] = v
	}
	for k, v := range pkg.DevDependencies {
		merged[k] = v
	}
	return merged
}

// HasDBDeps checks if the dependency map contains any known database packages.
func HasDBDeps(deps map[string]string) bool {
	dbPackages := []string{
		"prisma", "@prisma/client", "pg", "postgres",
		"typeorm", "drizzle-orm", "knex", "sequelize",
	}
	for _, pkg := range dbPackages {
		if _, ok := deps[pkg]; ok {
			return true
		}
	}
	return false
}

// IsLaravelWebProject returns true if the Laravel project has Jobs dir or scheduler references.
func IsLaravelWebProject(dir string) bool {
	jobsDir := filepath.Join(dir, "app", "Jobs")
	if info, err := os.Stat(jobsDir); err == nil && info.IsDir() {
		return true
	}

	consolePath := filepath.Join(dir, "routes", "console.php")
	if data, err := os.ReadFile(consolePath); err == nil {
		content := string(data)
		if strings.Contains(content, "Schedule") || strings.Contains(content, "schedule") {
			return true
		}
	}

	return false
}

// LaravelFeatures holds detected Laravel package features.
type LaravelFeatures struct {
	Horizon bool
	Reverb  bool
	Octane  bool
}

// DetectLaravelFeatures checks composer.json for Horizon, Reverb, and Octane packages.
func DetectLaravelFeatures(dir string) LaravelFeatures {
	composerPath := filepath.Join(dir, "composer.json")
	data, err := os.ReadFile(composerPath)
	if err != nil {
		return LaravelFeatures{}
	}

	var pkg struct {
		Require    map[string]string `json:"require"`
		RequireDev map[string]string `json:"require-dev"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return LaravelFeatures{}
	}

	has := func(name string) bool {
		_, ok := pkg.Require[name]
		if !ok {
			_, ok = pkg.RequireDev[name]
		}
		return ok
	}

	return LaravelFeatures{
		Horizon: has("laravel/horizon"),
		Reverb:  has("laravel/reverb"),
		Octane:  has("laravel/octane"),
	}
}

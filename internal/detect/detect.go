package detect

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fwartner/pnp/internal/types"
)

// DetectionResult holds the detected project type and confidence level.
type DetectionResult struct {
	Type       string // e.g. "laravel-web", "nextjs-static", "unknown"
	Confidence string // "high", "medium", "low"
}

// LaravelFeatures holds detected Laravel package features.
type LaravelFeatures struct {
	Horizon bool
	Reverb  bool
	Octane  bool
}

// DetectLaravelFeatures checks composer.json for Horizon, Reverb, and Octane packages.
func DetectLaravelFeatures(dir string) LaravelFeatures {
	f := types.DetectLaravelFeatures(dir)
	return LaravelFeatures{
		Horizon: f.Horizon,
		Reverb:  f.Reverb,
		Octane:  f.Octane,
	}
}

// DetectProjectType inspects the directory and returns the detected project type.
// Delegates to the type registry for detection.
func DetectProjectType(dir string) DetectionResult {
	pt, confidence := types.Detect(dir)
	if pt == nil || confidence == "" {
		return DetectionResult{Type: "unknown", Confidence: "low"}
	}
	return DetectionResult{Type: pt.Name(), Confidence: confidence}
}

// InferProjectName returns the base name of the directory as the project name.
func InferProjectName(dir string) string {
	return filepath.Base(dir)
}

// InferImageFromGitRemote reads the git remote origin URL and constructs a container
// image reference in the form registry/org/repo.
func InferImageFromGitRemote(dir, registry string) string {
	remoteURL := getGitRemoteURL(dir)
	if remoteURL == "" {
		return ""
	}

	orgRepo := parseGitRemoteURL(remoteURL)
	if orgRepo == "" {
		return ""
	}

	return registry + "/" + orgRepo
}

// getGitRemoteURL tries `git remote get-url origin`, falling back to parsing .git/config.
func getGitRemoteURL(dir string) string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}

	configPath := filepath.Join(dir, ".git", "config")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")
	inOrigin := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == `[remote "origin"]` {
			inOrigin = true
			continue
		}
		if inOrigin {
			if strings.HasPrefix(trimmed, "[") {
				break
			}
			if strings.HasPrefix(trimmed, "url = ") {
				return strings.TrimPrefix(trimmed, "url = ")
			}
		}
	}
	return ""
}

// parseGitRemoteURL extracts org/repo from SSH or HTTPS git URLs.
var (
	sshPattern   = regexp.MustCompile(`git@[^:]+:([^/]+)/([^/]+?)(?:\.git)?$`)
	httpsPattern = regexp.MustCompile(`https?://[^/]+/([^/]+)/([^/]+?)(?:\.git)?$`)
)

func parseGitRemoteURL(url string) string {
	if m := sshPattern.FindStringSubmatch(url); m != nil {
		return m[1] + "/" + m[2]
	}
	if m := httpsPattern.FindStringSubmatch(url); m != nil {
		return m[1] + "/" + m[2]
	}
	return ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// readPackageDeps reads package.json and returns a merged map of dependencies and devDependencies.
func readPackageDeps(path string) map[string]string {
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

// hasDBDeps checks if the dependency map contains any known database packages.
func hasDBDeps(deps map[string]string) bool {
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

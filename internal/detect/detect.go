package detect

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// DetectionResult holds the detected project type and confidence level.
type DetectionResult struct {
	Type       string // e.g. "laravel-web", "nextjs-static", "unknown"
	Confidence string // "high", "medium", "low"
}

// DetectProjectType inspects the directory and returns the detected project type.
func DetectProjectType(dir string) DetectionResult {
	// Check Laravel first (composer.json + artisan)
	if fileExists(filepath.Join(dir, "composer.json")) && fileExists(filepath.Join(dir, "artisan")) {
		if isLaravelWeb(dir) {
			return DetectionResult{Type: "laravel-web", Confidence: "high"}
		}
		return DetectionResult{Type: "laravel-api", Confidence: "high"}
	}

	// Check Node.js projects (package.json)
	pkgPath := filepath.Join(dir, "package.json")
	if fileExists(pkgPath) {
		deps := readPackageDeps(pkgPath)

		// Strapi
		if _, ok := deps["@strapi/strapi"]; ok {
			return DetectionResult{Type: "strapi", Confidence: "high"}
		}

		// Next.js
		if _, ok := deps["next"]; ok {
			if hasDBDeps(deps) {
				return DetectionResult{Type: "nextjs-fullstack", Confidence: "high"}
			}
			return DetectionResult{Type: "nextjs-static", Confidence: "high"}
		}
	}

	return DetectionResult{Type: "unknown", Confidence: "low"}
}

// isLaravelWeb returns true if the Laravel project has Jobs dir or scheduler references.
func isLaravelWeb(dir string) bool {
	// Check for Jobs directory
	jobsDir := filepath.Join(dir, "app", "Jobs")
	if info, err := os.Stat(jobsDir); err == nil && info.IsDir() {
		return true
	}

	// Check routes/console.php for Schedule references (Laravel 11+ style)
	consolePath := filepath.Join(dir, "routes", "console.php")
	if data, err := os.ReadFile(consolePath); err == nil {
		content := string(data)
		if strings.Contains(content, "Schedule") || strings.Contains(content, "schedule") {
			return true
		}
	}

	return false
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
		"prisma",
		"@prisma/client",
		"pg",
		"postgres",
		"typeorm",
		"drizzle-orm",
		"knex",
		"sequelize",
	}
	for _, pkg := range dbPackages {
		if _, ok := deps[pkg]; ok {
			return true
		}
	}
	return false
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
	// Try git command first
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}

	// Fallback: read .git/config
	configPath := filepath.Join(dir, ".git", "config")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}

	// Simple parsing: find url = ... under [remote "origin"]
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
				break // new section
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

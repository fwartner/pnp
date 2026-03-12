package ci

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fwartner/pnp/internal/config"
	"github.com/fwartner/pnp/internal/types"
)

// GenerateDockerfile creates a Dockerfile in the project directory based on the project type.
func GenerateDockerfile(projectType string, octaneCfg config.OctaneConfig, projectDir string) error {
	pt := types.Get(projectType)
	if pt == nil {
		return fmt.Errorf("unsupported project type for Dockerfile: %s", projectType)
	}

	// Build a minimal ProjectConfig with the Octane settings for Dockerfile selection
	cfg := config.ProjectConfig{
		Type:   projectType,
		Octane: octaneCfg,
	}

	content := pt.Dockerfile(cfg)
	ignoreContent := pt.Dockerignore()

	// Write Dockerfile
	outPath := filepath.Join(projectDir, "Dockerfile")
	if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	// Write .dockerignore if it doesn't exist
	ignorePath := filepath.Join(projectDir, ".dockerignore")
	if _, err := os.Stat(ignorePath); os.IsNotExist(err) {
		if err := os.WriteFile(ignorePath, []byte(ignoreContent), 0o644); err != nil {
			return fmt.Errorf("failed to write .dockerignore: %w", err)
		}
	}

	return nil
}

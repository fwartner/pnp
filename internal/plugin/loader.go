package plugin

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fwartner/pnp/internal/types"
)

// LoadAll scans ~/.pnp/plugins/ and registers all discovered plugins.
// Silently skips plugins that fail to load.
func LoadAll() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	pluginDir := filepath.Join(home, ".pnp", "plugins")
	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		return // no plugins directory
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		manifestPath := filepath.Join(pluginDir, entry.Name(), "plugin.yaml")
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			continue
		}

		manifest, err := LoadManifest(manifestPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load plugin %s: %v\n", entry.Name(), err)
			continue
		}

		pluginPath := filepath.Join(pluginDir, entry.Name())
		loadPlugin(manifest, pluginPath)
	}
}

func loadPlugin(manifest Manifest, pluginDir string) {
	// Register project types
	for _, t := range manifest.Provides.Types {
		binary := resolveBinary(pluginDir, t.Binary)
		ext := &ExternalProjectType{
			TypeName: t.Name,
			Binary:   binary,
		}
		// Don't panic on duplicate — skip silently
		if types.Get(t.Name) == nil {
			types.Register(ext)
		}
	}

	// Register commands
	for _, c := range manifest.Provides.Commands {
		entry := c
		entry.Binary = resolveBinary(pluginDir, c.Binary)
		registerCommand(manifest.Name, entry)
	}

	// Register hooks
	for _, h := range manifest.Provides.Hooks {
		binary := resolveBinary(pluginDir, h.Binary)
		RegisterHook(manifest.Name, h.Event, binary)
	}
}

func resolveBinary(pluginDir, binary string) string {
	if filepath.IsAbs(binary) {
		return binary
	}
	return filepath.Join(pluginDir, binary)
}

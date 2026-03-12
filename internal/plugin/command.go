package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/fwartner/pnp/internal/config"
	"github.com/spf13/cobra"
)

var registeredCommands []*cobra.Command

// Commands returns all plugin-provided cobra commands.
func Commands() []*cobra.Command {
	return registeredCommands
}

// registerCommand creates a cobra.Command that wraps a plugin binary.
func registerCommand(pluginName string, entry CommandEntry) {
	binary := entry.Binary

	cmd := &cobra.Command{
		Use:   entry.Name,
		Short: entry.Description,
		Long:  fmt.Sprintf("Provided by plugin: %s", pluginName),
		RunE: func(cmd *cobra.Command, args []string) error {
			execCmd := exec.Command(binary, args...)
			execCmd.Stdout = os.Stdout
			execCmd.Stderr = os.Stderr
			execCmd.Stdin = os.Stdin

			// Pass config as environment variables
			projCfg, _ := config.LoadProjectConfig()
			globalCfg, _ := config.LoadGlobalConfig()

			if projJSON, err := json.Marshal(projCfg); err == nil {
				execCmd.Env = append(os.Environ(), "PNP_PROJECT_CONFIG="+string(projJSON))
			}
			if globalJSON, err := json.Marshal(globalCfg); err == nil {
				execCmd.Env = append(execCmd.Env, "PNP_GLOBAL_CONFIG="+string(globalJSON))
			}

			return execCmd.Run()
		},
		DisableFlagParsing: true,
	}

	registeredCommands = append(registeredCommands, cmd)
}

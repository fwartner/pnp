package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/fwartner/pnp/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage project configuration (.cluster.yaml)",
	Long:  "View and modify the .cluster.yaml project configuration.",
}

var envListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show current configuration",
	RunE:  runEnvList,
}

var envSetCmd = &cobra.Command{
	Use:   "set <key=value>",
	Short: "Set a configuration value",
	Long:  "Set a field in .cluster.yaml. Use dot notation for nested fields (e.g., database.enabled=true).",
	Args:  cobra.ExactArgs(1),
	RunE:  runEnvSet,
}

var envEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open configuration in your editor",
	RunE:  runEnvEdit,
}

func init() {
	envCmd.AddCommand(envListCmd)
	envCmd.AddCommand(envSetCmd)
	envCmd.AddCommand(envEditCmd)
	rootCmd.AddCommand(envCmd)
}

func runEnvList(cmd *cobra.Command, args []string) error {
	projCfg, err := config.LoadProjectConfig()
	if err != nil {
		return fmt.Errorf("no .cluster.yaml found: %w", err)
	}

	data, err := yaml.Marshal(projCfg)
	if err != nil {
		return err
	}

	fmt.Println(titleStyle.Render("== Project Configuration =="))
	fmt.Println()
	fmt.Println(string(data))
	return nil
}

func runEnvSet(cmd *cobra.Command, args []string) error {
	parts := strings.SplitN(args[0], "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("expected key=value format, got %q", args[0])
	}
	key, value := parts[0], parts[1]

	// Load as raw YAML map for flexible key setting
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	path := cwd + "/.cluster.yaml"
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading .cluster.yaml: %w", err)
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parsing .cluster.yaml: %w", err)
	}

	// Set nested key using dot notation
	setNestedKey(raw, key, value)

	out, err := yaml.Marshal(raw)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, out, 0644); err != nil {
		return err
	}

	fmt.Println(successStyle.Render(fmt.Sprintf("Set %s = %s", key, value)))

	// Prompt to sync
	var doSync bool
	_ = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Run 'pnp sync' to apply changes?").
				Value(&doSync),
		),
	).Run()

	if doSync {
		syncCmd.RunE(cmd, nil)
	}

	return nil
}

func runEnvEdit(cmd *cobra.Command, args []string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	path := cwd + "/.cluster.yaml"

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("no .cluster.yaml found in current directory")
	}

	editorCmd := exec.Command(editor, path)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr
	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("editor failed: %w", err)
	}

	// Validate after edit
	_, err = config.LoadProjectConfig()
	if err != nil {
		fmt.Println(errorStyle.Render("Warning: .cluster.yaml may be invalid: " + err.Error()))
		return nil
	}

	fmt.Println(successStyle.Render("Configuration updated."))

	var doSync bool
	_ = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Run 'pnp sync' to apply changes?").
				Value(&doSync),
		),
	).Run()

	if doSync {
		syncCmd.RunE(cmd, nil)
	}

	return nil
}

// setNestedKey sets a value in a nested map using dot-separated key path.
func setNestedKey(m map[string]interface{}, key string, value string) {
	parts := strings.Split(key, ".")
	current := m

	for i, part := range parts {
		if i == len(parts)-1 {
			// Convert "true"/"false" to bool
			switch strings.ToLower(value) {
			case "true":
				current[part] = true
			case "false":
				current[part] = false
			default:
				current[part] = value
			}
			return
		}

		next, ok := current[part]
		if !ok {
			next = make(map[string]interface{})
			current[part] = next
		}
		if nextMap, ok := next.(map[string]interface{}); ok {
			current = nextMap
		} else {
			// Overwrite non-map with new map
			newMap := make(map[string]interface{})
			current[part] = newMap
			current = newMap
		}
	}
}

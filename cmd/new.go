package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/fwartner/pnp/internal/ci"
	"github.com/fwartner/pnp/internal/config"
	"github.com/fwartner/pnp/internal/types"
	"github.com/fwartner/pnp/internal/wizard"
	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new <type> <name>",
	Short: "Scaffold a new project with deployment configuration",
	Long:  "Creates a new project directory with scaffold files, .cluster.yaml, Dockerfile, and CI workflow.",
	Args:  cobra.ExactArgs(2),
	RunE:  runNew,
}

func init() {
	rootCmd.AddCommand(newCmd)
}

func runNew(cmd *cobra.Command, args []string) error {
	typeName := args[0]
	name := args[1]

	fmt.Println(titleStyle.Render("== PnP New =="))

	// Validate project type
	pt := types.Get(typeName)
	if pt == nil {
		fmt.Println(errorStyle.Render("Unknown project type: " + typeName))
		fmt.Println("Available types:")
		for _, n := range types.Names() {
			fmt.Printf("  - %s\n", n)
		}
		return fmt.Errorf("unknown project type: %s", typeName)
	}

	// Prompt for scope and environment
	scope := "agency"
	environment := "production"

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Project scope").
				Description("Determines naming convention, default org, domain, and visibility").
				Options(
					huh.NewOption("Customer project", "customer"),
					huh.NewOption("Private / internal project", "private"),
					huh.NewOption("Agency project (Pixel & Process)", "agency"),
				).
				Value(&scope),
			huh.NewSelect[string]().
				Title("Environment").
				Options(
					huh.NewOption("Preview", "preview"),
					huh.NewOption("Staging", "staging"),
					huh.NewOption("Production", "production"),
				).
				Value(&environment),
		),
	).Run()
	if err != nil {
		return err
	}

	// Construct scope-prefixed name
	fullName := name
	if !config.HasScopePrefix(name, scope) {
		fullName = config.ScopePrefixedName(scope, name)
	}

	fmt.Printf("  Project: %s (%s)\n", fullName, pt.DisplayName())

	// Create project directory
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	projectDir := filepath.Join(cwd, fullName)

	if _, err := os.Stat(projectDir); err == nil {
		return fmt.Errorf("directory %s already exists", fullName)
	}

	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return fmt.Errorf("creating project directory: %w", err)
	}

	// Write scaffold files
	fmt.Println(dimStyle.Render("  Generating scaffold files..."))
	scaffoldData := types.ScaffoldData{
		Name:      fullName,
		ShortName: config.ShortName(fullName, scope),
		Scope:     scope,
	}

	files := pt.ScaffoldFiles(scaffoldData)
	for relPath, content := range files {
		absPath := filepath.Join(projectDir, relPath)
		if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", relPath, err)
		}
		if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", relPath, err)
		}
	}
	fmt.Printf("  Wrote %d scaffold files\n", len(files))

	// Initialize git repo
	fmt.Println(dimStyle.Render("  Initializing git repository..."))
	gitInit := exec.Command("git", "init", "-b", "main")
	gitInit.Dir = projectDir
	if out, err := gitInit.CombinedOutput(); err != nil {
		fmt.Printf("  Warning: git init failed: %s\n", string(out))
	}

	// Generate .cluster.yaml with defaults
	globalCfg, _ := config.LoadGlobalConfig()
	projCfg := config.ProjectConfig{
		Name:        fullName,
		Type:        typeName,
		Scope:       scope,
		Environment: environment,
	}
	wizard.ApplyDefaults(&projCfg, globalCfg)

	clusterYAMLPath := filepath.Join(projectDir, ".cluster.yaml")
	if err := config.SaveProjectConfigTo(projCfg, clusterYAMLPath); err != nil {
		return fmt.Errorf("saving .cluster.yaml: %w", err)
	}
	fmt.Println(dimStyle.Render("  Generated .cluster.yaml"))

	// Generate Dockerfile
	if err := ci.GenerateDockerfile(typeName, projCfg.Octane, projectDir); err != nil {
		fmt.Println(warnStyle.Render("  Warning: could not generate Dockerfile: " + err.Error()))
	} else {
		fmt.Println(dimStyle.Render("  Generated Dockerfile"))
	}

	// Generate CI workflow
	if globalCfg.GitopsRemote != "" {
		image := fmt.Sprintf("ghcr.io/%s:latest", fullName)
		if err := ci.GenerateWorkflow(typeName, image, globalCfg.GitopsRemote, fullName, projectDir); err != nil {
			fmt.Println(warnStyle.Render("  Warning: could not generate CI workflow: " + err.Error()))
		} else {
			fmt.Println(dimStyle.Render("  Generated .github/workflows/deploy.yml"))
		}
	}

	// Initial commit
	gitAdd := exec.Command("git", "add", "-A")
	gitAdd.Dir = projectDir
	if _, err := gitAdd.CombinedOutput(); err == nil {
		gitCommit := exec.Command("git", "commit", "-m", "Initial scaffold via pnp new")
		gitCommit.Dir = projectDir
		gitCommit.CombinedOutput() // best-effort
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("Project %s created!", fullName)))
	fmt.Printf("  cd %s\n", fullName)
	fmt.Println("  pnp deploy")
	return nil
}

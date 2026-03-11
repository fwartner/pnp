package cmd

import (
	"fmt"
	"os"

	"github.com/fwartner/pnp/internal/ci"
	"github.com/fwartner/pnp/internal/config"
	"github.com/fwartner/pnp/internal/gitops"
	"github.com/fwartner/pnp/internal/secrets"
	"github.com/fwartner/pnp/internal/templates"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Re-sync gitops manifests from .cluster.yaml",
	Long:  "Non-interactive command that re-reads .cluster.yaml, renders templates, and updates the gitops repository. Use after changing configuration or in CI pipelines.",
	RunE:  runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	fmt.Println(titleStyle.Render("== PnP Sync =="))

	globalCfg, err := config.LoadGlobalConfig()
	if err != nil {
		return fmt.Errorf("loading global config: %w", err)
	}

	projCfg, err := config.LoadProjectConfig()
	if err != nil {
		return fmt.Errorf("loading .cluster.yaml: %w (run 'pnp deploy' first)", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Ensure secrets are generated
	isLaravel := projCfg.Type == "laravel-web" || projCfg.Type == "laravel-api"
	secretsChanged := false

	if isLaravel && projCfg.Secrets.AppKey == "" {
		key, err := secrets.GenerateAppKey()
		if err != nil {
			return fmt.Errorf("generating APP_KEY: %w", err)
		}
		projCfg.Secrets.AppKey = key
		secretsChanged = true
	}

	hasDB := isLaravel || projCfg.Type == "nextjs-fullstack" || projCfg.Type == "strapi"
	if hasDB && projCfg.Secrets.DBPassword == "" {
		pw, err := secrets.GeneratePassword(32)
		if err != nil {
			return fmt.Errorf("generating DB password: %w", err)
		}
		projCfg.Secrets.DBPassword = pw
		secretsChanged = true
	}

	if secretsChanged {
		if err := config.SaveProjectConfig(projCfg); err != nil {
			return fmt.Errorf("saving secrets: %w", err)
		}
	}

	// Render templates
	data := buildTemplateData(projCfg, globalCfg)

	tmpDir, err := os.MkdirTemp("", "pnp-sync-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := templates.Render(projCfg.Type, data, tmpDir); err != nil {
		return fmt.Errorf("rendering templates: %w", err)
	}

	// Write to gitops repo
	if globalCfg.GitopsRepo == "" {
		return fmt.Errorf("gitopsRepo is not set in ~/.pnp.yaml")
	}

	repo := gitops.NewRepo(globalCfg.GitopsRepo)
	if err := repo.Pull(); err != nil {
		fmt.Printf("  Warning: git pull failed: %v\n", err)
	}

	namespace := namespaceFromConfig(projCfg)

	if err := repo.WriteApp(projCfg.Name, projCfg.Environment, projCfg.Scope, tmpDir); err != nil {
		return fmt.Errorf("writing app: %w", err)
	}

	commitMsg := fmt.Sprintf("sync(%s): %s to %s", projCfg.Environment, projCfg.Name, namespace)
	if err := repo.CommitChanges(commitMsg); err != nil {
		return fmt.Errorf("committing: %w", err)
	}

	if err := repo.Push(); err != nil {
		return fmt.Errorf("pushing: %w", err)
	}

	fmt.Println(successStyle.Render("Gitops repo synced"))

	// Regenerate CI files if needed
	if err := ci.GenerateWorkflow(projCfg.Type, projCfg.Image, globalCfg.GitopsRemote, projCfg.Name, cwd); err != nil {
		fmt.Printf("  Warning: failed to update CI workflow: %v\n", err)
	}

	if err := ci.GenerateDockerfile(projCfg.Type, projCfg.Octane, cwd); err != nil {
		fmt.Printf("  Warning: failed to update Dockerfile: %v\n", err)
	}

	fmt.Println(successStyle.Render("Sync complete!"))
	return nil
}

package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/fwartner/pnp/internal/config"
	"github.com/fwartner/pnp/internal/gitops"
	"github.com/fwartner/pnp/internal/templates"
	"github.com/fwartner/pnp/internal/wizard"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Re-render and push templates for an existing project",
	Long:  "Reads .cluster.yaml, rebuilds templates from the current config, and pushes updated manifests to the gitops repository.",
	RunE:  runUpdate,
}

var updateFlagPR bool

func init() {
	updateCmd.Flags().BoolVar(&updateFlagPR, "pr", false, "Create a pull request instead of pushing directly")
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	fmt.Println(titleStyle.Render("== PnP Update =="))

	// 1. Load .cluster.yaml (must exist)
	projCfg, err := config.LoadProjectConfig()
	if err != nil {
		fmt.Println(errorStyle.Render("No .cluster.yaml found — run 'pnp deploy' first."))
		return fmt.Errorf("loading .cluster.yaml: %w", err)
	}
	fmt.Println(successStyle.Render("Loaded .cluster.yaml for " + projCfg.Name))

	// 2. Load global config
	globalCfg, err := config.LoadGlobalConfig()
	if err != nil {
		fmt.Println(errorStyle.Render("Failed to load global config: " + err.Error()))
		return err
	}

	// 3. Determine namespace
	namespace := projCfg.Name
	env := strings.ToLower(projCfg.Environment)
	if env == "preview" || env == "staging" {
		namespace = "preview-" + projCfg.Name
	}

	// 4. Build TemplateData
	isLaravelWeb := projCfg.Type == "laravel-web"
	data := templates.TemplateData{
		Name:      projCfg.Name,
		Namespace: namespace,
		Subdomain: wizard.Subdomain(projCfg.Domain, globalCfg.Defaults.Domain),
		Domain:    globalCfg.Defaults.Domain,
		Image:     projCfg.Image,
		Tag:       "latest",
		DBName:    projCfg.Database.Name,
		DBUsername: projCfg.Name,
		DBSize:    projCfg.Database.Size,

		RedisEnabled:     projCfg.Redis.Enabled,
		QueueEnabled:     isLaravelWeb,
		SchedulerEnabled: isLaravelWeb,

		PersistenceEnabled: isLaravelWeb || projCfg.Type == "strapi",
		PersistenceSize:    "5Gi",

		InfisicalProjectSlug: projCfg.Infisical.ProjectSlug,
		InfisicalEnvSlug:     projCfg.Infisical.EnvSlug,
		InfisicalSecretsPath: projCfg.Infisical.SecretsPath,
		InfisicalMailEnabled: isLaravelWeb || projCfg.Type == "laravel-api",

		CPU:    projCfg.Resources.CPU,
		Memory: projCfg.Resources.Memory,

		ChartPath: "apps/" + projCfg.Name,
		RepoURL:   globalCfg.GitopsRemote,
	}

	// 5. Render templates to temp dir
	fmt.Println(titleStyle.Render("Rendering templates..."))
	tmpDir, err := os.MkdirTemp("", "pnp-update-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := templates.Render(projCfg.Type, data, tmpDir); err != nil {
		fmt.Println(errorStyle.Render("Template rendering failed: " + err.Error()))
		return err
	}
	fmt.Println(successStyle.Render("Templates rendered"))

	// 6. Write to gitops repo
	if globalCfg.GitopsRepo == "" {
		return fmt.Errorf("gitopsRepo is not set in ~/.pnp.yaml")
	}

	fmt.Println(titleStyle.Render("Writing to gitops repo..."))
	repo := gitops.NewRepo(globalCfg.GitopsRepo)

	if err := repo.Pull(); err != nil {
		fmt.Printf("  Warning: git pull failed: %v\n", err)
	}

	if err := repo.WriteApp(projCfg.Name, projCfg.Environment, tmpDir); err != nil {
		fmt.Println(errorStyle.Render("Failed to write app: " + err.Error()))
		return err
	}
	fmt.Println(successStyle.Render("Manifests written to gitops repo"))

	// 7. Commit and push (or create PR)
	commitMsg := fmt.Sprintf("update(%s): %s in %s", projCfg.Environment, projCfg.Name, namespace)

	if updateFlagPR {
		fmt.Println(titleStyle.Render("Creating pull request..."))
		branch := fmt.Sprintf("update/%s-%s", projCfg.Name, projCfg.Environment)
		if err := repo.CreateBranchAndPush(branch, commitMsg); err != nil {
			fmt.Println(errorStyle.Render("Failed to create branch: " + err.Error()))
			return err
		}

		prURL, err := repo.CreatePR(commitMsg, fmt.Sprintf("Automated update of %s in %s", projCfg.Name, projCfg.Environment), branch)
		if err != nil {
			fmt.Println(errorStyle.Render("Failed to create PR: " + err.Error()))
			return err
		}
		fmt.Println(successStyle.Render("Pull request created: " + prURL))
	} else {
		fmt.Println(titleStyle.Render("Committing and pushing..."))
		if err := repo.CommitChanges(commitMsg); err != nil {
			fmt.Println(errorStyle.Render("Failed to commit: " + err.Error()))
			return err
		}
		if err := repo.Push(); err != nil {
			fmt.Println(errorStyle.Render("Failed to push: " + err.Error()))
			return err
		}
		fmt.Println(successStyle.Render("Changes pushed to gitops repo"))
	}

	fmt.Println(successStyle.Render("\nUpdate complete!"))
	return nil
}

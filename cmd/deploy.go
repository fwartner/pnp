package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/fwartner/pnp/internal/ci"
	"github.com/fwartner/pnp/internal/config"
	"github.com/fwartner/pnp/internal/detect"
	"github.com/fwartner/pnp/internal/gitops"
	"github.com/fwartner/pnp/internal/infisical"
	"github.com/fwartner/pnp/internal/templates"
	"github.com/fwartner/pnp/internal/wizard"
	"github.com/spf13/cobra"
)

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	successStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	errorStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy the current project to the cluster",
	Long:  "Detects the project type, runs the interactive wizard, renders templates, and pushes to the gitops repository.",
	RunE:  runDeploy,
}

var (
	flagPR          bool
	flagSkipSecrets bool
	flagWithCI      bool
)

func init() {
	deployCmd.Flags().BoolVar(&flagPR, "pr", false, "Create a pull request instead of pushing directly")
	deployCmd.Flags().BoolVar(&flagSkipSecrets, "skip-secrets", false, "Skip creating secrets in Infisical")
	deployCmd.Flags().BoolVar(&flagWithCI, "with-ci", false, "Generate GitHub Actions deploy workflow")
	rootCmd.AddCommand(deployCmd)
}

func runDeploy(cmd *cobra.Command, args []string) error {
	fmt.Println(titleStyle.Render("== PnP Deploy =="))

	// 1. Load global config
	globalCfg, err := config.LoadGlobalConfig()
	if err != nil {
		fmt.Println(errorStyle.Render("Failed to load global config: " + err.Error()))
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// 2. Check for existing .cluster.yaml
	var projCfg config.ProjectConfig
	existingConfig := false

	projCfg, err = config.LoadProjectConfig()
	if err == nil {
		existingConfig = true
		fmt.Println(successStyle.Render("Found existing .cluster.yaml"))
	} else {
		// 3. No .cluster.yaml: detect, infer, run wizard
		fmt.Println(titleStyle.Render("Detecting project type..."))
		detected := detect.DetectProjectType(cwd)
		fmt.Printf("  Detected: %s (%s confidence)\n", detected.Type, detected.Confidence)

		projectName := detect.InferProjectName(cwd)
		inferredImage := detect.InferImageFromGitRemote(cwd, globalCfg.Defaults.ImageRegistry)

		fmt.Println(titleStyle.Render("Running setup wizard..."))
		projCfg, err = wizard.Run(detected, inferredImage, projectName, globalCfg)
		if err != nil {
			fmt.Println(errorStyle.Render("Wizard failed: " + err.Error()))
			return err
		}
	}

	// 4. Build TemplateData
	namespace := namespaceFromConfig(projCfg)
	data := buildTemplateData(projCfg, globalCfg)

	// 6. Render templates to temp dir
	fmt.Println(titleStyle.Render("Rendering templates..."))
	tmpDir, err := os.MkdirTemp("", "pnp-deploy-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := templates.Render(projCfg.Type, data, tmpDir); err != nil {
		fmt.Println(errorStyle.Render("Template rendering failed: " + err.Error()))
		return err
	}
	fmt.Println(successStyle.Render("Templates rendered"))

	// 7. Write to gitops repo
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

	// 8. Create secrets via Infisical if applicable
	if !flagSkipSecrets && projCfg.Database.Enabled && globalCfg.Infisical.Token != "" {
		fmt.Println(titleStyle.Render("Creating secrets in Infisical..."))
		client := infisical.NewClient(globalCfg.Infisical.Host, globalCfg.Infisical.Token)

		pw, err := infisical.GeneratePassword()
		if err != nil {
			fmt.Println(errorStyle.Render("Failed to generate password: " + err.Error()))
			return err
		}

		secrets := map[string]string{
			"password": pw,
			"username": projCfg.Name,
		}

		err = client.CreateSecrets(
			secrets,
			projCfg.Infisical.ProjectSlug,
			projCfg.Infisical.EnvSlug,
			projCfg.Infisical.SecretsPath,
		)
		if err != nil {
			fmt.Println(errorStyle.Render("Warning: failed to create secrets: " + err.Error()))
			// Non-fatal: continue with deploy
		} else {
			fmt.Println(successStyle.Render("Secrets created in Infisical"))
		}
	}

	// 9. Commit and push (or create PR)
	commitMsg := fmt.Sprintf("deploy(%s): %s to %s", projCfg.Environment, projCfg.Name, namespace)

	if flagPR {
		fmt.Println(titleStyle.Render("Creating pull request..."))
		branch := fmt.Sprintf("deploy/%s-%s", projCfg.Name, projCfg.Environment)
		if err := repo.CreateBranchAndPush(branch, commitMsg); err != nil {
			fmt.Println(errorStyle.Render("Failed to create branch: " + err.Error()))
			return err
		}

		prURL, err := repo.CreatePR(commitMsg, fmt.Sprintf("Automated deploy of %s to %s", projCfg.Name, projCfg.Environment), branch)
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

	// 10. Generate GitHub Actions workflow if requested
	if flagWithCI {
		fmt.Println(titleStyle.Render("Generating GitHub Actions workflow..."))
		if err := ci.GenerateWorkflow(projCfg.Type, projCfg.Image, cwd); err != nil {
			fmt.Println(errorStyle.Render("Failed to generate CI workflow: " + err.Error()))
			return err
		}
		fmt.Println(successStyle.Render("GitHub Actions workflow generated at .github/workflows/deploy.yml"))
	}

	// 11. Save .cluster.yaml if not existing
	if !existingConfig {
		if err := config.SaveProjectConfig(projCfg); err != nil {
			fmt.Println(errorStyle.Render("Failed to save .cluster.yaml: " + err.Error()))
			return err
		}
		fmt.Println(successStyle.Render("Saved .cluster.yaml"))
	}

	fmt.Println(successStyle.Render("\nDeploy complete!"))
	return nil
}

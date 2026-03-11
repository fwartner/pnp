package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/fwartner/pnp/internal/ci"
	"github.com/fwartner/pnp/internal/config"
	"github.com/fwartner/pnp/internal/detect"
	"github.com/fwartner/pnp/internal/gh"
	"github.com/fwartner/pnp/internal/gitops"
	"github.com/fwartner/pnp/internal/secrets"
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
	flagPR     bool
	flagWithCI bool
)

func init() {
	deployCmd.Flags().BoolVar(&flagPR, "pr", false, "Create a pull request instead of pushing directly")
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

	// 4. Ensure git repo and GitHub remote exist (uses scope for defaults)
	if err := ensureGitRepo(cwd, projCfg.Scope, globalCfg); err != nil {
		return err
	}

	// 5. Generate secrets if not already set
	secretsChanged := false
	isLaravel := projCfg.Type == "laravel-web" || projCfg.Type == "laravel-api"
	if isLaravel && projCfg.Secrets.AppKey == "" {
		key, err := secrets.GenerateAppKey()
		if err != nil {
			return fmt.Errorf("generating APP_KEY: %w", err)
		}
		projCfg.Secrets.AppKey = key
		secretsChanged = true
		fmt.Println(successStyle.Render("Generated APP_KEY"))
	}

	hasDB := projCfg.Type == "laravel-web" || projCfg.Type == "laravel-api" ||
		projCfg.Type == "nextjs-fullstack" || projCfg.Type == "strapi"
	if hasDB && projCfg.Secrets.DBPassword == "" {
		pw, err := secrets.GeneratePassword(32)
		if err != nil {
			return fmt.Errorf("generating DB password: %w", err)
		}
		projCfg.Secrets.DBPassword = pw
		secretsChanged = true
		fmt.Println(successStyle.Render("Generated database password"))
	}

	// Persist generated secrets immediately so they survive re-runs
	if secretsChanged {
		if err := config.SaveProjectConfig(projCfg); err != nil {
			fmt.Println(errorStyle.Render("Failed to save secrets to .cluster.yaml: " + err.Error()))
			return err
		}
	}

	// 6. Build TemplateData
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

	// 8. Commit and push (or create PR)
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

	// 10. Generate GitHub Actions workflow and Dockerfile
	ciDir := filepath.Join(cwd, ".github", "workflows")
	workflowExists := false
	if _, err := os.Stat(filepath.Join(ciDir, "deploy.yml")); err == nil {
		workflowExists = true
	}

	var ciFilesGenerated []string

	if !workflowExists || flagWithCI {
		fmt.Println(titleStyle.Render("Generating CI/CD pipeline..."))
		if err := ci.GenerateWorkflow(projCfg.Type, projCfg.Image, globalCfg.GitopsRemote, projCfg.Name, cwd); err != nil {
			fmt.Println(errorStyle.Render("Failed to generate CI workflow: " + err.Error()))
			return err
		}
		ciFilesGenerated = append(ciFilesGenerated, ".github/workflows/deploy.yml")
		fmt.Println(successStyle.Render("GitHub Actions workflow generated"))

		dockerfilePath := filepath.Join(cwd, "Dockerfile")
		if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
			if err := ci.GenerateDockerfile(projCfg.Type, projCfg.Octane, cwd); err != nil {
				fmt.Println(errorStyle.Render("Failed to generate Dockerfile: " + err.Error()))
				return err
			}
			ciFilesGenerated = append(ciFilesGenerated, "Dockerfile")
			// .dockerignore is also generated if missing
			if _, err := os.Stat(filepath.Join(cwd, ".dockerignore")); err == nil {
				ciFilesGenerated = append(ciFilesGenerated, ".dockerignore")
			}
			fmt.Println(successStyle.Render("Dockerfile generated"))
		}
	}

	// 11. Save .cluster.yaml if not existing
	if !existingConfig {
		if err := config.SaveProjectConfig(projCfg); err != nil {
			fmt.Println(errorStyle.Render("Failed to save .cluster.yaml: " + err.Error()))
			return err
		}
		ciFilesGenerated = append(ciFilesGenerated, ".cluster.yaml")
		fmt.Println(successStyle.Render("Saved .cluster.yaml"))
	}

	// 12. Auto-commit and push generated CI files to the project repo
	if len(ciFilesGenerated) > 0 && gh.HasGitRepo(cwd) && gh.HasGitRemote(cwd) {
		fmt.Println(titleStyle.Render("Committing CI/CD files to project repo..."))
		if err := gh.CommitAndPush(cwd, "ci: add pnp deployment pipeline", ciFilesGenerated...); err != nil {
			fmt.Println(errorStyle.Render("Failed to commit CI files: " + err.Error()))
			fmt.Println("  You can manually commit and push the generated files.")
		} else {
			fmt.Println(successStyle.Render("CI/CD files committed and pushed"))
		}
	}

	// 13. Ensure GITOPS_TOKEN is configured on the project repo
	if gh.GHCLIAvailable() && gh.HasGitRepo(cwd) && gh.HasGitRemote(cwd) {
		if err := ensureGitopsToken(cwd); err != nil {
			fmt.Printf("  Warning: could not configure GITOPS_TOKEN: %v\n", err)
			fmt.Println("  The CI deploy job needs a GITOPS_TOKEN secret to push to the gitops repo.")
		}
	}

	fmt.Println(successStyle.Render("\nDeploy complete! Push to main to trigger automatic builds and deployments."))
	return nil
}

// ensureGitopsToken checks if GITOPS_TOKEN is set on the project repo.
// If not, it uses the current gh auth token to set it up.
func ensureGitopsToken(dir string) error {
	repoName, err := gh.GetRepoFullName(dir)
	if err != nil {
		return err
	}

	if gh.HasRepoSecret(repoName, "GITOPS_TOKEN") {
		return nil // already configured
	}

	fmt.Println(titleStyle.Render("Setting up GITOPS_TOKEN for cross-repo deployments..."))

	token, err := gh.GetAuthToken()
	if err != nil {
		return fmt.Errorf("could not get auth token: %w", err)
	}

	if err := gh.SetRepoSecret(repoName, "GITOPS_TOKEN", token); err != nil {
		return err
	}

	fmt.Println(successStyle.Render("GITOPS_TOKEN configured on " + repoName))
	return nil
}

// ensureGitRepo checks if the current directory has a git repo with a GitHub remote.
// If not, it offers to create one using the gh CLI. Uses scope-based defaults for org and visibility.
func ensureGitRepo(dir string, scope string, globalCfg config.GlobalConfig) error {
	if gh.HasGitRepo(dir) && gh.HasGitRemote(dir) {
		return nil // all good
	}

	if !gh.GHCLIAvailable() {
		if !gh.HasGitRepo(dir) {
			return fmt.Errorf("no git repository found and gh CLI is not available — initialize a git repo manually or install/authenticate the gh CLI")
		}
		fmt.Println(errorStyle.Render("Warning: no git remote found and gh CLI not available. Image inference may not work."))
		return nil
	}

	status := "No git repository found."
	if gh.HasGitRepo(dir) {
		status = "Git repository found but no GitHub remote."
	}

	var createRepo bool
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(status + " Create a GitHub repository?").
				Description("Uses the gh CLI to create a repo and set up the remote.").
				Value(&createRepo),
		),
	).Run()
	if err != nil {
		return err
	}

	if !createRepo {
		return nil
	}

	projectName := filepath.Base(dir)
	org := globalCfg.EffectiveGithubOrg(scope)

	var repoName string
	if org != "" {
		repoName = org + "/" + projectName
	} else {
		repoName = projectName
	}

	repoVisibility := globalCfg.EffectiveRepoVisibility(scope)

	err = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Repository name").
				Value(&repoName),
			huh.NewSelect[string]().
				Title("Visibility").
				Description(fmt.Sprintf("Default for %s scope: %s", scope, repoVisibility)).
				Options(
					huh.NewOption("Private", "private"),
					huh.NewOption("Public", "public"),
				).
				Value(&repoVisibility),
		),
	).Run()
	if err != nil {
		return err
	}

	fmt.Println(titleStyle.Render("Creating GitHub repository..."))
	remoteURL, err := gh.InitAndCreateRepo(gh.CreateRepoOptions{
		Name:        repoName,
		Description: projectName + " — managed by pnp",
		Private:     repoVisibility == "private",
		Dir:         dir,
	})
	if err != nil {
		return fmt.Errorf("creating GitHub repo: %w", err)
	}

	fmt.Println(successStyle.Render("Repository created: " + remoteURL))
	return nil
}

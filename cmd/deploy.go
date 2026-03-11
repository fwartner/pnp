package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/fwartner/pnp/internal/ci"
	"github.com/fwartner/pnp/internal/config"
	"github.com/fwartner/pnp/internal/detect"
	"github.com/fwartner/pnp/internal/doctor"
	"github.com/fwartner/pnp/internal/gh"
	"github.com/fwartner/pnp/internal/gitops"
	"github.com/fwartner/pnp/internal/progress"
	"github.com/fwartner/pnp/internal/secrets"
	"github.com/fwartner/pnp/internal/templates"
	"github.com/fwartner/pnp/internal/wizard"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy the current project to the cluster",
	Long:  "Detects the project type, runs the interactive wizard, renders templates, and pushes to the gitops repository.",
	RunE:  runDeploy,
}

var (
	flagPR           bool
	flagWithCI       bool
	flagAdvanced     bool
	flagWithPreviews bool
)

func init() {
	deployCmd.Flags().BoolVar(&flagPR, "pr", false, "Create a pull request instead of pushing directly")
	deployCmd.Flags().BoolVar(&flagWithCI, "with-ci", false, "Generate GitHub Actions deploy workflow")
	deployCmd.Flags().BoolVar(&flagAdvanced, "advanced", false, "Run the full advanced wizard with all options")
	deployCmd.Flags().BoolVar(&flagWithPreviews, "with-previews", false, "Generate preview environment workflow for PRs")
	rootCmd.AddCommand(deployCmd)
}

func runDeploy(cmd *cobra.Command, args []string) error {
	fmt.Println(titleStyle.Render("== PnP Deploy =="))

	// 0. Run doctor checks for critical prerequisites
	results := doctor.RunAll(false)
	if doctor.HasCriticalFailure(results) {
		fmt.Println()
		for _, r := range results {
			if !r.OK && r.Critical {
				fmt.Printf("  %s  %s: %s\n", errorStyle.Render("✗"), r.Name, r.Message)
			}
		}
		fmt.Println()
		fmt.Println(errorStyle.Render("Critical prerequisites missing. Run 'pnp doctor' to fix."))
		return fmt.Errorf("prerequisites check failed")
	}

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

		if flagAdvanced {
			fmt.Println(titleStyle.Render("Running advanced wizard..."))
			projCfg, err = wizard.RunAdvanced(detected, inferredImage, projectName, globalCfg)
		} else {
			fmt.Println(titleStyle.Render("Running setup wizard..."))
			projCfg, err = wizard.RunBasic(detected, inferredImage, projectName, globalCfg)
		}
		if err != nil {
			fmt.Println(errorStyle.Render("Wizard failed: " + err.Error()))
			return err
		}
	}

	// 4. Ensure git repo and GitHub remote exist (uses scope for defaults)
	if err := ensureGitRepo(cwd, projCfg.Scope, globalCfg); err != nil {
		return err
	}

	// Variables used across steps
	var tmpDir string
	namespace := namespaceFromConfig(projCfg)
	var ciFilesGenerated []string

	// Build tracked steps
	tracker := progress.NewTracker(
		progress.Step{
			Name: "Generate secrets",
			Action: func() error {
				secretsChanged := false
				isLaravel := projCfg.Type == "laravel-web" || projCfg.Type == "laravel-api"
				if isLaravel && projCfg.Secrets.AppKey == "" {
					key, err := secrets.GenerateAppKey()
					if err != nil {
						return fmt.Errorf("generating APP_KEY: %w", err)
					}
					projCfg.Secrets.AppKey = key
					secretsChanged = true
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
				}

				if secretsChanged {
					return config.SaveProjectConfig(projCfg)
				}
				return nil
			},
		},
		progress.Step{
			Name: "Render templates",
			Action: func() error {
				var err error
				tmpDir, err = os.MkdirTemp("", "pnp-deploy-*")
				if err != nil {
					return fmt.Errorf("creating temp dir: %w", err)
				}
				data := buildTemplateData(projCfg, globalCfg)
				return templates.Render(projCfg.Type, data, tmpDir)
			},
		},
		progress.Step{
			Name: "Pull gitops repo",
			Action: func() error {
				if globalCfg.GitopsRepo == "" {
					return fmt.Errorf("gitopsRepo is not set in ~/.pnp.yaml")
				}
				repo := gitops.NewRepo(globalCfg.GitopsRepo)
				_ = repo.Pull() // non-fatal
				return nil
			},
		},
		progress.Step{
			Name: "Write manifests",
			Action: func() error {
				repo := gitops.NewRepo(globalCfg.GitopsRepo)
				return repo.WriteApp(projCfg.Name, projCfg.Environment, projCfg.Scope, tmpDir)
			},
		},
		progress.Step{
			Name: "Push to gitops",
			Action: func() error {
				repo := gitops.NewRepo(globalCfg.GitopsRepo)
				commitMsg := fmt.Sprintf("deploy(%s): %s to %s", projCfg.Environment, projCfg.Name, namespace)

				if flagPR {
					branch := fmt.Sprintf("deploy/%s-%s", projCfg.Name, projCfg.Environment)
					if err := repo.CreateBranchAndPush(branch, commitMsg); err != nil {
						return err
					}
					prURL, err := repo.CreatePR(commitMsg, fmt.Sprintf("Automated deploy of %s to %s", projCfg.Name, projCfg.Environment), branch)
					if err != nil {
						return err
					}
					fmt.Printf("\r  Pull request: %s\n", prURL)
					return nil
				}

				if err := repo.CommitChanges(commitMsg); err != nil {
					return err
				}
				return repo.Push()
			},
		},
		progress.Step{
			Name: "Generate CI/CD pipeline",
			Action: func() error {
				ciDir := filepath.Join(cwd, ".github", "workflows")
				workflowExists := false
				if _, err := os.Stat(filepath.Join(ciDir, "deploy.yml")); err == nil {
					workflowExists = true
				}

				if !workflowExists || flagWithCI {
					if err := ci.GenerateWorkflow(projCfg.Type, projCfg.Image, globalCfg.GitopsRemote, projCfg.Name, cwd); err != nil {
						return err
					}
					ciFilesGenerated = append(ciFilesGenerated, ".github/workflows/deploy.yml")

					dockerfilePath := filepath.Join(cwd, "Dockerfile")
					if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
						if err := ci.GenerateDockerfile(projCfg.Type, projCfg.Octane, cwd); err != nil {
							return err
						}
						ciFilesGenerated = append(ciFilesGenerated, "Dockerfile")
						if _, err := os.Stat(filepath.Join(cwd, ".dockerignore")); err == nil {
							ciFilesGenerated = append(ciFilesGenerated, ".dockerignore")
						}
					}
				}

				// Generate preview workflow if requested
				if flagWithPreviews {
					scopeDomain := globalCfg.EffectiveDomain(projCfg.Scope)
					if err := ci.GeneratePreviewWorkflow(projCfg.Image, globalCfg.GitopsRemote, projCfg.Name, scopeDomain, cwd); err != nil {
						return err
					}
					ciFilesGenerated = append(ciFilesGenerated, ".github/workflows/preview.yml")
				}

				return nil
			},
		},
		progress.Step{
			Name: "Save configuration",
			Action: func() error {
				if !existingConfig {
					if err := config.SaveProjectConfig(projCfg); err != nil {
						return err
					}
					ciFilesGenerated = append(ciFilesGenerated, ".cluster.yaml")
				}
				return nil
			},
		},
		progress.Step{
			Name: "Commit CI files",
			Action: func() error {
				if len(ciFilesGenerated) > 0 && gh.HasGitRepo(cwd) && gh.HasGitRemote(cwd) {
					if err := gh.CommitAndPush(cwd, "ci: add pnp deployment pipeline", ciFilesGenerated...); err != nil {
						return nil // non-fatal, user can commit manually
					}
				}
				return nil
			},
		},
		progress.Step{
			Name: "Configure GITOPS_TOKEN",
			Action: func() error {
				if gh.GHCLIAvailable() && gh.HasGitRepo(cwd) && gh.HasGitRemote(cwd) {
					_ = ensureGitopsToken(cwd) // non-fatal
				}
				return nil
			},
		},
	)

	fmt.Println()
	if err := tracker.Run(); err != nil {
		fmt.Println()
		fmt.Println(errorStyle.Render("Deploy failed: " + err.Error()))
		return err
	}

	// Clean up temp dir
	if tmpDir != "" {
		os.RemoveAll(tmpDir)
	}

	fmt.Println()
	fmt.Println(successStyle.Render("Deploy complete! Push to main to trigger automatic builds and deployments."))
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

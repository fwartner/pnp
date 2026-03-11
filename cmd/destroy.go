package cmd

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/fwartner/pnp/internal/config"
	"github.com/fwartner/pnp/internal/gitops"
	"github.com/spf13/cobra"
)

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroy a deployed application from the cluster",
	Long:  "Removes the application manifests from the gitops repository, effectively destroying the deployment.",
	RunE:  runDestroy,
}

var (
	flagDestroyPR           bool
	flagDestroyCleanSecrets bool
)

func init() {
	destroyCmd.Flags().BoolVar(&flagDestroyPR, "pr", false, "Create a pull request instead of pushing directly")
	destroyCmd.Flags().BoolVar(&flagDestroyCleanSecrets, "clean-secrets", false, "Clean up associated secrets in Infisical (placeholder)")
	rootCmd.AddCommand(destroyCmd)
}

func runDestroy(cmd *cobra.Command, args []string) error {
	fmt.Println(titleStyle.Render("== PnP Destroy =="))

	// 1. Load .cluster.yaml (must exist)
	projCfg, err := config.LoadProjectConfig()
	if err != nil {
		fmt.Println(errorStyle.Render("No .cluster.yaml found in current directory."))
		fmt.Println("  Run this command from a project that has been deployed.")
		return fmt.Errorf("loading project config: %w", err)
	}
	fmt.Println(successStyle.Render(fmt.Sprintf("Found project: %s (%s)", projCfg.Name, projCfg.Environment)))

	// 2. Load global config
	globalCfg, err := config.LoadGlobalConfig()
	if err != nil {
		fmt.Println(errorStyle.Render("Failed to load global config: " + err.Error()))
		return err
	}

	if globalCfg.GitopsRepo == "" {
		return fmt.Errorf("gitopsRepo is not set in ~/.pnp.yaml")
	}

	// 3. Confirm destruction with huh
	var confirmed bool
	err = huh.NewConfirm().
		Title(fmt.Sprintf("Destroy %s (%s)?", projCfg.Name, projCfg.Environment)).
		Description("This will remove all manifests from the gitops repository. This action cannot be undone.").
		Affirmative("Yes, destroy").
		Negative("Cancel").
		Value(&confirmed).
		Run()
	if err != nil {
		return fmt.Errorf("confirmation prompt failed: %w", err)
	}

	if !confirmed {
		fmt.Println(titleStyle.Render("Destroy cancelled."))
		return nil
	}

	// 4. Delete app from gitops repo
	fmt.Println(titleStyle.Render("Removing from gitops repo..."))
	repo := gitops.NewRepo(globalCfg.GitopsRepo)

	if err := repo.Pull(); err != nil {
		fmt.Printf("  Warning: git pull failed: %v\n", err)
	}

	if !repo.AppExists(projCfg.Name, projCfg.Environment, projCfg.Scope) {
		fmt.Println(errorStyle.Render("App directory does not exist in gitops repo. Nothing to destroy."))
		return nil
	}

	if err := repo.DeleteApp(projCfg.Name, projCfg.Environment, projCfg.Scope); err != nil {
		fmt.Println(errorStyle.Render("Failed to delete app: " + err.Error()))
		return err
	}
	fmt.Println(successStyle.Render("App manifests removed"))

	// 5. Commit and push (or create PR)
	commitMsg := fmt.Sprintf("destroy(%s): remove %s", projCfg.Environment, projCfg.Name)

	if flagDestroyPR {
		fmt.Println(titleStyle.Render("Creating pull request..."))
		branch := fmt.Sprintf("destroy/%s-%s", projCfg.Name, projCfg.Environment)
		if err := repo.CreateBranchAndPush(branch, commitMsg); err != nil {
			fmt.Println(errorStyle.Render("Failed to create branch: " + err.Error()))
			return err
		}

		prURL, err := repo.CreatePR(commitMsg, fmt.Sprintf("Automated removal of %s from %s", projCfg.Name, projCfg.Environment), branch)
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

	// 6. Clean secrets placeholder
	if flagDestroyCleanSecrets {
		fmt.Println(titleStyle.Render("Cleaning up secrets..."))
		fmt.Println("  Secret cleanup is not yet implemented. Skipping.")
	}

	fmt.Println(successStyle.Render("\nDestroy complete!"))
	return nil
}

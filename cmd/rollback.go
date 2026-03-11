package cmd

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/fwartner/pnp/internal/config"
	"github.com/fwartner/pnp/internal/gitops"
	"github.com/spf13/cobra"
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Revert the last deployment",
	Long:  "Shows recent deployment commits and reverts the selected one by creating a revert commit in the gitops repo.",
	RunE:  runRollback,
}

func init() {
	rootCmd.AddCommand(rollbackCmd)
}

func runRollback(cmd *cobra.Command, args []string) error {
	fmt.Println(titleStyle.Render("== PnP Rollback =="))
	fmt.Println()

	projCfg, err := config.LoadProjectConfig()
	if err != nil {
		return fmt.Errorf("no .cluster.yaml found: %w", err)
	}

	globalCfg, err := config.LoadGlobalConfig()
	if err != nil {
		return fmt.Errorf("loading global config: %w", err)
	}

	repo := gitops.NewRepo(globalCfg.GitopsRepo)

	if err := repo.Pull(); err != nil {
		fmt.Printf("  Warning: git pull failed: %v\n", err)
	}

	appPath := repo.AppPath(projCfg.Name, projCfg.Environment, projCfg.Scope)

	// Get recent commits for this app
	commits, err := repo.GitLog(appPath, 10)
	if err != nil {
		return fmt.Errorf("getting git history: %w", err)
	}

	if len(commits) == 0 {
		fmt.Println(warnStyle.Render("No deployment history found for this application."))
		return nil
	}

	fmt.Println("  Recent deployment commits:")
	fmt.Println()

	options := make([]huh.Option[string], len(commits))
	for i, c := range commits {
		label := fmt.Sprintf("%s  %s", dimStyle.Render(c.Hash[:8]), c.Message)
		options[i] = huh.NewOption(label, c.Hash)
	}

	var selectedHash string
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select commit to revert").
				Options(options...).
				Value(&selectedHash),
		),
	).Run()
	if err != nil {
		return err
	}

	// Confirm
	var confirm bool
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Revert this commit? This will create a new commit in the gitops repo.").
				Value(&confirm),
		),
	).Run()
	if err != nil {
		return err
	}

	if !confirm {
		fmt.Println(dimStyle.Render("  Rollback cancelled."))
		return nil
	}

	// Perform revert
	if err := repo.Revert(selectedHash); err != nil {
		return fmt.Errorf("reverting commit: %w", err)
	}

	if err := repo.Push(); err != nil {
		return fmt.Errorf("pushing revert: %w", err)
	}

	fmt.Println(successStyle.Render("Rollback pushed! ArgoCD will sync the reverted state shortly."))
	return nil
}

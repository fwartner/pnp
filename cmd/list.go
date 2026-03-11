package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/fwartner/pnp/internal/config"
	"github.com/fwartner/pnp/internal/gitops"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all deployed applications",
	Long:  "Shows all applications in the gitops repository with their scope, environment, domain, and type.",
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	fmt.Println(titleStyle.Render("== Deployed Applications =="))
	fmt.Println()

	globalCfg, err := config.LoadGlobalConfig()
	if err != nil {
		return fmt.Errorf("loading global config: %w", err)
	}
	if globalCfg.GitopsRepo == "" {
		return fmt.Errorf("gitopsRepo is not set in ~/.pnp.yaml")
	}

	repo := gitops.NewRepo(globalCfg.GitopsRepo)
	apps, err := repo.ListApps()
	if err != nil {
		return fmt.Errorf("listing apps: %w", err)
	}

	if len(apps) == 0 {
		fmt.Println(dimStyle.Render("  No applications found in gitops repo."))
		return nil
	}

	// Table header
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	header := fmt.Sprintf("  %-25s %-12s %-12s %-40s %s",
		headerStyle.Render("NAME"),
		headerStyle.Render("SCOPE"),
		headerStyle.Render("ENV"),
		headerStyle.Render("DOMAIN"),
		headerStyle.Render("TYPE"),
	)
	fmt.Println(header)
	fmt.Println("  " + strings.Repeat("─", 100))

	for _, app := range apps {
		env := app.Environment
		if env == "" {
			env = "-"
		}
		domain := app.Domain
		if domain == "" {
			domain = "-"
		}
		appType := app.Type
		if appType == "" {
			appType = "-"
		}
		fmt.Printf("  %-25s %-12s %-12s %-40s %s\n",
			app.Name, app.Scope, env, domain, appType)
	}

	fmt.Printf("\n  %s\n", dimStyle.Render(fmt.Sprintf("%d application(s) found", len(apps))))
	return nil
}

package cmd

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/fwartner/pnp/internal/config"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the global PnP configuration",
	Long:  "Creates or updates ~/.pnp.yaml interactively, setting up gitops repo, Infisical credentials, and project defaults.",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	fmt.Println(titleStyle.Render("== PnP Init =="))

	// Load existing config (or defaults)
	cfg, err := config.LoadGlobalConfig()
	if err != nil {
		fmt.Println(errorStyle.Render("Failed to load existing config: " + err.Error()))
		return err
	}

	// Group 1: GitOps settings
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("GitOps Repo Path").
				Description("Local path to your gitops repository clone").
				Value(&cfg.GitopsRepo),
			huh.NewInput().
				Title("GitOps Repo Remote URL").
				Description("Remote URL of the gitops repository").
				Value(&cfg.GitopsRemote),
		),
	).Run()
	if err != nil {
		fmt.Println(errorStyle.Render("Form cancelled: " + err.Error()))
		return err
	}

	// Group 2: Infisical settings
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Infisical Vault URL").
				Description("URL of your Infisical instance").
				Value(&cfg.Infisical.Host),
			huh.NewInput().
				Title("Infisical Machine Identity Token").
				Description("Token for authenticating with Infisical").
				Value(&cfg.Infisical.Token),
		),
	).Run()
	if err != nil {
		fmt.Println(errorStyle.Render("Form cancelled: " + err.Error()))
		return err
	}

	// Group 3: Project defaults
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Default Domain").
				Description("Default domain for deployed applications").
				Value(&cfg.Defaults.Domain),
			huh.NewInput().
				Title("Default Image Registry").
				Description("Default container image registry (e.g. ghcr.io)").
				Value(&cfg.Defaults.ImageRegistry),
			huh.NewInput().
				Title("Default GitHub Org/User").
				Description("Default GitHub organization or username").
				Value(&cfg.Defaults.GithubOrg),
		),
	).Run()
	if err != nil {
		fmt.Println(errorStyle.Render("Form cancelled: " + err.Error()))
		return err
	}

	// Save config
	if err := config.SaveGlobalConfig(cfg); err != nil {
		fmt.Println(errorStyle.Render("Failed to save config: " + err.Error()))
		return err
	}

	path, err := config.GlobalConfigPath()
	if err != nil {
		fmt.Println(errorStyle.Render("Failed to determine config path: " + err.Error()))
		return err
	}
	fmt.Println(successStyle.Render(fmt.Sprintf("Configuration saved to %s", path)))
	return nil
}

package cmd

import (
	"fmt"

	"github.com/fwartner/pnp/internal/doctor"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check prerequisites and system health",
	Long:  "Validates that all required tools are installed and configured correctly for deploying with pnp.",
	RunE:  runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	fmt.Println(titleStyle.Render("== PnP Doctor =="))
	fmt.Println()

	results := doctor.RunAll(true)

	allOK := true
	for _, r := range results {
		var icon, style string
		if r.OK {
			icon = successStyle.Render("✓")
			style = r.Message
		} else if r.Critical {
			icon = errorStyle.Render("✗")
			style = errorStyle.Render(r.Message)
			allOK = false
		} else {
			icon = warnStyle.Render("!")
			style = warnStyle.Render(r.Message)
		}
		fmt.Printf("  %s  %-15s %s\n", icon, r.Name, style)
	}

	fmt.Println()
	if allOK {
		fmt.Println(successStyle.Render("All checks passed! You're ready to deploy."))
	} else {
		fmt.Println(errorStyle.Render("Some checks failed. Fix the issues above before deploying."))
	}

	return nil
}

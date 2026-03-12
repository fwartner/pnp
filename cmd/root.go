package cmd

import (
	"fmt"
	"os"

	"github.com/fwartner/pnp/internal/plugin"
	"github.com/spf13/cobra"

	// Register built-in project types via init().
	_ "github.com/fwartner/pnp/internal/types"
)

var rootCmd = &cobra.Command{
	Use:   "pnp",
	Short: "Pixel & Process deployment manager",
	Long:  "A CLI tool to manage Kubernetes deployments for Pixel & Process projects.",
}

func init() {
	// Load external plugins from ~/.pnp/plugins/
	plugin.LoadAll()
	for _, cmd := range plugin.Commands() {
		rootCmd.AddCommand(cmd)
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

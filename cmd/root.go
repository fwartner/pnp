package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "pnp",
	Short: "Pixel & Process deployment manager",
	Long:  "A CLI tool to manage Kubernetes deployments for Pixel & Process projects.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

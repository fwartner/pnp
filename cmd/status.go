package cmd

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/fwartner/pnp/internal/config"
	"github.com/fwartner/pnp/internal/kube"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the ArgoCD sync and health status of the current project",
	Long:  "Reads .cluster.yaml, looks up the ArgoCD application, and displays sync/health status with error explanations.",
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

// argoAppStatus holds the relevant fields from the ArgoCD application JSON.
type argoAppStatus struct {
	Status struct {
		Sync struct {
			Status string `json:"status"`
		} `json:"sync"`
		Health struct {
			Status string `json:"status"`
		} `json:"health"`
	} `json:"status"`
}

func runStatus(cmd *cobra.Command, args []string) error {
	fmt.Println(titleStyle.Render("== PnP Status =="))

	projCfg, err := config.LoadProjectConfig()
	if err != nil {
		fmt.Println(errorStyle.Render("No .cluster.yaml found in current directory."))
		return fmt.Errorf("loading project config: %w", err)
	}

	appName := "helm-" + projCfg.Name
	namespace := namespaceFromConfig(projCfg)

	// Try argocd CLI first, fall back to kubectl
	jsonBytes, err := exec.Command("argocd", "app", "get", appName, "-o", "json").Output()
	if err != nil {
		jsonBytes, err = exec.Command("kubectl", "get", "application", appName, "-n", "argocd", "-o", "json").Output()
		if err != nil {
			fmt.Println(errorStyle.Render("Could not retrieve ArgoCD application status."))
			return fmt.Errorf("failed to get application %s: %w", appName, err)
		}
	}

	var app argoAppStatus
	if err := json.Unmarshal(jsonBytes, &app); err != nil {
		return fmt.Errorf("parsing application JSON: %w", err)
	}

	syncStatus := app.Status.Sync.Status
	healthStatus := app.Status.Health.Status

	// Color helpers
	greenStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	redStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))

	colorize := func(value string) string {
		switch value {
		case "Synced", "Healthy":
			return greenStyle.Render(value)
		case "OutOfSync", "Degraded", "Missing":
			return redStyle.Render(value)
		default:
			return value
		}
	}

	fmt.Println()
	fmt.Printf("  App:         %s\n", appName)
	fmt.Printf("  Environment: %s\n", projCfg.Environment)
	fmt.Printf("  Domain:      %s\n", projCfg.Domain)
	fmt.Printf("  Sync:        %s\n", colorize(syncStatus))
	fmt.Printf("  Health:      %s\n", colorize(healthStatus))

	// Show error explanations for unhealthy states
	unhealthy := healthStatus != "Healthy" || syncStatus == "OutOfSync"
	if unhealthy {
		fmt.Println()

		// Explain sync status
		if explanation := kube.Explain(syncStatus); explanation != "" {
			fmt.Printf("  %s %s\n", warnStyle.Render("->"), explanation)
		}

		// Explain health status
		if explanation := kube.Explain(healthStatus); explanation != "" {
			fmt.Printf("  %s %s\n", warnStyle.Render("->"), explanation)
		}

		// Check pod-level errors if kubectl is available
		if kube.Available() {
			selector := "app.kubernetes.io/name=" + projCfg.Name
			statuses, err := kube.GetPodStatuses(namespace, selector)
			if err == nil {
				for _, status := range statuses {
					if explanation := kube.Explain(status); explanation != "" {
						fmt.Printf("  %s %s: %s\n", errorStyle.Render("!"), status, explanation)
					}
				}
			}

			// Show pods table
			pods, err := kube.GetPods(namespace, selector)
			if err == nil && len(pods) > 0 {
				fmt.Println()
				fmt.Println(titleStyle.Render("  Pods:"))
				fmt.Printf("  %-45s %-12s %s\n",
					dimStyle.Render("NAME"),
					dimStyle.Render("STATUS"),
					dimStyle.Render("READY"))
				fmt.Println("  " + strings.Repeat("─", 65))
				for _, pod := range pods {
					status := pod.Status
					if status == "Running" {
						status = greenStyle.Render(status)
					} else {
						status = redStyle.Render(status)
					}
					fmt.Printf("  %-45s %-12s %s\n", pod.Name, status, pod.Ready)
				}
			}
		}
	}

	fmt.Println()
	return nil
}

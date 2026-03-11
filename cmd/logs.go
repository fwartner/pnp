package cmd

import (
	"fmt"
	"strings"

	"github.com/fwartner/pnp/internal/config"
	"github.com/fwartner/pnp/internal/kube"
	"github.com/spf13/cobra"
)

var (
	flagFollow bool
	flagTail   int
)

var logsCmd = &cobra.Command{
	Use:   "logs [app-name]",
	Short: "Stream application logs",
	Long:  "Streams logs for the current project or a named application. Requires kubectl access to the cluster.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runLogs,
}

func init() {
	logsCmd.Flags().BoolVarP(&flagFollow, "follow", "f", true, "Follow log output")
	logsCmd.Flags().IntVar(&flagTail, "tail", 100, "Number of recent lines to show")
	rootCmd.AddCommand(logsCmd)
}

func runLogs(cmd *cobra.Command, args []string) error {
	if !kube.Available() {
		return fmt.Errorf("kubectl is not installed — required for log streaming")
	}

	var appName, namespace string

	if len(args) > 0 {
		appName = args[0]
		namespace = appName
	} else {
		projCfg, err := config.LoadProjectConfig()
		if err != nil {
			return fmt.Errorf("no .cluster.yaml found and no app name provided")
		}
		appName = projCfg.Name
		namespace = namespaceFromConfig(projCfg)
	}

	selector := "app.kubernetes.io/name=" + appName

	fmt.Println(titleStyle.Render(fmt.Sprintf("== Logs: %s ==", appName)))
	fmt.Println()

	// Find pods
	pods, err := kube.GetPods(namespace, selector)
	if err != nil {
		return fmt.Errorf("finding pods: %w", err)
	}

	if len(pods) == 0 {
		return fmt.Errorf("no pods found for %s in namespace %s", appName, namespace)
	}

	// Stream logs from first running pod
	for _, pod := range pods {
		if strings.EqualFold(pod.Status, "Running") {
			fmt.Printf("  Streaming from pod: %s\n\n", dimStyle.Render(pod.Name))
			return kube.StreamLogs(namespace, pod.Name, flagFollow, flagTail)
		}
	}

	// Fallback to first pod
	fmt.Printf("  Streaming from pod: %s (%s)\n\n", dimStyle.Render(pods[0].Name), pods[0].Status)
	return kube.StreamLogs(namespace, pods[0].Name, flagFollow, flagTail)
}

package kube

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// AppStatus holds the ArgoCD application status.
type AppStatus struct {
	Name         string
	SyncStatus   string
	HealthStatus string
	Message      string
}

// Pod represents a Kubernetes pod.
type Pod struct {
	Name   string
	Status string
	Ready  string
}

// Available checks if kubectl is in PATH.
func Available() bool {
	_, err := exec.LookPath("kubectl")
	return err == nil
}

// GetAppStatus queries an ArgoCD application's sync and health status.
func GetAppStatus(appName string) (AppStatus, error) {
	jsonBytes, err := exec.Command("kubectl", "get", "application", appName,
		"-n", "argocd", "-o", "json").Output()
	if err != nil {
		return AppStatus{}, fmt.Errorf("failed to get application %s: %w", appName, err)
	}

	var raw struct {
		Status struct {
			Sync struct {
				Status string `json:"status"`
			} `json:"sync"`
			Health struct {
				Status  string `json:"status"`
				Message string `json:"message"`
			} `json:"health"`
			Conditions []struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"conditions"`
		} `json:"status"`
	}
	if err := json.Unmarshal(jsonBytes, &raw); err != nil {
		return AppStatus{}, fmt.Errorf("parsing application JSON: %w", err)
	}

	msg := raw.Status.Health.Message
	if msg == "" && len(raw.Status.Conditions) > 0 {
		msg = raw.Status.Conditions[0].Message
	}

	return AppStatus{
		Name:         appName,
		SyncStatus:   raw.Status.Sync.Status,
		HealthStatus: raw.Status.Health.Status,
		Message:      msg,
	}, nil
}

// GetPods lists pods matching a label selector in a namespace.
func GetPods(namespace, selector string) ([]Pod, error) {
	out, err := exec.Command("kubectl", "get", "pods",
		"-n", namespace,
		"-l", selector,
		"-o", "jsonpath={range .items[*]}{.metadata.name}\t{.status.phase}\t{.status.containerStatuses[0].ready}{\"\\n\"}{end}",
	).Output()
	if err != nil {
		return nil, fmt.Errorf("listing pods: %w", err)
	}

	var pods []Pod
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		p := Pod{Name: parts[0]}
		if len(parts) > 1 {
			p.Status = parts[1]
		}
		if len(parts) > 2 {
			p.Ready = parts[2]
		}
		pods = append(pods, p)
	}
	return pods, nil
}

// StreamLogs streams logs for a pod to stdout.
func StreamLogs(namespace, pod string, follow bool, tail int) error {
	args := []string{"logs", pod, "-n", namespace, fmt.Sprintf("--tail=%d", tail)}
	if follow {
		args = append(args, "-f")
	}

	cmd := exec.Command("kubectl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// GetPodStatuses returns the container status reasons for pods in a namespace
// matching the given selector (e.g., "ImagePullBackOff", "CrashLoopBackOff").
func GetPodStatuses(namespace, selector string) ([]string, error) {
	out, err := exec.Command("kubectl", "get", "pods",
		"-n", namespace,
		"-l", selector,
		"-o", "jsonpath={range .items[*].status.containerStatuses[*]}{.state.waiting.reason}{\"\\n\"}{.state.terminated.reason}{\"\\n\"}{end}",
	).Output()
	if err != nil {
		return nil, err
	}

	var statuses []string
	seen := make(map[string]bool)
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !seen[line] {
			statuses = append(statuses, line)
			seen[line] = true
		}
	}
	return statuses, nil
}

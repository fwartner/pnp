package kube

// Explanations maps common Kubernetes error states to human-friendly descriptions
// with suggested fix commands.
var Explanations = map[string]string{
	"ImagePullBackOff":  "Container image not found. Check that the GHCR image exists and credentials are set.",
	"ErrImagePull":      "Container image not found. Check that the GHCR image exists and credentials are set.",
	"CrashLoopBackOff":  "App crashes on startup. Run 'pnp logs' to investigate.",
	"OOMKilled":         "Out of memory. Run 'pnp env set resources.memory=1Gi' then 'pnp sync'.",
	"Pending":           "Waiting for cluster resources. The cluster may be at capacity.",
	"OutOfSync":         "Changes pending in gitops repo. ArgoCD will sync shortly.",
	"CreateContainerConfigError": "Container configuration error. Check environment variables and secrets.",
	"RunContainerError": "Failed to start container. Check the Docker image entrypoint.",
}

// Explain returns a human-friendly explanation for a Kubernetes status.
// Returns empty string if no explanation is available.
func Explain(status string) string {
	if explanation, ok := Explanations[status]; ok {
		return explanation
	}
	return ""
}

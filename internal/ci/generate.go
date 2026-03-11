package ci

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

const workflowTemplate = `name: Build & Deploy

on:
  push:
    branches: [main]

permissions:
  contents: read
  packages: write

env:
  IMAGE: {{ .Image }}

jobs:
  build:
    runs-on: ubuntu-latest
    outputs:
      image-tag: ${{ "{{" }} steps.meta.outputs.version {{ "}}" }}

    steps:
      - uses: actions/checkout@v4

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Log in to GHCR
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ "{{" }} github.actor {{ "}}" }}
          password: ${{ "{{" }} secrets.GITHUB_TOKEN {{ "}}" }}

      - name: Docker meta
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ${{ "{{" }} env.IMAGE {{ "}}" }}
          tags: |
            type=sha
            type=raw,value=latest,enable=${{ "{{" }} github.ref == format('refs/heads/{0}', 'main') {{ "}}" }}

      - name: Build and push
        uses: docker/build-push-action@v6
        with:
          context: .
          push: true
          tags: ${{ "{{" }} steps.meta.outputs.tags {{ "}}" }}
          labels: ${{ "{{" }} steps.meta.outputs.labels {{ "}}" }}
          secrets: |
            composer_auth={"github-oauth":{"github.com":"${{ "{{" }} secrets.GITHUB_TOKEN {{ "}}" }}"}}
          cache-from: type=gha
          cache-to: type=gha,mode=max

  deploy:
    needs: build
    runs-on: ubuntu-latest

    steps:
      - name: Checkout gitops repo
        uses: actions/checkout@v4
        with:
          repository: {{ .GitopsRepo }}
          token: ${{ "{{" }} secrets.GITOPS_TOKEN || secrets.GITHUB_TOKEN {{ "}}" }}
          path: gitops

      - name: Update image tag
        run: |
          cd gitops
          TAG="${{ "{{" }} needs.build.outputs.image-tag {{ "}}" }}"
          VALUES_FILE="apps/{{ .AppName }}/values.yaml"
          if [ -f "$VALUES_FILE" ]; then
            sed -i "s|tag:.*|tag: ${TAG}|" "$VALUES_FILE"
            git config user.name "github-actions[bot]"
            git config user.email "github-actions[bot]@users.noreply.github.com"
            git add "$VALUES_FILE"
            git diff --cached --quiet || git commit -m "deploy({{ .AppName }}): update image tag to ${TAG}"
            git push
          else
            echo "::warning::Values file $VALUES_FILE not found"
          fi
`

type workflowData struct {
	Image      string
	GitopsRepo string
	AppName    string
}

// GenerateWorkflow creates a GitHub Actions deploy workflow in the given project directory.
func GenerateWorkflow(projectType string, image string, gitopsRemote string, appName string, projectDir string) error {
	// Extract org/repo from gitops remote URL
	gitopsRepo := extractGitHubRepo(gitopsRemote)

	data := workflowData{
		Image:      image,
		GitopsRepo: gitopsRepo,
		AppName:    appName,
	}

	// Validate project type.
	switch projectType {
	case "laravel-web", "laravel-api", "nextjs-fullstack", "nextjs-static", "strapi":
		// ok
	default:
		return fmt.Errorf("unsupported project type: %s", projectType)
	}

	tmpl, err := template.New("workflow").Parse(workflowTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse workflow template: %w", err)
	}

	outDir := filepath.Join(projectDir, ".github", "workflows")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("failed to create workflows directory: %w", err)
	}

	outPath := filepath.Join(outDir, "deploy.yml")
	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create workflow file: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("failed to execute workflow template: %w", err)
	}

	return nil
}

// extractGitHubRepo extracts "org/repo" from various GitHub URL formats.
func extractGitHubRepo(remote string) string {
	// Handle SSH format: git@github.com:org/repo.git
	if len(remote) > 15 && remote[:15] == "git@github.com:" {
		repo := remote[15:]
		if len(repo) > 4 && repo[len(repo)-4:] == ".git" {
			repo = repo[:len(repo)-4]
		}
		return repo
	}
	// Handle HTTPS format: https://github.com/org/repo.git
	prefix := "https://github.com/"
	if len(remote) > len(prefix) && remote[:len(prefix)] == prefix {
		repo := remote[len(prefix):]
		if len(repo) > 4 && repo[len(repo)-4:] == ".git" {
			repo = repo[:len(repo)-4]
		}
		return repo
	}
	return remote
}

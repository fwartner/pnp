package ci

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

const workflowTemplate = `name: Build & Push

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
          build-args: |
            COMPOSER_AUTH={"github-oauth":{"github.com":"${{ "{{" }} secrets.GITHUB_TOKEN {{ "}}" }}"}}
          cache-from: type=gha
          cache-to: type=gha,mode=max
`

type workflowData struct {
	Image string
}

// GenerateWorkflow creates a GitHub Actions deploy workflow in the given project directory.
func GenerateWorkflow(projectType string, image string, projectDir string) error {
	data := workflowData{
		Image: image,
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

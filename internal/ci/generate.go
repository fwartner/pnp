package ci

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

const workflowTemplate = `name: Deploy

on:
  push:
    branches: [main]

env:
  IMAGE: {{ .Image }}

jobs:
  build-and-push:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4
{{ if .IsLaravel }}
      - name: Setup PHP
        uses: shivammathur/setup-php@v2
        with:
          php-version: "8.3"

      - name: Install Composer dependencies
        run: composer install --no-interaction --prefer-dist --optimize-autoloader

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: "20"

      - name: Install npm dependencies
        run: npm ci

      - name: Build assets
        run: npm run build
{{ end }}{{ if .IsNode }}
      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: "20"

      - name: Install npm dependencies
        run: npm ci

      - name: Build
        run: npm run build
{{ end }}
      - name: Log in to GHCR
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ "{{" }} github.actor {{ "}}" }}
          password: ${{ "{{" }} secrets.GITHUB_TOKEN {{ "}}" }}

      - name: Build and push Docker image
        uses: docker/build-push-action@v5
        with:
          context: .
          push: true
          tags: |
            ${{ "{{" }} env.IMAGE {{ "}}" }}:${{ "{{" }} github.sha {{ "}}" }}
            ${{ "{{" }} env.IMAGE {{ "}}" }}:latest
`

type workflowData struct {
	Image     string
	IsLaravel bool
	IsNode    bool
}

// GenerateWorkflow creates a GitHub Actions deploy workflow in the given project directory.
func GenerateWorkflow(projectType string, image string, projectDir string) error {
	data := workflowData{
		Image: image,
	}

	switch projectType {
	case "laravel-web", "laravel-api":
		data.IsLaravel = true
	case "nextjs-fullstack", "nextjs-static", "strapi":
		data.IsNode = true
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

package ci

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

const previewWorkflowTemplate = `name: Preview Environments

on:
  pull_request:
    types: [opened, synchronize, closed]

permissions:
  contents: read
  packages: write
  pull-requests: write

env:
  IMAGE: {{ .Image }}

jobs:
  preview-deploy:
    if: github.event.action != 'closed'
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

      - name: Build and push preview image
        uses: docker/build-push-action@v6
        with:
          context: .
          push: true
          tags: ${{ "{{" }} env.IMAGE {{ "}}" }}:pr-${{ "{{" }} github.event.number {{ "}}" }}
          secrets: |
            composer_auth={"github-oauth":{"github.com":"${{ "{{" }} secrets.GITHUB_TOKEN {{ "}}" }}"}}
          cache-from: type=gha
          cache-to: type=gha,mode=max

      - name: Deploy preview to gitops
        env:
          GITOPS_TOKEN: ${{ "{{" }} secrets.GITOPS_TOKEN {{ "}}" }}
        run: |
          git clone https://x-access-token:${GITOPS_TOKEN}@github.com/{{ .GitopsRepo }}.git gitops
          cd gitops

          APP_DIR="apps/previews/{{ .AppName }}-pr-${{ "{{" }} github.event.number {{ "}}" }}"
          mkdir -p "${APP_DIR}"

          # Copy from main app as template and override values
          MAIN_DIR=""
          for d in apps/customer/{{ .AppName }} apps/agency/{{ .AppName }} apps/previews/{{ .AppName }}; do
            if [ -d "$d" ]; then
              MAIN_DIR="$d"
              break
            fi
          done

          if [ -n "$MAIN_DIR" ]; then
            cp -r "${MAIN_DIR}/"* "${APP_DIR}/"
            # Update image tag and domain for preview
            sed -i "s|tag:.*|tag: pr-${{ "{{" }} github.event.number {{ "}}" }}|" "${APP_DIR}/values.yaml"
            sed -i "s|subdomain:.*|subdomain: pr-${{ "{{" }} github.event.number {{ "}}" }}.preview|" "${APP_DIR}/values.yaml"
          fi

          git config user.name "github-actions[bot]"
          git config user.email "github-actions[bot]@users.noreply.github.com"
          git add -A
          git diff --cached --quiet || git commit -m "preview({{ .AppName }}): deploy PR #${{ "{{" }} github.event.number {{ "}}" }}"
          git diff --cached --quiet || git push

      - name: Comment PR with preview URL
        uses: actions/github-script@v7
        with:
          script: |
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: '🚀 Preview deployed: https://pr-${{ "{{" }} github.event.number {{ "}}" }}.preview.{{ .BaseDomain }}'
            })

  preview-cleanup:
    if: github.event.action == 'closed'
    runs-on: ubuntu-latest

    steps:
      - name: Remove preview from gitops
        env:
          GITOPS_TOKEN: ${{ "{{" }} secrets.GITOPS_TOKEN {{ "}}" }}
        run: |
          git clone https://x-access-token:${GITOPS_TOKEN}@github.com/{{ .GitopsRepo }}.git gitops
          cd gitops

          APP_DIR="apps/previews/{{ .AppName }}-pr-${{ "{{" }} github.event.number {{ "}}" }}"
          if [ -d "$APP_DIR" ]; then
            rm -rf "$APP_DIR"
            git config user.name "github-actions[bot]"
            git config user.email "github-actions[bot]@users.noreply.github.com"
            git add -A
            git commit -m "preview({{ .AppName }}): cleanup PR #${{ "{{" }} github.event.number {{ "}}" }}"
            git push
          fi
`

type previewData struct {
	Image      string
	GitopsRepo string
	AppName    string
	BaseDomain string
}

// GeneratePreviewWorkflow creates a GitHub Actions workflow for PR-based
// preview deployments.
func GeneratePreviewWorkflow(image, gitopsRemote, appName, baseDomain, projectDir string) error {
	gitopsRepo := extractGitHubRepo(gitopsRemote)

	data := previewData{
		Image:      image,
		GitopsRepo: gitopsRepo,
		AppName:    appName,
		BaseDomain: baseDomain,
	}

	tmpl, err := template.New("preview").Parse(previewWorkflowTemplate)
	if err != nil {
		return fmt.Errorf("parsing preview workflow template: %w", err)
	}

	outDir := filepath.Join(projectDir, ".github", "workflows")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("creating workflows directory: %w", err)
	}

	outPath := filepath.Join(outDir, "preview.yml")
	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("creating preview workflow file: %w", err)
	}
	defer f.Close()

	return tmpl.Execute(f, data)
}

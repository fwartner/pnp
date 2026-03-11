# PnP CLI — Design Document

**Date:** 2026-03-11
**Status:** Approved

## Overview

`pnp` is a Go CLI tool that automates Kubernetes deployments for Pixel & Process. Run `pnp deploy` in any project folder — it detects the project type, walks through an interactive wizard, and generates all Kubernetes resources in the gitops repo. Supports create, update, and destroy workflows.

## Reference

The gitops repo at `pixelandprocess-gitops` contains:
- `_templates/` — Helm chart templates for `laravel-web`, `laravel-api`, `nextjs-fullstack`, `nextjs-static`
- `charts/` — Reusable Helm charts for `laravel`, `nextjs`, `strapi`
- `apps/` — ArgoCD Application definitions
- `apps/previews/` — Dynamic preview deployments
- `scripts/create-preview.sh` — Existing shell-based automation (this CLI replaces it)

## Commands

```
pnp deploy          # Create or detect existing → wizard → deploy
pnp update          # Update existing deployment from .cluster.yaml changes
pnp destroy         # Remove deployment from gitops repo + cleanup
pnp status          # Show deployment status (ArgoCD sync state)
```

## Project Detection

On `pnp deploy`, the CLI:
1. Checks for `.cluster.yaml` → if found, pre-fills all settings
2. If not found, detects project type from files:
   - `composer.json` + `artisan` → Laravel (`laravel-web` if queue/scheduler detected, else `laravel-api`)
   - `package.json` with `next` dependency → Next.js (`nextjs-fullstack` if DB deps found, else `nextjs-static`)
   - `package.json` with `strapi` dependency → Strapi
3. Detects git remote → infers `ghcr.io/<org>/<repo>` image path
4. Runs interactive wizard for remaining/undetected values

## Wizard Flow

```
1. Project name         (auto: from folder name or git repo name)
2. Project type         (auto-detected, confirm or override)
3. Environment          (preview / staging / production)
4. Domain               (default based on environment pattern)
5. Image                (auto from git remote, allow override)
6. Database             (postgres size, db name — defaults provided)
7. Redis                (yes/no, defaults per project type)
8. Infisical project    (slug + env + secrets path)
9. Resources            (CPU/memory — sensible defaults per type)
10. Confirm & deploy
```

## Environment Domain Patterns

| Environment | Pattern | Example |
|---|---|---|
| Preview | `<name>.preview.pixelandprocess.de` | `acme.preview.pixelandprocess.de` |
| Staging | `<name>.staging.pixelandprocess.de` | `acme.staging.pixelandprocess.de` |
| Production | Custom domain (asked in wizard) | `acme-corp.de` |

## `.cluster.yaml` Schema

```yaml
name: acme-corp
type: laravel-web           # laravel-web | laravel-api | nextjs-fullstack | nextjs-static | strapi
environment: preview        # preview | staging | production
domain: acme-corp.preview.pixelandprocess.de
image: ghcr.io/fwartner/acme-corp
database:
  enabled: true
  size: 5Gi
  name: acme
redis:
  enabled: true
infisical:
  projectSlug: customer-apps-f-jq3
  envSlug: prod
  secretsPath: /acme-corp/db
resources:
  cpu: 100m
  memory: 256Mi
ci:
  enabled: false            # whether GitHub Actions workflow was generated
```

Saved to the project directory after first deploy. Re-running `pnp deploy` detects it and offers to update.

## Gitops Integration

- Clones or pulls the `pixelandprocess-gitops` repo (path configurable via `~/.pnp.yaml` or `PNP_GITOPS_REPO` env var)
- Generates files based on project type:
  - Preview/staging → `apps/previews/<name>/`
  - Production → `apps/<name>/`
- Renders templates using Go's `text/template` from the `_templates/` directory in the gitops repo
- **Default**: commits & pushes directly to `main`
- **`--pr` flag**: creates a feature branch + PR via GitHub CLI (`gh`)

## Generated Resources per Type

### Laravel Web
- ArgoCD Application (pointing to `charts/laravel`)
- CNPG Cluster (PostgreSQL)
- InfisicalSecret (DB credentials)
- Redis Deployment + Service
- Values: web deployment, queue worker, scheduler, PVC, ingress

### Laravel API
- ArgoCD Application (pointing to `charts/laravel`)
- CNPG Cluster (PostgreSQL)
- InfisicalSecret (DB credentials)
- Redis Deployment + Service
- Values: web deployment only, no queue/scheduler/PVC

### Next.js Fullstack
- ArgoCD Application (pointing to `charts/nextjs`)
- CNPG Cluster (PostgreSQL)
- InfisicalSecret (DB credentials)
- Values: deployment, ingress, DB connection env vars

### Next.js Static
- ArgoCD Application (pointing to `charts/nextjs`)
- Values: deployment, ingress (no database)

### Strapi
- ArgoCD Application (pointing to `charts/strapi`)
- CNPG Cluster (PostgreSQL)
- InfisicalSecret (DB credentials)
- Values: deployment, ingress, PVC for uploads

## Infisical Secrets Management

- Generates DB password, APP_KEY (Laravel), and other secrets automatically
- Pushes secrets to Infisical vault via REST API using machine identity authentication
- Generates `InfisicalSecret` CRD manifests in the gitops repo
- `--skip-secrets` flag: only generates CRD manifests, does not push to Infisical

## GitHub Actions CI (Optional)

`--with-ci` flag generates `.github/workflows/deploy.yml` in the current project repo:

- **Laravel**: Install PHP deps → build assets → Docker build → push to GHCR
- **Next.js**: Install Node deps → build → Docker build → push to GHCR
- **Strapi**: Install Node deps → build → Docker build → push to GHCR

Triggers on push to `main` branch.

## Global Configuration (`~/.pnp.yaml`)

```yaml
gitopsRepo: /Users/fwartner/Projects/Development/pixelandprocess-gitops
gitopsRemote: https://github.com/fwartner/pixelandprocess-gitops.git
infisical:
  host: https://vault.intern.pixelandprocess.de
  token: <machine-identity-token>
defaults:
  domain: pixelandprocess.de
  imageRegistry: ghcr.io
  githubOrg: fwartner
```

Created on first run if not present (via `pnp init` or auto-prompted).

## Tech Stack

- **Language**: Go
- **CLI framework**: [Cobra](https://github.com/spf13/cobra) — commands, flags, help text
- **TUI/Wizard**: [Bubbletea](https://github.com/charmbracelet/bubbletea) + [Huh](https://github.com/charmbracelet/huh) — interactive forms
- **Template rendering**: Go `text/template` — manifest generation
- **YAML**: `gopkg.in/yaml.v3` — config file parsing
- **Git operations**: `go-git` or shell exec — clone, commit, push
- **Distribution**: GitHub Releases + Homebrew tap (`fwartner/tap/pnp`)

## Architecture

```
cmd/
  root.go           # Root command, global flags
  deploy.go         # Deploy command
  update.go         # Update command
  destroy.go        # Destroy command
  status.go         # Status command
internal/
  config/           # Global config (~/.pnp.yaml) and project config (.cluster.yaml)
  detect/           # Project type detection logic
  wizard/           # Interactive wizard (Bubbletea/Huh forms)
  gitops/           # Gitops repo operations (clone, generate, commit, push, PR)
  templates/        # Template rendering engine
  infisical/        # Infisical API client (create secrets, manage projects)
  ci/               # GitHub Actions workflow generation
  registry/         # Image registry path inference from git remote
```

## Destroy Workflow

`pnp destroy` will:
1. Read `.cluster.yaml` to find the deployment name
2. Remove the app directory from the gitops repo
3. Optionally delete secrets from Infisical (`--clean-secrets`)
4. Commit & push (or PR with `--pr`)
5. ArgoCD prune policy handles Kubernetes resource cleanup

## Update Workflow

`pnp update` will:
1. Read `.cluster.yaml` for current config
2. Diff against what's in the gitops repo
3. Re-render templates with updated values
4. Commit & push changes

## Status Command

`pnp status` will:
1. Read `.cluster.yaml` for deployment name
2. Query ArgoCD API (or kubectl) for sync status
3. Display health, sync state, and recent events

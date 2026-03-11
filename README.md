# pnp

A CLI tool that automates Kubernetes deployments for [Pixel & Process](https://pixelandprocess.de) projects. Run `pnp deploy` inside any project folder and it handles everything — from creating the GitHub repository to deploying on the cluster.

## What it does

1. **Detects** your project type (Laravel, Next.js, Strapi) from source files
2. **Creates** a GitHub repository if one doesn't exist (via `gh` CLI)
3. **Walks you through** an interactive wizard to configure the deployment
4. **Generates** Kubernetes manifests (ArgoCD Application, CNPG PostgreSQL, Infisical secrets)
5. **Pushes** everything to the GitOps repo — ArgoCD takes it from there
6. **Creates secrets** in Infisical (database credentials, APP_KEY)
7. **Optionally generates** a GitHub Actions CI/CD pipeline

## Installation

### Homebrew (macOS/Linux)

```bash
brew install fwartner/tap/pnp
```

### Binary download

Download the latest release from [GitHub Releases](https://github.com/fwartner/pnp/releases):

```bash
# macOS (Apple Silicon)
curl -sL https://github.com/fwartner/pnp/releases/latest/download/pnp_0.1.0_darwin_arm64.tar.gz | tar xz
sudo mv pnp /usr/local/bin/

# macOS (Intel)
curl -sL https://github.com/fwartner/pnp/releases/latest/download/pnp_0.1.0_darwin_amd64.tar.gz | tar xz
sudo mv pnp /usr/local/bin/

# Linux (amd64)
curl -sL https://github.com/fwartner/pnp/releases/latest/download/pnp_0.1.0_linux_amd64.tar.gz | tar xz
sudo mv pnp /usr/local/bin/
```

### From source

```bash
go install github.com/fwartner/pnp@latest
```

## Prerequisites

- **[gh CLI](https://cli.github.com/)** — authenticated (`gh auth login`) — for GitHub repo creation and PRs
- **Git** — for interacting with the GitOps repository
- **kubectl** or **argocd CLI** — for `pnp status` (optional)

## Quick start

### 1. Initialize global config

```bash
pnp init
```

This creates `~/.pnp.yaml` with your settings:

```yaml
gitopsRepo: /path/to/your/gitops-repo      # local clone
gitopsRemote: https://github.com/org/gitops.git
infisical:
  host: https://vault.intern.pixelandprocess.de
  token: <machine-identity-token>
defaults:
  domain: pixelandprocess.de
  imageRegistry: ghcr.io
  githubOrg: your-org
```

### 2. Deploy a project

```bash
cd ~/projects/my-customer-app
pnp deploy
```

That's it. The CLI will:

- Detect the project type from source files
- Create a GitHub repo if needed (asks for confirmation)
- Run the interactive wizard for deployment settings
- Render Kubernetes manifests to the GitOps repo
- Push to `main` — ArgoCD auto-syncs the deployment
- Save `.cluster.yaml` in your project for future updates

### 3. Update after config changes

Edit `.cluster.yaml`, then:

```bash
pnp update
```

### 4. Tear it down

```bash
pnp destroy
```

## Commands

### `pnp deploy`

Create a new deployment or redeploy an existing one.

```bash
pnp deploy              # detect, wizard, deploy
pnp deploy --pr         # create a PR instead of pushing to main
pnp deploy --with-ci    # also generate .github/workflows/deploy.yml
pnp deploy --skip-secrets  # skip creating secrets in Infisical
```

**What happens:**

```
┌─────────────────┐    ┌───────────┐    ┌──────────┐    ┌────────┐    ┌─────────┐
│ Detect project  │───>│  Wizard   │───>│ Render   │───>│ GitOps │───>│ ArgoCD  │
│ type + git repo │    │ (confirm) │    │ manifests│    │  push  │    │  sync   │
└─────────────────┘    └───────────┘    └──────────┘    └────────┘    └─────────┘
```

### `pnp update`

Re-render manifests from `.cluster.yaml` and push changes.

```bash
pnp update          # push directly
pnp update --pr     # create a PR
```

### `pnp destroy`

Remove the deployment from the GitOps repo. ArgoCD's prune policy handles Kubernetes cleanup.

```bash
pnp destroy         # push directly
pnp destroy --pr    # create a PR for review
```

### `pnp status`

Show the ArgoCD sync and health status of the current project.

```bash
pnp status
```

```
App:         helm-my-customer-app
Environment: preview
Domain:      https://my-customer-app.preview.pixelandprocess.de
Sync:        Synced
Health:      Healthy
```

### `pnp init`

Interactive setup for global configuration (`~/.pnp.yaml`).

```bash
pnp init
```

### `pnp version`

```bash
pnp version
# pnp 0.1.0 (a1b2c3d)
```

## Supported project types

| Type | Detection | Database | Redis | Queue | Scheduler | Persistence |
|------|-----------|----------|-------|-------|-----------|-------------|
| **laravel-web** | `composer.json` + `artisan` + Jobs/scheduler | PostgreSQL (CNPG) | Yes | Yes | Yes | 1Gi |
| **laravel-api** | `composer.json` + `artisan` (no jobs) | PostgreSQL (CNPG) | Yes | No | No | No |
| **nextjs-fullstack** | `package.json` with `next` + DB deps | PostgreSQL (CNPG) | No | No | No | No |
| **nextjs-static** | `package.json` with `next` (no DB deps) | No | No | No | No | No |
| **strapi** | `package.json` with `@strapi/strapi` | PostgreSQL (CNPG) | No | No | No | 5Gi |

**DB dependency detection** (triggers `nextjs-fullstack` instead of `nextjs-static`):
`prisma`, `@prisma/client`, `pg`, `postgres`, `typeorm`, `drizzle-orm`, `knex`, `sequelize`

## Project config (`.cluster.yaml`)

Saved in your project directory after the first deploy. Editable for `pnp update`.

```yaml
name: my-customer-app
type: laravel-web
environment: preview
domain: my-customer-app.preview.pixelandprocess.de
image: ghcr.io/your-org/my-customer-app
database:
  enabled: true
  size: 5Gi
  name: app
redis:
  enabled: true
infisical:
  projectSlug: customer-apps-f-jq3
  envSlug: prod
  secretsPath: /my-customer-app/db
resources:
  cpu: 100m
  memory: 256Mi
ci:
  enabled: false
```

## Environment domains

| Environment | Domain pattern | Example |
|-------------|---------------|---------|
| Preview | `<name>.preview.pixelandprocess.de` | `acme.preview.pixelandprocess.de` |
| Staging | `<name>.staging.pixelandprocess.de` | `acme.staging.pixelandprocess.de` |
| Production | Custom (set in wizard) | `acme-corp.de` |

## Infrastructure stack

pnp generates manifests for an opinionated Kubernetes stack:

| Component | Purpose |
|-----------|---------|
| [ArgoCD](https://argo-cd.readthedocs.io/) | GitOps continuous delivery |
| [CloudNativePG](https://cloudnative-pg.io/) | PostgreSQL operator |
| [Infisical](https://infisical.com/) | Secret management |
| [cert-manager](https://cert-manager.io/) | TLS certificates (Let's Encrypt) |
| [external-dns](https://github.com/kubernetes-sigs/external-dns) | DNS automation (Cloudflare) |
| [Traefik](https://traefik.io/) | Ingress controller |

## Generated resources

When you run `pnp deploy`, the following resources are generated in the GitOps repo:

```
apps/previews/my-customer-app/
├── Chart.yaml                       # Helm chart metadata
├── values.yaml                      # Image, domain, DB config
└── templates/
    ├── application.yaml             # ArgoCD Application CRD
    ├── cnpg-cluster.yaml            # PostgreSQL cluster (if DB enabled)
    └── infisical-secrets.yaml       # Secret sync from Infisical vault
```

The ArgoCD Application points to a shared Helm chart (`charts/laravel`, `charts/nextjs`, or `charts/strapi`) in the same GitOps repo with project-specific values.

## GitHub repo creation

If you run `pnp deploy` in a directory without a git repository or without a GitHub remote, the CLI will offer to create one:

```
? No git repository found. Create a GitHub repository? Yes
? Repository name your-org/my-project
? Visibility Private

Creating GitHub repository...
Repository created: https://github.com/your-org/my-project
```

This requires the `gh` CLI to be installed and authenticated.

## CI/CD pipeline generation

Use `--with-ci` to generate a GitHub Actions workflow in your project:

```bash
pnp deploy --with-ci
```

This creates `.github/workflows/deploy.yml` that:
1. Triggers on push to `main`
2. Builds the project (PHP/Node.js depending on type)
3. Builds and pushes a Docker image to GHCR
4. Tags with both `latest` and the commit SHA

## Architecture

```
cmd/                    # Cobra commands
  deploy.go             # Main deploy flow
  update.go             # Re-render + push
  destroy.go            # Remove from GitOps
  status.go             # ArgoCD status check
  init.go               # Global config wizard
  helpers.go            # Shared TemplateData builder
internal/
  config/               # ~/.pnp.yaml and .cluster.yaml
  detect/               # Project type auto-detection
  wizard/               # Interactive TUI (charmbracelet/huh)
  templates/            # Manifest rendering (Go text/template)
  gitops/               # Git operations (clone, commit, push, PR)
  infisical/            # Infisical API client
  gh/                   # GitHub repo creation via gh CLI
  ci/                   # GitHub Actions workflow generation
```

## Development

```bash
# Clone
git clone https://github.com/fwartner/pnp.git
cd pnp

# Build
go build -o pnp .

# Test
go test ./...

# Run
./pnp version
```

### Releasing

Releases are automated via GitHub Actions + [GoReleaser](https://goreleaser.com/). Tag and push:

```bash
git tag v0.2.0
git push --tags
```

This builds binaries for Linux/macOS (amd64/arm64), creates a GitHub Release, and updates the Homebrew formula.

## License

MIT

# pnp

A CLI tool that automates Kubernetes deployments for [Pixel & Process](https://pixelandprocess.de) projects. Run `pnp deploy` inside any project folder and it handles everything — from creating the GitHub repository to deploying on the cluster.

## What it does

1. **Detects** your project type (Laravel, Next.js, Strapi — or any plugin-provided type) from source files
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
curl -sL https://github.com/fwartner/pnp/releases/latest/download/pnp_darwin_arm64.tar.gz | tar xz
sudo mv pnp /usr/local/bin/

# macOS (Intel)
curl -sL https://github.com/fwartner/pnp/releases/latest/download/pnp_darwin_amd64.tar.gz | tar xz
sudo mv pnp /usr/local/bin/

# Linux (amd64)
curl -sL https://github.com/fwartner/pnp/releases/latest/download/pnp_linux_amd64.tar.gz | tar xz
sudo mv pnp /usr/local/bin/
```

### From source

```bash
go install github.com/fwartner/pnp@latest
```

## Prerequisites

Run `pnp doctor` to check all prerequisites at once.

- **[gh CLI](https://cli.github.com/)** — authenticated (`gh auth login`) — for GitHub repo creation and PRs
- **Git** — for interacting with the GitOps repository
- **kubectl** — for `pnp status` and `pnp logs` (optional)
- **Docker** — for local builds (optional)

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
scopes:
  customer:
    domain: customerdomain.de
    githubOrg: customer-org
  agency:
    domain: pixelandprocess.de
    githubOrg: pixelandprocess
```

### 2. Scaffold a new project

```bash
pnp new laravel-web my-app
```

This creates a scope-prefixed project directory (e.g. `agency-my-app`) with:
- Scaffold files for the chosen project type
- `.cluster.yaml` with smart defaults
- `Dockerfile` and `.dockerignore`
- `.github/workflows/deploy.yml`
- Initialized git repo with initial commit

### 3. Deploy a project

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

### 4. Update after config changes

Edit `.cluster.yaml`, then:

```bash
pnp update
```

### 5. Tear it down

```bash
pnp destroy
```

## Commands

### `pnp new`

Scaffold a new project with all deployment configuration.

```bash
pnp new <type> <name>
pnp new laravel-web my-app        # creates agency-my-app/
pnp new nextjs-static dashboard   # creates customer-dashboard/
```

### `pnp deploy`

Create a new deployment or redeploy an existing one.

```bash
pnp deploy                   # detect, wizard, deploy
pnp deploy --pr              # create a PR instead of pushing to main
pnp deploy --with-ci         # also generate .github/workflows/deploy.yml
pnp deploy --advanced        # run the full advanced wizard with all options
pnp deploy --with-previews   # generate preview environment workflow for PRs
```

**What happens:**

```
┌─────────────┐   ┌──────────┐   ┌──────────┐   ┌────────┐   ┌─────────┐
│ Pre-deploy  │──>│  Doctor   │──>│  Wizard  │──>│ Render │──>│ GitOps  │
│   hooks     │   │  checks   │   │(confirm) │   │manifest│   │  push   │
└─────────────┘   └──────────┘   └──────────┘   └────────┘   └─────────┘
                                                                   │
                                                              ┌────▼────┐
                                                              │ ArgoCD  │
                                                              │  sync   │
                                                              └─────────┘
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

Show the ArgoCD sync and health status of the current project, including per-pod details and error explanations.

```bash
pnp status
```

```
App:         helm-my-customer-app
Environment: preview
Domain:      https://my-customer-app.preview.pixelandprocess.de
Sync:        Synced
Health:      Healthy

Pods:
  NAME                              STATUS   RESTARTS   AGE
  my-app-6d4f5b7c8-x2k9p           Running  0          2h
  my-app-worker-7f8a9b1c2-m3n4p     Running  0          2h
```

### `pnp doctor`

Check that all prerequisites are installed and configured.

```bash
pnp doctor
```

```
  ✓  git: installed
  ✓  gh: authenticated
  ✓  docker: running
  ✓  kubectl: configured
  ✓  global config: ~/.pnp.yaml found
  ✗  gitops repo: not cloned (run pnp init)
```

### `pnp list`

Show all deployed applications from the GitOps repo.

```bash
pnp list
```

### `pnp logs`

Stream logs from the running application pods.

```bash
pnp logs                # logs for current project
pnp logs my-app         # logs for a specific app
pnp logs --follow       # stream logs in real-time
pnp logs --tail 100     # show last 100 lines
```

### `pnp env`

Manage environment variables in `.cluster.yaml`.

```bash
pnp env list                        # show all env vars
pnp env set DATABASE_URL=postgres:// # set a variable
pnp env edit                        # open in $EDITOR
```

### `pnp rollback`

Revert a deployment to a previous version via interactive commit selection.

```bash
pnp rollback
```

### `pnp init`

Interactive setup for global configuration (`~/.pnp.yaml`).

```bash
pnp init
```

### `pnp version`

```bash
pnp version
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

New types can be added via the internal registry (one Go file) or as external plugins.

## Naming convention

Projects use scope-prefixed names for consistent organization across multi-tenant environments:

```
<scope>-<name>
```

| Scope | Example | Use case |
|-------|---------|----------|
| `customer` | `customer-acme-corp` | Client projects |
| `private` | `private-internal-tool` | Internal tools |
| `agency` | `agency-pixel-process` | Pixel & Process projects |

The scope determines default GitHub org, domain, repo visibility, and Infisical project — all configurable per scope in `~/.pnp.yaml`.

## Plugin system

Third-party plugins extend pnp with new project types, CLI commands, and deploy hooks.

### Plugin directory structure

```
~/.pnp/plugins/
└── django/
    ├── plugin.yaml          # manifest
    ├── django-type           # binary for project type
    └── pnp-django-lint       # binary for custom command
```

### Plugin manifest (`plugin.yaml`)

```yaml
name: django
version: 0.1.0
provides:
  types:
    - name: django
      binary: ./django-type
  commands:
    - name: lint
      binary: ./pnp-django-lint
      description: "Run Django linting"
  hooks:
    - event: pre-deploy
      binary: ./pre-deploy-check
```

### Plugin protocol

**Type plugins** are binaries called with subcommands, communicating via stdin/stdout JSON:

| Subcommand | Input | Output |
|-----------|-------|--------|
| `info` | — | `{ "name": "django", "displayName": "Django", "chartPath": "charts/django", "hasDatabase": true, "defaults": {...} }` |
| `detect <dir>` | — | `{ "confidence": "high" }` |
| `values-template` | — | Template string (stdout) |
| `application-template` | — | Template string (stdout) |
| `dockerfile` | ProjectConfig (stdin) | Dockerfile content (stdout) |
| `dockerignore` | — | Content (stdout) |
| `scaffold` | ScaffoldData (stdin) | `{ "path": "content" }` |

**Command plugins** receive project/global config as environment variables and pass through user args.

**Hook plugins** receive event JSON on stdin. Exit 0 to continue, non-zero to abort.

### Hook events

| Event | When | Abort on failure |
|-------|------|-----------------|
| `pre-deploy` | Before deploy steps run | Yes |
| `post-deploy` | After successful deploy | No (warning only) |

## Project config (`.cluster.yaml`)

Saved in your project directory after the first deploy. Editable for `pnp update`.

```yaml
name: agency-my-customer-app
type: laravel-web
scope: agency
environment: preview
domain: agency-my-customer-app.preview.pixelandprocess.de
image: ghcr.io/pixelandprocess/agency-my-customer-app
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
| Preview | `<name>.preview.<scope-domain>` | `agency-acme.preview.pixelandprocess.de` |
| Staging | `<name>.staging.<scope-domain>` | `agency-acme.staging.pixelandprocess.de` |
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
apps/<scope>/<environment>/<name>/
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
? Repository name pixelandprocess/agency-my-project
? Visibility Private

Creating GitHub repository...
Repository created: https://github.com/pixelandprocess/agency-my-project
```

This requires the `gh` CLI to be installed and authenticated. The default org and visibility are determined by the project scope.

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

Use `--with-previews` to also generate a preview environment workflow that deploys on every PR.

## Architecture

```
cmd/                    # Cobra commands
  deploy.go             # Main deploy flow with progress tracker
  new.go                # Project scaffolding
  update.go             # Re-render + push
  destroy.go            # Remove from GitOps
  status.go             # ArgoCD status check + pod details
  doctor.go             # Prerequisites checker
  list.go               # List deployed apps
  logs.go               # Stream pod logs
  rollback.go           # Revert deployments
  env.go                # Manage env vars
  init.go               # Global config wizard
  helpers.go            # Shared TemplateData builder
internal/
  types/                # Project type registry + built-in types
    types.go            # ProjectType interface
    registry.go         # Register, Get, All, Names, Detect
    laravel_web.go      # Laravel web implementation
    laravel_api.go      # Laravel API implementation
    nextjs_fullstack.go # Next.js fullstack implementation
    nextjs_static.go    # Next.js static implementation
    strapi.go           # Strapi implementation
  plugin/               # External plugin system
    manifest.go         # plugin.yaml parsing
    loader.go           # Plugin discovery (~/.pnp/plugins/)
    external_type.go    # ProjectType adapter for plugin binaries
    hooks.go            # Hook registry and runner
    command.go          # Plugin command registration
  config/               # ~/.pnp.yaml, .cluster.yaml, naming helpers
  detect/               # Project type auto-detection
  wizard/               # Interactive TUI (charmbracelet/huh)
  templates/            # Manifest rendering (Go text/template)
  gitops/               # Git operations (clone, commit, push, PR)
  infisical/            # Infisical API client
  gh/                   # GitHub repo creation via gh CLI
  ci/                   # GitHub Actions workflow + Dockerfile generation
  doctor/               # Prerequisites checker
  kube/                 # kubectl wrapper
  progress/             # Step tracker with spinner animation
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

### Adding a new project type

Create a single file in `internal/types/` implementing the `ProjectType` interface with an `init()` function that calls `Register()`:

```go
package types

func init() {
    Register(&MyType{})
}

type MyType struct{}

func (m *MyType) Name() string        { return "my-type" }
func (m *MyType) DisplayName() string { return "My Type" }
// ... implement remaining interface methods
```

No other files need to be modified — the registry picks it up automatically.

### Releasing

Releases are automated via GitHub Actions + [GoReleaser](https://goreleaser.com/). Tag and push:

```bash
git tag v1.4.0
git push --tags
```

This builds binaries for Linux/macOS (amd64/arm64), creates a GitHub Release, and updates the Homebrew formula.

## License

MIT

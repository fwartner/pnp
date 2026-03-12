package types

import (
	"path/filepath"

	"github.com/fwartner/pnp/internal/config"
)

func init() {
	Register(&Strapi{})
}

// Strapi implements ProjectType for Strapi CMS applications.
type Strapi struct{}

func (s *Strapi) Name() string        { return "strapi" }
func (s *Strapi) DisplayName() string  { return "Strapi" }
func (s *Strapi) ChartPath() string    { return "charts/strapi" }
func (s *Strapi) HasDatabase() bool    { return true }
func (s *Strapi) IsLaravel() bool      { return false }

func (s *Strapi) Detect(dir string) string {
	pkgPath := filepath.Join(dir, "package.json")
	if !FileExists(pkgPath) {
		return ""
	}
	deps := ReadPackageDeps(pkgPath)
	if _, ok := deps["@strapi/strapi"]; ok {
		return "high"
	}
	return ""
}

func (s *Strapi) DefaultConfig() TypeDefaults {
	return TypeDefaults{
		Database: true, Redis: false, Queue: false,
		Scheduler: false, Persistence: true,
		CPU: "200m", Memory: "512Mi",
	}
}

func (s *Strapi) ValuesTemplate() string {
	return strapiValuesYAML
}

func (s *Strapi) ApplicationTemplate() string {
	return strapiApplicationYAML
}

func (s *Strapi) Dockerfile(cfg config.ProjectConfig) string {
	return strapiDockerfile
}

func (s *Strapi) Dockerignore() string { return nodeDockerignore }

func (s *Strapi) ScaffoldFiles(data ScaffoldData) map[string]string {
	return map[string]string{
		"package.json": `{
  "name": "` + data.ShortName + `",
  "version": "0.1.0",
  "private": true,
  "scripts": {
    "develop": "strapi develop",
    "start": "strapi start",
    "build": "strapi build"
  },
  "dependencies": {
    "@strapi/strapi": "^5.0.0",
    "@strapi/plugin-users-permissions": "^5.0.0",
    "pg": "^8.0.0"
  }
}
`,
		"config/database.ts": `export default ({ env }) => ({
  connection: {
    client: 'postgres',
    connection: {
      connectionString: env('DATABASE_URL'),
    },
  },
});
`,
		"config/server.ts": `export default ({ env }) => ({
  host: env('HOST', '0.0.0.0'),
  port: env.int('PORT', 1337),
  app: {
    keys: env.array('APP_KEYS'),
  },
});
`,
		"src/.gitkeep":    "",
		"public/.gitkeep": "",
	}
}

const strapiValuesYAML = `spec:
  source:
    repoURL: << .RepoURL >>
    targetRevision: main
    path: << .ChartPath >>
  destination:
    namespace: << .Namespace >>
domain: << .Domain >>
subdomain: << .Subdomain >>
image:
  repository: << .Image >>
  tag: << .Tag >>
database:
  name: << .DBName >>
  username: << .DBUsername >>
`

const strapiApplicationYAML = `apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: helm-<< .Name >>
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: default
  source:
    repoURL: {{ .Values.spec.source.repoURL }}
    targetRevision: {{ .Values.spec.source.targetRevision }}
    path: {{ .Values.spec.source.path }}
    helm:
      values: |
        image:
          repository: {{ .Values.image.repository }}
          tag: {{ .Values.image.tag | quote }}
        imagePullSecrets:
          - name: ghcr-secret
        ingress:
          enabled: true
          className: traefik
          annotations:
            cert-manager.io/cluster-issuer: letsencrypt-prod
            traefik.ingress.kubernetes.io/router.entrypoints: websecure
            traefik.ingress.kubernetes.io/router.tls: "true"
          host: {{ .Values.subdomain }}.{{ .Values.domain }}
          tlsSecretName: {{ .Values.subdomain }}-tls
        env:
          DATABASE_URL: "postgresql://{{ .Values.database.username }}:$(DB_PASSWORD)@<< .Name >>-db-rw:5432/{{ .Values.database.name }}"
        database:
          enabled: true
          size: << .DBSize >>
          name: {{ .Values.database.name }}
          username: {{ .Values.database.username }}
          existingSecret: {{ .Release.Name }}-db-credentials
          existingSecretPasswordKey: password
        persistence:
          enabled: true
          size: << .PersistenceSize >>
          storageClass: hcloud-volumes
        resources:
          requests:
            cpu: << .CPU >>
            memory: << .Memory >>
  destination:
    server: https://kubernetes.default.svc
    namespace: {{ .Values.spec.destination.namespace }}
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
`

const strapiDockerfile = `FROM ghcr.io/fwartner/pnp/strapi:latest AS base

FROM base AS deps
WORKDIR /app
COPY package.json package-lock.json* yarn.lock* pnpm-lock.yaml* ./
RUN \
    if [ -f yarn.lock ]; then yarn install --frozen-lockfile; \
    elif [ -f pnpm-lock.yaml ]; then corepack enable pnpm && pnpm install --frozen-lockfile; \
    elif [ -f package-lock.json ]; then npm ci; \
    else npm install; \
    fi

FROM base AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

FROM base AS production
WORKDIR /app

COPY --from=builder --chown=strapi:strapi /app ./

USER strapi

CMD ["npm", "run", "start"]
`

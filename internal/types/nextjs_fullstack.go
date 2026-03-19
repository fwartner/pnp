package types

import (
	"path/filepath"

	"github.com/fwartner/pnp/internal/config"
)

func init() {
	Register(&NextjsFullstack{})
}

// NextjsFullstack implements ProjectType for Next.js with database.
type NextjsFullstack struct{}

func (n *NextjsFullstack) Name() string        { return "nextjs-fullstack" }
func (n *NextjsFullstack) DisplayName() string  { return "Next.js (Fullstack)" }
func (n *NextjsFullstack) ChartPath() string    { return "charts/nextjs" }
func (n *NextjsFullstack) HasDatabase() bool    { return true }
func (n *NextjsFullstack) IsLaravel() bool      { return false }

func (n *NextjsFullstack) Detect(dir string) string {
	pkgPath := filepath.Join(dir, "package.json")
	if !FileExists(pkgPath) {
		return ""
	}
	deps := ReadPackageDeps(pkgPath)
	if _, ok := deps["next"]; ok && HasDBDeps(deps) {
		return "high"
	}
	return ""
}

func (n *NextjsFullstack) DefaultConfig() TypeDefaults {
	return TypeDefaults{
		Database: true, Redis: false, Queue: false,
		Scheduler: false, Persistence: false,
		CPU: "200m", Memory: "512Mi",
	}
}

func (n *NextjsFullstack) ValuesTemplate() string {
	return nextjsFullstackValuesYAML
}

func (n *NextjsFullstack) ApplicationTemplate() string {
	return nextjsFullstackApplicationYAML
}

func (n *NextjsFullstack) Dockerfile(cfg config.ProjectConfig) string {
	return nextjsDockerfile
}

func (n *NextjsFullstack) Dockerignore() string { return nodeDockerignore }

func (n *NextjsFullstack) ScaffoldFiles(data ScaffoldData) map[string]string {
	return nextjsScaffoldFiles(data)
}

const nextjsFullstackValuesYAML = `spec:
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

const nextjsFullstackApplicationYAML = `apiVersion: argoproj.io/v1alpha1
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

const nextjsDockerfile = `FROM ghcr.io/pixel-process-ug/nextjs-base:latest AS base

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

COPY --from=builder /app/public ./public

# Standalone output — set output: 'standalone' in next.config.js
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

USER nextjs

CMD ["node", "server.js"]
`

const nodeDockerignore = `node_modules
.git
.github
.env
.env.*
.cluster.yaml
.next
out
dist
build
coverage
docker-compose*.yml
`

func nextjsScaffoldFiles(data ScaffoldData) map[string]string {
	return map[string]string{
		"package.json": `{
  "name": "` + data.ShortName + `",
  "version": "0.1.0",
  "private": true,
  "scripts": {
    "dev": "next dev",
    "build": "next build",
    "start": "next start",
    "lint": "next lint"
  },
  "dependencies": {
    "next": "^15.0.0",
    "react": "^19.0.0",
    "react-dom": "^19.0.0"
  },
  "devDependencies": {
    "typescript": "^5.0.0",
    "@types/node": "^22.0.0",
    "@types/react": "^19.0.0"
  }
}
`,
		"next.config.js": `/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'standalone',
}

module.exports = nextConfig
`,
		"tsconfig.json": `{
  "compilerOptions": {
    "target": "ES2017",
    "lib": ["dom", "dom.iterable", "esnext"],
    "allowJs": true,
    "skipLibCheck": true,
    "strict": true,
    "noEmit": true,
    "esModuleInterop": true,
    "module": "esnext",
    "moduleResolution": "bundler",
    "resolveJsonModule": true,
    "isolatedModules": true,
    "jsx": "preserve",
    "incremental": true,
    "paths": {
      "@/*": ["./src/*"]
    }
  },
  "include": ["next-env.d.ts", "**/*.ts", "**/*.tsx"],
  "exclude": ["node_modules"]
}
`,
		"src/app/layout.tsx": `export const metadata = {
  title: '` + data.Name + `',
  description: '` + data.Name + ` — managed by pnp',
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  )
}
`,
		"src/app/page.tsx": `export default function Home() {
  return (
    <main>
      <h1>` + data.Name + `</h1>
      <p>Your project is ready. Start building!</p>
    </main>
  )
}
`,
		"public/.gitkeep": "",
	}
}

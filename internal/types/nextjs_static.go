package types

import (
	"path/filepath"

	"github.com/fwartner/pnp/internal/config"
)

func init() {
	Register(&NextjsStatic{})
}

// NextjsStatic implements ProjectType for static Next.js sites (no database).
type NextjsStatic struct{}

func (n *NextjsStatic) Name() string        { return "nextjs-static" }
func (n *NextjsStatic) DisplayName() string  { return "Next.js (Static)" }
func (n *NextjsStatic) ChartPath() string    { return "charts/nextjs" }
func (n *NextjsStatic) HasDatabase() bool    { return false }
func (n *NextjsStatic) IsLaravel() bool      { return false }

func (n *NextjsStatic) Detect(dir string) string {
	pkgPath := filepath.Join(dir, "package.json")
	if !FileExists(pkgPath) {
		return ""
	}
	deps := ReadPackageDeps(pkgPath)
	if _, ok := deps["next"]; ok && !HasDBDeps(deps) {
		return "high"
	}
	return ""
}

func (n *NextjsStatic) DefaultConfig() TypeDefaults {
	return TypeDefaults{
		Database: false, Redis: false, Queue: false,
		Scheduler: false, Persistence: false,
		CPU: "200m", Memory: "512Mi",
	}
}

func (n *NextjsStatic) ValuesTemplate() string {
	return nextjsStaticValuesYAML
}

func (n *NextjsStatic) ApplicationTemplate() string {
	return nextjsStaticApplicationYAML
}

func (n *NextjsStatic) Dockerfile(cfg config.ProjectConfig) string {
	return nextjsDockerfile
}

func (n *NextjsStatic) Dockerignore() string { return nodeDockerignore }

func (n *NextjsStatic) ScaffoldFiles(data ScaffoldData) map[string]string {
	return nextjsScaffoldFiles(data)
}

const nextjsStaticValuesYAML = `spec:
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
`

const nextjsStaticApplicationYAML = `apiVersion: argoproj.io/v1alpha1
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
        database:
          enabled: false
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

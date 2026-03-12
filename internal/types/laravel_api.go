package types

import (
	"path/filepath"

	"github.com/fwartner/pnp/internal/config"
)

func init() {
	Register(&LaravelAPI{})
}

// LaravelAPI implements ProjectType for Laravel API applications.
type LaravelAPI struct{}

func (l *LaravelAPI) Name() string        { return "laravel-api" }
func (l *LaravelAPI) DisplayName() string  { return "Laravel (API)" }
func (l *LaravelAPI) ChartPath() string    { return "charts/laravel" }
func (l *LaravelAPI) HasDatabase() bool    { return true }
func (l *LaravelAPI) IsLaravel() bool      { return true }

func (l *LaravelAPI) Detect(dir string) string {
	if FileExists(filepath.Join(dir, "composer.json")) && FileExists(filepath.Join(dir, "artisan")) {
		if !IsLaravelWebProject(dir) {
			return "high"
		}
	}
	return ""
}

func (l *LaravelAPI) DefaultConfig() TypeDefaults {
	return TypeDefaults{
		Database: true, Redis: true, Queue: true,
		Scheduler: true, Persistence: false,
		CPU: "200m", Memory: "512Mi",
	}
}

func (l *LaravelAPI) ValuesTemplate() string { return laravelValuesYAML }

func (l *LaravelAPI) ApplicationTemplate() string {
	return laravelAPIApplicationYAML
}

func (l *LaravelAPI) Dockerfile(cfg config.ProjectConfig) string {
	return LaravelDockerfileFor(cfg)
}

func (l *LaravelAPI) Dockerignore() string { return laravelDockerignore }

func (l *LaravelAPI) ScaffoldFiles(data ScaffoldData) map[string]string {
	return LaravelScaffoldFiles(data)
}

const laravelAPIApplicationYAML = `apiVersion: argoproj.io/v1alpha1
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
        app:
          key: {{ .Values.app.key | quote }}
          url: "https://{{ .Values.subdomain }}.{{ .Values.domain }}"
          env: production
          debug: "false"
        database:
          enabled: true
          size: << .DBSize >>
          name: {{ .Values.database.name }}
          username: {{ .Values.database.username }}
          existingSecret: {{ .Release.Name }}-db-credentials
          existingSecretPasswordKey: password
        redis:
          enabled: << .RedisEnabled >>
        queue:
          enabled: << .QueueEnabled >>
          replicaCount: << .QueueReplicas >>
        scheduler:
          enabled: << .SchedulerEnabled >>
        horizon:
          enabled: << .HorizonEnabled >>
        reverb:
          enabled: << .ReverbEnabled >>
          port: << .ReverbPort >>
        octane:
          enabled: << .OctaneEnabled >>
          server: << .OctaneServer >>
        mail:
          mailer: smtp
          host: smtp.postmarkapp.com
          port: "587"
          existingSecret: {{ .Release.Name }}-mail-credentials
          from: {{ .Values.mail.from }}
          fromName: {{ .Release.Name | quote }}
        persistence:
          enabled: << .PersistenceEnabled >>
          size: << .PersistenceSize >>
          storageClass: hcloud-volumes
        resources:
          web:
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

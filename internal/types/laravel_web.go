package types

import (
	"path/filepath"

	"github.com/fwartner/pnp/internal/config"
)

func init() {
	Register(&LaravelWeb{})
}

// LaravelWeb implements ProjectType for full Laravel web applications.
type LaravelWeb struct{}

func (l *LaravelWeb) Name() string        { return "laravel-web" }
func (l *LaravelWeb) DisplayName() string  { return "Laravel (Full Web)" }
func (l *LaravelWeb) ChartPath() string    { return "charts/laravel" }
func (l *LaravelWeb) HasDatabase() bool    { return true }
func (l *LaravelWeb) IsLaravel() bool      { return true }

func (l *LaravelWeb) Detect(dir string) string {
	if FileExists(filepath.Join(dir, "composer.json")) && FileExists(filepath.Join(dir, "artisan")) {
		if IsLaravelWebProject(dir) {
			return "high"
		}
	}
	return ""
}

func (l *LaravelWeb) DefaultConfig() TypeDefaults {
	return TypeDefaults{
		Database: true, Redis: true, Queue: true,
		Scheduler: true, Persistence: true,
		CPU: "200m", Memory: "512Mi",
	}
}

func (l *LaravelWeb) ValuesTemplate() string { return laravelValuesYAML }

func (l *LaravelWeb) ApplicationTemplate() string {
	return laravelWebApplicationYAML
}

func (l *LaravelWeb) Dockerfile(cfg config.ProjectConfig) string {
	return LaravelDockerfileFor(cfg)
}

func (l *LaravelWeb) Dockerignore() string { return laravelDockerignore }

func (l *LaravelWeb) ScaffoldFiles(data ScaffoldData) map[string]string {
	return LaravelScaffoldFiles(data)
}

const laravelWebApplicationYAML = `apiVersion: argoproj.io/v1alpha1
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

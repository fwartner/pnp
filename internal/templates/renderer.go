package templates

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// chartYAML is generated for every project type.
const chartYAML = `apiVersion: v2
name: << .Name >>
version: 0.1.0
`

// ---------- values.yaml variants ----------

const laravelValuesYAML = `spec:
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
app:
  key: << .AppKey >>
database:
  name: << .DBName >>
  username: << .DBUsername >>
horizon:
  enabled: << .HorizonEnabled >>
reverb:
  enabled: << .ReverbEnabled >>
  port: << .ReverbPort >>
octane:
  enabled: << .OctaneEnabled >>
  server: << .OctaneServer >>
mail:
  from: info@pixelandprocess.de
`

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

// ---------- application.yaml variants ----------

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

// ---------- CNPG Cluster ----------

const cnpgClusterYAML = `apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: << .Name >>-db
  namespace: << .Namespace >>
spec:
  instances: 1
  storage:
    size: << .DBSize >>
  bootstrap:
    initdb:
      database: << .DBName >>
      owner: << .DBUsername >>
      secret:
        name: << .Name >>-db-credentials
`

// ---------- Infisical Secrets ----------

const infisicalDBOnlyYAML = `apiVersion: secrets.infisical.com/v1alpha1
kind: InfisicalSecret
metadata:
  name: << .Name >>-db-infisical
  namespace: << .Namespace >>
spec:
  hostAPI: << .InfisicalHost >>
  resyncInterval: 60
  authentication:
    universalAuth:
      credentialsRef:
        secretName: infisical-machine-identity
        secretNamespace: infisical-operator-system
      secretsScope:
        projectSlug: << .InfisicalProjectSlug >>
        envSlug: << .InfisicalEnvSlug >>
        secretsPath: << .InfisicalSecretsPath >>
  managedSecretReference:
    secretName: << .Name >>-db-credentials
    secretNamespace: << .Namespace >>
    secretType: kubernetes.io/basic-auth
`

const infisicalDBAndMailYAML = `apiVersion: secrets.infisical.com/v1alpha1
kind: InfisicalSecret
metadata:
  name: << .Name >>-db-infisical
  namespace: << .Namespace >>
spec:
  hostAPI: << .InfisicalHost >>
  resyncInterval: 60
  authentication:
    universalAuth:
      credentialsRef:
        secretName: infisical-machine-identity
        secretNamespace: infisical-operator-system
      secretsScope:
        projectSlug: << .InfisicalProjectSlug >>
        envSlug: << .InfisicalEnvSlug >>
        secretsPath: << .InfisicalSecretsPath >>
  managedSecretReference:
    secretName: << .Name >>-db-credentials
    secretNamespace: << .Namespace >>
    secretType: kubernetes.io/basic-auth
---
apiVersion: secrets.infisical.com/v1alpha1
kind: InfisicalSecret
metadata:
  name: << .Name >>-mail-infisical
  namespace: << .Namespace >>
spec:
  hostAPI: << .InfisicalHost >>
  resyncInterval: 60
  authentication:
    universalAuth:
      credentialsRef:
        secretName: infisical-machine-identity
        secretNamespace: infisical-operator-system
      secretsScope:
        projectSlug: << .InfisicalMailProjectSlug >>
        envSlug: << .InfisicalEnvSlug >>
        secretsPath: /smtp
  managedSecretReference:
    secretName: << .Name >>-mail-credentials
    secretNamespace: << .Namespace >>
    secretType: Opaque
`

// hasDB returns true if the project type requires a database.
func hasDB(projectType string) bool {
	switch projectType {
	case "laravel-web", "laravel-api", "nextjs-fullstack", "strapi":
		return true
	}
	return false
}

// isLaravel returns true if the project type is a Laravel variant.
func isLaravel(projectType string) bool {
	return projectType == "laravel-web" || projectType == "laravel-api"
}

// valuesTemplate returns the values.yaml template string for the given project type.
func valuesTemplate(projectType string) string {
	switch projectType {
	case "laravel-web", "laravel-api":
		return laravelValuesYAML
	case "nextjs-fullstack":
		return nextjsFullstackValuesYAML
	case "nextjs-static":
		return nextjsStaticValuesYAML
	case "strapi":
		return strapiValuesYAML
	}
	return ""
}

// applicationTemplate returns the application.yaml template string for the given project type.
func applicationTemplate(projectType string) string {
	switch projectType {
	case "laravel-web":
		return laravelWebApplicationYAML
	case "laravel-api":
		return laravelAPIApplicationYAML
	case "nextjs-fullstack":
		return nextjsFullstackApplicationYAML
	case "nextjs-static":
		return nextjsStaticApplicationYAML
	case "strapi":
		return strapiApplicationYAML
	}
	return ""
}

// renderTemplate parses a template string with << >> delimiters and writes the result to a file.
func renderTemplate(tmplStr, outPath string, data TemplateData) error {
	t, err := template.New("").Delims("<<", ">>").Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("parsing template for %s: %w", outPath, err)
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("creating directory for %s: %w", outPath, err)
	}

	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("creating file %s: %w", outPath, err)
	}
	defer f.Close()

	if err := t.Execute(f, data); err != nil {
		return fmt.Errorf("executing template for %s: %w", outPath, err)
	}
	return nil
}

// Render generates all Helm chart files for the given project type into outDir.
func Render(projectType string, data TemplateData, outDir string) error {
	// Chart.yaml — always
	if err := renderTemplate(chartYAML, filepath.Join(outDir, "Chart.yaml"), data); err != nil {
		return err
	}

	// values.yaml — varies by type
	vt := valuesTemplate(projectType)
	if vt == "" {
		return fmt.Errorf("unsupported project type: %s", projectType)
	}
	if err := renderTemplate(vt, filepath.Join(outDir, "values.yaml"), data); err != nil {
		return err
	}

	// templates/application.yaml — varies by type
	at := applicationTemplate(projectType)
	if at == "" {
		return fmt.Errorf("unsupported project type for application template: %s", projectType)
	}
	if err := renderTemplate(at, filepath.Join(outDir, "templates", "application.yaml"), data); err != nil {
		return err
	}

	// templates/cnpg-cluster.yaml — only for types with DB
	if hasDB(projectType) {
		if err := renderTemplate(cnpgClusterYAML, filepath.Join(outDir, "templates", "cnpg-cluster.yaml"), data); err != nil {
			return err
		}
	}

	// templates/infisical-secrets.yaml — only for types with DB
	if hasDB(projectType) {
		tmpl := infisicalDBOnlyYAML
		if isLaravel(projectType) {
			tmpl = infisicalDBAndMailYAML
		}
		if err := renderTemplate(tmpl, filepath.Join(outDir, "templates", "infisical-secrets.yaml"), data); err != nil {
			return err
		}
	}

	return nil
}

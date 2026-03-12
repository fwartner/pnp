package templates

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/fwartner/pnp/internal/types"
)

// chartYAML is generated for every project type.
const chartYAML = `apiVersion: v2
name: << .Name >>
version: 0.1.0
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

// ---------- Kubernetes Secrets (generated credentials) ----------

const dbCredentialsSecretYAML = `apiVersion: v1
kind: Secret
metadata:
  name: << .Name >>-db-credentials
  namespace: << .Namespace >>
type: kubernetes.io/basic-auth
stringData:
  username: << .DBUsername >>
  password: << .DBPassword >>
`

const laravelAppSecretYAML = `apiVersion: v1
kind: Secret
metadata:
  name: << .Name >>-env
  namespace: << .Namespace >>
type: Opaque
stringData:
  APP_KEY: << .AppKey >>
`

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
	pt := types.Get(projectType)
	if pt == nil {
		return fmt.Errorf("unsupported project type: %s", projectType)
	}

	// Chart.yaml — always
	if err := renderTemplate(chartYAML, filepath.Join(outDir, "Chart.yaml"), data); err != nil {
		return err
	}

	// values.yaml — from type
	if err := renderTemplate(pt.ValuesTemplate(), filepath.Join(outDir, "values.yaml"), data); err != nil {
		return err
	}

	// templates/application.yaml — from type
	if err := renderTemplate(pt.ApplicationTemplate(), filepath.Join(outDir, "templates", "application.yaml"), data); err != nil {
		return err
	}

	// templates/cnpg-cluster.yaml — only for types with DB
	if pt.HasDatabase() {
		if err := renderTemplate(cnpgClusterYAML, filepath.Join(outDir, "templates", "cnpg-cluster.yaml"), data); err != nil {
			return err
		}
	}

	// templates/infisical-secrets.yaml — only for types with DB and Infisical configured
	if pt.HasDatabase() && data.InfisicalProjectSlug != "" {
		tmpl := infisicalDBOnlyYAML
		if pt.IsLaravel() {
			tmpl = infisicalDBAndMailYAML
		}
		if err := renderTemplate(tmpl, filepath.Join(outDir, "templates", "infisical-secrets.yaml"), data); err != nil {
			return err
		}
	}

	// templates/db-credentials.yaml — generated DB credentials for CNPG bootstrap
	if pt.HasDatabase() && data.DBPassword != "" {
		if err := renderTemplate(dbCredentialsSecretYAML, filepath.Join(outDir, "templates", "db-credentials.yaml"), data); err != nil {
			return err
		}
	}

	// templates/app-secret.yaml — generated app secret (APP_KEY for Laravel)
	if pt.IsLaravel() && data.AppKey != "" {
		if err := renderTemplate(laravelAppSecretYAML, filepath.Join(outDir, "templates", "app-secret.yaml"), data); err != nil {
			return err
		}
	}

	return nil
}

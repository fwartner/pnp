package templates

// TemplateData holds all configuration values needed to render
// Helm chart templates for a given project deployment.
type TemplateData struct {
	Name      string
	Namespace string
	Subdomain string
	Domain    string
	Image     string
	Tag       string
	AppKey    string
	DBName    string
	DBUsername string
	DBSize    string

	RedisEnabled       bool
	QueueEnabled       bool
	SchedulerEnabled   bool
	PersistenceEnabled bool
	PersistenceSize    string

	InfisicalProjectSlug     string
	InfisicalEnvSlug         string
	InfisicalSecretsPath     string
	InfisicalMailEnabled     bool
	InfisicalHost            string
	InfisicalMailProjectSlug string

	CPU       string
	Memory    string
	ChartPath string
	RepoURL   string
}

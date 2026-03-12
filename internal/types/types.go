package types

import "github.com/fwartner/pnp/internal/config"

// ProjectType defines the contract that every project type must implement.
// Adding a new project type requires implementing this interface in a single file
// and calling Register() in an init() function.
type ProjectType interface {
	// Name returns the type identifier, e.g. "laravel-web", "nextjs-static".
	Name() string

	// DisplayName returns a human-readable label for wizard UIs.
	DisplayName() string

	// Detect inspects the given directory and returns a confidence level.
	// Returns "high", "medium", "low", or "" if not detected.
	Detect(dir string) string

	// DefaultConfig returns type-specific defaults for the wizard.
	DefaultConfig() TypeDefaults

	// ChartPath returns the Helm chart path in the gitops repo (e.g. "charts/laravel").
	ChartPath() string

	// HasDatabase returns whether this type uses a database by default.
	HasDatabase() bool

	// IsLaravel returns whether this type is a Laravel variant.
	IsLaravel() bool

	// ValuesTemplate returns the values.yaml Go template string (using << >> delims).
	ValuesTemplate() string

	// ApplicationTemplate returns the application.yaml Go template string.
	ApplicationTemplate() string

	// Dockerfile returns the Dockerfile content for this project type.
	Dockerfile(cfg config.ProjectConfig) string

	// Dockerignore returns the .dockerignore content.
	Dockerignore() string

	// ScaffoldFiles returns files to generate for `pnp new`.
	// Key is relative path, value is file content.
	ScaffoldFiles(data ScaffoldData) map[string]string
}

// TypeDefaults holds the default feature flags for a project type.
type TypeDefaults struct {
	Database    bool
	Redis       bool
	Queue       bool
	Scheduler   bool
	Persistence bool
	CPU         string
	Memory      string
}

// ScaffoldData holds data for project scaffolding templates.
type ScaffoldData struct {
	Name      string // scope-prefixed name (e.g. "agency-my-app")
	ShortName string // name without scope prefix (e.g. "my-app")
	Scope     string
	Domain    string
	Image     string
}

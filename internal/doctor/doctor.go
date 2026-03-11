package doctor

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/charmbracelet/huh"
	"github.com/fwartner/pnp/internal/config"
)

// Check represents a single prerequisite check.
type Check struct {
	Name     string
	Run      func() (ok bool, msg string)
	Fix      func() error // nil = not auto-fixable
	Critical bool         // if true, failure blocks deploy
}

// Result holds the outcome of a single check.
type Result struct {
	Name     string
	OK       bool
	Message  string
	Critical bool
}

// RunAll executes all checks and returns results. If autoFix is true,
// it prompts the user to fix failed checks that have a Fix function.
func RunAll(autoFix bool) []Result {
	checks := DefaultChecks()
	var results []Result

	for _, c := range checks {
		ok, msg := c.Run()
		if !ok && autoFix && c.Fix != nil {
			var confirm bool
			_ = huh.NewForm(
				huh.NewGroup(
					huh.NewConfirm().
						Title(fmt.Sprintf("%s: %s — attempt auto-fix?", c.Name, msg)).
						Value(&confirm),
				),
			).Run()

			if confirm {
				if err := c.Fix(); err != nil {
					msg = fmt.Sprintf("auto-fix failed: %v", err)
				} else {
					ok, msg = c.Run()
				}
			}
		}
		results = append(results, Result{
			Name:     c.Name,
			OK:       ok,
			Message:  msg,
			Critical: c.Critical,
		})
	}

	return results
}

// HasCriticalFailure returns true if any critical check failed.
func HasCriticalFailure(results []Result) bool {
	for _, r := range results {
		if !r.OK && r.Critical {
			return true
		}
	}
	return false
}

// DefaultChecks returns the standard set of prerequisite checks.
func DefaultChecks() []Check {
	return []Check{
		{
			Name:     "git",
			Critical: true,
			Run: func() (bool, string) {
				if _, err := exec.LookPath("git"); err != nil {
					return false, "git is not installed"
				}
				return true, "git is installed"
			},
			Fix: func() error {
				return runBrew("install", "git")
			},
		},
		{
			Name:     "gh CLI",
			Critical: true,
			Run: func() (bool, string) {
				if _, err := exec.LookPath("gh"); err != nil {
					return false, "gh CLI is not installed"
				}
				cmd := exec.Command("gh", "auth", "status")
				if err := cmd.Run(); err != nil {
					return false, "gh CLI is installed but not authenticated — run 'gh auth login'"
				}
				return true, "gh CLI is installed and authenticated"
			},
			Fix: func() error {
				if _, err := exec.LookPath("gh"); err != nil {
					if err := runBrew("install", "gh"); err != nil {
						return err
					}
				}
				fmt.Println("Please authenticate with GitHub:")
				cmd := exec.Command("gh", "auth", "login")
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				return cmd.Run()
			},
		},
		{
			Name:     "docker",
			Critical: false,
			Run: func() (bool, string) {
				if _, err := exec.LookPath("docker"); err != nil {
					return false, "docker is not installed"
				}
				return true, "docker is installed"
			},
			Fix: func() error {
				return runBrew("install", "--cask", "docker")
			},
		},
		{
			Name:     "kubectl",
			Critical: false,
			Run: func() (bool, string) {
				if _, err := exec.LookPath("kubectl"); err != nil {
					return false, "kubectl is not installed (optional — needed for status/logs)"
				}
				return true, "kubectl is installed"
			},
			Fix: nil,
		},
		{
			Name:     "global config",
			Critical: true,
			Run: func() (bool, string) {
				path, err := config.GlobalConfigPath()
				if err != nil {
					return false, "cannot determine config path"
				}
				if _, err := os.Stat(path); os.IsNotExist(err) {
					return false, "~/.pnp.yaml not found — run 'pnp init'"
				}
				_, err = config.LoadGlobalConfig()
				if err != nil {
					return false, fmt.Sprintf("~/.pnp.yaml is invalid: %v", err)
				}
				return true, "~/.pnp.yaml is valid"
			},
			Fix: nil, // handled by pnp init
		},
		{
			Name:     "gitops repo",
			Critical: true,
			Run: func() (bool, string) {
				cfg, err := config.LoadGlobalConfig()
				if err != nil || cfg.GitopsRepo == "" {
					return false, "gitopsRepo not configured in ~/.pnp.yaml"
				}
				if _, err := os.Stat(cfg.GitopsRepo); os.IsNotExist(err) {
					return false, fmt.Sprintf("gitops repo not found at %s", cfg.GitopsRepo)
				}
				return true, fmt.Sprintf("gitops repo found at %s", cfg.GitopsRepo)
			},
			Fix: func() error {
				cfg, err := config.LoadGlobalConfig()
				if err != nil || cfg.GitopsRemote == "" {
					return fmt.Errorf("gitopsRemote not set in ~/.pnp.yaml — run 'pnp init' first")
				}
				if cfg.GitopsRepo == "" {
					return fmt.Errorf("gitopsRepo path not set in ~/.pnp.yaml — run 'pnp init' first")
				}
				fmt.Printf("Cloning %s to %s...\n", cfg.GitopsRemote, cfg.GitopsRepo)
				cmd := exec.Command("git", "clone", cfg.GitopsRemote, cfg.GitopsRepo)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				return cmd.Run()
			},
		},
	}
}

func runBrew(args ...string) error {
	if _, err := exec.LookPath("brew"); err != nil {
		return fmt.Errorf("homebrew is not installed — install manually")
	}
	cmd := exec.Command("brew", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

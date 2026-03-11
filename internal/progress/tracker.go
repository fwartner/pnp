package progress

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	spinnerChars = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	checkMark    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42")).Render("✓")
	crossMark    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196")).Render("✗")
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

// Step represents a single tracked operation.
type Step struct {
	Name   string
	Action func() error
}

// Tracker runs a sequence of steps with visual progress indicators.
type Tracker struct {
	Steps []Step
}

// NewTracker creates a Tracker with the given steps.
func NewTracker(steps ...Step) *Tracker {
	return &Tracker{Steps: steps}
}

// Run executes all steps sequentially, printing a spinner while each runs
// and a check/cross mark on completion.
func (t *Tracker) Run() error {
	for i, step := range t.Steps {
		prefix := fmt.Sprintf("[%d/%d]", i+1, len(t.Steps))

		// Start spinner in background
		done := make(chan error, 1)
		go func() {
			done <- step.Action()
		}()

		// Animate spinner until step completes
		tick := 0
		ticker := time.NewTicker(80 * time.Millisecond)
		var err error

	loop:
		for {
			select {
			case err = <-done:
				ticker.Stop()
				break loop
			case <-ticker.C:
				spinner := spinnerChars[tick%len(spinnerChars)]
				fmt.Printf("\r  %s %s %s", dimStyle.Render(prefix), spinner, step.Name)
				tick++
			}
		}

		if err != nil {
			fmt.Printf("\r  %s %s %s\n", dimStyle.Render(prefix), crossMark, step.Name)
			return fmt.Errorf("%s: %w", step.Name, err)
		}
		elapsed := dimStyle.Render(fmt.Sprintf("(%dms)", tick*80))
		fmt.Printf("\r  %s %s %s %s\n", dimStyle.Render(prefix), checkMark, step.Name, elapsed)
	}
	return nil
}

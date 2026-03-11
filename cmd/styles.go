package cmd

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	successStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	errorStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	warnStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

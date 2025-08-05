package ux

import "github.com/charmbracelet/lipgloss"

// Theme defines the color scheme for the CLI.
type Theme struct {
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Success   lipgloss.Color
	Warning   lipgloss.Color
	Danger    lipgloss.Color
	Info      lipgloss.Color
	Faint     lipgloss.Color
}

// DefaultTheme is the default color scheme for the CLI.
var DefaultTheme = &Theme{
	Primary:   lipgloss.Color("#FF79C6"),
	Secondary: lipgloss.Color("#BD93F9"),
	Success:   lipgloss.Color("#50FA7B"),
	Warning:   lipgloss.Color("#F1FA8C"),
	Danger:    lipgloss.Color("#FF5555"),
	Info:      lipgloss.Color("#8BE9FD"),
	Faint:     lipgloss.Color("#6272A4"),
}

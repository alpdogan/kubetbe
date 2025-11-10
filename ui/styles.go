package ui

import "github.com/charmbracelet/lipgloss"

var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62")).
			Padding(0, 1)

	SelectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Padding(0, 1)

	NormalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Padding(0, 1)

	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	InfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("110")).
			Bold(true)
)

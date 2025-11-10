package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func Tick() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return TickMsg{}
	})
}

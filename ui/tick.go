package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"kubetbe/msg"
)

func Tick() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return TickMsg{}
	})
}

func StartLogLoadTimer(podName string) tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return msg.StartLogLoadMsg{PodName: podName}
	})
}

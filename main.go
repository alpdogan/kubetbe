package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"kubetbe/ui"
)

func main() {
	if len(os.Getenv("DEBUG")) > 0 {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			fmt.Printf("Failed to create log file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
	}

	// Get search term from command line arguments
	searchTerm := ""
	if len(os.Args) > 1 {
		searchTerm = os.Args[1]
	}

	model := ui.InitialModel(searchTerm)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"kubetbe/kubectl"
	"kubetbe/utils"
)

func (m *Model) Init() tea.Cmd {
	// Start namespace watch by default
	m.NamespaceWatch = true
	return tea.Batch(
		kubectl.FetchNamespaces(m.SearchTerm),
		Tick(),
		tea.EnterAltScreen,
	)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		if m.State == "panel_view" {
			// Update panel sizes
			if m.PodsPanel != nil {
				m.PodsPanel.MaxLines = m.Height / 3
			}
			for _, p := range m.LogsPanels {
				p.MaxLines = m.Height / 3
			}
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.Quit = true
			if m.State == "panel_view" {
				// Stop all watch commands
				if m.PodsPanel != nil && m.PodsPanel.UpdateCmd != nil {
					m.PodsPanel.UpdateCmd.Process.Kill()
				}
				for _, p := range m.LogsPanels {
					if p.UpdateCmd != nil {
						p.UpdateCmd.Process.Kill()
					}
				}
			}
			return m, tea.Quit

		case "up", "k":
			if m.State == "namespace_select" {
				m.DeleteConfirmation = "" // Clear delete confirmation on navigation
				if m.Cursor > 0 {
					m.Cursor--
				}
			} else if m.State == "panel_view" {
				if m.ActivePanel == 0 && m.PodsPanel != nil {
					podNames := ParsePodNames(m.PodsPanel.Content)
					if len(podNames) == 0 {
						m.PodCursor = 0
					} else {
						if m.PodCursor > 0 {
							m.PodCursor--
						}
						if m.PodCursor >= len(podNames) {
							m.PodCursor = len(podNames) - 1
						}
					}
					m.PodDeleteConfirmation = ""
				} else if m.ActivePanel > 0 && m.ActivePanel <= len(m.LogsPanels) {
					p := m.LogsPanels[m.ActivePanel-1]
					if p.ScrollPos > 0 {
						p.ScrollPos--
					}
				}
			}

		case "down", "j":
			if m.State == "namespace_select" {
				m.DeleteConfirmation = "" // Clear delete confirmation on navigation
				if m.Cursor < len(m.Namespaces)-1 {
					m.Cursor++
				}
			} else if m.State == "panel_view" {
				if m.ActivePanel == 0 && m.PodsPanel != nil {
					podNames := ParsePodNames(m.PodsPanel.Content)
					if len(podNames) == 0 {
						m.PodCursor = 0
					} else {
						if m.PodCursor < len(podNames)-1 {
							m.PodCursor++
						}
						if m.PodCursor >= len(podNames) {
							m.PodCursor = len(podNames) - 1
						}
					}
					m.PodDeleteConfirmation = ""
				} else if m.ActivePanel > 0 && m.ActivePanel <= len(m.LogsPanels) {
					p := m.LogsPanels[m.ActivePanel-1]
					maxScroll := utils.Max(0, len(p.Content)-p.MaxLines)
					if p.ScrollPos < maxScroll {
						p.ScrollPos++
					}
				}
			}

		case "pgup":
			if m.State == "panel_view" && len(m.LogsPanels) <= 4 {
				if m.ActivePanel == 0 && m.PodsPanel != nil {
					podNames := ParsePodNames(m.PodsPanel.Content)
					if len(podNames) == 0 {
						m.PodCursor = 0
					} else {
						step := utils.Max(1, m.PodsPanel.MaxLines)
						m.PodCursor = utils.Max(0, m.PodCursor-step)
						if m.PodCursor >= len(podNames) {
							m.PodCursor = len(podNames) - 1
						}
					}
					m.PodDeleteConfirmation = ""
				} else if m.ActivePanel > 0 && m.ActivePanel <= len(m.LogsPanels) {
					p := m.LogsPanels[m.ActivePanel-1]
					p.ScrollPos = utils.Max(0, p.ScrollPos-p.MaxLines)
				}
			}

		case "pgdown":
			if m.State == "panel_view" && len(m.LogsPanels) <= 4 {
				if m.ActivePanel == 0 && m.PodsPanel != nil {
					podNames := ParsePodNames(m.PodsPanel.Content)
					if len(podNames) == 0 {
						m.PodCursor = 0
					} else {
						step := utils.Max(1, m.PodsPanel.MaxLines)
						m.PodCursor = utils.Min(len(podNames)-1, m.PodCursor+step)
					}
					m.PodDeleteConfirmation = ""
				} else if m.ActivePanel > 0 && m.ActivePanel <= len(m.LogsPanels) {
					p := m.LogsPanels[m.ActivePanel-1]
					maxScroll := utils.Max(0, len(p.Content)-p.MaxLines)
					p.ScrollPos = utils.Min(maxScroll, p.ScrollPos+p.MaxLines)
				}
			}

		case "home":
			if m.State == "panel_view" && len(m.LogsPanels) <= 4 {
				if m.ActivePanel == 0 && m.PodsPanel != nil {
					podNames := ParsePodNames(m.PodsPanel.Content)
					if len(podNames) == 0 {
						m.PodCursor = 0
					} else {
						m.PodCursor = 0
					}
					m.PodDeleteConfirmation = ""
				} else if m.ActivePanel > 0 && m.ActivePanel <= len(m.LogsPanels) {
					m.LogsPanels[m.ActivePanel-1].ScrollPos = 0
				}
			}

		case "end":
			if m.State == "panel_view" && len(m.LogsPanels) <= 4 {
				if m.ActivePanel == 0 && m.PodsPanel != nil {
					podNames := ParsePodNames(m.PodsPanel.Content)
					if len(podNames) == 0 {
						m.PodCursor = 0
					} else {
						m.PodCursor = len(podNames) - 1
					}
					m.PodDeleteConfirmation = ""
				} else if m.ActivePanel > 0 && m.ActivePanel <= len(m.LogsPanels) {
					p := m.LogsPanels[m.ActivePanel-1]
					p.ScrollPos = utils.Max(0, len(p.Content)-p.MaxLines)
				}
			}

		case "enter":
			if m.State == "namespace_select" && len(m.Namespaces) > 0 {
				m.SelectedNS = m.Namespaces[m.Cursor]
				m.State = "panel_view"
				m.LogPageIndex = 0 // Reset to first page
				m.PodCursor = 0
				m.PodDeleteConfirmation = ""
				m.DeletingPod = ""
				// Initialize pods panel before starting watch
				m.PodsPanel = &Panel{
					Title:    fmt.Sprintf("Pods in %s", m.SelectedNS),
					Content:  []string{"Loading pods..."},
					MaxLines: m.Height / 3,
					Watch:    true,
				}
				return m, tea.Batch(
					kubectl.StartPodsWatch(m.SelectedNS),
					Tick(),
				)
			}

		case "r":
			if m.State == "namespace_select" {
				// Refresh namespace list
				m.DeleteConfirmation = "" // Clear any pending delete confirmation
				return m, tea.Batch(
					kubectl.FetchNamespaces(m.SearchTerm),
					Tick(), // Continue watch
				)
			}

		case "d":
			if m.State == "namespace_select" && len(m.Namespaces) > 0 {
				if m.DeletingNamespace != "" {
					// Already processing a delete; ignore additional delete requests
					break
				}
				selectedNamespace := m.Namespaces[m.Cursor]
				// If already confirming, delete the namespace
				if m.DeleteConfirmation == selectedNamespace {
					// Actually delete the namespace
					m.DeletingNamespace = selectedNamespace
					m.DeleteConfirmation = ""
					return m, kubectl.DeleteNamespace(selectedNamespace)
				} else {
					// Ask for confirmation
					m.DeleteConfirmation = selectedNamespace
				}
			} else if m.State == "panel_view" && m.PodsPanel != nil {
				if m.DeletingPod != "" {
					// Already processing a delete; ignore additional requests
					break
				}
				podNames := ParsePodNames(m.PodsPanel.Content)
				if len(podNames) == 0 {
					m.PodCursor = 0
					break
				}

				var selectedPod string

				// If a logs panel is active, use its pod name as the selected pod
				if m.ActivePanel > 0 && m.ActivePanel <= len(m.LogsPanels) {
					logPanel := m.LogsPanels[m.ActivePanel-1]
					selectedPod = strings.TrimPrefix(logPanel.Title, "Logs: ")
					if selectedPod != "" {
						for i, name := range podNames {
							if name == selectedPod {
								m.PodCursor = i
								break
							}
						}
					}
				}

				// Fallback to cursor-controlled selection
				if selectedPod == "" {
					if m.PodCursor < 0 {
						m.PodCursor = 0
					}
					if m.PodCursor >= len(podNames) {
						m.PodCursor = len(podNames) - 1
					}
					selectedPod = podNames[m.PodCursor]
				}

				if selectedPod == "" {
					break
				}
				if m.PodDeleteConfirmation == selectedPod {
					m.DeletingPod = selectedPod
					m.PodDeleteConfirmation = ""
					return m, kubectl.DeletePod(m.SelectedNS, selectedPod)
				}
				m.PodDeleteConfirmation = selectedPod
			}

		case "tab":
			if m.State == "panel_view" {
				if m.ActivePanel != 0 {
					m.PodDeleteConfirmation = ""
				}
				m.ActivePanel = (m.ActivePanel + 1) % (1 + len(m.LogsPanels))
			}

		case "shift+tab":
			if m.State == "panel_view" {
				if m.ActivePanel != 0 {
					m.PodDeleteConfirmation = ""
				}
				m.ActivePanel = (m.ActivePanel - 1 + 1 + len(m.LogsPanels)) % (1 + len(m.LogsPanels))
			}

		case "b":
			if m.State == "panel_view" {
				m.State = "namespace_select"
				m.LogPageIndex = 0 // Reset to first page
				m.PodCursor = 0
				m.PodDeleteConfirmation = ""
				m.DeletingPod = ""
				// Stop all watch commands
				if m.PodsPanel != nil && m.PodsPanel.UpdateCmd != nil {
					m.PodsPanel.UpdateCmd.Process.Kill()
					m.PodsPanel.UpdateCmd = nil
				}
				for _, p := range m.LogsPanels {
					if p.UpdateCmd != nil {
						p.UpdateCmd.Process.Kill()
						p.UpdateCmd = nil
					}
				}
				m.PodsPanel = nil
				m.LogsPanels = []*Panel{}
				m.ActivePanel = 0
				return m, tea.Batch(
					kubectl.FetchNamespaces(m.SearchTerm),
					Tick(), // Continue namespace watch
				)
			}

		// Clear delete confirmation on any other key press (except d)
		default:
			if m.State == "namespace_select" && m.DeleteConfirmation != "" {
				// Only clear if it's not a navigation key we already handle
				if msg.String() != "d" && msg.String() != "enter" && msg.String() != "q" && msg.String() != "r" {
					m.DeleteConfirmation = ""
				}
			}
			if m.State == "panel_view" && m.PodDeleteConfirmation != "" {
				if msg.String() != "d" && msg.String() != "tab" && msg.String() != "shift+tab" {
					m.PodDeleteConfirmation = ""
				}
			}
		}

	case NamespaceListMsg:
		m.Namespaces = msg.Namespaces
		if len(m.Namespaces) > 0 && m.Cursor >= len(m.Namespaces) {
			m.Cursor = len(m.Namespaces) - 1
		}

	case ErrorMsg:
		m.Err = msg.Err

	case NamespaceDeleteMsg:
		m.DeleteConfirmation = "" // Clear confirmation after delete attempt
		if m.DeletingNamespace == msg.Namespace {
			m.DeletingNamespace = ""
		}
		if msg.Err != nil {
			m.Err = msg.Err
		} else {
			// Successfully deleted, refresh namespace list
			return m, tea.Batch(
				kubectl.FetchNamespaces(m.SearchTerm),
				Tick(), // Continue namespace watch
			)
		}

	case PodDeleteMsg:
		m.PodDeleteConfirmation = ""
		if m.DeletingPod == msg.Pod {
			m.DeletingPod = ""
		}
		if msg.Err != nil {
			m.Err = msg.Err
		}

	case PodUpdateMsg:
		// Ensure podsPanel exists
		if m.PodsPanel == nil {
			m.PodsPanel = &Panel{
				Title:    fmt.Sprintf("Pods in %s", m.SelectedNS),
				Content:  []string{},
				MaxLines: m.Height / 3,
				Watch:    true,
			}
		}
		// CRITICAL: Limit pods panel content to prevent overflow
		// podsPanelHeight is 12, so max content lines = 12 - 5 (border/padding/title) = 7
		// Allow more content since we have scroll support
		podsContent := msg.Content
		// Don't limit content - let scroll handle it
		// But keep a reasonable limit to prevent memory issues (max 100 pods)
		if len(podsContent) > 100 {
			podsContent = podsContent[:100]
		}

		// Preserve scroll position if content length is similar
		oldContentLen := len(m.PodsPanel.Content)
		m.PodsPanel.Content = podsContent
		// Only reset scroll if content changed significantly
		if oldContentLen == 0 || utils.Abs(oldContentLen-len(podsContent)) > 5 {
			m.PodsPanel.ScrollPos = 0
		}
		// Ensure scroll position is valid
		if m.PodsPanel.ScrollPos > utils.Max(0, len(podsContent)-m.PodsPanel.MaxLines) {
			m.PodsPanel.ScrollPos = utils.Max(0, len(podsContent)-m.PodsPanel.MaxLines)
		}
		m.Err = msg.Err

		// Parse pods and start log watching for each
		if msg.Err == nil {
			podNames := ParsePodNames(msg.Content)
			if len(podNames) == 0 {
				m.PodCursor = 0
				m.PodDeleteConfirmation = ""
			} else {
				if m.PodCursor >= len(podNames) {
					m.PodCursor = len(podNames) - 1
				}
				if m.PodCursor < 0 {
					m.PodCursor = 0
				}
				if m.PodDeleteConfirmation != "" {
					found := false
					for _, name := range podNames {
						if name == m.PodDeleteConfirmation {
							found = true
							break
						}
					}
					if !found {
						m.PodDeleteConfirmation = ""
					}
				}
			}

			oldPodCount := len(m.LogsPanels)
			m.LogsPanels = UpdateLogsPanels(m.LogsPanels, podNames, m.SelectedNS)

			// Start log fetching for new pods
			if len(m.LogsPanels) > oldPodCount {
				var cmds []tea.Cmd
				for i := oldPodCount; i < len(m.LogsPanels); i++ {
					// Extract pod name from title
					title := m.LogsPanels[i].Title
					podName := strings.TrimPrefix(title, "Logs: ")
					cmds = append(cmds, kubectl.StartLogWatch(podName, m.SelectedNS))
				}
				return m, tea.Batch(cmds...)
			}
		}

	case LogUpdateMsg:
		if msg.Err == nil {
			for i, p := range m.LogsPanels {
				if strings.Contains(p.Title, msg.PodName) {
					// Store all log content - renderPanel will handle truncation based on maxHeight
					// This allows scrolling through more logs
					m.LogsPanels[i].Content = msg.Content
					// Scroll to end (show latest logs) by default
					// But allow user to scroll up to see older logs
					if len(msg.Content) > m.LogsPanels[i].MaxLines {
						m.LogsPanels[i].ScrollPos = len(msg.Content) - m.LogsPanels[i].MaxLines
					} else {
						m.LogsPanels[i].ScrollPos = 0
					}
					break
				}
			}
		}

	case TickMsg:
		var cmds []tea.Cmd

		// Refresh namespace list if watching
		if m.State == "namespace_select" && m.NamespaceWatch {
			cmds = append(cmds, kubectl.FetchNamespaces(m.SearchTerm))
		}

		// Refresh pods and logs if in panel view
		if m.State == "panel_view" && m.PodsPanel != nil && m.PodsPanel.Watch {
			cmds = append(cmds, kubectl.StartPodsWatch(m.SelectedNS))

			// Also refresh logs for all pods
			for _, logPanel := range m.LogsPanels {
				if logPanel.Watch {
					podName := strings.TrimPrefix(logPanel.Title, "Logs: ")
					cmds = append(cmds, kubectl.StartLogWatch(podName, m.SelectedNS))
				}
			}
		}

		if len(cmds) > 0 {
			return m, tea.Batch(append(cmds, Tick())...)
		}
		return m, nil
	}

	return m, nil
}

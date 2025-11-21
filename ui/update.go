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
		if m.State == "namespace_select" {
			m.updateNamespacePagination()
		}
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
		if m.State == "namespace_select" && m.ServiceIPInputActive {
			handled := true
			switch msg.Type {
			case tea.KeyEnter:
				ip := strings.TrimSpace(m.ServiceIPQuery)
				if ip == "" {
					m.ServiceIPErr = fmt.Errorf("please enter an IP address")
					m.ServiceIPResult = nil
					m.ServiceIPSearching = false
				} else {
					m.ServiceIPQuery = ip
					m.ServiceIPSearching = true
					m.ServiceIPErr = nil
					m.ServiceIPResult = nil
					m.ServiceIPInputActive = false
					return m, kubectl.FindServiceByIP(ip)
				}
			case tea.KeyEscape:
				m.ServiceIPInputActive = false
				m.ServiceIPErr = nil
				m.ServiceIPResult = nil
				m.ServiceIPSearching = false
				m.ServiceIPQuery = ""
			case tea.KeyBackspace, tea.KeyDelete:
				if len(m.ServiceIPQuery) > 0 {
					runes := []rune(m.ServiceIPQuery)
					m.ServiceIPQuery = string(runes[:len(runes)-1])
				}
			case tea.KeyRunes:
				m.ServiceIPQuery += string(msg.Runes)
			default:
				handled = false
			}
			if handled {
				return m, nil
			}
		}

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
				m.moveNamespaceCursor(-1)
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
				} else if m.DescribePanel != nil && m.ActivePanel == 1 {
					if m.DescribePanel.ScrollPos > 0 {
						m.DescribePanel.ScrollPos--
					}
				} else {
					logIndex := m.activeLogPanelIndex()
					if logIndex >= 0 && logIndex < len(m.LogsPanels) {
						p := m.LogsPanels[logIndex]
						if p.ScrollPos > 0 {
							p.ScrollPos--
						}
					}
				}
			}

		case "down", "j":
			if m.State == "namespace_select" {
				m.DeleteConfirmation = "" // Clear delete confirmation on navigation
				m.moveNamespaceCursor(1)
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
				} else if m.DescribePanel != nil && m.ActivePanel == 1 {
					maxScroll := utils.Max(0, len(m.DescribePanel.Content)-m.DescribePanel.MaxLines)
					if m.DescribePanel.ScrollPos < maxScroll {
						m.DescribePanel.ScrollPos++
					}
				} else {
					logIndex := m.activeLogPanelIndex()
					if logIndex >= 0 && logIndex < len(m.LogsPanels) {
						p := m.LogsPanels[logIndex]
						maxScroll := utils.Max(0, len(p.Content)-p.MaxLines)
						if p.ScrollPos < maxScroll {
							p.ScrollPos++
						}
					}
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
				m.DescribePanel = nil
				m.DescribeTarget = ""
				m.ServiceIPInputActive = false
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

		case "esc":
			if m.State == "namespace_select" {
				m.ServiceIPInputActive = false
				m.ServiceIPSearching = false
				m.ServiceIPErr = nil
				m.ServiceIPResult = nil
				m.ServiceIPQuery = ""
			}

		case "f":
			if m.State == "namespace_select" {
				m.ServiceIPInputActive = true
				m.ServiceIPSearching = false
				m.ServiceIPErr = nil
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
				selectedPod, podNames := m.selectedPodAndList()
				if len(podNames) == 0 || selectedPod == "" {
					m.PodCursor = 0
					break
				}
				if m.PodDeleteConfirmation == selectedPod {
					m.DeletingPod = selectedPod
					m.PodDeleteConfirmation = ""
					return m, kubectl.DeletePod(m.SelectedNS, selectedPod)
				}
				m.PodDeleteConfirmation = selectedPod
			}

		case "i":
			if m.State == "panel_view" && m.PodsPanel != nil {
				selectedPod, podNames := m.selectedPodAndList()
				if len(podNames) == 0 || selectedPod == "" {
					m.PodCursor = 0
					break
				}

				// Toggle off if describe already showing for selected pod
				if m.DescribePanel != nil && m.DescribeTarget == selectedPod {
					m.DescribePanel = nil
					m.DescribeTarget = ""
					if len(m.LogsPanels) == 0 {
						m.ActivePanel = 0
					} else {
						if m.ActivePanel > 1 {
							m.ActivePanel--
						} else if m.ActivePanel == 1 {
							m.ActivePanel = 0
						}
					}
					break
				}

				m.DescribeTarget = selectedPod
				m.DescribePanel = &Panel{
					Title:     fmt.Sprintf("Describe: %s", selectedPod),
					Content:   []string{"Fetching describe..."},
					MaxLines:  m.Height / 3,
					ScrollPos: 0,
					Watch:     false,
				}
				m.ActivePanel = 1
				return m, kubectl.DescribePod(m.SelectedNS, selectedPod)
			}

		case "tab":
			if m.State == "namespace_select" {
				m.moveNamespaceCursor(1)
			} else if m.State == "panel_view" {
				if m.DescribePanel != nil {
					if len(m.LogsPanels) == 0 {
						m.DescribePanel = nil
						m.DescribeTarget = ""
						m.ActivePanel = 0
						break
					}
					if m.ActivePanel == 1 {
						m.DescribePanel = nil
						m.DescribeTarget = ""
						m.ActivePanel = 1
						break
					}
				}
				totalPanels := m.totalPanelCount()
				if totalPanels <= 0 {
					break
				}
				if m.ActivePanel != 0 {
					m.PodDeleteConfirmation = ""
				}
				nextPanel := (m.ActivePanel + 1) % totalPanels
				m.ActivePanel = nextPanel

				// Lazy load: if switching to a log panel that doesn't exist yet, create it
				if nextPanel > 0 {
					var targetPodName string
					if m.DescribePanel != nil {
						if nextPanel == 1 {
							// Describe panel, skip
							break
						}
						targetPodIndex := nextPanel - 2
						if targetPodIndex >= 0 && targetPodIndex < len(m.AvailablePods) {
							targetPodName = m.AvailablePods[targetPodIndex]
						}
					} else {
						targetPodIndex := nextPanel - 1
						if targetPodIndex >= 0 && targetPodIndex < len(m.AvailablePods) {
							targetPodName = m.AvailablePods[targetPodIndex]
						}
					}

					if targetPodName != "" {
						// Check if log panel already exists
						exists := false
						for _, p := range m.LogsPanels {
							if strings.TrimPrefix(p.Title, "Logs: ") == targetPodName {
								exists = true
								break
							}
						}

						if !exists {
							// Create new log panel and start timer for delayed log loading (3 seconds)
							newPanel := &Panel{
								Title:     "Logs: " + targetPodName,
								Content:   []string{"Loading logs..."},
								MaxLines:  20,
								ScrollPos: 0,
								Watch:     true,
							}
							m.LogsPanels = append(m.LogsPanels, newPanel)
							m.PendingLogLoad = targetPodName
							return m, StartLogLoadTimer(targetPodName)
						}
					}
				}
			}

		case "shift+tab":
			if m.State == "namespace_select" {
				m.moveNamespaceCursor(-1)
			} else if m.State == "panel_view" {
				if m.DescribePanel != nil && m.ActivePanel == 1 {
					m.DescribePanel = nil
					m.DescribeTarget = ""
					m.ActivePanel = 0
					break
				}
				totalPanels := m.totalPanelCount()
				if totalPanels <= 0 {
					break
				}
				if m.ActivePanel != 0 {
					m.PodDeleteConfirmation = ""
				}
				nextPanel := (m.ActivePanel - 1 + totalPanels) % totalPanels
				m.ActivePanel = nextPanel

				// Lazy load: if switching to a log panel that doesn't exist yet, create it
				if nextPanel > 0 {
					var targetPodName string
					if m.DescribePanel != nil {
						if nextPanel == 1 {
							// Describe panel, skip
							break
						}
						targetPodIndex := nextPanel - 2
						if targetPodIndex >= 0 && targetPodIndex < len(m.AvailablePods) {
							targetPodName = m.AvailablePods[targetPodIndex]
						}
					} else {
						targetPodIndex := nextPanel - 1
						if targetPodIndex >= 0 && targetPodIndex < len(m.AvailablePods) {
							targetPodName = m.AvailablePods[targetPodIndex]
						}
					}

					if targetPodName != "" {
						// Check if log panel already exists
						exists := false
						for _, p := range m.LogsPanels {
							if strings.TrimPrefix(p.Title, "Logs: ") == targetPodName {
								exists = true
								break
							}
						}

						if !exists {
							// Create new log panel and start timer for delayed log loading (3 seconds)
							newPanel := &Panel{
								Title:     "Logs: " + targetPodName,
								Content:   []string{"Loading logs..."},
								MaxLines:  20,
								ScrollPos: 0,
								Watch:     true,
							}
							m.LogsPanels = append(m.LogsPanels, newPanel)
							m.PendingLogLoad = targetPodName
							return m, StartLogLoadTimer(targetPodName)
						}
					}
				}
			}

		case "right", "l":
			if m.State == "namespace_select" {
				m.changeNamespacePage(1)
			}

		case "left", "h":
			if m.State == "namespace_select" {
				m.changeNamespacePage(-1)
			}

		case "b":
			if m.State == "panel_view" {
				m.State = "namespace_select"
				m.LogPageIndex = 0 // Reset to first page
				m.PodCursor = 0
				m.PodDeleteConfirmation = ""
				m.DeletingPod = ""
				m.DescribePanel = nil
				m.DescribeTarget = ""
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
		m.updateNamespacePagination()

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

	case PodDescribeMsg:
		if msg.Err != nil {
			m.Err = msg.Err
		}
		if m.DescribePanel != nil && m.DescribeTarget == msg.Pod {
			m.DescribePanel.Content = msg.Content
			m.DescribePanel.ScrollPos = 0
		}

	case ServiceLookupMsg:
		m.ServiceIPSearching = false
		if msg.Err != nil {
			m.ServiceIPErr = msg.Err
			m.ServiceIPResult = nil
		} else {
			m.ServiceIPErr = nil
			if len(msg.Result) == 0 {
				m.ServiceIPResult = []string{fmt.Sprintf("No service found for IP %s", msg.IP)}
			} else {
				m.ServiceIPResult = msg.Result
			}
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
				if m.DescribePanel != nil {
					m.DescribePanel = nil
					m.DescribeTarget = ""
					if m.ActivePanel > 0 {
						m.ActivePanel = utils.Min(m.ActivePanel, 1+len(m.LogsPanels))
					}
				}
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
				if m.DescribePanel != nil {
					found := false
					for _, name := range podNames {
						if name == m.DescribeTarget {
							found = true
							break
						}
					}
					if !found {
						m.DescribePanel = nil
						m.DescribeTarget = ""
						if len(m.LogsPanels) > 0 {
							if m.ActivePanel > 1 {
								m.ActivePanel--
							} else if m.ActivePanel == 1 {
								m.ActivePanel = 0
							}
						} else {
							m.ActivePanel = 0
						}
					}
				}
			}

			// Update available pods list, but don't create log panels yet
			// Log panels will be created lazily when user navigates to them
			m.AvailablePods = podNames

			// Clean up log panels for pods that no longer exist
			var validLogPanels []*Panel
			for _, p := range m.LogsPanels {
				podName := strings.TrimPrefix(p.Title, "Logs: ")
				found := false
				for _, name := range podNames {
					if name == podName {
						found = true
						break
					}
				}
				if found {
					validLogPanels = append(validLogPanels, p)
				} else {
					// Stop watching logs for deleted pods
					if p.UpdateCmd != nil {
						p.UpdateCmd.Process.Kill()
						p.UpdateCmd = nil
					}
				}
			}
			m.LogsPanels = validLogPanels
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

	case StartLogLoadMsg:
		// 3 seconds have passed, start loading logs for the pending pod
		if m.PendingLogLoad == msg.PodName {
			m.PendingLogLoad = ""
			return m, kubectl.StartLogWatch(msg.PodName, m.SelectedNS)
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

			// Only refresh logs for panels that are already loaded (lazy loading)
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

func (m *Model) activeLogPanelIndex() int {
	if len(m.LogsPanels) == 0 {
		return -1
	}
	if m.DescribePanel != nil {
		if m.ActivePanel <= 1 {
			return -1
		}
		idx := m.ActivePanel - 2
		if idx >= 0 && idx < len(m.LogsPanels) {
			return idx
		}
		if len(m.LogsPanels) > 0 {
			return 0
		}
		return -1
	}
	if m.ActivePanel <= 0 {
		return -1
	}
	idx := m.ActivePanel - 1
	if idx >= 0 && idx < len(m.LogsPanels) {
		return idx
	}
	if len(m.LogsPanels) > 0 {
		return 0
	}
	return -1
}

func (m *Model) selectedPodAndList() (string, []string) {
	if m.PodsPanel == nil {
		return "", nil
	}
	podNames := ParsePodNames(m.PodsPanel.Content)
	if len(podNames) == 0 {
		m.PodCursor = 0
		return "", podNames
	}

	candidate := ""
	if m.DescribePanel != nil && m.ActivePanel == 1 && m.DescribeTarget != "" {
		candidate = m.DescribeTarget
	} else {
		logIndex := m.activeLogPanelIndex()
		if logIndex >= 0 && logIndex < len(m.LogsPanels) {
			title := m.LogsPanels[logIndex].Title
			candidate = strings.TrimPrefix(title, "Logs: ")
		}
	}
	if candidate != "" {
		for i, name := range podNames {
			if name == candidate {
				m.PodCursor = i
				return candidate, podNames
			}
		}
	}

	if m.PodCursor < 0 {
		m.PodCursor = 0
	}
	if m.PodCursor >= len(podNames) {
		m.PodCursor = len(podNames) - 1
	}
	return podNames[m.PodCursor], podNames
}

func (m *Model) totalPanelCount() int {
	// Count: pods panel (1) + describe panel (if exists) + all available pods
	count := 1 + len(m.AvailablePods)
	if m.DescribePanel != nil {
		count++
	}
	return count
}

func (m *Model) visibleNamespacesPerPage() int {
	// fixed max 10 per page, adjust for very small terminals
	lines := m.Height - 8
	if lines < 3 {
		lines = 3
	}
	if lines > 10 {
		lines = 10
	}
	return lines
}

func (m *Model) updateNamespacePagination() {
	perPage := m.visibleNamespacesPerPage()
	if perPage <= 0 {
		perPage = 1
	}
	total := len(m.Namespaces)
	if total == 0 {
		m.NSTotalPages = 1
		m.NSCurrentPage = 0
		m.Cursor = 0
		return
	}

	m.NSTotalPages = (total + perPage - 1) / perPage
	if m.NSTotalPages < 1 {
		m.NSTotalPages = 1
	}

	if m.Cursor >= total {
		m.Cursor = total - 1
	}
	m.NSCurrentPage = m.Cursor / perPage
}

func (m *Model) moveNamespaceCursor(delta int) {
	total := len(m.Namespaces)
	if total == 0 {
		m.Cursor = 0
		m.NSCurrentPage = 0
		m.NSTotalPages = 1
		return
	}
	m.Cursor += delta
	if m.Cursor < 0 {
		m.Cursor = 0
	} else if m.Cursor >= total {
		m.Cursor = total - 1
	}
	m.updateNamespacePagination()
}

func (m *Model) changeNamespacePage(delta int) {
	total := len(m.Namespaces)
	if total == 0 {
		return
	}
	perPage := m.visibleNamespacesPerPage()
	m.NSCurrentPage += delta
	if m.NSCurrentPage < 0 {
		m.NSCurrentPage = 0
	} else if m.NSCurrentPage >= m.NSTotalPages {
		m.NSCurrentPage = m.NSTotalPages - 1
	}
	start := m.NSCurrentPage * perPage
	if start >= total {
		start = total - 1
	}
	if start < 0 {
		start = 0
	}
	m.Cursor = start
	m.DeleteConfirmation = ""
	m.updateNamespacePagination()
}

func (m *Model) jumpNamespaceToStart() {
	if len(m.Namespaces) == 0 {
		return
	}
	m.Cursor = 0
	m.updateNamespacePagination()
}

func (m *Model) jumpNamespaceToEnd() {
	total := len(m.Namespaces)
	if total == 0 {
		return
	}
	m.Cursor = total - 1
	m.updateNamespacePagination()
}

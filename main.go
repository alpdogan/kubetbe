package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62")).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Padding(0, 1)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Padding(0, 1)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
)

type model struct {
	state        string // "namespace_select", "panel_view"
	namespaces   []string
	cursor       int
	selectedNS   string
	podsPanel    *panel
	logsPanels   []*panel
	activePanel  int
	logPageIndex int // Current page index for log panels (0-based, 4 panels per page)
	width        int
	height       int
	err          error
	quit         bool
	searchTerm   string // Search term for namespace filtering
}

type panel struct {
	title     string
	content   []string
	maxLines  int
	scrollPos int
	updateCmd *exec.Cmd
	watch     bool
}

type tickMsg time.Time

type podUpdateMsg struct {
	content []string
	err     error
}

type logUpdateMsg struct {
	podName string
	content []string
	err     error
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func initialModel(searchTerm string) model {
	return model{
		state:       "namespace_select",
		namespaces:  []string{},
		cursor:      0,
		podsPanel:   nil,
		logsPanels:  []*panel{},
		activePanel: 0,
		searchTerm:  searchTerm,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		fetchNamespaces(m.searchTerm),
		tea.EnterAltScreen,
	)
}

func fetchNamespaces(searchTerm string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd

		// If search term is provided, filter namespaces
		if searchTerm != "" {
			// Use grep with proper escaping to avoid shell injection
			cmd = exec.Command("sh", "-c", fmt.Sprintf("kubens | grep -i '%s'", strings.ReplaceAll(searchTerm, "'", "'\"'\"'")))
		} else {
			// Otherwise, get all namespaces
			cmd = exec.Command("kubens")
		}

		output, err := cmd.Output()
		if err != nil {
			// If grep doesn't find anything, it returns error but that's okay
			if searchTerm != "" {
				// Return empty list for grep not found
				return namespaceListMsg{namespaces: []string{}}
			}
			return errorMsg{err: fmt.Errorf("failed to run kubens command: %v", err)}
		}

		namespaces := []string{}
		scanner := bufio.NewScanner(strings.NewReader(string(output)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				namespaces = append(namespaces, line)
			}
		}

		return namespaceListMsg{namespaces: namespaces}
	}
}

type namespaceListMsg struct {
	namespaces []string
}

type errorMsg struct {
	err error
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.state == "panel_view" {
			// Update panel sizes
			if m.podsPanel != nil {
				m.podsPanel.maxLines = m.height / 3
			}
			for _, p := range m.logsPanels {
				p.maxLines = m.height / 3
			}
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quit = true
			if m.state == "panel_view" {
				// Stop all watch commands
				if m.podsPanel != nil && m.podsPanel.updateCmd != nil {
					m.podsPanel.updateCmd.Process.Kill()
				}
				for _, p := range m.logsPanels {
					if p.updateCmd != nil {
						p.updateCmd.Process.Kill()
					}
				}
			}
			return m, tea.Quit

		case "up", "k":
			if m.state == "namespace_select" {
				if m.cursor > 0 {
					m.cursor--
				}
			} else if m.state == "panel_view" {
				if m.activePanel == 0 && m.podsPanel != nil {
					if m.podsPanel.scrollPos > 0 {
						m.podsPanel.scrollPos--
					}
				} else if m.activePanel > 0 && m.activePanel <= len(m.logsPanels) {
					p := m.logsPanels[m.activePanel-1]
					if p.scrollPos > 0 {
						p.scrollPos--
					}
				}
			}

		case "down", "j":
			if m.state == "namespace_select" {
				if m.cursor < len(m.namespaces)-1 {
					m.cursor++
				}
			} else if m.state == "panel_view" {
				if m.activePanel == 0 && m.podsPanel != nil {
					p := m.podsPanel
					maxScroll := max(0, len(p.content)-p.maxLines)
					if p.scrollPos < maxScroll {
						p.scrollPos++
					}
				} else if m.activePanel > 0 && m.activePanel <= len(m.logsPanels) {
					p := m.logsPanels[m.activePanel-1]
					maxScroll := max(0, len(p.content)-p.maxLines)
					if p.scrollPos < maxScroll {
						p.scrollPos++
					}
				}
			}

		case "pgup":
			if m.state == "panel_view" && len(m.logsPanels) <= 4 {
				if m.activePanel == 0 && m.podsPanel != nil {
					p := m.podsPanel
					p.scrollPos = max(0, p.scrollPos-p.maxLines)
				} else if m.activePanel > 0 && m.activePanel <= len(m.logsPanels) {
					p := m.logsPanels[m.activePanel-1]
					p.scrollPos = max(0, p.scrollPos-p.maxLines)
				}
			}

		case "pgdown":
			if m.state == "panel_view" && len(m.logsPanels) <= 4 {
				if m.activePanel == 0 && m.podsPanel != nil {
					p := m.podsPanel
					maxScroll := max(0, len(p.content)-p.maxLines)
					p.scrollPos = min(maxScroll, p.scrollPos+p.maxLines)
				} else if m.activePanel > 0 && m.activePanel <= len(m.logsPanels) {
					p := m.logsPanels[m.activePanel-1]
					maxScroll := max(0, len(p.content)-p.maxLines)
					p.scrollPos = min(maxScroll, p.scrollPos+p.maxLines)
				}
			}

		case "home":
			if m.state == "panel_view" && len(m.logsPanels) <= 4 {
				if m.activePanel == 0 && m.podsPanel != nil {
					m.podsPanel.scrollPos = 0
				} else if m.activePanel > 0 && m.activePanel <= len(m.logsPanels) {
					m.logsPanels[m.activePanel-1].scrollPos = 0
				}
			}

		case "end":
			if m.state == "panel_view" && len(m.logsPanels) <= 4 {
				if m.activePanel == 0 && m.podsPanel != nil {
					p := m.podsPanel
					p.scrollPos = max(0, len(p.content)-p.maxLines)
				} else if m.activePanel > 0 && m.activePanel <= len(m.logsPanels) {
					p := m.logsPanels[m.activePanel-1]
					p.scrollPos = max(0, len(p.content)-p.maxLines)
				}
			}

		case "enter":
			if m.state == "namespace_select" && len(m.namespaces) > 0 {
				m.selectedNS = m.namespaces[m.cursor]
				m.state = "panel_view"
				m.logPageIndex = 0 // Reset to first page
				// Initialize pods panel before starting watch
				m.podsPanel = &panel{
					title:    fmt.Sprintf("Pods in %s", m.selectedNS),
					content:  []string{"Loading pods..."},
					maxLines: m.height / 3,
					watch:    true,
				}
				return m, tea.Batch(
					startPodsWatch(m.selectedNS),
					tick(),
				)
			}

		case "tab":
			if m.state == "panel_view" {
				m.activePanel = (m.activePanel + 1) % (1 + len(m.logsPanels))
			}

		case "shift+tab":
			if m.state == "panel_view" {
				m.activePanel = (m.activePanel - 1 + 1 + len(m.logsPanels)) % (1 + len(m.logsPanels))
			}

		case "b":
			if m.state == "panel_view" {
				m.state = "namespace_select"
				m.logPageIndex = 0 // Reset to first page
				// Stop all watch commands
				if m.podsPanel != nil && m.podsPanel.updateCmd != nil {
					m.podsPanel.updateCmd.Process.Kill()
					m.podsPanel.updateCmd = nil
				}
				for _, p := range m.logsPanels {
					if p.updateCmd != nil {
						p.updateCmd.Process.Kill()
						p.updateCmd = nil
					}
				}
				m.podsPanel = nil
				m.logsPanels = []*panel{}
				m.activePanel = 0
				return m, fetchNamespaces(m.searchTerm)
			}
		}

	case namespaceListMsg:
		m.namespaces = msg.namespaces
		if len(m.namespaces) > 0 && m.cursor >= len(m.namespaces) {
			m.cursor = len(m.namespaces) - 1
		}

	case errorMsg:
		m.err = msg.err

	case podUpdateMsg:
		// Ensure podsPanel exists
		if m.podsPanel == nil {
			m.podsPanel = &panel{
				title:    fmt.Sprintf("Pods in %s", m.selectedNS),
				content:  []string{},
				maxLines: m.height / 3,
				watch:    true,
			}
		}
		// CRITICAL: Limit pods panel content to prevent overflow
		// podsPanelHeight is 9, so max content lines = 9 - 5 (border/padding/title) = 4
		// Keep it very small to ensure pods panel always fits and stays visible
		podsContent := msg.content
		if len(podsContent) > 5 {
			// Keep only first 5 lines (header + 3-4 pod entries max)
			podsContent = podsContent[:5]
		}

		// Preserve scroll position if content length is similar
		oldContentLen := len(m.podsPanel.content)
		m.podsPanel.content = podsContent
		// Only reset scroll if content changed significantly
		if oldContentLen == 0 || abs(oldContentLen-len(podsContent)) > 5 {
			m.podsPanel.scrollPos = 0
		}
		// Ensure scroll position is valid
		if m.podsPanel.scrollPos > max(0, len(podsContent)-m.podsPanel.maxLines) {
			m.podsPanel.scrollPos = max(0, len(podsContent)-m.podsPanel.maxLines)
		}
		m.err = msg.err

		// Parse pods and start log watching for each
		if msg.err == nil {
			oldPodCount := len(m.logsPanels)
			podNames := parsePodNames(msg.content)
			m.logsPanels = updateLogsPanels(m.logsPanels, podNames, m.selectedNS)

			// Start log fetching for new pods
			if len(m.logsPanels) > oldPodCount {
				var cmds []tea.Cmd
				for i := oldPodCount; i < len(m.logsPanels); i++ {
					// Extract pod name from title
					title := m.logsPanels[i].title
					podName := strings.TrimPrefix(title, "Logs: ")
					cmds = append(cmds, startLogWatch(podName, m.selectedNS))
				}
				return m, tea.Batch(cmds...)
			}
		}

	case logUpdateMsg:
		if msg.err == nil {
			for i, p := range m.logsPanels {
				if strings.Contains(p.title, msg.podName) {
					// Store all log content - renderPanel will handle truncation based on maxHeight
					// This allows scrolling through more logs
					m.logsPanels[i].content = msg.content
					// Scroll to end (show latest logs) by default
					// But allow user to scroll up to see older logs
					if len(msg.content) > m.logsPanels[i].maxLines {
						m.logsPanels[i].scrollPos = len(msg.content) - m.logsPanels[i].maxLines
					} else {
						m.logsPanels[i].scrollPos = 0
					}
					break
				}
			}
		}

	case tickMsg:
		if m.state == "panel_view" && m.podsPanel != nil && m.podsPanel.watch {
			var cmds []tea.Cmd
			cmds = append(cmds, startPodsWatch(m.selectedNS))

			// Also refresh logs for all pods
			for _, logPanel := range m.logsPanels {
				if logPanel.watch {
					podName := strings.TrimPrefix(logPanel.title, "Logs: ")
					cmds = append(cmds, startLogWatch(podName, m.selectedNS))
				}
			}

			return m, tea.Batch(append(cmds, tick())...)
		}
		return m, nil
	}

	return m, nil
}

func parsePodNames(content []string) []string {
	podNames := []string{}
	for _, line := range content {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "NAME") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 0 {
			podNames = append(podNames, fields[0])
		}
	}
	return podNames
}

func updateLogsPanels(existing []*panel, podNames []string, namespace string) []*panel {
	// Create a map of existing panels by pod name
	existingMap := make(map[string]*panel)
	for _, p := range existing {
		for _, podName := range podNames {
			if strings.Contains(p.title, podName) {
				existingMap[podName] = p
				break
			}
		}
	}

	newPanels := []*panel{}
	for _, podName := range podNames {
		if p, exists := existingMap[podName]; exists {
			newPanels = append(newPanels, p)
		} else {
			// Create new panel for this pod
			newPanel := &panel{
				title:     fmt.Sprintf("Logs: %s", podName),
				content:   []string{"Loading logs..."},
				maxLines:  20,
				scrollPos: 0,
				watch:     true,
			}
			newPanels = append(newPanels, newPanel)
		}
	}

	// Stop and remove panels for pods that no longer exist
	for _, p := range existing {
		found := false
		for _, podName := range podNames {
			if strings.Contains(p.title, podName) {
				found = true
				break
			}
		}
		if !found && p.updateCmd != nil {
			p.updateCmd.Process.Kill()
			p.updateCmd = nil
		}
	}

	return newPanels
}

func startPodsWatch(namespace string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("kubectl", "get", "pods", "-n", namespace)
		output, err := cmd.Output()
		if err != nil {
			return podUpdateMsg{err: err}
		}

		lines := []string{}
		scanner := bufio.NewScanner(strings.NewReader(string(output)))
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}

		return podUpdateMsg{content: lines, err: nil}
	}
}

func startLogWatch(podName, namespace string) tea.Cmd {
	return func() tea.Msg {
		// Use --tail=50 to get more log lines
		// Since we show only one panel at a time, we can show more logs
		// renderPanel will truncate to fit the available height
		cmd := exec.Command("kubectl", "logs", "--tail=50", podName, "-n", namespace)
		output, err := cmd.CombinedOutput()

		if err != nil {
			return logUpdateMsg{
				podName: podName,
				content: []string{fmt.Sprintf("Log error: %v", err)},
				err:     err,
			}
		}

		lines := []string{}
		scanner := bufio.NewScanner(strings.NewReader(string(output)))
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}

		if len(lines) == 0 {
			lines = []string{"No logs yet..."}
		}

		return logUpdateMsg{
			podName: podName,
			content: lines,
			err:     nil,
		}
	}
}

func tick() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) View() string {
	if m.quit {
		return ""
	}

	if m.width == 0 {
		return "Loading..."
	}

	if m.state == "namespace_select" {
		return m.renderNamespaceSelect()
	}

	return m.renderPanelView()
}

func (m model) renderNamespaceSelect() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Kubernetes Helper - Select Namespace"))
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v\n\n", m.err)))
	}

	if len(m.namespaces) == 0 {
		if m.searchTerm != "" {
			b.WriteString(fmt.Sprintf("No namespaces found matching '%s'...\n", m.searchTerm))
			b.WriteString("Try running without search term to see all namespaces\n")
		} else {
			b.WriteString("No namespaces found...\n")
			b.WriteString("Command: kubens\n")
		}
	} else {
		for i, ns := range m.namespaces {
			cursor := " "
			style := normalStyle
			if i == m.cursor {
				cursor = ">"
				style = selectedStyle
			}
			b.WriteString(fmt.Sprintf("%s %s\n", cursor, style.Render(ns)))
		}
	}

	b.WriteString("\n")
	b.WriteString("↑↓: Select, Enter: Confirm, q: Quit")

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, b.String())
}

func (m model) renderPanelView() string {
	if m.podsPanel == nil {
		m.podsPanel = &panel{
			title:    fmt.Sprintf("Pods in %s", m.selectedNS),
			content:  []string{"Loading..."},
			maxLines: m.height / 3,
			watch:    true,
		}
		return "Loading pods..."
	}

	// Calculate available height for panels (subtract footer - footer needs ~6 lines)
	footerHeight := 6
	availableHeight := m.height - footerHeight

	// Pods panel is fixed at the top with a small, constant height
	// This ensures it stays visible and doesn't push content down
	// Keep it small to guarantee it always fits
	podsPanelHeight := 9 // Fixed small height for pods panel (includes border/padding)

	// Remaining height goes to single active log panel
	// Show only ONE log panel at a time (Tab to switch between them)
	logsPanelHeight := 0
	if len(m.logsPanels) > 0 {
		remainingHeight := availableHeight - podsPanelHeight
		// Use all remaining height for the single active log panel
		logsPanelHeight = remainingHeight
		// Ensure minimum height
		if logsPanelHeight < 5 {
			logsPanelHeight = 5
		}
	}

	// Update panel maxLines (subtract border and padding: ~3 lines)
	if m.podsPanel != nil {
		// Pods panel has fixed small height (9), limit content to 4 lines max
		// This ensures it never exceeds its allocated space and always stays visible
		m.podsPanel.maxLines = 4 // Maximum 4 lines of content (9 - 5 for border/padding/title)
	}
	for _, p := range m.logsPanels {
		// Log panels: use available height since we show only one at a time
		// Actual display is limited by maxHeight in renderPanel
		if logsPanelHeight > 0 {
			p.maxLines = logsPanelHeight - 3 // Subtract border/padding
		} else {
			p.maxLines = 20 // Default fallback
		}
	}

	var sections []string

	// CRITICAL: Pods panel - ALWAYS render first, fixed at top
	// This ensures it never gets covered and always stays visible
	// Render it with fixed height regardless of activePanel state
	if m.podsPanel != nil {
		podsContent := m.renderPanel(m.podsPanel, m.activePanel == 0, podsPanelHeight, m.width)
		// CRITICAL: Ensure pods panel doesn't exceed its allocated height
		// This prevents overlapping with log panels
		podsLines := strings.Split(podsContent, "\n")
		if len(podsLines) > podsPanelHeight {
			podsLines = podsLines[:podsPanelHeight]
			podsContent = strings.Join(podsLines, "\n")
		}
		sections = append(sections, podsContent)
	}

	// Logs panel - show only the active log panel (Tab to switch)
	if len(m.logsPanels) > 0 {
		// Find which log panel is active (activePanel 0 is pods, 1+ are log panels)
		activeLogIndex := -1
		if m.activePanel > 0 && m.activePanel <= len(m.logsPanels) {
			activeLogIndex = m.activePanel - 1
		} else if len(m.logsPanels) > 0 {
			// If no log panel is active, show the first one
			activeLogIndex = 0
			m.activePanel = 1
		}

		if activeLogIndex >= 0 && activeLogIndex < len(m.logsPanels) {
			logPanel := m.logsPanels[activeLogIndex]
			// Show single log panel with full width and remaining height
			logContent := m.renderPanel(logPanel, true, logsPanelHeight, m.width)
			// CRITICAL: Ensure log panel doesn't exceed its allocated height
			// This prevents overlapping with pods panel or footer
			logLines := strings.Split(logContent, "\n")
			if len(logLines) > logsPanelHeight {
				logLines = logLines[:logsPanelHeight]
				logContent = strings.Join(logLines, "\n")
			}
			sections = append(sections, logContent)
		}
	}

	// Combine all panels - ensure each panel doesn't exceed its allocated height
	// This prevents panels from overlapping
	var combinedSections []string
	for i, section := range sections {
		// Split section into lines and ensure it doesn't exceed its intended height
		sectionLines := strings.Split(section, "\n")
		var maxSectionHeight int
		if i == 0 && m.podsPanel != nil {
			// First section is pods panel
			maxSectionHeight = podsPanelHeight
		} else {
			// Other sections are log panels
			maxSectionHeight = logsPanelHeight
		}
		// Truncate if exceeds max height
		if len(sectionLines) > maxSectionHeight {
			sectionLines = sectionLines[:maxSectionHeight]
			section = strings.Join(sectionLines, "\n")
		}
		combinedSections = append(combinedSections, section)
	}
	combined := lipgloss.JoinVertical(lipgloss.Left, combinedSections...)

	// Add footer - show current log panel info
	var footer string
	if len(m.logsPanels) > 0 {
		activeLogIndex := -1
		if m.activePanel > 0 && m.activePanel <= len(m.logsPanels) {
			activeLogIndex = m.activePanel - 1
		} else {
			activeLogIndex = 0
		}
		currentPanel := activeLogIndex + 1
		totalPanels := len(m.logsPanels)
		footer = fmt.Sprintf(
			"\n%s | Tab: Switch log panel (%d/%d) | ↑↓: Scroll | b: Back | q: Quit",
			titleStyle.Render(fmt.Sprintf("Namespace: %s", m.selectedNS)),
			currentPanel, totalPanels,
		)
	} else {
		footer = fmt.Sprintf(
			"\n%s | Tab: Switch panel | ↑↓: Scroll | PgUp/PgDn: Page | Home/End: Jump | b: Back | q: Quit",
			titleStyle.Render(fmt.Sprintf("Namespace: %s", m.selectedNS)),
		)
	}

	return combined + footer
}

func (m model) renderPanel(p *panel, active bool, maxHeight int, width int) string {
	if p == nil {
		return ""
	}

	content := strings.Join(p.content, "\n")

	// Scroll the content
	lines := strings.Split(content, "\n")

	// CRITICAL: Calculate available content lines from maxHeight ONLY
	// maxHeight includes border (~2) + padding (~2) + title (~1) + content
	// So content should be maxHeight - 5 lines maximum
	availableContentLines := max(1, maxHeight-5)

	// ALWAYS use availableContentLines as the hard limit for ALL panels
	// This ensures panels never grow beyond their allocated space
	// p.maxLines is just a suggestion, but availableContentLines is the absolute limit
	displayLines := min(p.maxLines, availableContentLines)
	// But never exceed availableContentLines - this is the hard limit for all panels
	if displayLines > availableContentLines {
		displayLines = availableContentLines
	}

	// Calculate max scroll position
	maxScroll := max(0, len(lines)-displayLines)
	// Ensure scroll position is within bounds
	if p.scrollPos > maxScroll {
		p.scrollPos = maxScroll
	}
	if p.scrollPos < 0 {
		p.scrollPos = 0
	}

	// Get visible lines - exactly displayLines, no more
	// CRITICAL: Never exceed availableContentLines
	start := p.scrollPos
	maxAllowedLines := min(displayLines, availableContentLines)
	end := min(len(lines), start+maxAllowedLines)
	// Ensure we never take more than maxAllowedLines
	if end-start > maxAllowedLines {
		end = start + maxAllowedLines
	}

	visibleLines := lines[start:end]
	// Ensure we have exactly maxAllowedLines or less
	if len(visibleLines) > maxAllowedLines {
		visibleLines = visibleLines[:maxAllowedLines]
	}
	content = strings.Join(visibleLines, "\n")

	// Add scroll indicator to title
	title := p.title
	if len(lines) > displayLines {
		currentPage := p.scrollPos/displayLines + 1
		totalPages := (len(lines) + displayLines - 1) / displayLines
		scrollIndicator := fmt.Sprintf(" (%d/%d)", min(currentPage, totalPages), totalPages)
		title += scrollIndicator
	}

	style := panelStyle
	if active {
		style = style.BorderForeground(lipgloss.Color("229"))
	}

	// Build panel content: title + content
	titleRendered := titleStyle.Render(title)

	// CRITICAL: For pods panel, be extra strict - ensure content fits exactly
	// Calculate exact space for content (maxHeight - title - blank lines)
	titleLines := strings.Split(titleRendered, "\n")
	titleHeight := len(titleLines)
	// maxHeight includes: border (2) + padding (2) + title + blank (1) + content
	// So content should be maxHeight - titleHeight - 3 (border/padding/blank)
	strictContentLines := max(1, maxHeight-titleHeight-3)

	// If content exceeds strict limit, truncate it
	contentLines := strings.Split(content, "\n")
	if len(contentLines) > strictContentLines {
		contentLines = contentLines[:strictContentLines]
		content = strings.Join(contentLines, "\n")
	}

	panelContent := titleRendered + "\n\n" + content

	// CRITICAL: Final safety check - truncate panelContent to exactly maxHeight lines
	// This ensures the panel never exceeds its allocated height, even with ANSI codes
	fullContentLines := strings.Split(panelContent, "\n")
	if len(fullContentLines) > maxHeight {
		// Truncate to exactly maxHeight lines - no more, no less
		fullContentLines = fullContentLines[:maxHeight]
		panelContent = strings.Join(fullContentLines, "\n")
	}

	if width > 0 {
		return style.Height(maxHeight).Width(width).Render(panelContent)
	}
	return style.Height(maxHeight).Render(panelContent)
}

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

	p := tea.NewProgram(initialModel(searchTerm), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

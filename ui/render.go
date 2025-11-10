package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"kubetbe/utils"
)

func (m *Model) View() string {
	if m.Quit {
		return ""
	}

	if m.Width == 0 {
		return "Loading..."
	}

	if m.State == "namespace_select" {
		return m.renderNamespaceSelect()
	}

	return m.renderPanelView()
}

func (m *Model) renderNamespaceSelect() string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render("Kubernetes Helper - Select Namespace"))
	b.WriteString("\n\n")

	if m.Err != nil {
		b.WriteString(ErrorStyle.Render(fmt.Sprintf("Error: %v\n\n", m.Err)))
	}

	if len(m.Namespaces) == 0 {
		if m.SearchTerm != "" {
			b.WriteString(fmt.Sprintf("No namespaces found matching '%s'...\n", m.SearchTerm))
			b.WriteString("Try running without search term to see all namespaces\n")
		} else {
			b.WriteString("No namespaces found...\n")
			b.WriteString("Command: kubectl get namespaces\n")
		}
	} else {
		for i, ns := range m.Namespaces {
			cursor := " "
			style := NormalStyle
			if i == m.Cursor {
				cursor = ">"
				style = SelectedStyle
			}
			displayName := ns
			if ns == m.DeletingNamespace {
				displayName = fmt.Sprintf("%s (deleting...)", ns)
			}
			b.WriteString(fmt.Sprintf("%s %s\n", cursor, style.Render(displayName)))
		}
	}

	b.WriteString("\n")

	if m.DeletingNamespace != "" {
		b.WriteString(InfoStyle.Render(fmt.Sprintf("Deleting namespace '%s'...\n", m.DeletingNamespace)))
	}

	// Show delete confirmation if pending
	if m.DeleteConfirmation != "" {
		b.WriteString(ErrorStyle.Render(fmt.Sprintf("\n⚠️  Delete namespace '%s'? Press 'd' again to confirm, any other key to cancel\n", m.DeleteConfirmation)))
	}

	// Show help text
	helpText := "↑↓: Select, Enter: Confirm"
	if m.NamespaceWatch {
		helpText += ", R: Refresh, d: Delete"
	} else {
		helpText += ", R: Refresh, d: Delete"
	}
	helpText += ", q: Quit"
	b.WriteString(helpText)

	return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, b.String())
}

func (m *Model) renderPanelView() string {
	if m.PodsPanel == nil {
		return "Loading pods..."
	}

	// Calculate available height for panels (subtract footer - footer needs ~6 lines)
	footerHeight := 6
	availableHeight := m.Height - footerHeight

	// Pods panel is fixed at the top with a reasonable height
	// This ensures it stays visible and can show more pods with scroll
	// Increased height to show more pods (12 lines can show ~8-10 pods)
	podsPanelHeight := 12 // Fixed height for pods panel (includes border/padding)

	// Remaining height goes to single active log panel
	// Show only ONE log panel at a time (Tab to switch between them)
	logsPanelHeight := 0
	if len(m.LogsPanels) > 0 {
		remainingHeight := availableHeight - podsPanelHeight
		// Use all remaining height for the single active log panel
		logsPanelHeight = remainingHeight
		// Ensure minimum height
		if logsPanelHeight < 5 {
			logsPanelHeight = 5
		}
	}

	// Update panel maxLines (subtract border and padding: ~3 lines)
	if m.PodsPanel != nil {
		// Pods panel has fixed height (12), allow more content with scroll
		// This ensures it never exceeds its allocated space but can scroll to show more
		m.PodsPanel.MaxLines = 7 // Maximum 7 lines of visible content (12 - 5 for border/padding/title)
	}
	for _, p := range m.LogsPanels {
		// Log panels: use available height since we show only one at a time
		// Actual display is limited by maxHeight in renderPanel
		if logsPanelHeight > 0 {
			p.MaxLines = logsPanelHeight - 3 // Subtract border/padding
		} else {
			p.MaxLines = 20 // Default fallback
		}
	}

	var sections []string

	// Find active pod name for highlighting in pods panel
	activePodName := ""
	if len(m.LogsPanels) > 0 {
		activeLogIndex := -1
		if m.ActivePanel > 0 && m.ActivePanel <= len(m.LogsPanels) {
			activeLogIndex = m.ActivePanel - 1
		} else if len(m.LogsPanels) > 0 {
			activeLogIndex = 0
		}
		if activeLogIndex >= 0 && activeLogIndex < len(m.LogsPanels) {
			// Extract pod name from log panel title (format: "Logs: pod-name-...")
			title := m.LogsPanels[activeLogIndex].Title
			activePodName = strings.TrimPrefix(title, "Logs: ")
		}
	}

	selectedPodName := ""
	if m.PodsPanel != nil {
		podNames := ParsePodNames(m.PodsPanel.Content)
		if len(podNames) > 0 {
			podIndex := m.PodCursor
			if podIndex < 0 {
				podIndex = 0
			}
			if podIndex >= len(podNames) {
				podIndex = len(podNames) - 1
			}
			if podIndex >= 0 && podIndex < len(podNames) {
				selectedPodName = podNames[podIndex]
			}
		}
	}

	highlightPodName := activePodName
	if m.ActivePanel == 0 && selectedPodName != "" {
		highlightPodName = selectedPodName
	}

	// CRITICAL: Pods panel - ALWAYS render first, fixed at top
	// This ensures it never gets covered and always stays visible
	// Render it with fixed height regardless of activePanel state
	// Pass activePodName to highlight the active pod in the list
	if m.PodsPanel != nil {
		podsContent := m.renderPanelWithHighlight(m.PodsPanel, m.ActivePanel == 0, podsPanelHeight, m.Width, highlightPodName)
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
	if len(m.LogsPanels) > 0 {
		// Find which log panel is active (activePanel 0 is pods, 1+ are log panels)
		activeLogIndex := -1
		if m.ActivePanel > 0 && m.ActivePanel <= len(m.LogsPanels) {
			activeLogIndex = m.ActivePanel - 1
		} else if len(m.LogsPanels) > 0 {
			// If no log panel is active, show the first one (without changing active panel)
			activeLogIndex = 0
		}

		if activeLogIndex >= 0 && activeLogIndex < len(m.LogsPanels) {
			logPanel := m.LogsPanels[activeLogIndex]
			// Show single log panel with full width and remaining height
			isActive := m.ActivePanel == activeLogIndex+1
			logContent := m.renderPanel(logPanel, isActive, logsPanelHeight, m.Width)
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
		if i == 0 && m.PodsPanel != nil {
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

	// Add footer - show current log panel info and active pod name
	var footer string
	if len(m.LogsPanels) > 0 {
		activeLogIndex := -1
		if m.ActivePanel > 0 && m.ActivePanel <= len(m.LogsPanels) {
			activeLogIndex = m.ActivePanel - 1
		} else {
			activeLogIndex = 0
		}
		currentPanel := activeLogIndex + 1
		totalPanels := len(m.LogsPanels)

		// Get active pod name for display
		activePodDisplay := ""
		if activeLogIndex >= 0 && activeLogIndex < len(m.LogsPanels) {
			title := m.LogsPanels[activeLogIndex].Title
			activePodDisplay = strings.TrimPrefix(title, "Logs: ")
			// Truncate long pod names for display
			if len(activePodDisplay) > 40 {
				activePodDisplay = activePodDisplay[:37] + "..."
			}
		}

		footer = fmt.Sprintf(
			"\n%s | Active: %s | Tab: Switch (%d/%d) | ↑↓: Scroll | d: Delete pod | b: Back | q: Quit",
			TitleStyle.Render(fmt.Sprintf("Namespace: %s", m.SelectedNS)),
			activePodDisplay,
			currentPanel, totalPanels,
		)
	} else {
		footer = fmt.Sprintf(
			"\n%s | Tab: Switch panel | ↑↓: Scroll | PgUp/PgDn: Page | Home/End: Jump | d: Delete pod | b: Back | q: Quit",
			TitleStyle.Render(fmt.Sprintf("Namespace: %s", m.SelectedNS)),
		)
	}

	if m.DeletingPod != "" {
		footer += "\n" + InfoStyle.Render(fmt.Sprintf("Deleting pod '%s'...", m.DeletingPod))
	}
	if m.PodDeleteConfirmation != "" {
		footer += "\n" + ErrorStyle.Render(fmt.Sprintf("⚠️  Delete pod '%s'? Press 'd' again to confirm, any other key to cancel", m.PodDeleteConfirmation))
	}

	return combined + footer
}

func (m Model) renderPanelWithHighlight(p *Panel, active bool, maxHeight int, width int, highlightPodName string) string {
	// Same as renderPanel but highlights the pod in the list if it matches highlightPodName
	if p == nil {
		return ""
	}

	content := strings.Join(p.Content, "\n")

	// If this is pods panel and we have a pod name to highlight, highlight it and auto-scroll
	activePodLineIndex := -1
	if highlightPodName != "" && strings.Contains(p.Title, "Pods in") {
		lines := strings.Split(content, "\n")
		highlightedLines := make([]string, len(lines))
		for i, line := range lines {
			// Check if this line contains the pod name (pod name is usually the first field)
			// Skip header lines
			if strings.HasPrefix(strings.TrimSpace(line), "NAME") || strings.TrimSpace(line) == "" {
				highlightedLines[i] = line
				continue
			}
			fields := strings.Fields(line)
			if len(fields) > 0 {
				podName := fields[0]
				// Check if this pod matches the active pod
				// Pod names should match exactly, but we also check prefix match for cases where
				// pod names might have slight variations
				matches := podName == highlightPodName ||
					strings.HasPrefix(podName, highlightPodName) ||
					strings.HasPrefix(highlightPodName, podName)
				if matches {
					// Highlight this line and remember its index
					highlightedLines[i] = SelectedStyle.Render(line)
					activePodLineIndex = i
				} else {
					highlightedLines[i] = line
				}
			} else {
				highlightedLines[i] = line
			}
		}
		content = strings.Join(highlightedLines, "\n")

		// Auto-scroll to active pod if it's not visible
		if activePodLineIndex >= 0 {
			// Calculate how many content lines we can show (after header)
			availableContentLines := utils.Max(1, maxHeight-5)
			// Find header line (usually first line or second line)
			headerLineIndex := -1
			for i, line := range lines {
				if strings.HasPrefix(strings.TrimSpace(line), "NAME") {
					headerLineIndex = i
					break
				}
			}
			if headerLineIndex < 0 {
				headerLineIndex = 0
			}

			// Calculate visible range (scrollPos is offset from start, header is included)
			visibleStart := p.ScrollPos
			visibleEnd := visibleStart + availableContentLines

			// If active pod is not in visible range, scroll to it
			if activePodLineIndex < visibleStart || activePodLineIndex >= visibleEnd {
				// Scroll so that active pod is visible (preferably in the middle)
				// Make sure we account for header
				scrollOffset := utils.Max(0, activePodLineIndex-(availableContentLines/2))
				// Ensure scroll doesn't go before header
				if scrollOffset <= headerLineIndex {
					scrollOffset = headerLineIndex + 1
				}
				p.ScrollPos = scrollOffset
			}
		}
	}

	// Scroll the content
	lines := strings.Split(content, "\n")

	// CRITICAL: Calculate available content lines from maxHeight ONLY
	// maxHeight includes border (~2) + padding (~2) + title (~1) + content
	// So content should be maxHeight - 5 lines maximum
	availableContentLines := utils.Max(1, maxHeight-5)

	// ALWAYS use availableContentLines as the hard limit for ALL panels
	// This ensures panels never grow beyond their allocated space
	// p.maxLines is just a suggestion, but availableContentLines is the absolute limit
	displayLines := utils.Min(p.MaxLines, availableContentLines)
	// But never exceed availableContentLines - this is the hard limit for all panels
	if displayLines > availableContentLines {
		displayLines = availableContentLines
	}

	// Calculate max scroll position
	maxScroll := utils.Max(0, len(lines)-displayLines)
	// Ensure scroll position is within bounds
	if p.ScrollPos > maxScroll {
		p.ScrollPos = maxScroll
	}
	if p.ScrollPos < 0 {
		p.ScrollPos = 0
	}

	// Get visible lines - exactly displayLines, no more
	// CRITICAL: Never exceed availableContentLines
	start := p.ScrollPos
	maxAllowedLines := utils.Min(displayLines, availableContentLines)
	end := utils.Min(len(lines), start+maxAllowedLines)
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
	title := p.Title
	if len(lines) > displayLines {
		currentPage := p.ScrollPos/displayLines + 1
		totalPages := (len(lines) + displayLines - 1) / displayLines
		scrollIndicator := fmt.Sprintf(" (%d/%d)", utils.Min(currentPage, totalPages), totalPages)
		title += scrollIndicator
	}

	style := PanelStyle
	if active {
		style = style.BorderForeground(lipgloss.Color("229"))
	}

	// Build panel content: title + content
	titleRendered := TitleStyle.Render(title)

	// CRITICAL: For pods panel, be extra strict - ensure content fits exactly
	// Calculate exact space for content (maxHeight - title - blank lines)
	titleLines := strings.Split(titleRendered, "\n")
	titleHeight := len(titleLines)
	// maxHeight includes: border (2) + padding (2) + title + blank (1) + content
	// So content should be maxHeight - titleHeight - 3 (border/padding/blank)
	strictContentLines := utils.Max(1, maxHeight-titleHeight-3)

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

func (m Model) renderPanel(p *Panel, active bool, maxHeight int, width int) string {
	if p == nil {
		return ""
	}

	content := strings.Join(p.Content, "\n")

	// Scroll the content
	lines := strings.Split(content, "\n")

	// CRITICAL: Calculate available content lines from maxHeight ONLY
	// maxHeight includes border (~2) + padding (~2) + title (~1) + content
	// So content should be maxHeight - 5 lines maximum
	availableContentLines := utils.Max(1, maxHeight-5)

	// ALWAYS use availableContentLines as the hard limit for ALL panels
	// This ensures panels never grow beyond their allocated space
	// p.maxLines is just a suggestion, but availableContentLines is the absolute limit
	displayLines := utils.Min(p.MaxLines, availableContentLines)
	// But never exceed availableContentLines - this is the hard limit for all panels
	if displayLines > availableContentLines {
		displayLines = availableContentLines
	}

	// Calculate max scroll position
	maxScroll := utils.Max(0, len(lines)-displayLines)
	// Ensure scroll position is within bounds
	if p.ScrollPos > maxScroll {
		p.ScrollPos = maxScroll
	}
	if p.ScrollPos < 0 {
		p.ScrollPos = 0
	}

	// Get visible lines - exactly displayLines, no more
	// CRITICAL: Never exceed availableContentLines
	start := p.ScrollPos
	maxAllowedLines := utils.Min(displayLines, availableContentLines)
	end := utils.Min(len(lines), start+maxAllowedLines)
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
	title := p.Title
	if len(lines) > displayLines {
		currentPage := p.ScrollPos/displayLines + 1
		totalPages := (len(lines) + displayLines - 1) / displayLines
		scrollIndicator := fmt.Sprintf(" (%d/%d)", utils.Min(currentPage, totalPages), totalPages)
		title += scrollIndicator
	}

	style := PanelStyle
	if active {
		style = style.BorderForeground(lipgloss.Color("229"))
	}

	// Build panel content: title + content
	titleRendered := TitleStyle.Render(title)

	// CRITICAL: For pods panel, be extra strict - ensure content fits exactly
	// Calculate exact space for content (maxHeight - title - blank lines)
	titleLines := strings.Split(titleRendered, "\n")
	titleHeight := len(titleLines)
	// maxHeight includes: border (2) + padding (2) + title + blank (1) + content
	// So content should be maxHeight - titleHeight - 3 (border/padding/blank)
	strictContentLines := utils.Max(1, maxHeight-titleHeight-3)

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

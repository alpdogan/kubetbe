package ui

import "strings"

func ParsePodNames(content []string) []string {
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

func UpdateLogsPanels(existing []*Panel, podNames []string, namespace string) []*Panel {
	// Create a map of existing panels by pod name
	existingMap := make(map[string]*Panel)
	for _, p := range existing {
		for _, podName := range podNames {
			if strings.Contains(p.Title, podName) {
				existingMap[podName] = p
				break
			}
		}
	}

	newPanels := []*Panel{}
	for _, podName := range podNames {
		if p, exists := existingMap[podName]; exists {
			newPanels = append(newPanels, p)
		} else {
			// Create new panel for this pod
			newPanel := &Panel{
				Title:     "Logs: " + podName,
				Content:   []string{"Loading logs..."},
				MaxLines:  20,
				ScrollPos: 0,
				Watch:     true,
			}
			newPanels = append(newPanels, newPanel)
		}
	}

	// Stop and remove panels for pods that no longer exist
	for _, p := range existing {
		found := false
		for _, podName := range podNames {
			if strings.Contains(p.Title, podName) {
				found = true
				break
			}
		}
		if !found && p.UpdateCmd != nil {
			p.UpdateCmd.Process.Kill()
			p.UpdateCmd = nil
		}
	}

	return newPanels
}


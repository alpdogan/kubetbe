package kubectl

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"kubetbe/msg"
)

func FetchNamespaces(searchTerm string) tea.Cmd {
	return func() tea.Msg {
		// Use kubectl to get namespaces
		cmd := exec.Command("kubectl", "get", "namespaces", "-o", "jsonpath={.items[*].metadata.name}")

		output, err := cmd.Output()
		if err != nil {
			return msg.ErrorMsg{Err: fmt.Errorf("failed to run kubectl get namespaces command: %v", err)}
		}

		// Parse output - jsonpath returns space-separated names
		allNamespaces := strings.Fields(string(output))

		// Filter by search term if provided
		namespaces := []string{}
		if searchTerm != "" {
			searchLower := strings.ToLower(searchTerm)
			for _, ns := range allNamespaces {
				if strings.Contains(strings.ToLower(ns), searchLower) {
					namespaces = append(namespaces, ns)
				}
			}
		} else {
			namespaces = allNamespaces
		}

		return msg.NamespaceListMsg{Namespaces: namespaces}
	}
}

func DeleteNamespace(namespace string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("kubectl", "delete", "namespace", namespace)
		err := cmd.Run()
		if err != nil {
			return msg.NamespaceDeleteMsg{
				Namespace: namespace,
				Err:       fmt.Errorf("failed to delete namespace: %v", err),
			}
		}
		// After successful delete, refresh the namespace list
		return msg.NamespaceDeleteMsg{
			Namespace: namespace,
			Err:       nil,
		}
	}
}

func DeletePod(namespace, pod string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("kubectl", "delete", "pod", pod, "-n", namespace)
		err := cmd.Run()
		if err != nil {
			return msg.PodDeleteMsg{
				Namespace: namespace,
				Pod:       pod,
				Err:       fmt.Errorf("failed to delete pod: %v", err),
			}
		}
		return msg.PodDeleteMsg{
			Namespace: namespace,
			Pod:       pod,
			Err:       nil,
		}
	}
}

func DescribePod(namespace, pod string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("kubectl", "describe", "pod", pod, "-n", namespace)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return msg.PodDescribeMsg{
				Namespace: namespace,
				Pod:       pod,
				Content:   []string{fmt.Sprintf("Describe error: %v", err)},
				Err:       err,
			}
		}

		lines := []string{}
		scanner := bufio.NewScanner(strings.NewReader(string(output)))
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		if len(lines) == 0 {
			lines = []string{"No describe output..."}
		}

		return msg.PodDescribeMsg{
			Namespace: namespace,
			Pod:       pod,
			Content:   lines,
			Err:       nil,
		}
	}
}

func StartPodsWatch(namespace string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("kubectl", "get", "pods", "-n", namespace)
		output, err := cmd.Output()
		if err != nil {
			return msg.PodUpdateMsg{Err: err}
		}

		lines := []string{}
		scanner := bufio.NewScanner(strings.NewReader(string(output)))
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}

		return msg.PodUpdateMsg{Content: lines, Err: nil}
	}
}

func StartLogWatch(podName, namespace string) tea.Cmd {
	return func() tea.Msg {
		// Use --tail=50 to get more log lines
		// Since we show only one panel at a time, we can show more logs
		// renderPanel will truncate to fit the available height
		cmd := exec.Command("kubectl", "logs", "--tail=50", podName, "-n", namespace)
		output, err := cmd.CombinedOutput()

		if err != nil {
			return msg.LogUpdateMsg{
				PodName: podName,
				Content: []string{fmt.Sprintf("Log error: %v", err)},
				Err:     err,
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

		return msg.LogUpdateMsg{
			PodName: podName,
			Content: lines,
			Err:     nil,
		}
	}
}

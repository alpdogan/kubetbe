# Kubernetes Helper (kubetbe)

A Kubernetes helper tool similar to k9s. Written in Go, it allows you to view pods and their logs in different panels using a TUI (Terminal User Interface).

## Features

- 🎯 **Namespace Selection**: Lists namespaces with optional filtering via command line argument
- 🔍 **Search Filtering**: Filter namespaces by search term (e.g., `kubetbe zeus` shows only namespaces containing "zeus")
- 📊 **Pod Monitoring**: Displays pods in the selected namespace in real-time
- 📝 **Log Monitoring**: Shows logs for each pod in separate panels
- ⌨️ **Keyboard Control**: Easy navigation with arrow keys and Tab
- 🎨 **Modern TUI**: Beautiful and user-friendly interface with Bubbletea

## Requirements

- Go 1.21 or higher
- `kubectl` installed and configured
- `kubens` tool installed (for namespace listing)

## Installation

```bash
# Download dependencies
go mod download

# Run
go run main.go
```

Or build as binary:

```bash
go build -o kubetbe main.go
./kubetbe
```

## Usage

### Command Line Arguments

You can optionally provide a search term to filter namespaces:

```bash
# Show all namespaces
./kubetbe

# Show only namespaces containing "zeus" (case-insensitive)
./kubetbe zeus

# Show only namespaces containing "prod"
./kubetbe prod
```

### Navigation

1. **Namespace Selection**:
   - When the application starts, it lists namespaces (filtered if search term provided)
   - Use arrow keys (↑↓) to select a namespace
   - Press Enter to confirm selection

2. **Panel View**:
   - Top panel: Pod list (`kubectl get pods`)
   - Bottom panels: Logs for each pod (`kubectl logs`)
   - Use Tab key to switch between panels
   - Use arrow keys (↑↓) to scroll through logs

3. **Keyboard Shortcuts**:
   - `↑` / `k`: Scroll up (line by line)
   - `↓` / `j`: Scroll down (line by line)
   - `PgUp`: Scroll up one page
   - `PgDn`: Scroll down one page
   - `Home`: Jump to top
   - `End`: Jump to bottom
   - `Tab`: Switch to next panel
   - `Shift+Tab`: Switch to previous panel
   - `Enter`: Confirm selection in namespace view
   - `b`: Go back to namespace selection
   - `q` / `Ctrl+C`: Quit

## Panel Layout

- Pod panel is displayed at full width with scrollable content
- Log panels are arranged side by side (2 columns) or vertically
- Each panel is highlighted when active (yellow border)
- Scroll position indicator shows current page (e.g., "Pods in namespace (2/5)")
- Pod panel gets at least 1/3 of screen height for better visibility
- All panels are automatically updated every 2 seconds
- Scroll position is preserved when content updates (unless significant changes occur)

## Notes

- If no command line argument is provided, all namespaces are shown
- Search term filtering is case-insensitive (uses `grep -i`)
- Logs show the last 100 lines using `--tail=100` parameter
- Pods are refreshed every 2 seconds
- Logs are also updated every 2 seconds
- Panels are automatically updated when pods are deleted or new pods are created

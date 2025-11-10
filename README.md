# Kubernetes Helper (`kubetbe`)

`kubetbe` is a lightweight TUI companion for `kubectl`. It focuses on the daily workflow of jumping between namespaces, finding pods, watching logs, deleting resources and quickly answering questions like “which service owns this IP?” – all from the terminal.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and Go.

## Highlights

- 🎯 **Namespace navigator** with optional CLI filtering (`kubetbe prod`) and built‑in paging (10 items per page).
- 🔁 **Live pod view** that refreshes automatically while preserving scroll position.
- 🪵 **Structured log panes** – each pod gets its own scrollable panel.
- 📝 **Describe on demand**: press `i` to fetch `kubectl describe pod`, rendered inline.
- ❌ **Resource actions**: delete namespaces (`d` in namespace view) and pods (`d` in pod view) with confirmation.
- 🔎 **Find service by IP**: press `f`, enter an IP, immediately see matching `kubectl get services --all-namespaces -o wide` rows.
- 🧭 **Keyboard-first UX** with Vim style movement, tab cycling between panels, and page navigation via `Tab`, `Shift+Tab`, `←`, `→`.

## Requirements

- `kubectl` configured to talk to the cluster you want to inspect.
- To **build from source**: Go 1.21+.
- To **use prebuilt binaries**: no Go toolchain required.

## Installation

### Option 1: Using the install script (recommended)

Prebuilt binaries are published per release. Run:

```bash
curl -fsSL https://raw.githubusercontent.com/alpdogan/kubetbe/main/install.sh | bash
```

The script:

1. Detects your platform (macOS Intel/Apple Silicon, Linux AMD64).
2. Downloads the matching binary from the latest GitHub Release.
3. Installs it to `~/.local/bin` (change via `INSTALL_DIR=/custom/path`).

> **PATH reminder**  
> Add `~/.local/bin` to your shell configuration if it is not already there:
>
> ```bash
> # zsh (default on macOS)
> echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc && source ~/.zshrc
>
> # bash (often default on Linux)
> echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc && source ~/.bashrc
> ```

You can also clone the repo and run the script locally:

```bash
git clone https://github.com/alpdogan/kubetbe.git
cd kubetbe
INSTALL_DIR=/usr/local/bin bash install.sh   # requires sudo, optional
```

### Option 2: Build from source

```bash
git clone https://github.com/alpdogan/kubetbe.git
cd kubetbe
go build -o kubetbe .
./kubetbe           # optional search term: ./kubetbe prod
```

## Usage & Shortcuts

### Namespace view (startup screen)

| Key(s)           | Action |
|------------------|--------|
| `↑` / `k` / `↓` / `j` | Move selection (automatically flips pages) |
| `Tab` / `→` / `l`     | Next page (10 namespaces per page) |
| `Shift+Tab` / `←` / `h` | Previous page |
| `Enter`           | Open selected namespace (switch to panel view) |
| `d`               | Delete namespace (confirmation required) |
| `f`               | Find service by IP (enter IP, `Esc` to cancel) |
| `Esc`             | Close service lookup results |
| `r`               | Refresh namespace list |
| `q`, `Ctrl+C`     | Quit |

Tip: you can start the app filtered by a string: `kubetbe zeus` shows only namespaces containing “zeus”.

### Panel view (after selecting a namespace)

Layout:
- **Pods Panel** (top, fixed height) – follows `kubectl get pods`.
- **Log Panel(s)** (bottom) – one per pod; only the active log pane is shown at a time.
- `Describe`: appears in place of logs when toggled.

| Key(s)                  | Action |
|-------------------------|--------|
| `Tab` / `Shift+Tab`     | Cycle between pods panel, describe (if open), log panels |
| `↑` / `k` / `↓` / `j`   | Scroll pods or logs (depending on active panel) |
| `PgUp` / `PgDn`         | Page scroll logs/describe |
| `Home` / `End`          | Jump to top/bottom of logs/describe |
| `i`                     | Toggle describe for the selected pod |
| `d`                     | Delete highlighted pod (with confirmation) |
| `b`                     | Back to namespace view |
| `q`, `Ctrl+C`           | Quit |

## Service Lookup (`f`)

While in the namespace selection screen press `f`:
1. Enter an IP (e.g., `0.0.0.0`).
2. Hit `Enter` – matching rows from `kubectl get services --all-namespaces -o wide` appear.
3. `Esc` clears the results.

## How It Works

- Pods and logs refresh continuously using Bubble Tea commands.
- Log tail is currently `--tail=50`; tweak in `kubectl/commands.go`.
- Namespace pagination adapts to terminal height but caps list length at 10 per page.
- The application keeps `kubectl` invocations simple so you can reason about what is happening under the hood.

## Developing

```bash
go test ./...
go run main.go
```

Binary builds for release:

```bash
GOOS=darwin GOARCH=amd64 go build -o kubetbe-darwin-amd64 .
GOOS=darwin GOARCH=arm64 go build -o kubetbe-darwin-arm64 .
GOOS=linux  GOARCH=amd64 go build -o kubetbe-linux-amd64 .
```

Upload these artifacts to a GitHub Release so the install script can fetch them.

---

Happy debugging! Contributions, bug reports and feature suggestions are welcome. Data plane feeling a little foggy? `kubetbe` is here to help. 🎉

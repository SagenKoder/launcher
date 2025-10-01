# Launcher

![License](https://img.shields.io/github/license/SagenKoder/launcher)
![Repo Size](https://img.shields.io/github/repo-size/SagenKoder/launcher)
![Last Commit](https://img.shields.io/github/last-commit/SagenKoder/launcher)
![GitHub Stars](https://img.shields.io/github/stars/SagenKoder/launcher?style=social)
![GitHub Forks](https://img.shields.io/github/forks/SagenKoder/launcher?style=social)
![GitHub Watchers](https://img.shields.io/github/watchers/SagenKoder/launcher?style=social)
![GitHub Issues](https://img.shields.io/github/issues/SagenKoder/launcher)
![GitHub Pull Requests](https://img.shields.io/github/issues-pr/SagenKoder/launcher)
![Go Report Card](https://goreportcard.com/badge/github.com/SagenKoder/launcher)
[![Build Status](https://github.com/SagenKoder/launcher/actions/workflows/release.yml/badge.svg)](https://github.com/SagenKoder/launcher/actions/workflows/release.yml)
![GitHub Contributors](https://img.shields.io/github/contributors/SagenKoder/launcher)
![GitHub All Releases](https://img.shields.io/github/downloads/SagenKoder/launcher/total)
![Lines of Code](https://img.shields.io/tokei/lines/github/SagenKoder/launcher)


Launcher is a lightweight command-palette style app launcher with first-class plugin support. Hit your hotkey, start typing, and run applications, open dashboards, or chat with an AI assistant without leaving your keyboard. Everything is written in Go and built on top of the Fyne toolkit, so the codebase stays small, cross-platform, and easy to extend.

## Why Launcher?

- **Fast to use** – a single search box for all applications and plugins.
- **Simple to extend** – drop in small Go plugins or configure links in YAML.
- **Native experience** – packaged as a tiny desktop app with minimal dependencies.
- **Streaming UI** – plugins can stream responses (great for AI or long-running tasks).

## Features at a Glance

- Powerful fuzzy search for installed desktop applications.
- Plugin registry with rich UI components (badges, markdown rendering, copy buttons).
- Built-in plugins for AI Chat plus configurable external links (use config to add dashboards, search tools, and more).
- Configurable via `config.yaml` (API keys, custom links, etc.).
- Packaged `.deb` for quick installation on Debian/Ubuntu.

## Installation

### Debian / Ubuntu

Download the latest release artifact and install it with `dpkg`:

```bash
curl -LO https://github.com/SagenKoder/launcher/releases/download/v0.1.1/launcher_0.1.1_amd64.deb
sudo dpkg -i launcher_0.1.1_amd64.deb
```

This will place the executable in `/usr/bin/launcher` and stage a default config at `/etc/launcher/config.yaml`. Adjust the path if you are installing a different version or architecture.

> **Tip:** Add a custom system shortcut (for example `Alt+Space`) that launches `/usr/bin/launcher` to get a command palette workflow.

### Building from Source

Prerequisites:

- Go 1.22+
- `make`

Clone the repo and build:

```bash
git clone https://github.com/SagenKoder/launcher.git
cd launcher
make build            # go build ./cmd/launcher
make package          # optional: produce dist/launcher_VERSION_amd64.deb
```

The resulting binary is in `dist/launcher`. Run it directly (`./dist/launcher`) or install the generated `.deb`.

## Configuration

Launcher looks for `config.yaml` in the following locations (first match wins):

1. `LAUNCHER_CONFIG` environment variable
2. `./config.yaml` (current working directory)
3. `${XDG_CONFIG_HOME}/launcher/config.yaml`
4. `${HOME}/.config/launcher/config.yaml`
5. `/etc/launcher/config.yaml`

A starter file is provided as `config.example.yaml`. Copy it to one of the paths above and edit the relevant sections. Example:

```yaml
chat:
  api_key: sk-your-openai-key
  base_url: https://api.openai.com
  model: gpt-4o

links:
  - name: Team Wiki
    url: https://intranet.example.com/wiki
    icon: help-browser
  - name: Status Page
    url: https://status.example.com
    icon: web-browser
  - name: Observability Dashboard
    url: https://grafana.example.com/dashboards
    icon: utilities-monitor
  - name: Log Search
    url: https://kibana.example.com/app/discover#/?query=__QUERY__
    replacement: __QUERY__
    icon: system-search
```

Values under `links` become plugin entries. When `replacement` is omitted the link opens immediately. If you provide a `replacement`, the launcher prompts for input, URL-encodes it, and swaps it into the configured URL before opening the browser. This lets you replicate more complex plugins—like log searches—purely through configuration.

## Usage

1. Launch `launcher` (bind it to a global hotkey for best results).
2. Start typing to search installed applications via fuzzy matching.
3. Hit `Enter` to launch the highlighted application.
4. Type a plugin trigger (for example choose “AI Chat” from the list or `/` prefix if you add shortcuts) and the UI switches to the plugin view with badges and streaming output.

Keyboard shortcuts:

- `↑` / `↓` – move selection in the list
- `Enter` – launch the selected entry or submit input to the active plugin
- `Esc` – close the launcher window

## Built-in Plugins

| Plugin ID | Description | Notes |
|-----------|-------------|-------|
| `chat` | Streaming AI assistant backed by OpenAI-compatible APIs | Requires `chat` config with API key | 
| `link-*` | Config-driven links defined in `config.yaml` | Closes on launch |

Each plugin registers itself during startup (see `internal/plugins`). Feel free to remove or modify those registrations to fit your environment.

## Writing Your Own Plugin

Plugins live in Go code and register an instance of `plugins.Info`. The launcher handles rendering, input, and streaming for you.

### 1. Create a Go file in `internal/plugins`

```go
package plugins

import (
    "fmt"

    "github.com/SagenKoder/launcher/internal/applications"
)

func init() {
    Register(Info{
        ID:            "hello",
        Name:          "Hello World",
        IconPath:      applications.DebugResolveIcon("applications-utilities"),
        Intro:         "Small demo plugin",
        Hint:          "Type anything",
        CloseOnSubmit: false,
        OnSubmit: func(input string) (string, error) {
            return fmt.Sprintf("Hello, %s!", input), nil
        },
    })
}
```

### 2. (Optional) Stream responses

If your plugin needs to stream output (for example, from an API that sends incremental data), implement `OnSubmitStream` instead:

```go
OnSubmitStream: func(ctx context.Context, input string, emit func(string, bool)) error {
    emit("Working on "+input+"...", false)
    // Emit more chunks here
    emit("Done!", true)
    return nil
},
```

Return errors to display them inline in the plugin panel. Use `CloseOnSubmit: true` if the launcher should exit after the plugin finishes.

### 3. Customize icons and intro text

- `IconPath` should point to an image file; `applications.DebugResolveIcon(name)` is convenient during development because it resolves system icons.
- `Intro` is rendered as Markdown in the plugin pane when the plugin activates.
- `Hint` replaces the search box placeholder while your plugin is focused.

Once registered, the plugin appears alongside applications in the list. Selecting it switches the UI into plugin mode automatically.

## Development

- Format & lint: `go fmt ./...` and `go vet ./...`
- Build: `make build`
- Tests: `go test ./...` (unit tests coming soon)
- Package: `make package` (produces `.deb` in `dist/`)

The codebase is structured into small packages:

- `cmd/launcher` – thin binary entry point
- `internal/launcher` – UI orchestration, search, plugin host
- `internal/applications` – desktop entry discovery
- `internal/plugins` – plugin registry and built-ins
- `internal/search` – fuzzy search helpers
- `internal/ui` – shared widget implementations

## Contributing

1. Fork the repository and create a feature branch.
2. Make your changes and run `go test ./...`.
3. Open a pull request describing what you added or fixed.

Ideas welcome! New plugins, better hotkey integration, testing, and docs are all great ways to help.

## License

This project will ship with an OSI-approved license (MIT/Apache-2.0 in progress). For now, treat the code as all rights reserved until the license lands.

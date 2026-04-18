```text
██████╗ ███████╗ ██████╗ █████╗ ██████╗
██╔══██╗██╔════╝██╔════╝██╔══██╗██╔══██╗
██████╔╝█████╗  ██║     ███████║██████╔╝
██╔══██╗██╔══╝  ██║     ██╔══██║██╔═══╝
██║  ██║███████╗╚██████╗██║  ██║██║
╚═╝  ╚═╝╚══════╝ ╚═════╝╚═╝  ╚═╝╚═╝
```

A terminal-based meeting notes app with AI-powered structuring

![CI](https://github.com/glieske/recap/actions/workflows/ci.yml/badge.svg) [![Go Report Card](https://goreportcard.com/badge/github.com/glieske/recap)](https://goreportcard.com/report/github.com/glieske/recap) [![Latest Release](https://img.shields.io/github/v/release/glieske/recap)](https://github.com/glieske/recap/releases/latest) [![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

`recap` is designed for fast note capture during live meetings and instant AI-assisted post-processing.
Capture raw notes in a keyboard-first TUI, structure them into consistent markdown, and generate share-ready email summaries without leaving your terminal.

## Features

- 📝 TUI for capturing meeting notes in real-time
- 🤖 AI-powered structuring (raw notes → formatted Markdown with Summary, Attendees, Key Decisions, Action Points, Discussion Notes, Next Steps)
- 📧 AI-generated professional email summaries with subject line
- 📂 Project organization with auto-generated ticket IDs (e.g., INFRA-001, TEAM-042)
- 🖥️ Cross-platform: macOS, Linux, Windows
- 🧠 3 AI providers: GitHub Models, OpenRouter, LM Studio (local)
- 💾 Auto-save every 2 seconds
- 📋 Clipboard support for email sharing
- 🪟 Modal overlay system for help, settings, and confirmations
- ⚙️ In-app settings editor (Ctrl+,)
- ⚡ AI provider quick-switch (Ctrl+O)

## Installation

Choose the installation method that best fits your workflow.

### Homebrew (macOS & Linux)

```bash
brew tap glieske/tap
brew install recap
```

### Go Install

Great for users who already manage Go-based CLI tools via `go install`.

```bash
go install github.com/glieske/recap/cmd/recap@latest
```

### From Source

Use this when you want to build from source or contribute to development.

```bash
git clone https://github.com/glieske/recap.git
cd recap
make build
```

## Configuration

Config file location:
- Linux/macOS: `~/.config/recap/config.yaml`
- Windows: `%APPDATA%\recap\config.yaml`

First run creates default config automatically.

**Example `config.yaml`:**
```yaml
notes_dir: ~/recap
ai_provider: github_models # github_models | openrouter | lm_studio
github_model: gpt-4o
openrouter_model: anthropic/claude-3-opus
openrouter_api_key: your-api-key-here
lm_studio_url: http://localhost:1234/v1
lm_studio_model: # optional, uses whatever model is currently loaded
```

## AI Provider Setup

### GitHub Models (default)
- Requires GitHub CLI: `gh auth login`
- Uses `gh auth token` for authentication
- Endpoint: https://models.inference.ai.azure.com

### OpenRouter
- Get API key from https://openrouter.ai
- Set `ai_provider: openrouter` and `openrouter_api_key: your-key` in `config.yaml`

### LM Studio (local)
- Install [LM Studio](https://lmstudio.ai), load a model
- Set `ai_provider: lm_studio` in `config.yaml`
- Default URL: http://localhost:1234/v1

## Keybindings

### Global
| Key | Action |
|---|---|
| `q` / `Ctrl+C` | Quit |
| `?` | Help overlay |
| `Ctrl+,` | Open settings |
| `Esc` | Back / close |

### Welcome Screen
| Key | Action |
|---|---|
| `↑`/`↓` / `j`/`k` | Navigate menu |
| `Enter` | Select item |

### Meeting List
| Key | Action |
|---|---|
| `Enter` | Open meeting |
| `n` | New meeting |
| `f` | Cycle project filter |
| `t` | Cycle tag filter |
| `d` | Delete meeting (modal confirm) |
| `/` | Search |

### Editor
| Key | Action |
|---|---|
| `Ctrl+S` | Save |
| `Ctrl+T` | Insert timestamp |
| `Ctrl+A` | AI structure notes |
| `Ctrl+E` | Generate email |
| `Ctrl+O` | Switch AI provider |
| `Ctrl+,` | Open settings |
| `Ctrl+P` | Toggle preview |

### Email
| Key | Action |
|---|---|
| `c` | Copy to clipboard |
| `r` | Regenerate |

### Confirmation Dialogs
| Key | Action |
|---|---|
| `y` / `Enter` | Confirm |
| `n` / `Esc` | Cancel |
| `←`/`→`/`Tab` | Switch Yes/No |

## Project / Ticket System

- Create projects with unique prefixes (e.g., `INFRA`, `TEAM`, `DEV`)
- Meetings auto-assigned sequential ticket IDs within project
- Directory structure: `notes_dir/PREFIX/YYYY-MM-DD-slug/`
- Each meeting has:
  - `raw.txt` (raw notes)
  - `structured.md` (AI output)
  - `meta.json` (metadata)

## Development

Common development commands:

```bash
make build    # Build binary
make install  # Install to $GOPATH/bin
make test     # Run tests
make lint     # Run go vet
make fmt      # Format code
make clean    # Remove binary
```

## Contributing

Thanks for your interest in improving `recap`.

Contributions are welcome! Please:
1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the Apache License 2.0 — see the [LICENSE](LICENSE) file for details.

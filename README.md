# RepoBird CLI

<img width="1254" height="726" alt="cli-logo-loading" src="https://github.com/user-attachments/assets/12e06a97-161f-4286-a241-6060fd8d9f2c" />


[![Go Version](https://img.shields.io/badge/Go-1.20+-00ADD8?style=flat&logo=go)](https://go.dev)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](LICENSE)
[![CI Status](https://img.shields.io/github/actions/workflow/status/repobird/repobird-cli/ci.yml?branch=main)](https://github.com/repobird/repobird-cli/actions)
[![Release](https://img.shields.io/github/v/release/repobird/repobird-cli)](https://github.com/repobird/repobird-cli/releases)

**One-shot issue to PR with OpenCode-powered cloud agents**

RepoBird CLI is the command-line interface for [RepoBird.ai](https://repobird.ai) - OpenCode-powered cloud coding agents that handle everything from issue to PR. No chat loop, no local workstation babysitting, no manual Git operations. Write the task once, track the run, and review the PR when it is ready.

## 🎯 What is RepoBird?

RepoBird provides one-shot coding agents with complete Git automation. Unlike chat-based AI tools that require back-and-forth iterations and manual Git operations, RepoBird is simple: **issue in, PR out**.

Write your issue description once, and autonomous OpenCode-based agents handle the workflow: repository setup, implementation, testing, commits, and PR creation. Credits are the customer-facing unit for cloud agent work, so RepoBird no longer models availability around fixed Basic/Pro monthly run counts.

### Key Features

- 🚀 **One-Shot Execution**: No chat, no iterations - write once, ship automatically
- 🔧 **Complete Git Automation**: Never touch Git - perfect commits, branches, and PRs every time
- ⚡ **OpenCode Cloud Execution**: Launch OpenCode-backed coding agents without tying up your local machine
- 🤖 **Autonomous Agents**: Full cycle from repository setup to PR without human intervention
- 📊 **Real-Time Monitoring**: Track progress of all parallel runs in the TUI dashboard
- 🔒 **Isolated VM Execution**: Each agent runs in its own secure Debian microVM with full development tools
- 💳 **Credit-Based Usage**: Runs consume credits based on cloud agent work instead of fixed Basic/Pro run limits
- 🌐 **Complete Dev Environment**: Multi-language support, package managers, databases - everything needed to build real software

## 🎯 Why RepoBird?

### The Problem with Other AI Tools
- **IDE-based tools**: Run locally with resource constraints, handle one task at a time, require IDE context switching and manual Git operations
- **Other AI Agents**: Chat interfaces requiring multiple iterations, manual PR creation, no parallel execution capabilities, lack native GitHub integration

### The RepoBird Difference
**GitHub-Native Integration**: Lives entirely within your GitHub workflow as a GitHub App. Complete automation from issue to PR - no external tools, no context switching.

**Cloud-Based Execution**: Launch coding agents in managed cloud environments with full resources and no local constraints.

**One-Shot Simplicity**: Write your issue once, get a production-ready PR back. No chat, no iterations, no manual steps. 73% of PRs merge without changes.

**Complete Git Automation**: Our agents handle everything - branching, atomic commits with proper messages, comprehensive PR descriptions. You never touch Git.

**Enterprise-Grade Environment**: Each agent runs in an isolated cloud VM with full development tools, package managers, and internet access. RepoBird's forward-looking agent workflow is OpenCode-based.

**Credit-Based Runs**: Credits cover cloud agent work across model usage, orchestration, runtime, logs, and storage. Basic and Pro language may still appear as capability or model-selection presets in some API responses, but it should not be interpreted as fixed monthly run-count availability.

## 📦 Installation

### Quick Install (Linux/macOS)

```bash
curl -sSL https://raw.githubusercontent.com/RepoBird/repobird-cli/main/scripts/install.sh | bash
```

### Direct Download

#### macOS
```bash
# Apple Silicon
curl -L https://github.com/RepoBird/repobird-cli/releases/latest/download/repobird-cli_darwin_arm64.tar.gz | tar xz && \
sudo mv repobird /usr/local/bin/

# Intel
curl -L https://github.com/RepoBird/repobird-cli/releases/latest/download/repobird-cli_darwin_amd64.tar.gz | tar xz && \
sudo mv repobird /usr/local/bin/
```

#### Linux
```bash
curl -L https://github.com/RepoBird/repobird-cli/releases/latest/download/repobird-cli_linux_amd64.tar.gz | tar xz && \
sudo mv repobird /usr/local/bin/
```

#### Windows
Download the latest ZIP from the [releases page](https://github.com/RepoBird/repobird-cli/releases) and extract `repobird.exe`.

### Build from Source

```bash
# Requires Go 1.20+
git clone https://github.com/RepoBird/repobird-cli.git
cd repobird-cli
make build
sudo cp build/repobird /usr/local/bin/
```

## 🚀 Quick Start

### 1. Get Your API Key

Sign up for a free account at [RepoBird.ai](https://repobird.ai) and get your API key from [Dashboard → API Keys](https://repobird.ai/dashboard/user-profile/api-keys).

### 2. Authenticate

```bash
# One-time setup
repobird login
# Enter your API key when prompted
```

### 3. Submit Your First Task (One-Shot)

```bash
# Quickest way - direct command with flags (no file needed)
repobird run -r your-org/your-repo -p "Fix the login bug where users get stuck on loading screen"

# Simple cloud-agent presets
repobird basic -r your-org/your-repo "Fix a small bug"  # DeepSeek V4 Flash
repobird pro -r your-org/your-repo "Implement OAuth"    # Kimi K2.6

# Push commits to an output branch without opening a PR
repobird run -r your-org/your-repo -p "Update generated docs" --output-branch automation/docs --branch-only

# Resend only after reviewing a prompt-risk acknowledgement error
repobird run -r your-org/your-repo -p @reviewed-task.md --acknowledge-prompt-risk

# Inside a git repo with an origin remote, the repo can be auto-detected
repobird pro "Fix the login bug where users get stuck on loading screen"

# Or read prompt from a file using @ prefix
echo "Fix the login bug where users get stuck on loading screen" > task.txt
repobird run -r your-org/your-repo -p @task.txt

# Or use a JSON file for more options
echo '{
  "repository": "your-org/your-repo",
  "prompt": "Fix the login bug where users get stuck on loading screen"
}' > fix.json

repobird run fix.json --follow
# That's it. PR will be created automatically. No further action needed.
```

## 📖 Usage Guide

### Authentication Management

```bash
repobird login          # Interactive login with API key
repobird verify         # Verify your API key is valid
repobird info           # Show authentication status
repobird logout         # Remove stored credentials

# Alternative: Use environment variable
export REPOBIRD_API_KEY=your-api-key

# Optional: disable colored human-readable output
repobird config set color never
```

### Submitting Tasks

```bash
# Single task
repobird run task.json --follow
repobird basic "Fix a small bug"
repobird pro "Implement OAuth"
repobird run --basic -r myorg/webapp -p "Fix a small bug"
repobird run --pro -r myorg/webapp -p "Implement OAuth"

# From different formats
repobird run task.yaml          # YAML format
repobird run task.md            # Markdown with frontmatter
cat task.json | repobird run -  # From stdin
```

| Command | Use When | Default Model |
|---|---|---|
| `repobird basic "prompt"` | Quick Basic preset from inside a git repo | DeepSeek V4 Flash |
| `repobird pro "prompt"` | Quick Pro preset from inside a git repo | Kimi K2.6 |
| `repobird run --basic -r owner/repo -p "prompt"` | Basic preset with explicit repository | DeepSeek V4 Flash |
| `repobird run --pro -r owner/repo -p "prompt"` | Pro preset with explicit repository | Kimi K2.6 |

The `basic` and `pro` commands auto-detect the repository from the current git remote when `-r/--repo` is omitted. After submission, the CLI prints the selected run type and model before showing the run ID/status.

### Monitoring & Management

```bash
# Check status
repobird status                 # List all runs
repobird status RUN_ID          # Check specific run
repobird status --follow RUN_ID # Live updates

# Interactive dashboard
repobird tui                    # Launch terminal UI
```

### Repository Defaults

When repository branch defaults are enabled on the API, the CLI can inspect and update persisted defaults. Per-run flags such as `--base-branch`, `--pr-target-branch`, `--output-branch`, and `--branch-only` still override repository defaults.

```bash
repobird repo list
repobird repo show repo_123
repobird repo defaults repo_123 --base develop --pr-target release
repobird repo defaults repo_123 --clear-base --clear-pr-target
repobird repo defaults repo_123 --clear-output  # branch-only runs generate an output branch
```

### Terminal UI Navigation
<img width="1251" height="723" alt="tui-example" src="https://github.com/user-attachments/assets/263d96bf-2b53-4152-943c-de5529ad40d6" />

The interactive dashboard features a **Miller column layout** inspired by the ranger file manager, providing hierarchical navigation through repositories → runs → details in a three-column view. This intuitive layout allows you to see context at every level while drilling down into specific run details.

| Key | Action |
|-----|--------|
| `Tab` / `→` | Navigate forward between columns |
| `Shift+Tab` / `←` | Navigate backward |
| `↑` / `↓` | Move selection up/down |
| `Enter` | Select item |
| `f` | Fuzzy search in current column |
| `n` | Create new run |
| `r` | Refresh data |
| `?` | Show help |
| `q` | Quit |

### Example Templates

```bash
# Generate example configurations
repobird examples generate minimal -o task.json
repobird examples schema  # View full schema documentation
```

## 📝 Task Configuration

Tasks are defined in JSON, YAML, or Markdown files with two required fields:

- `repository` - Target repository (format: "owner/repo")
- `prompt` - Task description for the AI

### Simple Example

```json
{
  "repository": "myorg/webapp",
  "prompt": "Add user authentication with JWT tokens"
}
```

### Advanced Example

```json
{
  "repository": "myorg/webapp",
  "prompt": "Implement OAuth2 authentication",
  "source": "main",
  "target": "feature/oauth",
  "title": "Add OAuth2 support",
  "context": "Use Google and GitHub as providers",
  "files": ["src/auth/", "config/oauth.json"]
}
```

For complete configuration options and examples, see the [Run Configuration Guide](docs/RUN-CONFIG-FORMATS.md).

## 🛡️ Advanced Features

### Smart Caching

- Local caching reduces API calls and improves performance
- Repository and run data cached for quick access
- Automatic cache invalidation on updates

### Retry Logic

- Automatic exponential backoff for transient failures
- Configurable retry attempts and timeouts
- Graceful handling of rate limits

## 📚 Documentation

For the live product documentation, start at [repobird.ai/docs](https://repobird.ai/docs).

### Getting Started
- [RepoBird Docs](https://repobird.ai/docs) - Product guides, GitHub setup, workflow docs, and feature documentation
- [Getting Started Guide](https://repobird.ai/docs/getting-started) - Connect GitHub and trigger your first RepoBird run
- [CLI Quick Start](https://repobird.ai/docs/cli-quickstart) - Install the CLI, authenticate, and submit your first terminal run
- [Installation Guide](docs/INSTALLATION.md) - Platform-specific CLI setup instructions
- [Configuration Guide](docs/CONFIGURATION-GUIDE.md) - Authentication and settings

### User Guides
- [Terminal UI Guide](docs/TUI-GUIDE.md) - Master the interactive dashboard
- [Run Configuration Formats](docs/RUN-CONFIG-FORMATS.md) - Task file examples
- [Branch Workflow](https://repobird.ai/docs/branch-workflow) - How RepoBird creates branches and pull requests
- [Troubleshooting Guide](docs/TROUBLESHOOTING.md) - CLI troubleshooting
- [Error Messages Guide](https://repobird.ai/docs/error-messages-guide) - Product error codes and recovery steps

### Reference
- [CLI Command Reference](docs/cli-reference.md) - Complete command documentation
- [API Reference](docs/API-REFERENCE.md) - REST API integration
- [Keyboard Shortcuts](docs/QUICK-REFERENCE.md) - TUI navigation cheat sheet

### Development
- [Architecture Overview](docs/ARCHITECTURE.md) - System design and components
- [Development Guide](docs/DEVELOPMENT-GUIDE.md) - Setup for contributors
- [Testing Guide](docs/TESTING-GUIDE.md) - Test strategies and patterns

## 🤝 Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development Setup

```bash
# Clone your fork
git clone https://github.com/your-username/repobird-cli.git
cd repobird-cli

# Install dependencies
make deps

# Run tests
make test

# Build locally
make build
```

## 📄 License

This project is licensed under the GNU Affero General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

## 🌟 Support

- **Application Documentation**: [repobird.ai/docs](https://repobird.ai/docs)
- **Technical/Developer Docs**: [docs/](docs/) - Architecture, API reference, development guides
- **Issues**: [GitHub Issues](https://github.com/repobird/repobird-cli/issues)
- **Discussions**: [GitHub Discussions](https://github.com/repobird/repobird-cli/discussions)

## 🙏 Acknowledgments

Built with:
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Terminal UI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling

---

Made with ❤️ by the RepoBird team

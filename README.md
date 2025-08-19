# RepoBird CLI

[![Go Version](https://img.shields.io/badge/Go-1.20+-00ADD8?style=flat&logo=go)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![CI Status](https://img.shields.io/github/actions/workflow/status/repobird/repobird-cli/ci.yml?branch=main)](https://github.com/repobird/repobird-cli/actions)
[![Release](https://img.shields.io/github/v/release/repobird/repobird-cli)](https://github.com/repobird/repobird-cli/releases)

**One-Shot Issue to PR with Complete Git Automation**

RepoBird CLI is the command-line interface for [RepoBird.ai](https://repobird.ai) - one-shot coding agents that handle everything from issue to PR. No chat, no iterations, no manual Git operations. Write your issue once, get a perfect PR back. Clear entire backlogs with bulk parallel runs.

## 🎯 What is RepoBird?

RepoBird provides one-shot coding agents with complete Git automation. Unlike chat-based AI tools that require back-and-forth iterations and manual Git operations, RepoBird is simple: **issue in, PR out**. 

Write your issue description once, and our autonomous agents handle everything - research, implementation, testing, commits, and PR creation. No chat interface, no copy-pasting code, no Git commands. The CLI enables massive scale with bulk parallel runs - clear your entire backlog in one command.

### Key Features

- 🚀 **One-Shot Execution**: No chat, no iterations - write once, ship automatically
- 🔧 **Complete Git Automation**: Never touch Git - perfect commits, branches, and PRs every time
- ⚡ **Bulk Parallel Runs**: Clear 50+ issues simultaneously with one command
- 🤖 **Autonomous Agents**: Full cycle from research to PR without human intervention
- 📊 **Real-Time Monitoring**: Track progress of all parallel runs in the TUI dashboard
- 🎯 **73% Auto-Merge Rate**: PRs so good they merge without changes
- 🛡️ **Bulletproof Git Operations**: Impossible to mess up - predetermined workflows, not AI experiments
- 🔐 **Enterprise Ready**: Secure API key management and team collaboration

## 🎯 Why RepoBird?

### The Problem with Other AI Tools
- **Copilot/Cursor**: Requires constant interaction, you still handle all Git operations manually
- **ChatGPT/Claude**: Copy-paste code snippets, manage Git yourself, lose context between sessions
- **Other AI Agents**: Chat interfaces, multiple iterations, manual PR creation

### The RepoBird Difference
**One-Shot Simplicity**: Write your issue once, get a PR back. No chat, no iterations, no manual steps.

**Complete Git Automation**: Our agents handle everything - branching, commits with proper messages, PR creation with descriptions. You never touch Git.

**Massive Scale**: Submit 50+ tasks in parallel. While you're in a meeting, RepoBird clears your entire backlog.

**Perfect Every Time**: Atomic commits, proper commit messages, clean Git history. Impossible to mess up because it's not AI making Git decisions - it's bulletproof predetermined workflows.

## 📦 Installation

### Quick Install (Linux/macOS)

```bash
curl -sSL https://raw.githubusercontent.com/RepoBird/repobird-cli/main/scripts/install.sh | bash
```

### Direct Download

#### macOS
```bash
# Apple Silicon
curl -L https://github.com/RepoBird/repobird-cli/releases/latest/download/repobird-cli_darwin_arm64.tar.gz | tar xz
sudo mv repobird /usr/local/bin/

# Intel
curl -L https://github.com/RepoBird/repobird-cli/releases/latest/download/repobird-cli_darwin_amd64.tar.gz | tar xz
sudo mv repobird /usr/local/bin/
```

#### Linux
```bash
curl -L https://github.com/RepoBird/repobird-cli/releases/latest/download/repobird-cli_linux_amd64.tar.gz | tar xz
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

## 🗑️ Uninstallation

### Using the Uninstall Script

The easiest way to completely remove RepoBird CLI and its data:

```bash
# If you have the repository cloned
./scripts/uninstall.sh

# Or download and run the script directly
curl -sSL https://raw.githubusercontent.com/RepoBird/repobird-cli/main/scripts/uninstall.sh | bash
```

The uninstall script will:
- Remove the `repobird` binary and `rb` alias from your system
- Delete configuration files (including API keys)
- Clean up cache directories
- Prompt for confirmation before each removal

### Manual Uninstallation

If you prefer to uninstall manually:

```bash
# Remove the binary (location depends on installation method)
sudo rm -f /usr/local/bin/repobird
sudo rm -f /usr/local/bin/rb
# Or if installed with go install
rm -f ~/go/bin/repobird
rm -f ~/go/bin/rb

# Remove configuration and cache
rm -rf ~/.config/repobird
rm -rf ~/.repobird  # Legacy location
```


## 🚀 Quick Start

### 1. Get Your API Key

Sign up for a free account at [RepoBird.ai](https://repobird.ai) to get your API key.

### 2. Authenticate

```bash
# One-time setup
repobird login
# Enter your API key when prompted
```

### 3. Submit Your First Task (One-Shot)

```bash
# Create a task - just describe what you want
echo '{
  "repository": "your-org/your-repo",
  "prompt": "Fix the login bug where users get stuck on loading screen"
}' > fix.json

# Submit and watch the magic happen
repobird run fix.json --follow
# That's it. PR will be created automatically. No further action needed.
```

### 4. Clear Your Entire Backlog (Bulk Mode)

```bash
# Submit multiple issues at once
echo '{
  "repository": "your-org/your-repo",
  "runs": [
    {"prompt": "Fix login bug"},
    {"prompt": "Add dark mode"},
    {"prompt": "Improve error handling"},
    {"prompt": "Update dependencies"},
    {"prompt": "Add unit tests for auth module"}
  ]
}' > backlog.json

# Fire and forget - all PRs created in parallel
repobird bulk backlog.json

# Monitor all runs in real-time
repobird tui
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
```

### Submitting Tasks

```bash
# Single task
repobird run task.json --follow

# From different formats
repobird run task.yaml          # YAML format
repobird run task.md            # Markdown with frontmatter
cat task.json | repobird run -  # From stdin

# Bulk operations
repobird bulk tasks.json        # Submit multiple tasks
```

### Monitoring & Management

```bash
# Check status
repobird status                 # List all runs
repobird status RUN_ID          # Check specific run
repobird status --follow RUN_ID # Live updates

# Interactive dashboard
repobird tui                    # Launch terminal UI
```

### Terminal UI Navigation

The interactive dashboard provides a rich interface for managing your runs:

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
repobird examples generate bulk -o bulk.json
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

### Bulk Operations

Submit multiple tasks in a single file:

```json
{
  "repository": "myorg/webapp",
  "runs": [
    {"prompt": "Fix login bug", "target": "fix/login"},
    {"prompt": "Add password reset", "target": "feature/reset"},
    {"prompt": "Improve error handling", "target": "fix/errors"}
  ]
}
```

For complete configuration options and examples, see the [Run Configuration Guide](docs/RUN-CONFIG-FORMATS.md).

## 🛡️ Advanced Features

### Duplicate Prevention

RepoBird automatically prevents accidental duplicate submissions:
- File content hashing detects when you're re-running the same task
- Visual indicators in the TUI show duplicate status
- Easy override option when you intentionally want to re-run

### Smart Caching

- Local caching reduces API calls and improves performance
- Repository and run data cached for quick access
- Automatic cache invalidation on updates

### Retry Logic

- Automatic exponential backoff for transient failures
- Configurable retry attempts and timeouts
- Graceful handling of rate limits

## 📚 Documentation

### Getting Started
- [Installation Guide](https://repobird.ai/docs/cli/installation) - Platform-specific setup instructions
- [Quick Start Tutorial](https://repobird.ai/docs/cli/quickstart) - Your first RepoBird task
- [Configuration Guide](docs/CONFIGURATION-GUIDE.md) - Authentication and settings

### User Guides
- [Terminal UI Guide](docs/TUI-GUIDE.md) - Master the interactive dashboard
- [Run Configuration Formats](docs/RUN-CONFIG-FORMATS.md) - Task file examples
- [Bulk Operations Guide](docs/BULK-RUNS.md) - Managing multiple tasks
- [Troubleshooting Guide](docs/TROUBLESHOOTING.md) - Common issues and solutions

### Reference
- [CLI Command Reference](docs/cli-reference.md) - Complete command documentation
- [API Reference](docs/API-REFERENCE.md) - REST API integration
- [Keyboard Shortcuts](docs/QUICK-REFERENCE.md) - TUI navigation cheat sheet

### Development
- [Architecture Overview](docs/ARCHITECTURE.md) - System design and components
- [Development Guide](docs/DEVELOPMENT-GUIDE.md) - Setup for contributors
- [Testing Guide](docs/TESTING-GUIDE.md) - Test strategies and patterns

## 🤝 Contributing

We welcome contributions! RepoBird CLI is open source and community-driven.

### How to Contribute

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to your branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

Please read our [Contributing Guidelines](CONTRIBUTING.md) for details on our code of conduct and development process.

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

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🌟 Support

- **Documentation**: [repobird.ai/docs](https://repobird.ai/docs)
- **Issues**: [GitHub Issues](https://github.com/repobird/repobird-cli/issues)
- **Discussions**: [GitHub Discussions](https://github.com/repobird/repobird-cli/discussions)

## 🙏 Acknowledgments

Built with:
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Terminal UI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling

---

Made with ❤️ by the RepoBird team

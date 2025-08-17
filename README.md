# RepoBird CLI

[![Go Version](https://img.shields.io/badge/Go-1.20+-00ADD8?style=flat&logo=go)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![CI Status](https://img.shields.io/github/actions/workflow/status/repobird/repobird-cli/ci.yml?branch=main)](https://github.com/repobird/repobird-cli/actions)
[![Release](https://img.shields.io/github/v/release/repobird/repobird-cli)](https://github.com/repobird/repobird-cli/releases)

RepoBird CLI is a powerful command-line interface and terminal UI for [RepoBird.ai](https://repobird.ai) - the AI-powered code generation platform. Submit coding tasks to AI agents, track progress in real-time, and manage your development workflows efficiently from your terminal.

## 🎯 What is RepoBird?

RepoBird is an AI platform that automates code generation and software development tasks. Simply describe what you want to build or fix, and RepoBird's AI agents will create pull requests with the changes. The CLI tool gives you full control over this process directly from your terminal.

### Key Features

- 🤖 **AI-Powered Development**: Submit tasks in natural language and get production-ready code
- 📊 **Real-Time Monitoring**: Track AI agent progress with live updates
- 🎨 **Interactive Dashboard**: Rich terminal UI with intuitive navigation
- 🔄 **Batch Operations**: Submit and manage multiple tasks simultaneously
- 🔍 **Smart Search**: Fuzzy search across repositories and runs
- 🛡️ **Duplicate Prevention**: Automatic detection prevents accidental re-submissions
- 🔐 **Secure Authentication**: Safe API key management with multiple auth methods

## 📦 Installation

### macOS

#### Using Homebrew (Recommended)
```bash
brew tap repobird/tap
brew install repobird-cli
```

#### Direct Download
```bash
# Download latest release for macOS (Apple Silicon)
curl -L https://github.com/repobird/repobird-cli/releases/latest/download/repobird-darwin-arm64 -o repobird
chmod +x repobird
sudo mv repobird /usr/local/bin/

# For Intel Macs
curl -L https://github.com/repobird/repobird-cli/releases/latest/download/repobird-darwin-amd64 -o repobird
chmod +x repobird
sudo mv repobird /usr/local/bin/
```

### Linux

#### Using Script
```bash
curl -sSL https://raw.githubusercontent.com/repobird/repobird-cli/main/install.sh | bash
```

#### Direct Download
```bash
# Download latest release for Linux
curl -L https://github.com/repobird/repobird-cli/releases/latest/download/repobird-linux-amd64 -o repobird
chmod +x repobird
sudo mv repobird /usr/local/bin/
```

### Windows

#### Using Scoop
```powershell
scoop bucket add repobird https://github.com/repobird/scoop-bucket
scoop install repobird
```

#### Direct Download
Download the latest Windows executable from the [releases page](https://github.com/repobird/repobird-cli/releases).

### Build from Source

```bash
# Requires Go 1.20+
git clone https://github.com/repobird/repobird-cli.git
cd repobird-cli
make build

# Install globally (optional)
sudo cp build/repobird /usr/local/bin/
```

## 🚀 Quick Start

### 1. Get Your API Key

Sign up for a free account at [RepoBird.ai](https://repobird.ai) to get your API key.

### 2. Authenticate

```bash
# Interactive login (recommended)
repobird login
# Enter your API key when prompted

# Verify authentication
repobird verify
```

### 3. Submit Your First Task

```bash
# Create a simple task file
echo '{
  "repository": "your-org/your-repo",
  "prompt": "Add a README file with project documentation"
}' > task.json

# Submit the task
repobird run task.json --follow
```

### 4. Monitor Progress

```bash
# Launch the interactive dashboard
repobird tui

# Or check status via CLI
repobird status
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

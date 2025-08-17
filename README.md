# RepoBird CLI

CLI and TUI (Terminal User Interface) for [RepoBird.ai](https://repobird.ai) - trigger AI coding agents, submit batch runs, and monitor your AI agent runs through an interactive dashboard.

## Quick Start

### Installation

#### From Source
```bash
# Clone the repository
git clone https://github.com/repobird/repobird-cli.git
cd repobird-cli

# Build the binary
make build

# Install to PATH (optional)
sudo cp build/repobird /usr/local/bin/
```

#### Download Binary
Download the latest release from the [releases page](https://github.com/yourusername/repobird-cli/releases).

### Configuration

#### Authentication
```bash
# Secure login (recommended)
repobird login

# Verify your API key
repobird verify

# Check authentication status
repobird info

# Logout (remove stored API key)
repobird logout
```

#### Alternative Methods
```bash
# Set API key via config command
repobird config set api-key YOUR_API_KEY

# Or use environment variable
export REPOBIRD_API_KEY=YOUR_API_KEY
```

### Basic Usage

#### Generate Example Configurations
```bash
# View configuration schema and examples
repobird examples schema

# Generate example files
repobird examples generate minimal -o task.json
repobird examples generate run -f yaml -o task.yaml
repobird examples generate bulk -o bulk.json
```

#### Submit a Task
```bash
# Run a single task from a JSON file
repobird run task.json

# Run and follow progress
repobird run task.json --follow

# Run from YAML or Markdown file
repobird run task.yaml
repobird run task.md

# Run multiple tasks (bulk) from a file with runs array
repobird run tasks.json  # Automatically detects bulk format
```

#### Check Status
```bash
# View all runs
repobird status

# Check specific run
repobird status RUN_ID

# Follow run progress
repobird status --follow RUN_ID
```

#### Interactive TUI
```bash
# Launch the interactive dashboard
repobird tui

# TUI Navigation:
# - Tab/Arrow keys: Navigate between columns
# - Enter: Select item and move to next column
# - f: Activate fuzzy search (FZF mode) for current column
# - n: Create new run
# - s: Show status info
# - r: Refresh data
# - ?: Toggle help
# - q: Quit
```

### Task Configuration Formats

The `repobird run` command supports multiple formats and automatically detects single vs bulk runs:
- **JSON/YAML** - Single or bulk runs with `prompt` (required) and `repository` (required)
- **Markdown** - YAML frontmatter with documentation in body
- **JSONL** - JSON Lines for bulk operations
- **Stdin** - Pipe JSON directly

#### Required Fields
- `repository` - Repository name in format "owner/repo"
- `prompt` - Task description/instructions for the AI

#### Optional Fields
- `source` - Source branch (defaults to repository's default branch if not specified)
- `target` - Target branch name (auto-generated if not specified)
- `title` - Human-readable title (auto-generated if not specified)
- `runType` - Type: "run" or "plan" (default: "run")
- `context` - Additional context or instructions
- `files` - List of specific files to include

#### Example: Minimal Configuration
```json
{
  "repository": "myorg/webapp",
  "prompt": "Fix the authentication bug where users cannot log in after 5 failed attempts"
}
```

#### Example: Full Configuration
```json
{
  "repository": "myorg/webapp",
  "prompt": "Add user authentication to the application",
  "source": "main",
  "target": "feature/auth",
  "title": "Add authentication system",
  "runType": "run",
  "context": "Use JWT tokens and bcrypt for password hashing",
  "files": ["src/auth.js", "src/models/user.js"]
}
```

#### Example: Bulk Configuration
```json
{
  "repository": "myorg/webapp",
  "source": "main",
  "runType": "run",
  "runs": [
    {
      "prompt": "Fix authentication bug",
      "title": "Fix auth issue",
      "target": "fix/auth"
    },
    {
      "prompt": "Add logging to API",
      "title": "Add API logging",
      "target": "feature/logging"
    }
  ]
}
```

#### Example: YAML Format
```yaml
repository: myorg/webapp
prompt: Add user authentication to the application
source: main
target: feature/auth
title: Add authentication system
runType: run
context: Use JWT tokens and bcrypt for password hashing
files:
  - src/auth.js
  - src/models/user.js
```

#### Example: Markdown Format
```markdown
---
repository: myorg/webapp
prompt: Add user authentication to the application
---

# Additional Context

Implement a secure authentication system using:
- JWT tokens for session management
- bcrypt for password hashing
- Rate limiting for login attempts
```

For complete format documentation and more examples, see [Run Config Formats](docs/run-config-formats.md).

### Duplicate Run Prevention

RepoBird CLI automatically detects and prevents duplicate task submissions using file hashing:

- **Universal File Support**: Works with JSON, YAML, Markdown, or any file type - calculates SHA-256 hash of file content
- **Visual Indicator**: The TUI shows a validation status indicator next to the Submit button:
  - ✓ Ready to submit (green) - Task is valid and not a duplicate
  - ⚠️ Duplicate detected (yellow) - This task file has already been submitted
- **User-Friendly Override**: When a duplicate is detected during submission:
  - **No Error Page**: Instead of showing a confusing error, you get a clear prompt
  - **Yellow Status Bar**: `[DUPLICATE] ⚠️ DUPLICATE RUN DETECTED (ID: 123) - Override? [y] yes [n] no`
  - **One-Click Retry**: Press `y` to automatically override and submit, or `n` to cancel
- **Smart Caching**: File hashes are cached locally and synced with the server to prevent accidental re-submissions

This feature helps prevent:
- Accidental double-clicks or re-submissions
- Running the same task file multiple times by mistake
- Wasting API credits on duplicate runs

The duplicate detection works across all your devices as the hash tracking is server-side.

### Common Commands

```bash
# View help
repobird --help
repobird run --help

# Check version
repobird version

# List configuration
repobird config list

# Authentication commands
repobird login      # Secure login with API key
repobird logout     # Remove stored API key
repobird verify     # Verify API key is valid
repobird info       # Show authentication status
```

## Features

- 🚀 Submit AI-powered code generation tasks
- 📊 Real-time status tracking with progress updates
- 🎨 Rich terminal UI with interactive dashboard
- 🔍 Fuzzy search (FZF) for quick navigation and selection
- 🔐 Secure API key management
- 🔄 Automatic retry with exponential backoff
- 📝 Support for both run and approval workflows
- 🛡️ Duplicate run prevention with file hash tracking
- 🌍 Cross-platform support (Linux, macOS, Windows)

### Terminal UI Features

#### Dashboard View
- **Miller Columns Layout**: Navigate repositories, runs, and details in three columns
- **Fuzzy Search**: Press `f` on any column to activate FZF mode for quick filtering
- **Keyboard Navigation**: Vim-style keys (h/j/k/l) or arrow keys
- **Real-time Updates**: Auto-refresh with customizable intervals
- **Status Indicators**: Visual icons for run status (✓ success, ⚡ running, ✗ failed)

#### Create Run View
- **Repository Selection**: Fuzzy search through repository history
- **Form Validation**: Real-time validation with helpful error messages
- **Keyboard Shortcuts**:
  - `Ctrl+F`: Activate fuzzy search for repository field
  - `f` (in normal mode): Fuzzy search when on repository field
  - `Ctrl+S`: Submit run
  - `Tab`: Navigate between fields

## Requirements

- Go 1.20+ (for building from source)
- Git (for repository operations)
- Internet connection

## Development

For development information, see [DEV.md](DEV.md).

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Documentation

- [Quick Reference](docs/quick-reference.md) - Keyboard shortcuts and commands cheat sheet
- [Terminal UI Guide](docs/tui-guide.md) - Complete guide to the interactive interface
- [Architecture Overview](docs/architecture.md)
- [API Reference](docs/api-reference.md)
- [Configuration Guide](docs/configuration-guide.md)
- [Development Guide](docs/development-guide.md)
- [Troubleshooting Guide](docs/troubleshooting.md)

## License

[Add your license here]

## Support

For issues and feature requests, please use the [GitHub issue tracker](https://github.com/yourusername/repobird-cli/issues).

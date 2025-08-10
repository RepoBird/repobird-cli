# RepoBird CLI

A command-line interface for interacting with the https://RepoBird.ai AI platform to automate code generation and modifications.

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

Set your API key:
```bash
repobird config set api-key YOUR_API_KEY
```

Or use environment variable:
```bash
export REPOBIRD_API_KEY=YOUR_API_KEY
```

### Basic Usage

#### Submit a Task
```bash
# Run a task from a JSON file
repobird run task.json

# Run and follow progress
repobird run task.json --follow
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
```

### Task File Format

Create a `task.json` file:
```json
{
  "prompt": "Add user authentication to the application",
  "repository": "org/repo",
  "source": "main",
  "target": "feature/auth",
  "runType": "run",
  "title": "Add authentication",
  "context": "Use JWT tokens for authentication",
  "files": ["src/auth.js", "src/routes.js"]
}
```

### Common Commands

```bash
# View help
repobird --help
repobird run --help

# Check version
repobird version

# List configuration
repobird config list

# Remove API key
repobird config unset api-key
```

## Features

- 🚀 Submit AI-powered code generation tasks
- 📊 Real-time status tracking with progress updates
- 🎨 Rich terminal UI with interactive dashboard
- 🔐 Secure API key management
- 🔄 Automatic retry with exponential backoff
- 📝 Support for both run and approval workflows
- 🌍 Cross-platform support (Linux, macOS, Windows)

## Requirements

- Go 1.20+ (for building from source)
- Git (for repository operations)
- Internet connection

## Development

For development information, see [DEV.md](DEV.md).

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Documentation

- [Architecture Overview](docs/architecture.md)
- [API Reference](docs/api-reference.md)
- [Configuration Guide](docs/configuration-guide.md)
- [Troubleshooting Guide](docs/troubleshooting.md)

## License

[Add your license here]

## Support

For issues and feature requests, please use the [GitHub issue tracker](https://github.com/yourusername/repobird-cli/issues).

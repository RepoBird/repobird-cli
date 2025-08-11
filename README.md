# RepoBird CLI

A command-line interface for interacting with the [RepoBird.ai](https://repobird.ai) platform to automate code generation and modifications.

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

# Remove API key
repobird config unset api-key
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
- **Smart Defaults**: Auto-detects current git repository
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

## Run Configuration File Formats

RepoBird CLI supports three configuration file formats for defining run tasks. All formats support the same fields, with some being required and others optional.

### Required Fields
- `prompt` - The task description for the AI
- `repository` - Repository in `owner/repo` format
- `target` - Target branch for changes
- `title` - Short title for the run

### Optional Fields
- `source` - Source branch (defaults to `main`)
- `runType` - Type of run: `run`, `approval` (defaults to `run`)
- `context` - Additional context for the AI
- `files` - List of relevant files

### JSON Format

Traditional JSON format for defining run configurations:

```json
{
  "prompt": "Implement user authentication with JWT tokens",
  "repository": "acme/webapp",
  "source": "main",
  "target": "feature/jwt-auth",
  "runType": "run",
  "title": "Add JWT authentication system",
  "context": "Need secure authentication with JWT tokens",
  "files": ["src/auth.js", "src/middleware.js", "src/routes.js"]
}
```

### YAML Format

YAML provides a cleaner, more readable format with support for multiline strings:

```yaml
# task.yaml
prompt: |
  Implement a complete user authentication system with the following requirements:
  - Use JWT tokens for stateless authentication
  - Implement refresh token rotation
  - Add rate limiting for login attempts
  - Include password reset functionality
  - Add email verification for new accounts
  
  Follow security best practices and add comprehensive error handling.
  
repository: acme/webapp
source: main
target: feature/jwt-auth
runType: run
title: Add JWT authentication system
context: |
  The application currently has no authentication.
  We're using Express.js with TypeScript.
  Database is PostgreSQL with Prisma ORM.
  
files:
  - src/auth/jwt.ts
  - src/middleware/auth.ts
  - src/routes/auth.ts
  - src/models/user.ts
```

Minimal YAML example with defaults:

```yaml
# fix-bug.yml
prompt: Fix the login timeout issue affecting mobile users
repository: myorg/mobile-app
target: fix/login-timeout
title: Fix mobile login timeout bug
```

### Markdown Format

Markdown files with YAML frontmatter combine configuration with rich documentation:

```markdown
---
prompt: Implement comprehensive API documentation
repository: acme/api-service
source: main
target: feature/api-docs
runType: run
title: Add OpenAPI documentation
context: Generate OpenAPI 3.0 specification for all endpoints
files:
  - src/routes/
  - src/controllers/
  - src/models/
---

# API Documentation Task

## Overview
The API service currently lacks comprehensive documentation, making it difficult for developers to integrate with our platform.

## Requirements

### OpenAPI Specification
- Generate OpenAPI 3.0 compliant specification
- Include all REST endpoints
- Document request/response schemas
- Add authentication requirements
- Include example requests and responses

### Interactive Documentation
- Set up Swagger UI for interactive testing
- Configure ReDoc for beautiful static docs
- Add postman collection export

### Code Integration
- Add JSDoc comments to all route handlers
- Implement schema validation matching the OpenAPI spec
- Add automated tests to ensure docs stay in sync

## Technical Considerations
- The API uses Express.js with TypeScript
- Authentication is handled via JWT tokens
- Current version is v2, maintain backwards compatibility
- Consider versioning strategy for future changes

## Definition of Done
- [ ] All endpoints documented in OpenAPI spec
- [ ] Swagger UI accessible at /api-docs
- [ ] ReDoc accessible at /docs
- [ ] Schema validation implemented
- [ ] CI/CD validates spec on each commit
- [ ] Team review and approval
```

### Using Configuration Files

```bash
# Run with any format
repobird run task.json
repobird run task.yaml
repobird run task.yml
repobird run task.md

# Follow run status after creation
repobird run task.yaml --follow

# Validate without creating (dry run)
repobird run task.md --dry-run
```

### File Discovery in TUI

The TUI's Create Run view can load configuration files:
1. Navigate to "Load Config" field
2. Press `Enter` or `f` to open file selector
3. Automatically discovers `.json`, `.yaml`, `.yml`, `.md`, and `.markdown` files
4. Files are shown with icons: 📄 JSON, 📋 YAML, 📝 Markdown
5. Supports fuzzy search for quick filtering

## Support

For issues and feature requests, please use the [GitHub issue tracker](https://github.com/yourusername/repobird-cli/issues).

# Development Guide

## Overview

Complete guide for setting up, building, and contributing to RepoBird CLI.

## Related Documentation
- **[Architecture Overview](architecture.md)** - System design and structure
- **[Testing Guide](testing-guide.md)** - Testing patterns and coverage
- **[API Reference](api-reference.md)** - API client implementation
- **[TUI Guide](tui-guide.md)** - Terminal UI development

## Prerequisites

**Required:**
- Go 1.20+ ([Download](https://golang.org/dl/))
- Git
- Make

**Optional:**
- golangci-lint (code quality)
- Docker (containerized builds)

## Quick Start

```bash
# Clone and setup
git clone https://github.com/repobird/cli.git
cd cli
go mod download

# Build and run
make build
./build/repobird version

# Install locally
make install
repobird --help
```

## Project Structure

```
.
├── cmd/repobird/       # Main entry point
├── internal/           # Private packages
│   ├── api/           # API client
│   ├── commands/      # CLI commands
│   ├── tui/           # Terminal UI
│   │   ├── views/     # TUI views
│   │   ├── components/# Shared components
│   │   ├── cache/     # Caching layer
│   │   └── keymap/    # Key handling
│   ├── config/        # Configuration
│   └── errors/        # Error handling
├── pkg/               # Public packages
├── docs/              # Documentation
└── Makefile          # Build automation
```

## Development Workflow

### 1. Environment Setup
```bash
# Development API (optional)
export REPOBIRD_API_URL=http://localhost:8080
export REPOBIRD_API_KEY=test_key

# Enable debug logging
export REPOBIRD_DEBUG_LOG=1

# Important: For testing, set REPOBIRD_API_KEY to empty
# to avoid environment pollution and API key errors
# The Makefile already does this for 'make test'
# REPOBIRD_API_KEY="" go test ./...
```

### 2. Common Tasks
```bash
# Run tests
make test

# Run with coverage
make coverage

# Lint code
make lint

# Fix lint issues
make lint-fix

# Format code
make fmt

# Full check (fmt, lint, test)
make check

# CI pipeline (includes security)
make ci
```

### 3. Building

**Single Platform:**
```bash
make build
```

**Cross-Platform:**
```bash
make build-all
# Creates binaries for:
# - darwin/amd64, darwin/arm64
# - linux/amd64, linux/arm64  
# - windows/amd64
```

**With Version Info:**
```bash
VERSION=v1.2.3 make build
```

## Adding Features

### New CLI Command
1. Create command file in `internal/commands/`
2. Implement command with Cobra
3. Register in `root.go`
4. Add tests
5. Update help text

Example:
```go
// internal/commands/mycommand.go
package commands

func NewMyCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "mycommand",
        Short: "Does something useful",
        RunE: func(cmd *cobra.Command, args []string) error {
            // Implementation
            return nil
        },
    }
}
```

### New TUI View
1. Create view in `internal/tui/views/`
2. Implement Bubble Tea model interface
3. Register in app router
4. Use WindowLayout for consistency
5. Add navigation messages

Template:
```go
type MyView struct {
    client APIClient
    cache  *cache.SimpleCache
    layout *components.WindowLayout
}

func NewMyView(client APIClient, cache *cache.SimpleCache) *MyView {
    return &MyView{
        client: client,
        cache:  cache,
    }
}

func (v *MyView) Init() tea.Cmd {
    return v.loadData()
}

func (v *MyView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Handle messages
}

func (v *MyView) View() string {
    // Render UI
}
```

### New API Endpoint
1. Add method to `internal/api/client.go`
2. Define request/response models
3. Add retry logic if needed
4. Write tests

## Testing

### Running Tests
```bash
# All tests
make test

# Specific package
go test ./internal/api

# With verbose output
go test -v ./...

# With race detection
go test -race ./...
```

### Writing Tests
```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:  "valid input",
            input: "test",
            want:  "TEST",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := MyFunction(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("got = %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Code Style

### Go Conventions
- Use `gofmt` for formatting
- Follow [Effective Go](https://golang.org/doc/effective_go)
- Keep functions small (<50 lines)
- Handle errors explicitly
- Use meaningful variable names

### Project Conventions
- Minimal constructors (max 3 params)
- Message-based navigation in TUI
- Table-driven tests
- Structured error handling
- No global state in runtime code

## Debugging

### Debug Logging
```go
import "github.com/repobird/cli/internal/tui/debug"

debug.LogToFilef("Operation: %s, Result: %v", op, result)
```

View logs:
```bash
tail -f /tmp/repobird_debug.log
```

### TUI Debugging
```bash
# Enable debug mode
REPOBIRD_DEBUG_LOG=1 repobird tui

# Check specific operations
grep "CACHE" /tmp/repobird_debug.log
grep "NAV" /tmp/repobird_debug.log
```

## Git Workflow

### Branch Naming
- `feature/description` - New features
- `fix/description` - Bug fixes
- `chore/description` - Maintenance
- `docs/description` - Documentation

### Commit Messages
```
<type>: <short summary>

[optional body]

[optional footer]
```

Types: feat, fix, docs, style, refactor, test, chore

### Pull Requests
1. Create feature branch
2. Make changes with tests
3. Run `make check`
4. Push branch
5. Open PR with description
6. Address review feedback
7. Squash merge to main

## Release Process

1. Update version:
```bash
VERSION=v1.2.3 make build
```

2. Run full CI:
```bash
make ci
```

3. Create tag:
```bash
git tag v1.2.3
git push origin v1.2.3
```

4. Build releases:
```bash
make release
```

## Performance Tips

- Use caching to reduce API calls
- Batch operations when possible
- Profile with `go tool pprof`
- Monitor with `go test -bench`
- Use context for cancellation

## Security

- Never log sensitive data
- Use structured errors
- Validate all inputs
- Keep dependencies updated
- Run security checks: `make security`

## Contributing

1. Fork the repository
2. Create feature branch
3. Write tests first (TDD)
4. Implement feature
5. Update documentation
6. Submit pull request

See [CONTRIBUTING.md](../CONTRIBUTING.md) for details.
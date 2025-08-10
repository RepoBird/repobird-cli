# RepoBird CLI - Developer Guide

This guide provides detailed information for developers working on the RepoBird CLI codebase.

## Development Setup

### Prerequisites
- Go 1.20 or higher
- Git
- Make
- golangci-lint (for linting)

### Initial Setup
```bash
# Clone the repository
git clone https://github.com/yourusername/repobird-cli.git
cd repobird-cli

# Install dependencies
go mod download

# Verify setup
make test
make build
```

## Project Structure

```
repobird-cli/
├── cmd/repobird/          # Main application entry point
│   └── main.go           # CLI initialization and command setup
├── internal/              # Private application code
│   ├── api/              # API client and communication
│   │   ├── client.go     # HTTP client implementation
│   │   └── models.go     # API request/response types
│   ├── commands/         # Cobra command implementations
│   │   ├── config.go     # Configuration management commands
│   │   ├── run.go        # Task submission command
│   │   ├── status.go     # Status checking command
│   │   └── tui.go        # Terminal UI command
│   ├── config/           # Configuration management
│   │   └── config.go     # Viper-based config handling
│   ├── errors/           # Error handling
│   │   └── errors.go     # Custom error types and formatting
│   ├── models/           # Data models
│   │   └── types.go      # Core data structures
│   ├── retry/            # Retry logic
│   │   └── retry.go      # Exponential backoff implementation
│   ├── tui/              # Terminal UI components
│   │   ├── dashboard/    # Main dashboard view
│   │   ├── components/   # Reusable UI components
│   │   └── models/       # TUI-specific models
│   └── utils/            # Utility functions
│       ├── git.go        # Git operations
│       ├── polling.go    # Status polling
│       └── security.go   # Security utilities
├── pkg/                   # Public library code
│   ├── utils/            # Exported utilities
│   └── version/          # Version information
├── docs/                  # Documentation
├── tasks/                 # Example task files
└── build/                 # Build artifacts
```

## Build System

### Makefile Targets

```bash
# Core build commands
make build          # Build binary for current OS/arch
make build-all      # Build for all platforms
make install        # Build and install to /usr/local/bin

# Testing
make test           # Run all tests
make test-verbose   # Run tests with verbose output
make coverage       # Generate coverage report
make test-race      # Run tests with race detector

# Code quality
make lint           # Run golangci-lint
make lint-fix       # Auto-fix linting issues
make fmt            # Format code with gofmt
make fmt-check      # Check if code is formatted
make vet            # Run go vet

# Composite commands
make check          # Run fmt-check, vet, and lint
make ci             # Run all CI checks
make all            # Clean, check, test, and build

# Utilities
make clean          # Remove build artifacts
make deps           # Download dependencies
make mod-tidy       # Clean up go.mod and go.sum
make run            # Build and run with debug flag
```

### Cross-Platform Building

```bash
# Build for specific platforms
GOOS=linux GOARCH=amd64 make build
GOOS=darwin GOARCH=arm64 make build
GOOS=windows GOARCH=amd64 make build

# Or use the convenience target
make build-all
```

## Testing

### Test Structure
- Unit tests: Alongside source files (`*_test.go`)
- Integration tests: In `test/integration/`
- Mocks: Generated in `mocks/` directory

### Writing Tests

```go
// Table-driven test example
func TestParseTask(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    *Task
        wantErr bool
    }{
        {
            name:  "valid task",
            input: `{"prompt": "test"}`,
            want:  &Task{Prompt: "test"},
        },
        // Add more test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseTask(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("ParseTask() error = %v, wantErr %v", err, tt.wantErr)
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("ParseTask() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Running Tests

```bash
# Run all tests
make test

# Run specific package tests
go test ./internal/api

# Run with coverage
make coverage

# Run with race detection
make test-race

# Run specific test
go test -run TestParseTask ./internal/models
```

## Debugging

### Debug Mode

Enable debug output:
```bash
# Via flag
repobird --debug status

# Via environment
export REPOBIRD_DEBUG=true
repobird status
```

### Local Development

```bash
# Run with local API server
export REPOBIRD_API_URL=http://localhost:8080
./build/repobird status

# Use debug TUI script
./debug-tui.sh
```

### Logging

Debug logging locations:
- API requests/responses: When `--debug` flag is set
- Config operations: `~/.repobird/debug.log` (when enabled)
- TUI events: Stderr when run with `--debug`

## Code Style Guide

### Go Conventions
- Use `gofmt` for formatting
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use meaningful variable names
- Keep functions small (<50 lines)
- Handle errors explicitly

### Error Handling

```go
// Use custom error types
if err != nil {
    return errors.NewAPIError("failed to fetch run", err)
}

// Check error types
if errors.IsRetryable(err) {
    // Retry logic
}

// Format for users
fmt.Fprintf(os.Stderr, "Error: %s\n", errors.FormatUserError(err))
```

### Context Usage

```go
// Always accept context for cancellation
func FetchData(ctx context.Context, id string) (*Data, error) {
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
        // Proceed with operation
    }
}
```

## API Integration

### Client Methods

The API client provides these core methods:
- `CreateRun(task)` - Submit a new task
- `GetRun(id)` - Fetch run details
- `ListRuns()` - List all runs
- `GetUserInfo()` - Get user information
- `VerifyAuth()` - Verify API key

All methods support:
- Automatic retry with exponential backoff
- Context cancellation
- Structured error responses

### Adding New Endpoints

1. Define models in `internal/api/models.go`
2. Add client method in `internal/api/client.go`
3. Implement retry wrapper if needed
4. Add command in `internal/commands/`
5. Write tests for all components

## TUI Development

### Bubble Tea Framework

The TUI uses [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the interactive interface.

Key concepts:
- **Model**: Application state
- **Update**: Handle messages and update state
- **View**: Render the UI

### Adding TUI Features

1. Create component in `internal/tui/components/`
2. Define messages in `internal/tui/messages/`
3. Update dashboard model in `internal/tui/dashboard/`
4. Handle updates in dashboard update function
5. Render in dashboard view function

## Configuration Management

### Viper Configuration

Configuration hierarchy (highest to lowest priority):
1. Command-line flags
2. Environment variables
3. Config file (`~/.repobird/config.yaml`)
4. Default values

### Adding Configuration Options

```go
// In internal/config/config.go
func init() {
    viper.SetDefault("new_option", "default_value")
    viper.BindEnv("new_option", "REPOBIRD_NEW_OPTION")
}

// In command
flags.String("new-option", "", "Description")
viper.BindPFlag("new_option", cmd.Flags().Lookup("new-option"))
```

## Release Process

### Version Management

1. Update version in `pkg/version/version.go`
2. Update CHANGELOG.md
3. Commit changes: `git commit -m "chore: bump version to vX.Y.Z"`
4. Tag release: `git tag vX.Y.Z`
5. Push with tags: `git push origin main --tags`

### Building Release Binaries

```bash
# Build all platforms
make build-all

# Creates binaries in build/
# - repobird-linux-amd64
# - repobird-linux-arm64
# - repobird-darwin-amd64
# - repobird-darwin-arm64
# - repobird-windows-amd64.exe
```

### GitHub Release

1. Create release on GitHub
2. Upload binaries from `build/` directory
3. Include changelog in release notes
4. Mark as pre-release if applicable

## Performance Profiling

### CPU Profiling

```go
import _ "net/http/pprof"

func init() {
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
}
```

Access profiles at `http://localhost:6060/debug/pprof/`

### Memory Profiling

```bash
# Run with memory profiling
go test -memprofile=mem.prof -bench=.

# Analyze profile
go tool pprof mem.prof
```

## Troubleshooting Development Issues

### Common Issues

1. **Module errors**: Run `go mod tidy`
2. **Build failures**: Check Go version with `go version`
3. **Test failures**: Ensure clean state with `make clean`
4. **Lint errors**: Run `make lint-fix`

### Development Tools

Recommended tools:
- **golangci-lint**: Comprehensive linting
- **delve**: Go debugger
- **go-task**: Task runner alternative to Make
- **air**: Live reload for development

## CI/CD Pipeline

### GitHub Actions Workflow

The CI pipeline runs:
1. Linting checks
2. Format verification
3. Unit tests with coverage
4. Integration tests
5. Cross-platform builds
6. Security scanning

### Pre-commit Hooks

Install pre-commit hooks:
```bash
cat > .git/hooks/pre-commit << 'EOF'
#!/bin/sh
make check
EOF
chmod +x .git/hooks/pre-commit
```

## Security Considerations

### API Key Handling
- Never log API keys
- Use secure storage (OS keychain when possible)
- Clear sensitive data from memory after use
- Validate API keys before storage

### Input Validation
- Sanitize all user inputs
- Validate JSON schemas
- Check file paths for directory traversal
- Limit request sizes

## Additional Resources

- [Project Documentation](docs/)
- [Architecture Overview](docs/architecture.md)
- [API Reference](docs/api-reference.md)
- [Testing Guide](docs/testing-guide.md)
- [Contributing Guidelines](CONTRIBUTING.md)
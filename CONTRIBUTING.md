# Contributing to RepoBird CLI

Thank you for your interest in contributing to RepoBird CLI! This document provides guidelines and instructions for contributing to the project.

## Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct:
- Be respectful and inclusive
- Welcome newcomers and help them get started
- Focus on constructive criticism
- Accept feedback gracefully
- Prioritize the project's best interests

## How to Contribute

### Reporting Issues

Before creating an issue:
1. Check existing issues to avoid duplicates
2. Use the issue search to see if it's already reported
3. Check if the issue is fixed in the latest version

When creating an issue, include:
- Clear, descriptive title
- Steps to reproduce the problem
- Expected vs actual behavior
- System information (OS, Go version)
- Relevant logs or error messages
- Screenshots if applicable

### Suggesting Enhancements

Enhancement suggestions are welcome! Please:
1. Check if the feature already exists
2. Search existing issues for similar suggestions
3. Provide a clear use case
4. Explain why this would be useful
5. Consider implementation complexity

### Pull Requests

#### Before You Start

1. **Discuss First**: For significant changes, open an issue first
2. **Check Issues**: Look for issues tagged `good first issue` or `help wanted`
3. **Read Documentation**: Familiarize yourself with [CLAUDE.md](CLAUDE.md) and project documentation in [docs/](docs/)

#### Development Process

1. **Fork and Clone**
   ```bash
   git clone https://github.com/yourusername/repobird-cli.git
   cd repobird-cli
   ```

2. **Create a Branch**
   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b fix/issue-description
   ```

3. **Make Changes**
   - Follow the code style guide
   - Write clear, self-documenting code
   - Add comments for complex logic
   - Update documentation as needed

4. **Write Tests**
   - Add unit tests for new functions
   - Ensure existing tests pass
   - Aim for >70% coverage on new code
   ```bash
   make test
   make coverage
   ```

5. **Lint and Format**
   ```bash
   make fmt
   make lint-fix
   make check
   ```

6. **Commit Changes**
   ```bash
   git add .
   git commit -m "type: brief description

   Detailed explanation of what changed and why.
   Reference any related issues: Fixes #123"
   ```

7. **Push and Create PR**
   ```bash
   git push origin your-branch-name
   ```

#### Commit Message Guidelines

Format: `<type>: <short summary>`

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code restructuring
- `test`: Test additions or fixes
- `chore`: Maintenance tasks
- `perf`: Performance improvements

Examples:
```
feat: add JSON schema validation for task files
fix: resolve timeout issue in status polling
docs: update installation instructions
refactor: simplify error handling in API client
```

#### Pull Request Guidelines

Your PR should:
1. Have a clear, descriptive title
2. Reference any related issues
3. Include a description of changes
4. List any breaking changes
5. Include test results
6. Have all CI checks passing

PR Description Template:
```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Manual testing completed

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Comments added for complex code
- [ ] Documentation updated
- [ ] No new warnings generated
- [ ] Tests added/updated
- [ ] All tests passing
```

## Development Guide

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

### Project Structure

```
repobird-cli/
├── cmd/repobird/          # Main application entry point
│   └── main.go           # CLI initialization and command setup
├── internal/              # Private application code
│   ├── api/              # API client and communication
│   │   ├── client.go     # HTTP client implementation
│   │   └── models.go     # API request/response types
│   ├── bulk/             # Bulk operations handling
│   ├── cache/            # Caching implementations
│   ├── commands/         # Cobra command implementations
│   │   ├── config.go     # Configuration management commands
│   │   ├── run.go        # Task submission command
│   │   ├── status.go     # Status checking command
│   │   └── tui.go        # Terminal UI command
│   ├── config/           # Configuration management
│   │   └── manager.go    # Configuration handling
│   ├── container/        # Container-related utilities
│   ├── domain/           # Domain models and logic
│   ├── errors/           # Error handling
│   │   └── types.go      # Custom error types and formatting
│   ├── models/           # Data models
│   ├── prompts/          # Prompt generation
│   ├── repository/       # Repository operations
│   ├── retry/            # Retry logic
│   │   └── retry.go      # Exponential backoff implementation
│   ├── services/         # Business logic services
│   ├── testutils/        # Testing utilities
│   ├── tui/              # Terminal UI components
│   │   ├── views/        # TUI views
│   │   ├── components/   # Reusable UI components
│   │   ├── cache/        # TUI caching layer
│   │   └── keymap/       # Key binding management
│   └── utils/            # Utility functions
├── pkg/                   # Public library code
│   ├── utils/            # Exported utilities
│   └── version/          # Version information
├── docs/                  # Documentation
├── examples/              # Example configurations
│   └── single-runs/      # Single run examples
├── scripts/               # Build and utility scripts
├── tests/                 # Test files
│   ├── integration/      # Integration tests
│   └── testdata/         # Test data files
└── testdata/              # Additional test fixtures
```

### Build System

#### Makefile Targets

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

#### Cross-Platform Building

```bash
# Build for specific platforms
GOOS=linux GOARCH=amd64 make build
GOOS=darwin GOARCH=arm64 make build
GOOS=windows GOARCH=amd64 make build

# Or use the convenience target
make build-all
```

### Code Style Guidelines

#### Go Code Standards

1. **Formatting**: Use `gofmt` (enforced by CI)
2. **Linting**: Pass `golangci-lint` checks
3. **Naming**: Follow Go naming conventions
   - Exported names start with capital letter
   - Use camelCase for variables
   - Use descriptive names

4. **Error Handling**:
   ```go
   // Good
   if err != nil {
       return fmt.Errorf("failed to parse task: %w", err)
   }
   
   // Avoid
   if err != nil {
       panic(err)
   }
   ```

5. **Comments**:
   ```go
   // Package api provides the HTTP client for RepoBird API communication.
   package api
   
   // CreateRun submits a new task to the RepoBird API.
   // It returns the created run or an error if the request fails.
   func CreateRun(task *Task) (*Run, error) {
       // Implementation
   }
   ```

6. **Context Usage**:
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

### Testing

#### Test Structure
- Unit tests: Alongside source files (`*_test.go`)
- Integration tests: In `tests/integration/`
- Mocks: Generated in `mocks/` directory

#### Writing Tests

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

#### Running Tests

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

### Debugging

#### Debug Mode

Enable debug output:
```bash
# Via flag
repobird --debug status

# Via environment
export REPOBIRD_DEBUG=true
repobird status
```

#### Local Development

```bash
# Run with local API server
export REPOBIRD_API_URL=http://localhost:8080
./build/repobird status

# Use debug TUI script
./debug-tui.sh
```

#### Logging

Debug logging locations:
- API requests/responses: When `--debug` flag is set
- Config operations: `~/.repobird/debug.log` (when enabled)
- TUI events: Stderr when run with `--debug`

### API Integration

#### Client Methods

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

#### Adding New Endpoints

1. Define models in `internal/api/models.go`
2. Add client method in `internal/api/client.go`
3. Implement retry wrapper if needed
4. Add command in `internal/commands/`
5. Write tests for all components

### TUI Development

#### Bubble Tea Framework

The TUI uses [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the interactive interface.

Key concepts:
- **Model**: Application state
- **Update**: Handle messages and update state
- **View**: Render the UI

#### Adding TUI Features

1. Create component in `internal/tui/components/`
2. Define messages in `internal/tui/messages/`
3. Update dashboard model in `internal/tui/dashboard/`
4. Handle updates in dashboard update function
5. Render in dashboard view function

### Configuration Management

#### Viper Configuration

Configuration hierarchy (highest to lowest priority):
1. Command-line flags
2. Environment variables
3. Config file (`~/.repobird/config.yaml`)
4. Default values

#### Adding Configuration Options

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

### Performance Profiling

#### CPU Profiling

```go
import _ "net/http/pprof"

func init() {
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
}
```

Access profiles at `http://localhost:6060/debug/pprof/`

#### Memory Profiling

```bash
# Run with memory profiling
go test -memprofile=mem.prof -bench=.

# Analyze profile
go tool pprof mem.prof
```

### CI/CD Pipeline

#### GitHub Actions Workflow

The CI pipeline runs:
1. Linting checks
2. Format verification
3. Unit tests with coverage
4. Integration tests
5. Cross-platform builds
6. Security scanning

#### Pre-commit Hooks

Install pre-commit hooks:
```bash
cat > .git/hooks/pre-commit << 'EOF'
#!/bin/sh
make check
EOF
chmod +x .git/hooks/pre-commit
```

### Security Considerations

#### API Key Handling
- Never log API keys
- Use secure storage (OS keychain when possible)
- Clear sensitive data from memory after use
- Validate API keys before storage

#### Input Validation
- Sanitize all user inputs
- Validate JSON schemas
- Check file paths for directory traversal
- Limit request sizes

## Development Roadmap

### High Priority Tasks

#### Enhanced Pagination for 100+ Runs
- [ ] Implement cache infrastructure for run pagination
- [ ] Add API pagination enhancement with retry logic  
- [ ] Modify TUI navigation to prevent wrap-around behavior
- [ ] Add Load More button for manual pagination
- [ ] Show "X of Y total runs loaded" in status line
- [ ] Test with users having 1000+ runs

**Current Status:** ✅ Simple fix implemented - increased limit to 1000 runs

#### Usage Progress Display
- [ ] Fix usage progress bar showing 0.00% on status page
- [X] Hard-code free/pro tier usage limits
- [X] Allow admin-credited extra runs to exceed limits

### Medium Priority Tasks

#### Global Configuration Support
- [ ] Implement global config file support (~/.repobird/global.yaml)
- [ ] Add environment-specific configurations (dev, staging, prod)
- [ ] Support config inheritance and overrides
- [ ] Add config validation and schema

#### Improve FZF Integration
- [ ] Add inline FZF selection for run IDs
- [ ] Implement FZF for task file selection
- [ ] Add FZF preview for task JSON files
- [ ] Support FZF for branch selection
- [ ] Handle FZF binary detection gracefully

#### Enhanced TUI Features
- [ ] Add real-time log streaming in TUI
- [ ] Implement split pane view for multiple runs
- [ ] Add keyboard shortcuts for common actions
- [ ] Support theme customization
- [ ] Add TUI configuration persistence

#### Batch Operations Support
- [ ] Support running multiple tasks from directory
- [ ] Add batch status checking
- [ ] Implement parallel task execution
- [ ] Add batch cancellation support
- [ ] Create batch results summary view

#### Offline Mode Implementation
- [ ] Cache run data locally
- [ ] Queue tasks for later submission
- [ ] Sync when connection restored
- [ ] Add offline status indicators
- [ ] Implement conflict resolution

### Low Priority Tasks

#### Plugin System
- [ ] Design plugin architecture
- [ ] Add plugin discovery mechanism
- [ ] Implement plugin API
- [ ] Create example plugins
- [ ] Add plugin marketplace support

#### Multi-Repository Support
- [ ] Support GitLab repositories
- [ ] Add Bitbucket integration
- [ ] Implement generic Git support
- [ ] Add repository switching
- [ ] Create repository profiles

See the full roadmap and detailed task descriptions in [docs/roadmap.md](docs/roadmap.md).

## Release Process

### Version Management

1. Bump version using Makefile commands:
   - `make bump-patch` for patch releases (0.0.X)
   - `make bump-minor` for minor releases (0.X.0)
   - `make bump-major` for major releases (X.0.0)
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

## Troubleshooting Development Issues

### Common Issues

1. **Module errors**: Run `go mod tidy`
2. **Build failures**: Check Go version with `go version`
3. **Test failures**: Ensure clean state with `make clean`
4. **Lint errors**: Run `make lint-fix`

### Development Tools

Recommended tools:
```bash
# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install golang.org/x/tools/cmd/goimports@latest
```

### Useful Commands

```bash
# Quick development cycle
make clean build test

# Full CI simulation
make ci

# Debug mode
./build/repobird --debug status
```

## Getting Help

If you need help:
1. Check existing documentation
2. Search closed issues
3. Ask in PR comments
4. Open a discussion issue

## Community

### Communication Channels

- **Issues**: Bug reports and feature requests
- **Pull Requests**: Code contributions and discussions
- **Discussions**: General questions and ideas

### Recognition

Contributors are recognized in:
- Release notes
- Contributors file
- Project documentation

## Quick Checklist for Contributors

Before submitting your PR, ensure:

- [ ] Fork is up to date with main branch
- [ ] Branch follows naming convention
- [ ] Code compiles without warnings
- [ ] All tests pass locally
- [ ] New code has tests
- [ ] Documentation is updated
- [ ] Commit messages follow guidelines
- [ ] PR description is complete
- [ ] CI checks are passing

## Additional Resources

- [Project Documentation](docs/)
- [Architecture Overview](docs/architecture.md)
- [API Reference](docs/api-reference.md)
- [Testing Guide](docs/testing-guide.md)
- [TUI Guide](docs/tui-guide.md)
- [Configuration Guide](docs/configuration-guide.md)

## Thank You!

Your contributions make RepoBird CLI better for everyone. We appreciate your time and effort in improving the project!

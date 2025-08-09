# RepoBird CLI - Development Guide

## Prerequisites

### Required Software
- **Go 1.20+**: [Download Go](https://golang.org/dl/)
- **Git**: Version control system
- **Make**: Build automation tool
- **golangci-lint**: Code quality tool

### Optional Tools
- **GoReleaser**: For creating releases
- **Docker**: For containerized builds
- **ngrok**: For testing with local API server

## Setting Up Development Environment

### 1. Clone the Repository
```bash
git clone https://github.com/repobird/cli.git
cd cli
```

### 2. Install Dependencies
```bash
# Install Go dependencies
go mod download

# Install development tools
make install-tools

# Verify installation
make check
```

### 3. Environment Configuration
```bash
# Create local config directory
mkdir -p ~/.repobird

# Set development API endpoint (optional)
export REPOBIRD_API_URL=http://localhost:8080

# Set API key for testing
export REPOBIRD_API_KEY=your_test_key

# Enable debug mode
export REPOBIRD_DEBUG=true
```

## Project Structure

```
.
├── cmd/repobird/          # Main entry point
├── internal/              # Private packages
│   ├── api/              # API client
│   ├── cache/            # Caching layer
│   ├── commands/         # CLI commands
│   ├── config/           # Configuration
│   ├── domain/           # Business logic
│   ├── errors/           # Error handling
│   ├── models/           # Data models
│   ├── retry/            # Retry logic
│   ├── tui/              # Terminal UI
│   └── utils/            # Utilities
├── pkg/                   # Public packages
│   ├── utils/            # Public utilities
│   └── version/          # Version info
├── docs/                  # Documentation
├── scripts/              # Build scripts
└── Makefile              # Build automation
```

## Building the Project

### Quick Build
```bash
# Build for current platform
make build

# Run without installing
make run ARGS="status"

# Install locally
make install
```

### Cross-Platform Builds
```bash
# Build for all platforms
make build-all

# Build for specific platform
GOOS=linux GOARCH=amd64 make build
GOOS=darwin GOARCH=arm64 make build
GOOS=windows GOARCH=amd64 make build
```

### Build with Version Info
```bash
# Set version
VERSION=v1.2.3 make build

# Build with git info
make build-release
```

## Development Workflow

### 1. Creating a New Feature

#### Branch Setup
```bash
# Create feature branch
git checkout -b feature/your-feature

# Or for bug fixes
git checkout -b fix/issue-description
```

#### Code Organization
1. **Commands**: Add to `/internal/commands/`
2. **API Methods**: Add to `/internal/api/client.go`
3. **Models**: Add to `/internal/models/`
4. **TUI Views**: Add to `/internal/tui/views/`

### 2. Adding a New Command

Create `/internal/commands/newcmd.go`:
```go
package commands

import (
    "fmt"
    "github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
    Use:   "newcmd [args]",
    Short: "Brief description",
    Long:  `Detailed description`,
    RunE:  runNewCmd,
}

func init() {
    // Add flags
    newCmd.Flags().StringP("option", "o", "", "Option description")
    
    // Register with root
    rootCmd.AddCommand(newCmd)
}

func runNewCmd(cmd *cobra.Command, args []string) error {
    // Implementation
    return nil
}
```

### 3. Adding API Endpoints

Update `/internal/api/client.go`:
```go
// Add method to Client
func (c *Client) NewEndpoint(ctx context.Context, param string) (*Response, error) {
    endpoint := fmt.Sprintf("/api/v1/resource/%s", param)
    
    req, err := c.newRequest(ctx, "GET", endpoint, nil)
    if err != nil {
        return nil, err
    }
    
    var response Response
    err = c.doRequest(req, &response)
    return &response, err
}

// Add retry wrapper
func (c *Client) NewEndpointWithRetry(ctx context.Context, param string) (*Response, error) {
    return retry.Do(ctx, func() (*Response, error) {
        return c.NewEndpoint(ctx, param)
    }, c.retryConfig)
}
```

### 4. Adding TUI Views

Create `/internal/tui/views/newview.go`:
```go
package views

import (
    tea "github.com/charmbracelet/bubbletea"
)

type NewView struct {
    // State fields
}

func NewNewView() *NewView {
    return &NewView{}
}

func (v *NewView) Init() tea.Cmd {
    return nil
}

func (v *NewView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Handle keys
    }
    return v, nil
}

func (v *NewView) View() string {
    return "View content"
}
```

## Code Style Guidelines

### Go Conventions
```go
// Package comment
package mypackage

import (
    // Standard library
    "context"
    "fmt"
    
    // External packages
    "github.com/spf13/cobra"
    
    // Internal packages
    "github.com/repobird/cli/internal/api"
)

// Constants group related values
const (
    DefaultTimeout = 30 * time.Second
    MaxRetries     = 3
)

// Interfaces define contracts
type Service interface {
    Method(ctx context.Context) error
}

// Structs implement interfaces
type serviceImpl struct {
    client *api.Client
}

// Methods follow receiver-name-method pattern
func (s *serviceImpl) Method(ctx context.Context) error {
    // Always handle errors explicitly
    if err := s.validate(); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    
    return nil
}
```

### Error Handling
```go
// Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to process %s: %w", item, err)
}

// Use custom error types
type ValidationError struct {
    Field string
    Value interface{}
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("invalid %s: %v", e.Field, e.Value)
}

// Check error types
var validationErr ValidationError
if errors.As(err, &validationErr) {
    // Handle validation error
}
```

### Testing Patterns
```go
func TestFunction(t *testing.T) {
    // Table-driven tests
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid input", "test", "TEST", false},
        {"empty input", "", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Function(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("got = %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Running Tests

### Unit Tests
```bash
# Run all tests
make test

# Run with coverage
make coverage

# Run specific package
go test ./internal/api/...

# Run with verbose output
go test -v ./...

# Run specific test
go test -run TestClientCreate ./internal/api
```

### Integration Tests
```bash
# Run integration tests
make test-integration

# With real API (requires API key)
REPOBIRD_API_KEY=real_key make test-integration
```

### Benchmarks
```bash
# Run benchmarks
make benchmark

# Run specific benchmark
go test -bench=BenchmarkCache ./internal/cache
```

## Debugging

### Enable Debug Logging
```bash
# Via environment
export REPOBIRD_DEBUG=true

# Via flag
repobird status --debug

# In code
debug.LogToFile("Debug message")
debug.LogToFilef("Formatted: %v", data)
```

### Debug TUI Application
```bash
# Run TUI with debug output
./debug-tui.sh

# Or manually
REPOBIRD_DEBUG=true repobird tui 2>debug.log
```

### Using Delve Debugger
```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug run
dlv debug ./cmd/repobird -- status

# Set breakpoint
(dlv) break main.main
(dlv) continue
```

### Memory Profiling
```bash
# Build with profiling
go build -gcflags="all=-N -l" ./cmd/repobird

# Run with profiling
GODEBUG=gctrace=1 ./repobird status

# Analyze with pprof
go tool pprof mem.prof
```

## Common Development Tasks

### Adding Configuration Options
```go
// In internal/config/config.go
type Config struct {
    NewOption string `mapstructure:"new_option"`
}

// In internal/commands/root.go
rootCmd.PersistentFlags().String("new-option", "", "Description")
viper.BindPFlag("new_option", rootCmd.PersistentFlags().Lookup("new-option"))
```

### Implementing Retry Logic
```go
import "github.com/repobird/cli/internal/retry"

result, err := retry.Do(ctx, func() (*Result, error) {
    return apiCall()
}, retry.Config{
    MaxRetries: 3,
    InitialDelay: time.Second,
})
```

### Adding Cache Support
```go
import "github.com/repobird/cli/internal/cache"

// Check cache
if cached, found := cache.Get(key); found {
    return cached.(*MyType), nil
}

// Store in cache
cache.Set(key, value, 30*time.Second)
```

### Creating User-Friendly Errors
```go
import "github.com/repobird/cli/internal/errors"

// Create error
err := errors.NewAPIError(401, "Unauthorized")

// Check type
if errors.IsAuthError(err) {
    fmt.Println("Please login first")
}

// Format for user
userMsg := errors.FormatUserError(err)
```

## Git Hooks

### Pre-commit Hook
Create `.git/hooks/pre-commit`:
```bash
#!/bin/sh
make fmt-check
make lint
make test-unit
```

### Commit Message Format
```
<type>(<scope>): <subject>

<body>

<footer>
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `style`: Formatting
- `refactor`: Code restructuring
- `test`: Adding tests
- `chore`: Maintenance

## Release Process

### 1. Update Version
```bash
# Update version in code
VERSION=v1.2.3 make set-version

# Commit changes
git commit -am "chore: bump version to v1.2.3"
```

### 2. Create Tag
```bash
git tag -a v1.2.3 -m "Release v1.2.3"
git push origin v1.2.3
```

### 3. Build Release
```bash
# Build all platforms
make release

# Or use GoReleaser
goreleaser release --clean
```

### 4. Create GitHub Release
```bash
gh release create v1.2.3 \
  --title "Release v1.2.3" \
  --notes "Release notes" \
  ./dist/*
```

## Performance Optimization

### Profiling
```go
import _ "net/http/pprof"

// Add to main.go
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()

// Profile CPU
go tool pprof http://localhost:6060/debug/pprof/profile

// Profile memory
go tool pprof http://localhost:6060/debug/pprof/heap
```

### Optimization Tips
1. **Use sync.Pool** for frequently allocated objects
2. **Preallocate slices** when size is known
3. **Avoid string concatenation** in loops
4. **Use buffered channels** for async operations
5. **Cache expensive computations**

## Troubleshooting Development Issues

### Dependency Issues
```bash
# Clear module cache
go clean -modcache

# Update dependencies
go get -u ./...
go mod tidy

# Vendor dependencies
go mod vendor
```

### Build Issues
```bash
# Clean build cache
go clean -cache

# Rebuild everything
make clean
make build
```

### Test Failures
```bash
# Run tests with race detector
go test -race ./...

# Increase timeout
go test -timeout 30s ./...

# Skip integration tests
go test -short ./...
```

## IDE Setup

### VS Code
`.vscode/settings.json`:
```json
{
  "go.useLanguageServer": true,
  "go.lintTool": "golangci-lint",
  "go.lintFlags": ["--fast"],
  "go.testFlags": ["-v"],
  "go.buildTags": "integration",
  "go.testEnvVars": {
    "REPOBIRD_DEBUG": "true"
  }
}
```

### GoLand
1. Set Go SDK to 1.20+
2. Enable Go Modules
3. Configure golangci-lint
4. Set environment variables in Run Configuration

## Contributing

### Code Review Checklist
- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] Lint passes
- [ ] No sensitive data exposed
- [ ] Error handling complete
- [ ] Performance considered
- [ ] Backwards compatible

### Pull Request Template
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
- [ ] Documentation updated
- [ ] No warnings generated
```

## Resources

### Documentation
- [Go Documentation](https://golang.org/doc/)
- [Cobra Documentation](https://cobra.dev/)
- [Bubble Tea Documentation](https://github.com/charmbracelet/bubbletea)

### Tools
- [golangci-lint](https://golangci-lint.run/)
- [GoReleaser](https://goreleaser.com/)
- [Delve Debugger](https://github.com/go-delve/delve)

### Community
- GitHub Issues: Report bugs and request features
- Discussions: Ask questions and share ideas
- Discord: Real-time chat with developers
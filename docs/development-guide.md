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

The TUI uses a message-based navigation pattern with minimal view constructors and shared cache management.

#### Constructor Pattern

**✅ NEW: Minimal Constructor Pattern**
Create `/internal/tui/views/newview.go`:
```go
package views

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/repobird/repobird-cli/internal/tui/cache"
    "github.com/repobird/repobird-cli/internal/tui/components"
    "github.com/repobird/repobird-cli/internal/tui/messages"
)

type NewView struct {
    client APIClient           // API client dependency
    cache  *cache.SimpleCache  // Shared cache from app-level
    id     string              // Resource ID for loading data
    
    // UI components (use shared components when possible)
    list   *components.ScrollableList
    form   *components.Form
    
    // State fields (NO parent state)
    width  int
    height int
    loading bool
    // ... other view-specific state only
}

// ✅ NEW: Minimal constructor - maximum 3 parameters
func NewNewView(client APIClient, cache *cache.SimpleCache, id string) *NewView {
    return &NewView{
        client:  client,
        cache:   cache,         // Shared cache instance from App
        id:      id,
        loading: true,          // Always start loading
        list:    components.NewScrollableList(),
        form:    components.NewForm(),
    }
}

func (v *NewView) Init() tea.Cmd {
    // Load view's own data
    return v.loadData()
}

func (v *NewView) loadData() tea.Cmd {
    return func() tea.Msg {
        // Check cache first
        if data := v.cache.GetData(v.id); data != nil {
            return dataLoadedMsg{data: data}
        }
        
        // Load from API
        data, err := v.client.GetData(v.id)
        if err != nil {
            return errMsg{err: err}
        }
        
        // Cache for next time
        v.cache.SetData(v.id, data)
        return dataLoadedMsg{data: data}
    }
}

func (v *NewView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "b", "esc":
            // Navigate back using message
            return v, func() tea.Msg {
                return messages.NavigateBackMsg{}
            }
        case "d":
            // Navigate to dashboard using message
            return v, func() tea.Msg {
                return messages.NavigateToDashboardMsg{}
            }
        case "enter":
            // Navigate to another view with context
            return v, func() tea.Msg {
                return messages.NavigateToDetailsMsg{
                    RunID: v.getSelectedRunID(),
                }
            }
        }
    case tea.WindowSizeMsg:
        v.width = msg.Width
        v.height = msg.Height
        // Update component dimensions
        v.list.Update(msg)
        v.form.Update(msg)
    }
    
    // Update child components
    var cmds []tea.Cmd
    listModel, listCmd := v.list.Update(msg)
    v.list = listModel.(*components.ScrollableList)
    cmds = append(cmds, listCmd)
    
    formModel, formCmd := v.form.Update(msg)
    v.form = formModel.(*components.Form)
    cmds = append(cmds, formCmd)
    
    return v, tea.Batch(cmds...)
}

func (v *NewView) View() string {
    if v.width == 0 || v.height == 0 {
        return ""
    }
    
    // Use shared components for consistent UI
    return v.list.View() + "\n" + v.form.View()
}

// Helper methods
func (v *NewView) getSelectedRunID() string {
    // Implementation
    return ""
}
```

#### Using Navigation Context

Views can share data through navigation context:

```go
// In the originating view (e.g., Dashboard)
func (v *DashboardView) navigateToCreate() tea.Cmd {
    // Set context before navigation
    v.cache.SetNavigationContext("selected_repo", v.getSelectedRepo())
    v.cache.SetNavigationContext("user_preference", "advanced")
    
    return func() tea.Msg {
        return messages.NavigateToCreateMsg{
            SelectedRepository: v.getSelectedRepo(),
        }
    }
}

// In the target view (e.g., CreateRunView)
func (v *CreateRunView) Init() tea.Cmd {
    // Retrieve context
    if repo := v.cache.GetNavigationContext("selected_repo"); repo != nil {
        v.setRepository(repo.(string))
    }
    
    if pref := v.cache.GetNavigationContext("user_preference"); pref != nil {
        v.setMode(pref.(string))
    }
    
    return nil
}
```

### 5. Using Shared Components

The TUI provides reusable components to ensure consistency and reduce code duplication.

#### ScrollableList Component

Use for multi-column scrollable lists with keyboard navigation:

```go
import "github.com/repobird/repobird-cli/internal/tui/components"

// Create with configuration
list := components.NewScrollableList(
    components.WithColumns(3),
    components.WithKeyNavigation(true),
    components.WithValueNavigation(true),
    components.WithDimensions(100, 50),
    components.WithColumnWidths([]int{40, 30, 30}),
)

// Set data
items := [][]string{
    {"Run ID", "Status", "Repository"},
    {"123", "DONE", "org/repo"},
    {"456", "RUNNING", "org/other"},
}
list.SetItems(items)

// Handle in Update method
listModel, cmd := list.Update(msg)
v.list = listModel.(*components.ScrollableList)

// Get selection
selected := list.GetSelected()           // Current row data
index := list.GetSelectedIndex()         // Current row index
list.SetSelected(5)                      // Set selection programmatically

// Render
listView := list.View()
```

#### Form Component

Use for input forms with validation and field management:

```go
// Create form
form := components.NewForm(
    components.WithFormDimensions(80, 30),
    components.WithValidation(true),
)

// Define fields
fields := []components.FormField{
    {
        Key:         "repository",
        Label:       "Repository",
        Type:        components.TextInput,
        Required:    true,
        Placeholder: "org/repo",
        Validator: func(value string) error {
            if !strings.Contains(value, "/") {
                return errors.New("repository must be in org/repo format")
            }
            return nil
        },
    },
    {
        Key:      "description",
        Label:    "Description",
        Type:     components.TextArea,
        Required: false,
    },
    {
        Key:     "run_type",
        Label:   "Run Type",
        Type:    components.Select,
        Options: []string{"run", "plan", "approval"},
    },
}
form.SetFields(fields)

// Handle in Update method
formModel, cmd := form.Update(msg)
v.form = formModel.(*components.Form)

// Get form data
if form.IsComplete() {
    data := form.GetData()
    repository := data["repository"].(string)
    description := data["description"].(string)
}

// Set values programmatically
form.SetValue("repository", "my-org/my-repo")

// Render
formView := form.View()
```

#### ErrorView Component

Use for consistent error display with recovery options:

```go
// Navigate to error view using message
return v, func() tea.Msg {
    return messages.NavigateToErrorMsg{
        Error:       err,
        Message:     "Failed to load run details",
        Recoverable: true, // Allow going back
    }
}

// For non-recoverable errors
return v, func() tea.Msg {
    return messages.NavigateToErrorMsg{
        Error:       err,
        Message:     "Critical system error",
        Recoverable: false, // Clears navigation stack
    }
}
```

#### FZF Selector Component

The FZF selector provides fuzzy search functionality:

```go
// internal/tui/components/fzf_selector.go
import "github.com/sahilm/fuzzy"

// Create FZF mode
fzf := components.NewFZFMode(items, width, height)
fzf.Activate()

// Handle in Update method
if fzf.IsActive() {
    newFzf, cmd := fzf.Update(msg)
    // Handle FZFSelectedMsg
}

// Render dropdown
fzfView := fzf.View()
```

#### Repository Selector

Provides repository selection with history:

```go
// internal/tui/components/repository_selector.go
selector := components.NewRepositorySelector()
repo, err := selector.SelectRepository()
```

#### Integrating FZF in Views

1. Add FZF state to view struct:
```go
type MyView struct {
    fzfMode   *components.FZFMode
    fzfActive bool
}
```

2. Handle activation:
```go
case "f": // Activate FZF
    items := getItems()
    v.fzfMode = components.NewFZFMode(items, width, height)
    v.fzfMode.Activate()
```

3. Handle selection:
```go
case components.FZFSelectedMsg:
    if !msg.Result.Canceled {
        // Process selection
        selected := msg.Result.Selected
    }
    v.fzfMode = nil
```

4. Render with overlay:
```go
if v.fzfMode != nil && v.fzfMode.IsActive() {
    return v.renderWithFZFOverlay(baseView)
}
```

#### Navigation Best Practices

**✅ DO:**
- Use navigation messages for all view transitions
- Embed `*cache.SimpleCache` in view structs for state management
- Use shared components for consistency
- Set navigation context before navigating
- Handle window resize in all views
- Clear navigation context when returning to dashboard

**❌ DON'T:**
- Create child views directly in Update methods
- Use global variables for view state
- Store navigation state in view fields
- Bypass the App router for navigation
- Create duplicate UI components

**Message-Based Navigation Pattern:**
```go
// ✅ CORRECT: Use navigation messages
case "enter":
    return v, func() tea.Msg {
        return messages.NavigateToDetailsMsg{
            RunID: selectedID,
            FromCreate: true,
        }
    }

// ❌ INCORRECT: Direct view creation
case "enter":
    detailsView := views.NewRunDetailsView(v.client, selectedID)
    return detailsView, detailsView.Init()
```

**Context Management Pattern:**
```go
// ✅ CORRECT: Use navigation context
v.cache.SetNavigationContext("form_data", formValues)
return v, func() tea.Msg {
    return messages.NavigateToConfirmMsg{}
}

// ❌ INCORRECT: Pass data in message fields
return v, func() tea.Msg {
    return messages.NavigateToConfirmMsg{
        FormData: formValues, // Tight coupling
    }
}
```

**Component Update Pattern:**
```go
// ✅ CORRECT: Proper component updates
var cmds []tea.Cmd
listModel, listCmd := v.list.Update(msg)
v.list = listModel.(*components.ScrollableList)
cmds = append(cmds, listCmd)

return v, tea.Batch(cmds...)

// ❌ INCORRECT: Ignoring component updates
v.list.Update(msg) // Lost return values
return v, nil
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

## Cache Concurrency Patterns

### Safe Cache Operations
The cache system uses a layered approach to prevent deadlocks:

```go
// Pattern 1: Proper lock ordering in SimpleCache
func (c *SimpleCache) Get(key string) (*Run, error) {
    c.mu.RLock()
    // Release lock BEFORE calling HybridCache to prevent deadlocks
    c.mu.RUnlock()
    
    return c.hybridCache.Get(key)
}

// Pattern 2: Single-decision routing in HybridCache  
func (h *HybridCache) Get(key string) (*Run, error) {
    // Make routing decision WITHOUT holding locks
    if h.shouldUseSession(key) {
        return h.sessionCache.Get(key)
    }
    return h.permanentCache.Get(key)
}

// Pattern 3: Lock-free file I/O in PermanentCache
func (p *PermanentCache) saveToFile(data []byte) error {
    tempFile := p.filePath + ".tmp"
    // Write to temp file, then atomic rename
    if err := os.WriteFile(tempFile, data, 0644); err != nil {
        return err
    }
    return os.Rename(tempFile, p.filePath)
}
```

### Batch Update Pattern
For TUI views handling multiple cache operations:

```go
// Collect all updates first
updates := make(map[string]*Run)
for _, runID := range runIDs {
    if run, err := fetchFromAPI(runID); err == nil {
        updates[runID] = run
    }
}

// Apply all updates in a single batch
cache.BatchUpdate(updates)
```

### Race Detection
Run tests with race detection enabled:

```bash
# Enable race detector for all tests
go test -race ./...

# Focus on cache-related tests
go test -race ./internal/cache/...

# Integration tests with race detection
make test-integration GOFLAGS=-race
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

### Debug Cache Issues
```bash
# Enable cache-specific debugging
export REPOBIRD_DEBUG_CACHE=true

# Monitor cache operations
tail -f /tmp/repobird_debug.log | grep -i cache

# Check for deadlock patterns
go test -race -v ./internal/cache/... 2>&1 | grep -i "race\|deadlock"
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

## Dependencies Management

### Core Dependencies

```go
// Terminal UI
github.com/charmbracelet/bubbletea  // TUI framework
github.com/charmbracelet/lipgloss   // Styling
github.com/charmbracelet/bubbles    // UI components

// Fuzzy Search
github.com/sahilm/fuzzy              // Fuzzy string matching
github.com/ktr0731/go-fuzzyfinder   // Interactive fuzzy finder

// CLI Framework
github.com/spf13/cobra               // Command-line interface
github.com/spf13/viper               // Configuration

// Utilities
github.com/briandowns/spinner       // Progress indicators
github.com/fatih/color              // Terminal colors
```

### Adding Dependencies

```bash
# Add new dependency
go get github.com/package/name

# Update go.mod and go.sum
go mod tidy

# Verify
go mod verify
```

### Updating Dependencies

```bash
# Update all dependencies
go get -u ./...

# Update specific dependency
go get -u github.com/package/name

# Clean up
go mod tidy
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
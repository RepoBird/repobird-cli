# RepoBird CLI - Code Smells & Refactoring Tasks

## Executive Summary
This document identifies critical code quality issues, architectural violations, and refactoring opportunities in the RepoBird CLI codebase. Issues are prioritized by impact and effort, with actionable tasks for resolution.

## Critical Issues (P0)

### 1. Global State Anti-Pattern
**Problem**: Multiple global variables violate Go best practices and make testing difficult
- `globalCache` in `/internal/cache/cache.go:64`
- `dashboardCache` in `/internal/cache/dashboard_cache.go:41`
- Package-level `var` commands throughout `/internal/commands/*.go`
- `init()` function initializes cache at package import time

**Impact**: 
- Race conditions in concurrent scenarios
- Difficult to test in isolation
- Hidden dependencies
- Violates dependency injection principles

**Solution**:
```go
// Replace global cache with injected service
type Container struct {
    Cache      cache.Interface
    Config     config.Service
    APIClient  *api.Client
}

// Commands become functions that accept container
func NewRunCommand(c *Container) *cobra.Command {
    return &cobra.Command{
        RunE: func(cmd *cobra.Command, args []string) error {
            // Use c.Cache instead of globalCache
        },
    }
}
```

### 2. Duplicate Code - Truncate Functions
**Problem**: 6+ different truncate implementations doing the same thing
- `/internal/tui/components/statusline.go:276` - `truncateWithEllipsis`
- `/internal/tui/views/bulk_fzf.go:50` - `truncateString`
- `/internal/tui/views/dashboard.go:3407` - `truncateString` (marked deprecated but still used)
- `/internal/tui/components/help_view.go:545` - `truncateString`
- `/internal/tui/components/table.go:250` - `truncate`
- `/internal/commands/status.go:183` - `truncate`

**Solution**:
```go
// Create single utility function
// internal/utils/strings.go
package utils

import "github.com/charmbracelet/lipgloss"

func TruncateWithEllipsis(s string, maxWidth int) string {
    if maxWidth <= 3 {
        return "..."
    }
    if lipgloss.Width(s) <= maxWidth {
        return s
    }
    // Handle unicode properly
    runes := []rune(s)
    if len(runes) > maxWidth-3 {
        return string(runes[:maxWidth-3]) + "..."
    }
    return s
}
```

## High Priority Issues (P1)

### 3. Cache Architecture Chaos
**Problem**: Multiple overlapping cache implementations without clear boundaries
- `GlobalCache` mixes runs, details, forms, user info, file hashes
- `DashboardCache` duplicates functionality
- No consistent TTL strategy
- Missing invalidation logic
- No abstraction interface

**Current State**:
```
GlobalCache (kitchen sink)
├── runs (list cache)
├── details (temporary)
├── terminalDetails (permanent)
├── formData (UI state)
├── userInfo (auth)
└── fileHashCache (bulk)
```

**Solution**:
```go
// Define clear cache interfaces
type CacheService interface {
    RunCache
    UserCache
    FormCache
}

type RunCache interface {
    GetRuns(ctx context.Context, userID int) ([]models.Run, error)
    SetRuns(ctx context.Context, userID int, runs []models.Run, ttl time.Duration) error
    InvalidateRuns(ctx context.Context, userID int) error
}

// Separate implementations
type MemoryCache struct {
    runs *ttlcache.Cache
}

type PersistentCache struct {
    dir string
}
```

### 4. StatusLine State Management
**Problem**: StatusLine state scattered across views with duplicate update logic
- Manual width propagation in every view
- Duplicate spinner update calls
- Inconsistent temporary message handling
- No central state management

**Issues Found**:
- `/internal/tui/views/dashboard.go:559` - Duplicate spinner updates
- Every view manually sets `statusLine.SetWidth(v.width)`
- Temporary message logic mixed with rendering

**Solution**:
```go
// Centralized StatusLine manager
type StatusManager struct {
    statusLine *StatusLine
    width      int
    loading    bool
}

func (sm *StatusManager) Update(msg tea.Msg) tea.Cmd {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        sm.width = msg.Width
        sm.statusLine.SetWidth(msg.Width)
    case spinner.TickMsg:
        if sm.loading {
            sm.statusLine.UpdateSpinner()
        }
    }
    return nil
}
```

### 5. Constructor Explosion
**Problem**: Multiple constructor variants for views violate DRY principle
- `NewRunListView`
- `NewRunListViewWithCache`
- `NewRunListViewWithCacheAndDimensions`
- Similar pattern across all views

**Solution**: Use functional options pattern
```go
type ViewOption func(*RunListView)

func WithCache(cache cache.Interface) ViewOption {
    return func(v *RunListView) {
        v.cache = cache
    }
}

func WithDimensions(width, height int) ViewOption {
    return func(v *RunListView) {
        v.width = width
        v.height = height
    }
}

func NewRunListView(client APIClient, opts ...ViewOption) *RunListView {
    v := &RunListView{
        client: client,
        // defaults
    }
    for _, opt := range opts {
        opt(v)
    }
    return v
}
```

## Medium Priority Issues (P2)

### 6. Clean Architecture Violations
**Problem**: No clear separation between layers
- Business logic in command handlers
- Direct Viper access throughout code
- Views directly access global state
- No domain/use-case/adapter separation

**Example Violations**:
- `/internal/commands/auth.go:266` - `var cacheTimeout = 5 * time.Minute` (business logic in command)
- Direct `viper.GetString()` calls scattered everywhere
- Views import cache package directly

**Solution**: Implement clean architecture layers
```
cmd/repobird/
├── main.go (wire dependencies)
internal/
├── domain/       (entities, no external deps)
│   └── run.go
├── usecases/     (business logic)
│   └── create_run.go
├── adapters/     (external interfaces)
│   ├── cli/      (cobra commands)
│   ├── tui/      (bubble tea views)
│   └── api/      (HTTP client)
└── infrastructure/ (implementations)
    ├── cache/
    └── config/
```

### 7. Bubble Tea Anti-Patterns
**Problem**: Framework misuse causing performance issues
- Potential blocking operations in Update()
- Inconsistent Cmd batching
- Missing cancellation for async operations
- Direct state mutation without messages

**Issues**:
- Some Update() methods don't batch Cmds properly
- No consistent cancellation pattern for long-running operations
- State changes without going through message flow

**Solution**:
```go
// Proper async operation with cancellation
type loadDataMsg struct {
    data []models.Run
    err  error
}

func loadDataCmd(ctx context.Context) tea.Cmd {
    return func() tea.Msg {
        ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
        defer cancel()
        
        data, err := fetchData(ctx)
        return loadDataMsg{data, err}
    }
}
```

### 8. Cobra Command Structure
**Problem**: Package-level var commands instead of functions
- All commands defined as `var cmdName = &cobra.Command{}`
- No dependency injection support
- Difficult to test in isolation
- Flag validation scattered

**Solution**:
```go
// Convert to functions that return commands
func NewRootCommand(container *Container) *cobra.Command {
    cmd := &cobra.Command{
        Use: "repobird",
    }
    
    // Add subcommands with dependencies
    cmd.AddCommand(
        NewRunCommand(container),
        NewStatusCommand(container),
    )
    
    // Use Cobra's flag grouping
    cmd.MarkFlagsRequiredTogether("source", "target")
    
    return cmd
}
```

## Low Priority Issues (P3)

### 9. Configuration Management
**Problem**: Direct Viper access without abstraction
- `viper.GetString()` calls throughout code
- No validation at startup
- Mixed environment variable handling

**Solution**:
```go
type Config struct {
    APIKey string `validate:"required"`
    APIURL string `validate:"url"`
}

type ConfigService interface {
    Load() (*Config, error)
    Validate(cfg *Config) error
}
```

### 10. Error Handling Duplication
**Problem**: Repeated error formatting logic
- Same error display patterns in multiple views
- No consistent error presenter

**Solution**: Create error presenter utility

## Action Plan

### Phase 1: Eliminate Global State (Week 1)
1. [ ] Create dependency injection container
2. [ ] Convert cache from global to injected service
3. [ ] Refactor commands to functions accepting container
4. [ ] Update tests to use mock implementations

### Phase 2: Consolidate Duplicate Code (Week 2)
1. [ ] Create `utils.TruncateWithEllipsis()` function
2. [ ] Replace all truncate implementations
3. [ ] Implement functional options for view constructors
4. [ ] Consolidate error handling patterns

### Phase 3: Cache Architecture (Week 3)
1. [ ] Define cache interfaces
2. [ ] Separate cache responsibilities
3. [ ] Implement proper TTL and invalidation
4. [ ] Add cache metrics/debugging

### Phase 4: Clean Architecture (Week 4)
1. [ ] Create domain layer
2. [ ] Extract use cases from commands
3. [ ] Implement adapter pattern for external services
4. [ ] Add integration tests for layers

## Metrics for Success
- [ ] Zero global variables (except main)
- [ ] Single truncate implementation
- [ ] All views use functional options
- [ ] Cache operations through interfaces
- [ ] 80%+ test coverage
- [ ] Clean `golangci-lint` output

## Code Smell Locations Reference

### Global Variables
- `/internal/cache/cache.go:64` - `var globalCache`
- `/internal/cache/dashboard_cache.go:41` - `var dashboardCache`
- `/internal/commands/*.go` - All command vars

### Duplicate Truncate Functions
- `/internal/tui/components/statusline.go:276`
- `/internal/tui/views/bulk_fzf.go:50`
- `/internal/tui/views/dashboard.go:3407`
- `/internal/tui/components/help_view.go:545`
- `/internal/tui/components/table.go:250`
- `/internal/commands/status.go:183`

### Cache Issues
- `/internal/cache/cache.go:67` - `init()` function
- `/internal/cache/cache.go:29-62` - Mixed responsibilities in GlobalCache

### StatusLine Issues
- `/internal/tui/views/dashboard.go:559` - Duplicate spinner update
- All views have manual `statusLine.SetWidth()` calls

### Constructor Explosion
- `/internal/tui/views/list.go:49,56,79` - Three constructors
- `/internal/tui/views/details.go:68,90,149,190,214` - Five constructors
- `/internal/tui/views/create.go:103,118,300` - Three constructors

## Notes
- Priority based on: impact to code quality, ease of testing, and user experience
- Each refactoring should be done in a separate PR with tests
- Run `make lint-fix fmt` after each change
- Update documentation as architecture changes
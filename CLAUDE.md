# CLAUDE.md - RepoBird CLI Project Guidelines

## Project Overview
RepoBird CLI is a Go-based command-line tool for interacting with the RepoBird AI platform. It enables users to submit AI-powered code generation tasks, track their progress, and manage runs through both CLI commands and a rich Terminal User Interface.

## Documentation
Core documentation is in the `docs/` directory:
- **[Architecture Overview](docs/architecture.md)** - System design and components
- **[API Reference](docs/api-reference.md)** - REST endpoints and client
- **[TUI Guide](docs/tui-guide.md)** - Terminal UI implementation
- **[Keymap Architecture](docs/keymap-architecture.md)** - Key handling system
- **[Development Guide](docs/development-guide.md)** - Setup and contributing
- **[Testing Guide](docs/testing-guide.md)** - Testing strategies
- **[Configuration Guide](docs/configuration-guide.md)** - Settings and auth
- **[Troubleshooting Guide](docs/troubleshooting.md)** - Common issues
- **[Bulk Runs Guide](docs/bulk-runs.md)** - Batch operations
- **[Dashboard Layouts](docs/dashboard-layouts.md)** - Miller columns UI

## Quick Reference

### Core Technologies
- **Go 1.20+** with standard library preferred
- **Cobra** for CLI commands
- **Bubble Tea** for TUI with message-based navigation
- **Lipgloss** for terminal styling

### Project Structure
```
/cmd/repobird/      # Entry point
/internal/          # Private packages
  /api/             # API client
  /commands/        # CLI commands
  /tui/             # Terminal UI
    /views/         # TUI views
    /components/    # Shared components
    /cache/         # Caching layer
    /keymap/        # Key handling
  /config/          # Configuration
  /errors/          # Error handling
/pkg/               # Public packages
/docs/              # Documentation
```

## Development Guidelines

### Code Style
- Follow Go conventions and idioms
- Keep functions small (<50 lines)
- Explicit error handling
- No global state in runtime code
- Table-driven tests

### Critical Architecture Knowledge

#### Core Entry Points
- `cmd/repobird/main.go` - Application entry
- `internal/commands/root.go` - CLI command tree
- `internal/tui/app.go` - TUI router and navigation
- `internal/api/client.go` - API client implementation

#### Message-Based Navigation (TUI)
```go
// Never: view := NewDetailsView(...)
// Always: return v, func() tea.Msg { return NavigateToDetailsMsg{} }
```
App router (`internal/tui/app.go`) handles all view transitions.

#### View Constructor Pattern
```go
NewView(client, cache, id)  // Max 3 params, self-loading in Init()
```

#### Cache System
- `internal/tui/cache/simple_cache.go` - Main cache interface
- `internal/tui/cache/hybrid_cache.go` - Routing layer
- Terminal runs persist to disk, active runs in memory (5min TTL)
- Location: `~/.config/repobird/cache/users/{hash}/`

#### WindowLayout Critical Pattern
```go
// In view struct - start nil
layout *components.WindowLayout

// In Update() on WindowSizeMsg only
if v.layout == nil {
    v.layout = components.NewWindowLayout(width, height)
}
```
Never initialize in constructor - causes width issues.

#### Lipgloss Border Calculations (CRITICAL)
**Borders add 2 chars to width** - always subtract when calculating:
```go
// WRONG: Causes right-side cutoff
leftBox := lipgloss.NewStyle().Width(termWidth/2).Border(...)
rightBox := lipgloss.NewStyle().Width(termWidth/2).Border(...)

// CORRECT: Account for border expansion
totalWidth := termWidth - 4  // 2 boxes * 2 border chars
leftBox := lipgloss.NewStyle().Width(totalWidth/2).Border(...)
rightBox := lipgloss.NewStyle().Width(totalWidth/2).Border(...)
```

**Height calculations for full-screen views:**
```go
// Standard view with status bar
availableHeight := height - 1  // 1 for status

// With title and status
availableHeight := height - 3  // 2 for title, 1 for status

// Fix top border cutoff
availableHeight := height - 3  // Extra space
content := lipgloss.NewStyle().MarginTop(1).Render(boxes)
```

#### Key Processing Flow
`internal/tui/app.go::processKeyWithFiltering()` → Check disabled → Custom handler → Navigation → View Update

#### Dashboard Miller Columns Layout
Three-column hierarchical navigation (Repositories → Runs → Details):
```
│ Repositories │ Runs        │ Details     │
│ > myorg/app │ ✓ Fix bug   │ Run: 123    │
│   team/api  │ ⚡ Running   │ Status: ... │
```
- **Navigation**: Tab (forward), Shift+Tab/h (back), Enter (select)
- **Column widths**: 30%/35%/35% (standard), adjusts for terminal size
- **Status icons**: ✓ done, ⚡ running, ✗ failed, ○ pending
- **FZF search**: Press 'f' on any column for fuzzy filter

#### Dashboard File Split
Large views split for maintainability:
- `dashboard.go` - Core Bubble Tea methods
- `dash_navigation.go` - Key handling and column movement
- `dash_rendering.go` - Miller columns layout rendering
- `dash_data.go` - Data operations
- Pattern: `{view}_*.go` for large views

#### Error Handling
- `internal/errors/types.go` - Error types and classification
- Always use `errors.FormatUserError()` for CLI output
- `IsRetryable()`, `IsAuthError()` for error type checking

#### URL Management
- `internal/config/urls.go` - Centralized RepoBird URLs
- Adapts URLs based on `REPOBIRD_API_URL` environment
- Pricing URL shown for quota errors

#### Testing Patterns
- Mock API: `internal/api/mock_client.go`
- Cache isolation: `t.Setenv("XDG_CONFIG_HOME", tmpDir)`
- Navigation test: Check message types, not view instances

## Common Commands

### Development
```bash
make build          # Build binary
make test           # Run tests
make coverage       # Test coverage
make lint-fix       # Fix linting
make fmt            # Format code
make check          # Run all checks
```

### CLI Usage
```bash
repobird config set api-key YOUR_KEY
repobird run task.json --follow
repobird tui
repobird bulk config.json
```

## Testing Requirements
- Minimum 70% coverage for new code
- Use table-driven tests
- Mock external dependencies
- Use `XDG_CONFIG_HOME` for test isolation
See [Testing Guide](docs/testing-guide.md) for patterns.

## Configuration
- API key via environment (`REPOBIRD_API_KEY`) or config
- Debug logging: `REPOBIRD_DEBUG_LOG=1`
- Cache location: `~/.config/repobird/cache/`
See [Configuration Guide](docs/configuration-guide.md) for details.

## Known Limitations
- Maximum 45-minute timeout for operations
- Dashboard loads up to 1000 runs
- No offline mode currently
- GitHub/GitLab repositories only

## AI Assistant Instructions

When working on this codebase:

### Development Rules
1. **Maintain backward compatibility** for CLI commands
2. **Follow established patterns** - study existing code first
3. **Write tests** for new functionality (70%+ coverage)
4. **Use message-based navigation** in TUI - never create views directly
5. **Prefer editing existing files** over creating new ones
6. **Handle errors explicitly** with user-friendly messages
7. **Never log sensitive data** (API keys, tokens)

### Development Workflow
1. Read existing code before making changes
2. Follow Go idioms and project conventions
3. Test with `make check` before finalizing
4. Update relevant docs/ files for significant changes
5. Use `debug.LogToFilef()` for debugging TUI issues

### Testing Requirements for Todo Lists
When making changes to Go application code:
- **Always add `make test` as the final todo item** when modifying any `.go` files
- **Additionally add `make test-integration`** if CLI commands in `internal/commands/` are changed
- This ensures all code changes are validated before completion

### Critical Implementation Files
- **API Client**: `internal/api/client.go`, `internal/api/models.go`
- **TUI Router**: `internal/tui/app.go` (all navigation goes through here)
- **Cache**: `internal/tui/cache/simple_cache.go` (main interface)
- **Components**: `internal/tui/components/` (reusable UI elements)
- **Config**: `internal/config/manager.go` (settings management)
- **URLs**: `internal/config/urls.go` (dynamic URL management)
- **Retry Logic**: `internal/retry/retry.go` (exponential backoff)

### When Debugging
```bash
# Enable debug logging
REPOBIRD_DEBUG_LOG=1 repobird tui
tail -f /tmp/repobird_debug.log

# Check specific patterns
grep "CACHE\|NAV\|KEY" /tmp/repobird_debug.log
```

## Current Branch Context
Working on branch: `code-smells`
Main branch: `main`

## Final Checklist for Significant Changes
When completing major features:
1. ☐ Update relevant docs/ files if patterns changed
2. ☐ Run `make lint-fix fmt` to clean up code
3. ☐ Run `make test` to ensure tests pass
4. ☐ Update this file only if critical patterns changed
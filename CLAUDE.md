# CLAUDE.md - RepoBird CLI Project Guidelines

## Project Overview
RepoBird CLI is a Go-based command-line tool for interacting with the RepoBird AI platform. It enables users to submit AI-powered code generation tasks, track their progress, and manage runs through a simple CLI interface.

## Key Features
- Submit AI tasks from JSON files
- Track run status with real-time polling
- Manage API configuration securely
- Auto-detect repository and branch information
- Support for both run and approval workflows
- Rich Terminal User Interface (TUI) with Bubble Tea

## Documentation
Comprehensive documentation is available in the `docs/` directory:

### Core Architecture & Design
- **[Architecture Overview](docs/architecture.md)** - System design, components, patterns, and overall codebase structure
- **[API Reference](docs/api-reference.md)** - REST endpoints, client methods, error handling, and authentication patterns
- **[TUI Guide](docs/tui-guide.md)** - Terminal UI implementation, Bubble Tea patterns, and view architecture
- **[Keymap Architecture](docs/keymap-architecture.md)** - Centralized key processing system, per-view customization, and implementation guide

### Development & Operations  
- **[Development Guide](docs/development-guide.md)** - Setup, building, contributing, and coding standards
- **[Testing Guide](docs/testing-guide.md)** - Test strategies, patterns, coverage requirements, and best practices
- **[Configuration Guide](docs/configuration-guide.md)** - Settings management, environment variables, and security practices
- **[Troubleshooting Guide](docs/troubleshooting.md)** - Common issues, debugging techniques, and solutions

### Feature-Specific Guides
- **[Bulk Runs](docs/bulk-runs.md)** - Bulk operation workflows, configuration files, and batch processing
- **[Dashboard Layouts](docs/dashboard-layouts.md)** - Multi-column layouts, navigation patterns, and view states

## Architecture

### Directory Structure
```
/cmd/repobird/      - Main entry point for the CLI
/internal/          - Private application code
  /api/             - API client implementation
  /commands/        - Cobra command implementations
  /config/          - Configuration management
  /errors/          - Error handling and recovery
  /models/          - Data models and types
  /retry/           - Retry logic with exponential backoff
  /utils/           - Utility functions (git, polling, security)
/pkg/               - Public library code
  /utils/           - Utility functions (git helpers)
  /version/         - Version information
/build/             - Build output directory
/docs/              - Documentation
/tasks/             - Task tracking files
```

### Core Technologies
- **Go 1.20+** - Primary language
- **Cobra** - CLI framework
- **Viper** - Configuration management
- **Bubble Tea** - Terminal UI framework
- **Lipgloss** - Terminal styling
- **Standard library HTTP client** - API communication
- **Fuzzy** - Fuzzy string matching for search

## Development Guidelines

### Code Style
- Follow standard Go conventions and idioms
- Use `gofmt` for formatting (enforced by CI)
- Use `golangci-lint` for comprehensive linting
- Prefer explicit error handling
- Avoid global variables except for command flags
- Keep functions small and focused (<50 lines)
- Write clear, self-documenting code

### State Management & Caching (IMPORTANT)
- **TUI Views**: Embed cache instances in view structs (no globals)
- **Cache Pattern**: Use `*cache.SimpleCache` embedded in each view
- **Initialization**: Create cache with `cache.NewSimpleCache()` in view constructors
- **Testing**: Use temporary directories for cache in tests (set `XDG_CONFIG_HOME`)
- **No Global State**: Never use package-level variables for runtime state
- **Bubble Tea Pattern**: All state flows through Model.Update() method

#### Cache Architecture
- **Hybrid Cache**: Automatic routing between disk (permanent) and memory (session) storage
- **Terminal Runs**: Automatically persisted to disk (DONE/FAILED/CANCELLED states)
- **Stuck Runs**: Runs older than 2 hours are permanently cached (likely stuck in invalid state)
- **Active Runs**: Kept in memory with 5-minute TTL (RUNNING/PENDING states less than 2 hours old)
- **User Isolation**: Each user has separate cache directory (`~/.config/repobird/cache/users/{hash}/`)
- **No Manual Save/Load**: Persistence is automatic - `SaveToDisk()`/`LoadFromDisk()` are now no-ops
- **Performance**: Terminal/old runs load in <10ms from disk, 90% reduction in API calls
- **Concurrency Safety**: Fixed deadlock issues through proper lock ordering and single-decision routing
- **Lock-Free File I/O**: PermanentCache uses atomic file operations without holding locks
- **Batch Updates**: Dashboard view uses batch cache updates to prevent lock contention

### Testing Requirements
- Minimum 70% test coverage for new code
- Write unit tests for all public functions
- Use table-driven tests where appropriate
- Mock external dependencies (API calls, file I/O)
- Run tests before committing: `make test`

### Git Workflow
- Branch naming: `feature/description`, `fix/description`, `chore/description`
- Commit messages: `<type>: <short summary>` (e.g., `feat: add status polling`)
- Always create PRs for review
- Squash commits when merging
- Keep main branch stable and deployable

## Terminal UI (TUI) Implementation

### FZF Integration
The TUI includes built-in fuzzy search functionality:

#### Dashboard View File Organization
The dashboard view is split across multiple files for maintainability:

- **`dashboard.go`** (838 lines) - Core Update/View/Init methods only
- **`dash_navigation.go`** (454 lines) - Navigation logic and keymap handling
- **`dash_updates.go`** (572 lines) - Viewport and content update methods
- **`dash_state.go`** (263 lines) - State management and validation helpers
- **`dash_status_info.go`** (548 lines) - Status overlay + help/docs rendering
- **`dash_rendering.go`** (584 lines) - All layout rendering (columns, layouts, status lines)
- **`dash_data.go`** (461 lines) - Data loading, repository operations, and cache management
- **`dash_formatting.go`** (222 lines) - Text formatting, truncation, and icon utilities
- **`dash_fzf.go`** (124 lines) - FZF overlay positioning and activation logic
- **`dash_messages.go`** (42 lines) - Custom message types for dashboard operations  
- **`dash_clipboard.go`** (38 lines) - Clipboard operations and yank animations

**Total**: ~4,146 lines across 11 focused files (main file reduced 56% from 1,906 to 838 lines)

#### Dashboard FZF Integration
- Press `f` on any column to activate FZF mode
- Shows dropdown overlay at cursor position
- Filters repositories, runs, or details based on input
- Enter selects and moves to next column
- ESC cancels FZF mode

#### Create Run View (`internal/tui/views/create.go`)
- `Ctrl+F` in insert mode activates FZF for repository field
- `f` in normal mode (after ESC) activates FZF for repository field
- Shows repository history and current git repo
- Smart icon indicators (📁 current, 🔄 history, ✏️ edited)

#### FZF Component (`internal/tui/components/fzf_selector.go`)
- Reusable fuzzy search component
- Compact dropdown style with rounded borders
- Real-time filtering using sahilm/fuzzy library
- Keyboard navigation (arrows, Ctrl+j/k)
- Scroll indicators for long lists

### Key Bindings
- `f` - Activate FZF mode (dashboard)
- `Ctrl+F` - Activate FZF mode (create view, insert mode)
- `↑↓` or `j/k` - Navigate items
- `Enter` - Select item
- `ESC` - Cancel FZF mode
- `Tab` - Next field/column
- `n` - New run
- `s` - Status info
- `r` - Refresh
- `q` - Quit

## Common Commands

### Development
```bash
# Build the binary
make build

# Run tests
make test

# Run with coverage
make coverage

# Lint code
make lint

# Fix linting issues automatically
make lint-fix

# Format code
make fmt

# Run all checks
make check

# Clean build artifacts
make clean
```

### CLI Usage
```bash
# Configure API key
repobird config set api-key YOUR_KEY

# Submit a task
repobird run task.json
repobird run task.json --follow

# Check status
repobird status
repobird status RUN_ID
repobird status --follow RUN_ID

# View version
repobird version
```

### Example Task JSON
```json
{
  "prompt": "Fix the authentication bug in the login flow",
  "repository": "org/repo",
  "source": "main",
  "target": "fix/auth-bug",
  "runType": "run",
  "title": "Fix authentication bug",
  "context": "Users are unable to login with valid credentials",
  "files": ["src/auth.js", "src/login.js"]
}
```

## API Integration

### Endpoints
- `POST /api/v1/runs` - Create new run
- `GET /api/v1/runs/{id}` - Get run status
- `GET /api/v1/runs` - List all runs
- `GET /api/v1/auth/verify` - Verify API key

### Authentication
- Bearer token authentication using API key
- Store keys securely (never in plain text)
- Support environment variable: `REPOBIRD_API_KEY`

### Error Handling
- **Custom Error Types**: Use `internal/errors` package for structured errors
- **User-Friendly Messages**: Call `errors.FormatUserError(err)` for CLI output
- **Error Classification**: Use `errors.IsRetryable()`, `errors.IsAuthError()`, etc.
- **Retry Logic**: Import `internal/retry` for exponential backoff with circuit breaker
- **API Errors**: Use `client.CreateRunWithRetry()` and `client.GetRunWithRetry()` methods
- **Polling**: Use `utils.NewPoller()` for status updates with graceful interruption

## Configuration

### Environment Variables
- `REPOBIRD_API_KEY` - API authentication key
- `REPOBIRD_API_URL` - Override API endpoint (for development)
- `REPOBIRD_ENV` - Environment setting (prod/dev) - affects frontend URL generation

### Config File
- Location: `~/.repobird/config.yaml`
- Never commit config files with sensitive data
- Use Viper for config management

## Dependencies
- Go 1.20 or higher
- Git (for repository detection)
- Internet connection for API calls
- github.com/charmbracelet/bubbletea (TUI framework)
- github.com/charmbracelet/lipgloss (styling)
- github.com/sahilm/fuzzy (fuzzy matching)
- github.com/ktr0731/go-fuzzyfinder (repository selector fallback)

## Known Issues & Limitations
- Maximum timeout for runs is 45 minutes
- No offline mode currently
- Limited to GitHub repositories for auto-detection
- Dashboard and list views load up to 1000 runs (increased from 100 for better UX)

## CI/CD Requirements
- All tests must pass
- Code must be formatted (`make fmt-check`)
- Linting must pass (`make lint`)
- Coverage should not decrease
- Binary must build successfully

## Release Process
1. Update version in code
2. Run full test suite: `make ci`
3. Create git tag: `git tag vX.Y.Z`
4. Build release binaries: `make build-all`
5. Create GitHub release with binaries

## Performance Considerations
- Keep API calls minimal
- Use pagination for list operations
- Cache user info when possible
- Implement request timeout (45 min default)
- Use context for cancellation

## Security Guidelines
- Never log API keys or sensitive data
- Validate all user input
- Use HTTPS for all API communication
- Store credentials securely using OS keychain when possible
- Regular dependency updates for security patches

## Debugging
- Use `--debug` flag for verbose output
- Check `~/.repobird/` for config issues
- API requests are logged when debug is enabled
- Use `make run` for local development

## Contributing Guidelines
- Read existing code before making changes
- Follow the established patterns
- Write tests for new features
- Update documentation as needed
- Run `make check` before submitting PR

## AI Assistant Instructions

When working on this codebase:
1. Always maintain backward compatibility for CLI commands
2. Prioritize user experience - clear error messages, helpful defaults
3. Follow Go idioms and best practices
4. Add tests for any new functionality
5. Update CLI help text when adding new features
6. Keep dependencies minimal - prefer standard library
7. Ensure cross-platform compatibility (Linux, macOS, Windows)
8. For significant changes, always create these final todo items (in order):
   - `☐ Update detailed documentation in docs/ directory (if applicable)`
   - `☐ Update CLAUDE.md with general/critical info needed for future development`
   - `☐ Run linting and formatting: make lint-fix fmt`
9. Document any non-obvious design decisions in code comments and docs/ markdown files.
10. When debugging TUI issues, use `debug.LogToFilef()` to write to `/tmp/repobird_debug.log` and check logs with `tail -f /tmp/repobird_debug.log`

### TUI Implementation Patterns
- **Bubble Tea Model-Update-View**: All TUI views implement `Update(tea.Msg) (tea.Model, tea.Cmd)` pattern
- **Message-Based Navigation**: Views use navigation messages to transition between views via central App router
- **App Router**: Central navigation controller in `internal/tui/app.go` handles all view transitions and maintains view history stack
- **Navigation Messages**: Type-safe navigation messages in `internal/tui/messages/navigation.go` (NavigateToCreateMsg, NavigateBackMsg, etc.)
- **Minimal Constructors**: Views created with max 3 params: `NewView(client, cache, id)` pattern
- **Shared Cache**: Single cache instance from app-level passed to all views
- **Self-Loading Views**: Views load their own data in `Init()` method, no parent state coupling
- **Shared Components**: Reusable UI components in `internal/tui/components/` (ScrollableList, Form, ErrorView)
- **Navigation Context**: Temporary state sharing via `cache.SetNavigationContext()` without tight coupling
- **View History Stack**: Back navigation support with `NavigateBackMsg`, dashboard reset with `NavigateToDashboardMsg`
- **🆕 Core Keymap System**: Centralized key processing with per-view customization (see Key Management section)
- **🆕 View File Organization**: Large views split into focused files (e.g., dashboard split into 8 files by functionality)
- **🆕 Global Layout System**: Centralized sizing via `WindowLayout` component for consistent borders across all views
- **Debug Logging**: Use `debug.LogToFilef()` from `internal/tui/debug` package (configurable via `REPOBIRD_DEBUG_LOG`)

#### Navigation Architecture
- **Anti-Pattern**: Never create child views directly in Update methods - use navigation messages instead
- **Message Flow**: View → Navigation Message → App Router → NewView(client, cache, id) → Push to Stack
- **Constructor Pattern**: All views use minimal `NewView(client, cache, id)` pattern, max 3 parameters
- **Self-Loading**: Views load their own data in `Init()`, no parent state passing
- **Shared State**: Single cache instance shared across all views from app-level
- **Context Management**: Use navigation context for temporary data, cleared on dashboard return
- **Error Recovery**: Recoverable errors allow back navigation, non-recoverable errors clear history stack

### BulkView Architecture (Reference Implementation)

The BulkView serves as a reference implementation demonstrating proper navigation patterns and WindowLayout usage:

#### Split File Organization
- **`bulk.go`** (main controller) - Core Update/View/Init methods and mode management
- **`bulk_commands.go`** - File loading, submission, and progress tracking commands
- **`bulk_messages.go`** - Custom message types for bulk operations
- **`bulk_test.go`** - Comprehensive test coverage for all modes and navigation

#### Navigation Pattern Compliance
```go
// ✅ Correct: Message-based navigation instead of direct view creation
func (v *BulkView) handleFileSelectKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    case key.Matches(msg, v.keys.Quit):
        return v, func() tea.Msg {
            return messages.NavigateBackMsg{}  // Navigation message
        }
}

// ✅ Correct: WindowSizeMsg handling for proper initialization
func (v *BulkView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    case tea.WindowSizeMsg:
        v.width = msg.Width
        v.height = msg.Height
        v.layout = components.NewWindowLayout(v.width, v.height) // Global layout
        // Update components...
}
```

#### WindowLayout System Usage
- Uses `components.NewWindowLayout()` for consistent borders and sizing
- Status line integration with mode-specific help text
- Proper viewport calculations for content areas
- Responsive design that handles terminal resizing

#### Multi-Mode State Management
- **File Selection Mode**: Browse and select configuration files
- **Run List Mode**: Review and toggle individual runs
- **Progress Mode**: Real-time progress tracking
- **Results Mode**: Summary with success/failure details

This architecture serves as the template for implementing other complex views that require multiple modes and consistent navigation behavior.

### Key Management (Core Keymap System)

#### **Architecture Overview**
The TUI uses a centralized key processing system that provides consistent, extensible key handling across all views:

```
Key Press → App.processKeyWithFiltering() → View Keymap Check → Action Execution
```

#### **Core Components**

**1. CoreKeyRegistry (`internal/tui/keymap/core.go`)**
- Central registry of all keys and their default actions
- Maps keystrings to actions: `ActionNavigateBack`, `ActionGlobalQuit`, `ActionViewSpecific`, etc.
- Extensible: new keys and actions can be registered

**2. CoreViewKeymap Interface**
```go
type CoreViewKeymap interface {
    IsKeyDisabled(keyString string) bool
    HandleKey(keyMsg tea.KeyMsg) (handled bool, model tea.Model, cmd tea.Cmd)
}
```

**3. Centralized Processing (`App.processKeyWithFiltering()`)**
- Single point where ALL keys are processed
- Checks view keymaps before executing actions
- Handles global actions (force quit) regardless of view state
- Routes navigation actions through proper channels

#### **Key Action Types**
- **Navigation Actions**: `b` (back), `B` (bulk), `n` (new), `r` (refresh), `q` (quit), `?` (help)
- **Global Actions**: `Q` (force quit), `ctrl+c` (force quit) - always work regardless of view
- **View-Specific Actions**: `s` (status), `f` (filter), `enter`, `tab`, arrow keys - handled by views

#### **Per-View Key Customization**

**Disable Keys Example (Dashboard)**:
```go
type DashboardView struct {
    disabledKeys map[string]bool
}

func NewDashboardView(client APIClient) *DashboardView {
    return &DashboardView{
        disabledKeys: map[string]bool{
            "b": true,    // Disable back navigation
            "esc": true,  // Disable escape key
        },
    }
}

func (d *DashboardView) IsKeyDisabled(keyString string) bool {
    return d.disabledKeys[keyString]
}
```

**Custom Key Handling Example**:
```go
func (v *CreateView) HandleKey(keyMsg tea.KeyMsg) (handled bool, model tea.Model, cmd tea.Cmd) {
    if keyMsg.String() == "ctrl+s" && v.isFormValid() {
        // Custom save behavior
        return true, v, v.submitForm()
    }
    return false, v, nil // Let system handle other keys
}
```

#### **Implementation Guidelines**

**For New Views**:
1. Implement `CoreViewKeymap` interface if you need key customization
2. Use `IsKeyDisabled()` to disable unwanted keys
3. Use `HandleKey()` for custom key behaviors
4. Views without interface work normally (all keys enabled)

**Key Processing Priority**:
1. **Disabled Check**: If `IsKeyDisabled(key)` returns true → ignore completely
2. **Custom Handler**: If `HandleKey()` returns handled=true → use custom result  
3. **Global Actions**: Force quit, etc. → handled by app regardless of view
4. **Navigation Actions**: Back, bulk, etc. → converted to navigation messages
5. **View-Specific**: All other keys → delegated to view's Update() method

**Benefits**:
- ✅ **Consistent**: Same key behavior across all views
- ✅ **Extensible**: Any view can disable/customize any key
- ✅ **Maintainable**: Single place to understand key processing
- ✅ **Debuggable**: Easy to trace what happens with any key press
- ✅ **Backward Compatible**: Existing views work unchanged

**Note**: View architecture refactored (2025-08-12) to follow clean Bubble Tea patterns:
- **Dashboard**: Uses shared `ScrollableList` component instead of child views
- **CreateRunView**: Refactored to use `NewCreateRunView(client)` exclusively with navigation context
- **RunListView**: Simplified to single constructor, removed cache metadata fields, uses cache methods directly
- All views use minimal constructors: `NewView(client, cache, id)` pattern
- Navigation via messages only - no direct view creation
- Views are self-loading in `Init()` - no parent state coupling
- Cache encapsulation: Views use `cache.GetRuns()`, `cache.SetRuns()` instead of managing cache state

### Global Layout System (WindowLayout)

**CRITICAL**: All views except Dashboard MUST use the global WindowLayout system for consistent sizing and borders.

#### Purpose
Eliminates the architectural nightmare of each view manually calculating its own dimensions and borders. Previously, views had inconsistent border cutoffs and duplicate sizing logic that broke when lipgloss rendering behavior changed.

#### Usage Pattern
```go
// In view struct
type MyView struct {
    layout *components.WindowLayout
    // ... other fields
}

// In constructor
func NewMyView(client APIClient, cache *cache.SimpleCache) *MyView {
    return &MyView{
        layout: components.NewWindowLayout(80, 24), // default dimensions
        // ... other initialization
    }
}

// In handleWindowSizeMsg
func (v *MyView) handleWindowSizeMsg(msg tea.WindowSizeMsg) {
    v.width = msg.Width
    v.height = msg.Height
    v.layout.Update(msg.Width, msg.Height) // Update layout calculations
    
    // Get viewport dimensions from layout
    viewportWidth, viewportHeight := v.layout.GetViewportDimensions()
    v.viewport.Width = viewportWidth
    v.viewport.Height = viewportHeight
}

// In View() method
func (v *MyView) View() string {
    if !v.layout.IsValidDimensions() {
        return v.layout.GetMinimalView("My View - Loading...")
    }
    
    // Use layout for all sizing
    boxStyle := v.layout.CreateStandardBox()
    titleStyle := v.layout.CreateTitleStyle()
    contentStyle := v.layout.CreateContentStyle()
    
    // ... render with consistent styling
}
```

#### Key Methods
- `NewWindowLayout(width, height)` - Create layout calculator
- `Update(width, height)` - Recalculate on resize
- `GetBoxDimensions()` - Box width/height for lipgloss containers
- `GetViewportDimensions()` - Content area for bubble tea viewports
- `CreateStandardBox()` - Consistent rounded border box
- `CreateTitleStyle()` - Standard title formatting
- `CreateContentStyle()` - Content area styling
- `IsValidDimensions()` - Check if terminal is large enough

#### Views That MUST Use WindowLayout
- ✅ **Details View** (single-box content display)
- ✅ **Status View** (single-box info display)  
- ✅ **Create Run View** (form-based layout)
- ✅ **Error View** (error message display)
- ✅ **List View** (single-column list display)
- ✅ **Bulk View** (multi-mode view with consistent borders and status line)
- ❌ **Dashboard** (uses custom 3-column Miller Columns layout)

#### Benefits
- **Consistent Borders**: All views have perfect borders without cutoffs
- **Single Source of Truth**: Change layout logic once, applies everywhere
- **Lipgloss Compatibility**: Automatically accounts for border expansion (2px wider than set width)
- **Responsive Design**: Handles terminal resizing gracefully
- **Easy Maintenance**: No more hunting through views to fix border issues
- **Future Proof**: New views just adopt the pattern

#### Anti-Patterns to Avoid
- ❌ Manual border calculations in view code
- ❌ Hardcoded margins and padding
- ❌ Direct lipgloss sizing without using layout
- ❌ Copy-pasting sizing logic between views
- ❌ Ignoring the layout system for "simple" views

### Testing Patterns  
- **Table-Driven Tests**: Use struct slices with test cases for systematic testing
- **Test Isolation**: Set `XDG_CONFIG_HOME` to temp directory in tests for cache isolation
- **Mocking**: Use `github.com/stretchr/testify` for assertions and mocks
- **Coverage Target**: Maintain 70%+ test coverage for new code
- **Navigation Testing**: Test navigation messages, App router transitions, and view history stack
- **Component Testing**: Test shared components (ScrollableList, Form) with proper Update/View cycles
- **Integration Testing**: Test complete navigation flows (Dashboard → Create → Details → Back)
- **Context Testing**: Test navigation context sharing and cleanup between views

### Build & Development
- **make ci vs make check**: `ci` includes security checks and coverage, `check` is faster (fmt-check, vet, lint, test)
- **Cross-compilation**: `make build-all` targets: darwin/amd64, darwin/arm64, linux/amd64, linux/arm64, windows/amd64  
- **Environment Modes**: `REPOBIRD_ENV=dev` (development) vs `REPOBIRD_ENV=prod` (production) affects frontend URL generation

### Code Organization
- **Interfaces**: Define in `internal/domain/interfaces.go` and `internal/tui/interfaces.go`
- **internal/ vs pkg/**: `internal/` for private code, `pkg/` only for truly reusable public APIs
- **Error Handling**: Use structured `internal/errors` package with `ErrorType` classification and user-friendly formatting
- **Context Usage**: All API calls and cancellable operations use context.Context for timeouts and cancellation

### Performance Guidelines
- **HTTP Client**: Pre-configured with 45-minute timeout, circuit breaker, and retry logic with exponential backoff
- **No Explicit Concurrency**: Standard library HTTP client handles connection pooling, avoid manual goroutines unless needed
- **Memory Management**: TUI views handle large datasets (1000+ runs) with efficient caching and pagination

## Quick Troubleshooting

### API Key Issues
```bash
# Verify API key is set
repobird config get api-key

# Test authentication
REPOBIRD_API_KEY=your_key repobird status
```

### Build Issues
```bash
# Clean and rebuild
make clean
go mod tidy
make build
```

### Test Failures
```bash
# Run tests with verbose output
go test -v ./...

# Check specific package
go test -v ./internal/api
```

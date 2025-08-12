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

- **[Architecture Overview](docs/architecture.md)** - System design, components, patterns
- **[API Reference](docs/api-reference.md)** - Endpoints, client methods, error handling
- **[Development Guide](docs/development-guide.md)** - Setup, building, contributing
- **[Testing Guide](docs/testing-guide.md)** - Test strategies, patterns, coverage
- **[Configuration Guide](docs/configuration-guide.md)** - Settings, environment, security
- **[Troubleshooting Guide](docs/troubleshooting.md)** - Common issues and solutions

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

#### Dashboard View (`internal/tui/views/dashboard.go`)
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
8. create final todo list item of run linting and formatting after all other todo changes: `make lint-fix fmt`
9. Document any non-obvious design decisions in code comments in docs/ markdown files.
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
- **Debug Logging**: Use `debug.LogToFilef()` from `internal/tui/debug` package (configurable via `REPOBIRD_DEBUG_LOG`)

#### Navigation Architecture
- **Anti-Pattern**: Never create child views directly in Update methods - use navigation messages instead
- **Message Flow**: View → Navigation Message → App Router → NewView(client, cache, id) → Push to Stack
- **Constructor Pattern**: All views use minimal `NewView(client, cache, id)` pattern, max 3 parameters
- **Self-Loading**: Views load their own data in `Init()`, no parent state passing
- **Shared State**: Single cache instance shared across all views from app-level
- **Context Management**: Use navigation context for temporary data, cleared on dashboard return
- **Error Recovery**: Recoverable errors allow back navigation, non-recoverable errors clear history stack

**Note**: View architecture refactored (2025-08-12) to follow clean Bubble Tea patterns:
- **Dashboard**: Uses shared `ScrollableList` component instead of child views
- **CreateRunView**: Refactored to use `NewCreateRunView(client)` exclusively with navigation context
- **RunListView**: Simplified to single constructor, removed cache metadata fields, uses cache methods directly
- All views use minimal constructors: `NewView(client, cache, id)` pattern
- Navigation via messages only - no direct view creation
- Views are self-loading in `Init()` - no parent state coupling
- Cache encapsulation: Views use `cache.GetRuns()`, `cache.SetRuns()` instead of managing cache state

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

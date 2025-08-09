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
- **Standard library HTTP client** - API communication

## Development Guidelines

### Code Style
- Follow standard Go conventions and idioms
- Use `gofmt` for formatting (enforced by CI)
- Use `golangci-lint` for comprehensive linting
- Prefer explicit error handling
- Avoid global variables except for command flags
- Keep functions small and focused (<50 lines)
- Write clear, self-documenting code

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

### Config File
- Location: `~/.repobird/config.yaml`
- Never commit config files with sensitive data
- Use Viper for config management

## Dependencies
- Go 1.20 or higher
- Git (for repository detection)
- Internet connection for API calls

## Known Issues & Limitations
- Maximum timeout for runs is 45 minutes
- No offline mode currently
- Limited to GitHub repositories for auto-detection
- No support for batch operations yet

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

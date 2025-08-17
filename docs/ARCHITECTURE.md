# RepoBird CLI - Architecture Overview

## System Architecture

RepoBird CLI is a Go-based terminal application for interacting with the RepoBird AI platform, featuring both CLI commands and a rich TUI interface.

```
┌─────────────────────────────────────────────────────────────┐
│                     CLI Application                         │
├─────────────────────────────────────────────────────────────┤
│ Commands Layer (Cobra)  │  TUI Layer (Bubble Tea)         │
├─────────────────────────┼─────────────────────────────────────┤
│       Services Layer (Domain Logic)                        │
├─────────────────────────────────────────────────────────────┤
│ Repository Layer │ Cache Layer │ Config Layer │ Utils      │
├─────────────────────────────────────────────────────────────┤
│              External APIs & File System                   │
└─────────────────────────────────────────────────────────────┘
```

## Related Documentation
- **[TUI Guide](TUI-GUIDE.md)** - Terminal UI implementation details
- **[API Reference](API-REFERENCE.md)** - REST API client implementation  
- **[Configuration Guide](CONFIGURATION-GUIDE.md)** - Settings and authentication
- **[Testing Guide](TESTING-GUIDE.md)** - Testing patterns and strategies
- **[Development Guide](DEVELOPMENT-GUIDE.md)** - Setup and contributing

## Core Components

### 1. Command Layer (`/internal/commands/`)
Cobra-based CLI commands for user interaction:
- **run**: Submit AI tasks from JSON files
- **status**: Monitor run progress with polling
- **config**: Manage API keys and settings  
- **auth**: Authentication workflows
- **tui**: Launch interactive terminal interface
- **bulk**: Batch run submission

### 2. TUI Layer (`/internal/tui/`)
Bubble Tea-based terminal UI with message-driven navigation:

**Architecture:**
- **App Router** (`app.go`): Central navigation with view history stack
- **Navigation Messages**: Type-safe view transitions via messages  
- **Views**: Dashboard, Create, Details, List, Bulk, Error
- **Components**: ScrollableList, Form, WindowLayout, FZF selector
- **Pattern**: `NewView(client, cache, id)` minimal constructors

**Key Features:**
- Message-based navigation (no direct view creation)
- Shared cache instance across all views
- Self-loading views via `Init()` method
- Vim-style keybindings with centralized keymap
- Global WindowLayout for consistent sizing
- FZF fuzzy search integration

See **[TUI Guide](TUI-GUIDE.md)** and **[Keymap Architecture](KEYMAP-ARCHITECTURE.md)** for implementation details.

### 3. API Client (`/internal/api/`)
HTTP client with resilience patterns:
- Bearer token authentication
- Exponential backoff retry logic
- Circuit breaker for failure prevention
- Request/response logging (debug mode)
- Structured error handling

See **[API Reference](API-REFERENCE.md)** for endpoints and methods.

### 4. Domain Layer (`/internal/domain/`)
Business logic and models:
- Core entities (Run, Task, User)
- Service interfaces
- Validation rules

### 5. Cache System (`/internal/tui/cache/`)
Hybrid cache with automatic persistence:

**Architecture:**
- **PermanentCache** (Disk): Terminal runs, user info, stuck runs (>2h old)
- **SessionCache** (Memory): Active runs, dashboard data (5min TTL)
- **HybridCache**: Intelligent routing between layers

**Key Features:**
- Automatic persistence of completed runs
- User-isolated storage (`~/.config/repobird/cache/users/{hash}/`)
- 90% reduction in API calls
- <10ms disk load time
- Test isolation via `XDG_CONFIG_HOME`

### 6. Configuration Management (`/internal/config/`)
Multi-backend secure configuration:
1. Environment variables (`REPOBIRD_API_KEY`)
2. System keyring (secure desktop storage)
3. Encrypted file fallback (`~/.repobird/config.yaml`)

See **[Configuration Guide](CONFIGURATION-GUIDE.md)** for details.

### 7. Error Handling (`/internal/errors/`)
Structured errors with user-friendly messages:
- Typed errors (API, Network, Auth, Quota, Validation)
- Retryable error detection
- User-friendly formatting via `FormatUserError()`

## Data Flow

**Run Creation:** JSON input → Validation → Git detection → API call → Cache update → Display

**Status Polling:** Cache check → API request (if miss) → Cache update → Display → Poll loop

## Security

**API Key Storage:**
- Environment variables (plain text, isolated)
- System keyring (native secure storage)
- Encrypted file (AES-256-GCM)

**Encryption:** Machine-specific key derivation, random nonces, authenticated encryption

## Concurrency & Performance

**Thread Safety:**
- Lock ordering: SimpleCache → HybridCache → Session/Permanent
- TUI uses message passing (actor model)
- Lock-free file I/O in PermanentCache
- Batch cache updates to reduce contention

**Performance:**
- Connection pooling for HTTP
- Adaptive polling intervals
- Lazy loading of large datasets
- 90% API call reduction via caching

## Extension Points

**New Commands:** Create in `/internal/commands/`, register in root, add tests

**New API Endpoints:** Define in `/internal/api/`, add retry logic, update models

**New TUI Views:** Create in `/internal/tui/views/`, register in app router, follow WindowLayout pattern

## Design Patterns

- **Repository Pattern**: Data access abstraction via interfaces
- **Factory Pattern**: Configured instance creation
- **Strategy Pattern**: Switchable storage backends
- **Observer Pattern**: Message-based TUI updates
- **Command Pattern**: CLI command encapsulation

## Testing

**Test Distribution:** 70% unit, 25% integration, 5% E2E

**Strategies:**
- Mock external dependencies
- Table-driven tests
- Test isolation via `XDG_CONFIG_HOME`
- Coverage target: 70%+

See **[Testing Guide](TESTING-GUIDE.md)** for patterns and best practices.

## Deployment & Operations

**Binary Distribution:** Cross-platform builds for darwin/linux/windows (amd64/arm64)

**Debug Mode:** `REPOBIRD_DEBUG_LOG=1` or `--debug` flag for verbose logging

**Monitoring:** Logs to `/tmp/repobird_debug.log` when debug enabled

See **[Troubleshooting Guide](TROUBLESHOOTING.md)** for debugging techniques.

## Key Architectural Patterns

**Message-Based Navigation:** Views emit navigation messages, app router handles transitions

**Shared State:** Single cache instance passed to all views

**Self-Loading Views:** Views fetch their own data in `Init()`

**Component Reuse:** ScrollableList, Form, WindowLayout shared across views

**Context Management:** Navigation context for temporary state sharing

## Technology Stack

**Core:** Go 1.20+, Cobra (CLI), Bubble Tea (TUI), Lipgloss (styling)

**Development:** Make, golangci-lint, testify

**Best Practices:**
- Clean architecture with layered design
- SOLID principles
- Message-based loose coupling
- Security-first approach
- Go idioms and conventions
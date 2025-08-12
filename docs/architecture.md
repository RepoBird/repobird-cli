# RepoBird CLI - Architecture Overview

## System Architecture

RepoBird CLI follows a clean, layered architecture designed for maintainability, testability, and extensibility.

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

## Core Components

### 1. Command Layer (`/internal/commands/`)
The command layer handles user interaction through CLI commands using the Cobra framework.

- **Root Command**: Global configuration, error handling, command registration
- **Run Command**: Creates new AI runs from JSON specifications
- **Status Command**: Monitors run progress with polling support
- **Config Command**: Manages API keys and settings
- **Auth Command**: Handles authentication workflows
- **TUI Command**: Launches the interactive terminal interface

### 2. TUI Layer (`/internal/tui/`)
The Terminal User Interface provides rich, interactive experiences using Bubble Tea.

```
┌──────────────────────────────────┐
│         TUI Application          │
├──────────────────────────────────┤
│  Views  │ Components │  Styles   │
├──────────────────────────────────┤
│    Forms   │   Debug Utilities   │
└──────────────────────────────────┘
```

Key features:
- Multi-view navigation (List → Details → Create)
- Real-time status updates
- Vim-style keybindings
- Clipboard integration
- Persistent state management

### 3. API Client (`/internal/api/`)
Robust HTTP client implementation with enterprise-grade features.

**Core Capabilities:**
- Bearer token authentication
- Exponential backoff retry logic
- Circuit breaker pattern
- Request/response logging
- Error classification and handling

**Resilience Patterns:**
```go
// Retry with exponential backoff
client.CreateRunWithRetry(ctx, request)

// Circuit breaker prevents cascade failures
if circuitBreaker.IsOpen() {
    return ErrServiceUnavailable
}
```

### 4. Domain Layer (`/internal/domain/`)
Contains business logic and domain models, isolated from external concerns.

- **Models**: Core entities (Run, Task, User)
- **Interfaces**: Service and repository contracts
- **Business Rules**: Validation and state transitions

### 5. Cache System (`/internal/tui/cache/`)
**Hybrid layered cache architecture** with automatic persistence and intelligent data routing.

```
┌─────────────────────────────────────────────────────────────┐
│                   HybridCache (Facade)                      │
├─────────────────────────────────────────────────────────────┤
│  PermanentCache (Disk)      │  SessionCache (Memory)       │
│  ~/.config/repobird/cache/  │  TTL-based in-memory         │
│  users/{user-hash}/         │                              │
│  ├── Terminal Runs (∞)      │  ├── Active Runs (5min)     │
│  ├── User Info (∞)          │  ├── Dashboard (5min)       │
│  ├── File Hashes (∞)        │  └── Form Data (30min)      │
│  └── Repositories (∞)       │                              │
└─────────────────────────────────────────────────────────────┘
```

**Cache Layers:**

1. **PermanentCache** - Disk storage for stable data:
   - Automatically persists DONE/FAILED/CANCELLED runs
   - Automatically persists any run older than 2 hours (stuck runs)
   - Never expires, survives restarts
   - User-isolated directories with hashed IDs
   - Instant loading (<10ms)

2. **SessionCache** - Memory storage for active data:
   - Caches RUNNING/PENDING runs less than 2 hours old
   - Dashboard and form data with configurable TTLs
   - Automatically removes terminal or old runs
   - Fast access for frequently changing data

3. **HybridCache** - Intelligent routing facade:
   - Routes data to appropriate storage based on state
   - Merges results from both layers
   - Transparent fallback if disk cache fails
   - Maintains backward compatibility

**Implementation Pattern:**

```go
type DashboardView struct {
    cache *cache.SimpleCache  // Now wraps HybridCache internally
}

func NewDashboardView(client APIClient) *DashboardView {
    cache := cache.NewSimpleCache()  // Automatic user detection
    // No manual LoadFromDisk() needed - automatic
}
```

**Storage Strategy:**

| Data Type | Storage | TTL | Rationale |
|-----------|---------|-----|-----------|
| Terminal Runs | Disk | Never | Immutable, frequently accessed |
| Stuck Runs (>2h) | Disk | Never | Likely stuck in invalid state, won't change |
| Active Runs (<2h) | Memory | 5 min | Changes frequently, needs updates |
| User Info | Disk | Never | Stable across sessions |
| File Hashes | Disk | Never | Deduplication across sessions |
| Dashboard | Memory | 5 min | Aggregated view, can rebuild |
| Form Data | Memory | 30 min | Temporary UI state |

**Directory Structure:**
```
~/.config/repobird/cache/
└── users/
    ├── user-a1b2c3d4/         # Hashed user ID
    │   ├── runs/
    │   │   ├── run-123.json    # Terminal run (DONE)
    │   │   └── run-456.json    # Terminal run (FAILED)
    │   ├── user-info.json      # User profile
    │   ├── file-hashes.json    # Dedup hashes
    │   └── repositories/
    │       └── list.json       # Repo list
    └── anonymous/              # Unauthenticated users
        └── runs/
```

**Performance Benefits:**
- **90% reduction** in API calls for completed runs
- **<10ms load time** for terminal runs from disk
- **Offline support** for viewing completed work
- **User isolation** prevents cache conflicts
- **Automatic persistence** eliminates manual save/load

**Key Features:**

1. **Automatic State Routing**: Run status determines storage location automatically
2. **Zero Configuration**: No manual persistence calls needed
3. **Backward Compatible**: Existing code works without changes
4. **Test Friendly**: Uses `XDG_CONFIG_HOME` for test isolation
5. **Graceful Degradation**: Falls back to memory-only if disk fails

### 6. Configuration Management (`/internal/config/`)
Secure, flexible configuration with multiple storage backends.

**Storage Priority:**
1. Environment variables (CI/CD friendly)
2. System keyring (Desktop secure storage)
3. Encrypted file (Universal fallback)

### 7. Error Handling (`/internal/errors/`)
Structured error system with user-friendly messaging.

```go
type ErrorType int

const (
    APIError          // HTTP API errors
    NetworkError      // Connectivity issues
    AuthError         // Authentication failures
    QuotaError        // Usage limits
    ValidationError   // Input validation
    RateLimitError    // Rate limiting
)
```

## Data Flow

### Creating a Run
```
User Input (JSON)
    ↓
Command Parser
    ↓
Validation Layer
    ↓
Git Auto-detection
    ↓
API Client (with retry)
    ↓
Cache Update
    ↓
Response Display
```

### Status Polling
```
Status Request
    ↓
Cache Check (Memory → Persistent)
    ↓ (cache miss)
API Request
    ↓
Cache Update
    ↓
Status Display
    ↓ (if --follow)
Poll Loop (with interruption handling)
```

## Security Architecture

### API Key Management
```
┌─────────────────────────────────┐
│     API Key Input/Storage       │
├─────────────────────────────────┤
│  Environment  │  Keyring  │ File │
├───────────────┼───────────┼──────┤
│   Plain Text  │  Native   │ AES  │
│   (Isolated)  │  Secure   │ 256  │
└───────────────┴───────────┴──────┘
```

### Encryption Details
- **Algorithm**: AES-256-GCM
- **Key Derivation**: Machine-specific (hardware + user info)
- **Nonce**: Random per encryption
- **Authentication**: GCM mode provides integrity

## Concurrency Model

### Thread-Safe Operations
- Cache operations use sync.RWMutex
- API client uses context for cancellation
- TUI uses message passing (actor model)

### Concurrent Patterns
```go
// Parallel API calls
var wg sync.WaitGroup
for _, id := range runIDs {
    wg.Add(1)
    go func(id string) {
        defer wg.Done()
        fetchRun(id)
    }(id)
}
wg.Wait()
```

## Performance Optimizations

### 1. Caching Strategy
- Memory cache for active data (30s TTL)
- Persistent cache for terminal states
- API response caching to reduce requests

### 2. Efficient Polling
- Adaptive intervals based on run status
- Graceful interruption handling
- Backoff for long-running operations

### 3. Resource Management
- Connection pooling for HTTP client
- Lazy loading of large datasets
- Cache size limits and cleanup

## Extension Points

### Adding New Commands
1. Create command file in `/internal/commands/`
2. Implement command logic
3. Register in root command
4. Add tests

### Adding New API Endpoints
1. Define endpoint in `/internal/api/endpoints.go`
2. Implement client method
3. Add retry logic if needed
4. Update models if required

### Adding TUI Views
1. Create view in `/internal/tui/views/`
2. Define model and update functions
3. Register in app navigation
4. Add styling and keybindings

## Design Patterns

### 1. Repository Pattern
Abstracts data access behind interfaces:
```go
type RunRepository interface {
    Create(ctx context.Context, run *Run) error
    Get(ctx context.Context, id string) (*Run, error)
    List(ctx context.Context) ([]*Run, error)
}
```

### 2. Factory Pattern
Creates configured instances:
```go
func NewAPIClient(config Config) *Client {
    return &Client{
        httpClient: buildHTTPClient(config),
        retrier:    buildRetrier(config),
    }
}
```

### 3. Strategy Pattern
Switches between storage strategies:
```go
type StorageStrategy interface {
    Store(key string, value []byte) error
    Retrieve(key string) ([]byte, error)
}
```

### 4. Observer Pattern
TUI uses message-based updates:
```go
type Model struct {
    subscriptions []chan Msg
}

func (m *Model) Notify(msg Msg) {
    for _, sub := range m.subscriptions {
        sub <- msg
    }
}
```

## Testing Architecture

### Test Pyramid
```
        ╱╲
       ╱  ╲      E2E Tests (5%)
      ╱────╲
     ╱      ╲    Integration Tests (25%)
    ╱────────╲
   ╱          ╲  Unit Tests (70%)
  ╱────────────╲
```

### Testing Strategies
- **Unit Tests**: Mock external dependencies
- **Integration Tests**: Test component interactions
- **E2E Tests**: Full workflow validation
- **Property Tests**: Fuzz testing for models
- **Benchmark Tests**: Performance validation

## Deployment Architecture

### Binary Distribution
```
repobird-cli/
├── darwin-amd64/
├── darwin-arm64/
├── linux-amd64/
├── linux-arm64/
└── windows-amd64/
```

### Docker Support
```dockerfile
FROM golang:1.20 AS builder
# Build static binary
RUN CGO_ENABLED=0 go build

FROM scratch
# Minimal runtime
COPY --from=builder /app/repobird /
```

## Monitoring & Observability

### Debug Mode
Enable with `--debug` flag:
- API request/response logging
- Performance timing
- Cache hit/miss rates
- Error stack traces

### Metrics Collection (Future)
- Command usage statistics
- API latency tracking
- Error rate monitoring
- User engagement metrics

## Scalability Considerations

### Current Limits
- Single-user CLI application
- 45-minute timeout for long operations
- Local cache storage

### Future Scalability
- Team workspaces support
- Distributed caching
- Batch operations
- Webhook notifications

## Technology Stack

### Core Technologies
- **Go 1.20+**: Primary language
- **Cobra**: CLI framework
- **Viper**: Configuration management
- **Bubble Tea**: TUI framework
- **Standard library**: HTTP, crypto, encoding

### Development Tools
- **Make**: Build automation
- **golangci-lint**: Code quality
- **gosec**: Security analysis
- **go test**: Testing framework

## Best Practices Applied

1. **Clean Architecture**: Clear separation of concerns
2. **SOLID Principles**: Interface segregation, dependency inversion
3. **12-Factor App**: Configuration, logging, disposability
4. **Go Idioms**: Error handling, interfaces, channels
5. **Security First**: Secure by default, defense in depth
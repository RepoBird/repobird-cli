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

### 5. Cache System (`/internal/cache/`)
Multi-level caching for performance and offline support with user isolation.

```
Memory Cache (30s TTL)
    ↓
Persistent Cache (Terminal runs never expire)
    ↓
Global Cache (Cross-view state)
```

**User-Based Cache Separation:**
The cache system now supports user-specific storage to prevent data mixing when multiple users share the same machine:

```
~/.cache/repobird/
├── users/
│   ├── user-123/
│   │   ├── runs/
│   │   │   ├── run-456.json
│   │   │   └── ...
│   │   └── repository_history.json
│   └── user-789/
│       ├── runs/
│       └── repository_history.json
└── shared/ (fallback for unknown users)
    ├── runs/
    └── repository_history.json
```

**User Service (`/internal/services/user_service.go`):**
- Manages current authenticated user context
- Automatically initializes user-specific cache on authentication
- Provides user ID extraction from API responses
- Handles cache switching when users login/logout

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
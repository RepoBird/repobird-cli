# RepoBird CLI - API Reference

## API Client Configuration

### Base Configuration
```go
const (
    BaseURL        = "https://api.repobird.ai"
    DefaultTimeout = 45 * time.Minute
    MaxRetries     = 5
    UserAgent      = "repobird-cli/{version}"
)
```

### Authentication
All API requests require Bearer token authentication:
```
Authorization: Bearer <API_KEY>
```

## API Endpoints

### 1. Create Run
Creates a new AI-powered code generation run.

**Endpoint:** `POST /api/v1/runs`

**Request Body:**
```json
{
  "prompt": "string",           // Required: Task description
  "repository": "string",        // Required: Format: "org/repo"
  "source": "string",           // Required: Source branch
  "target": "string",           // Optional: Target branch
  "runType": "string",          // Required: "run" or "approval"
  "title": "string",            // Optional: PR title
  "context": "string",          // Optional: Additional context
  "files": ["string"],          // Optional: Specific files
  "directories": ["string"],    // Optional: Specific directories
  "messageToReviewer": "string", // Optional: For approval runs
  "messageToApplier": "string",  // Optional: For approval runs
  "gitProvider": "string",      // Optional: "github", "gitlab", etc.
  "modelOverride": "string"     // Optional: Model selection
}
```

**Response:**
```json
{
  "data": {
    "run": {
      "id": "string",
      "status": "string",
      "createdAt": "2024-01-01T00:00:00Z",
      "repository": "org/repo",
      "sourceBranch": "main",
      "targetBranch": "feature/branch",
      "prUrl": "string"
    }
  }
}
```

**Go Implementation:**
```go
func (c *Client) CreateRun(ctx context.Context, req *RunRequest) (*Run, error)
func (c *Client) CreateRunWithRetry(ctx context.Context, req *RunRequest) (*Run, error)
```

### 2. Get Run Status
Retrieves the current status and details of a run.

**Endpoint:** `GET /api/v1/runs/{id}`

**Path Parameters:**
- `id` (string): The run ID

**Response:**
```json
{
  "id": "string",
  "status": "pending|running|completed|failed|cancelled",
  "createdAt": "2024-01-01T00:00:00Z",
  "updatedAt": "2024-01-01T00:00:00Z",
  "completedAt": "2024-01-01T00:00:00Z",
  "repository": "org/repo",
  "sourceBranch": "main",
  "targetBranch": "feature/branch",
  "prUrl": "https://github.com/org/repo/pull/123",
  "prompt": "string",
  "runType": "run",
  "error": "string",
  "progress": {
    "percentage": 75,
    "message": "Processing files..."
  }
}
```

**Go Implementation:**
```go
func (c *Client) GetRun(ctx context.Context, id string) (*Run, error)
func (c *Client) GetRunWithRetry(ctx context.Context, id string) (*Run, error)
```

### 3. List Runs
Lists all runs for the authenticated user.

**Endpoint:** `GET /api/v1/runs`

**Query Parameters:**
- `page` (int): Page number (default: 1)
- `limit` (int): Items per page (default: 20, max: 100)
- `status` (string): Filter by status
- `repository` (string): Filter by repository
- `sort` (string): Sort field (createdAt, updatedAt)
- `order` (string): Sort order (asc, desc)

**Response:**
```json
{
  "runs": [
    {
      "id": "string",
      "status": "string",
      "createdAt": "2024-01-01T00:00:00Z",
      "repository": "org/repo",
      "prompt": "string"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 100,
    "totalPages": 5
  }
}
```

**Go Implementation:**
```go
func (c *Client) ListRuns(ctx context.Context, opts ListOptions) (*RunList, error)
```

### 4. Cancel Run
Cancels an active run.

**Endpoint:** `POST /api/v1/runs/{id}/cancel`

**Response:**
```json
{
  "id": "string",
  "status": "cancelled",
  "cancelledAt": "2024-01-01T00:00:00Z"
}
```

**Go Implementation:**
```go
func (c *Client) CancelRun(ctx context.Context, id string) error
```

### 5. Verify Authentication
Verifies API key and retrieves account information.

**Endpoint:** `GET /api/v1/auth/verify`

**Response:**
```json
{
  "valid": true,
  "user": {
    "id": "string",
    "email": "user@example.com",
    "name": "string",
    "quota": {
      "used": 10,
      "limit": 100,
      "period": "month"
    }
  }
}
```

**Go Implementation:**
```go
func (c *Client) VerifyAuth(ctx context.Context) (*AuthInfo, error)
```

## Error Responses

### Standard Error Format
```json
{
  "error": {
    "code": "string",
    "message": "string",
    "details": {},
    "retryable": false
  }
}
```

### Error Codes
| Code | HTTP Status | Description | Retryable |
|------|-------------|-------------|-----------|
| `AUTH_FAILED` | 401 | Invalid or missing API key | No |
| `FORBIDDEN` | 403 | Insufficient permissions | No |
| `NOT_FOUND` | 404 | Resource not found | No |
| `VALIDATION_ERROR` | 400 | Invalid request data | No |
| `QUOTA_EXCEEDED` | 429 | Usage limit reached | No |
| `RATE_LIMITED` | 429 | Too many requests | Yes |
| `SERVER_ERROR` | 500 | Internal server error | Yes |
| `SERVICE_UNAVAILABLE` | 503 | Service temporarily down | Yes |
| `TIMEOUT` | 504 | Request timeout | Yes |

## Client Configuration

### Environment Variables
```bash
REPOBIRD_API_KEY=your_api_key_here
REPOBIRD_API_URL=https://custom.api.url  # Optional
REPOBIRD_DEBUG=true                      # Enable debug logging
```

### Programmatic Configuration
```go
client := api.NewClient(api.Config{
    APIKey:     "your_api_key",
    BaseURL:    "https://api.repobird.ai",
    Timeout:    45 * time.Minute,
    MaxRetries: 5,
    Debug:      false,
})
```

## Retry Logic

### Exponential Backoff Configuration
```go
type RetryConfig struct {
    MaxRetries     int           // Default: 5
    InitialDelay   time.Duration // Default: 1s
    MaxDelay       time.Duration // Default: 30s
    Multiplier     float64       // Default: 2.0
    JitterFraction float64       // Default: 0.1
}
```

### Retryable Conditions
- Network errors (connection refused, timeout)
- HTTP 429 (Rate Limited)
- HTTP 500, 502, 503, 504
- Circuit breaker not open

### Circuit Breaker
```go
type CircuitBreaker struct {
    FailureThreshold int           // Default: 5
    RecoveryTimeout  time.Duration // Default: 30s
    HalfOpenRequests int           // Default: 3
}
```

States:
- **Closed**: Normal operation
- **Open**: All requests fail immediately
- **Half-Open**: Limited requests to test recovery

## Rate Limiting

### Client-Side Rate Limiting
```go
rateLimiter := rate.NewLimiter(
    rate.Every(time.Second/10), // 10 requests per second
    20,                          // Burst of 20
)
```

### Server-Side Rate Limits
- **Per-minute**: 60 requests
- **Per-hour**: 1000 requests
- **Concurrent runs**: 5

Response headers:
```
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 45
X-RateLimit-Reset: 1704067200
Retry-After: 30
```

## Polling Operations

### Status Polling Configuration
```go
type PollerConfig struct {
    InitialInterval time.Duration // Default: 2s
    MaxInterval     time.Duration // Default: 30s
    Multiplier      float64       // Default: 1.5
    Timeout         time.Duration // Default: 45m
}
```

### Usage Example
```go
poller := utils.NewPoller(config)
err := poller.Poll(ctx, func() (bool, error) {
    run, err := client.GetRun(ctx, runID)
    if err != nil {
        return false, err
    }
    return run.IsTerminal(), nil
})
```

## WebSocket Events (Future)

### Connection
```javascript
ws://api.repobird.ai/v1/ws?token=<API_KEY>
```

### Event Types
```json
{
  "type": "run.status",
  "data": {
    "id": "string",
    "status": "string",
    "progress": {}
  }
}
```

## SDK Usage Examples

### Basic Run Creation
```go
package main

import (
    "context"
    "github.com/repobird/cli/internal/api"
)

func main() {
    client := api.NewClient(api.Config{
        APIKey: "your_api_key",
    })
    
    run, err := client.CreateRun(context.Background(), &api.RunRequest{
        Prompt:     "Fix the authentication bug",
        Repository: "org/repo",
        Source:     "main",
        RunType:    "run",
    })
    
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Run created: %s\n", run.ID)
}
```

### Polling with Cancellation
```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Handle interruption
go func() {
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt)
    <-sigCh
    cancel()
}()

// Poll for completion
err := client.PollRunStatus(ctx, runID, func(run *Run) {
    fmt.Printf("Status: %s (%.0f%%)\n", 
        run.Status, run.Progress.Percentage)
})
```

### Error Handling
```go
run, err := client.CreateRun(ctx, request)
if err != nil {
    switch {
    case errors.IsAuthError(err):
        // Handle authentication error
        fmt.Println("Please check your API key")
    case errors.IsQuotaExceeded(err):
        // Handle quota error
        fmt.Println("Monthly quota exceeded")
    case errors.IsRetryable(err):
        // Retry operation
        run, err = client.CreateRunWithRetry(ctx, request)
    default:
        // Handle other errors
        fmt.Printf("Error: %v\n", err)
    }
}
```

## Performance Considerations

### Connection Pooling
```go
transport := &http.Transport{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 10,
    IdleConnTimeout:     90 * time.Second,
    DisableCompression:  false,
}
```

### Request Optimization
- Use conditional requests with ETags
- Enable HTTP/2 for multiplexing
- Implement response caching
- Use pagination for list operations

### Metrics
Track these metrics for monitoring:
- Request latency (p50, p95, p99)
- Error rates by type
- Retry attempts and success rate
- Circuit breaker state changes

## Migration Guide

### From v1 to v2
```go
// v1
client := api.NewClient(apiKey)

// v2
client := api.NewClient(api.Config{
    APIKey: apiKey,
})
```

### Deprecated Methods
| Old Method | New Method | Deprecation Date |
|------------|------------|------------------|
| `client.Status()` | `client.GetRun()` | v2.0.0 |
| `client.Submit()` | `client.CreateRun()` | v2.0.0 |

## API Versioning

### Version Header
```
X-API-Version: v1
```

### Breaking Changes Policy
- Major version changes may break compatibility
- Minor versions add functionality
- Patch versions fix bugs
- Deprecation notices given 3 months in advance

## Support

### API Status
- Status page: https://status.repobird.ai
- Health check: `GET /health`

### Rate Limit Information
```bash
curl -H "Authorization: Bearer $API_KEY" \
  https://api.repobird.ai/api/v1/auth/rate-limit
```

### Debug Mode
Enable detailed logging:
```bash
export REPOBIRD_DEBUG=true
repobird status --debug
```
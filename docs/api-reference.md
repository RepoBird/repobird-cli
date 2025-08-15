# RepoBird CLI - API Reference

## Overview

REST API client implementation for RepoBird AI platform with resilience patterns and enterprise features.

## Related Documentation
- **[Architecture Overview](architecture.md)** - System design and patterns
- **[Configuration Guide](configuration-guide.md)** - API key and environment setup
- **[Troubleshooting Guide](troubleshooting.md)** - Debug mode and error handling
- **[Development Guide](development-guide.md)** - Client usage examples

## Configuration

**Base URL:** `https://repobird.ai/api/v1`  
**Authentication:** Bearer token via `Authorization: Bearer <API_KEY>`  
**Timeout:** 45 minutes default  
**User Agent:** `repobird-cli/{version}`

## Core Endpoints

### Create Run
`POST /api/v1/runs` - Submit AI-powered code generation task

**Request:**
```json
{
  "prompt": "string",           // Required: Task description
  "repository": "org/repo",     // Required: GitHub/GitLab repo
  "source": "main",            // Required: Source branch
  "target": "feature/xyz",     // Optional: Target branch
  "runType": "run|approval",   // Required: Execution mode
  "files": ["path/to/file"]    // Optional: Specific files
}
```

**Methods:**
```go
func (c *Client) CreateRun(ctx context.Context, req *RunRequest) (*Run, error)
func (c *Client) CreateRunWithRetry(ctx context.Context, req *RunRequest) (*Run, error)
```

### Get Run Status
`GET /api/v1/runs/{id}` - Retrieve run details and progress

**Response:**
```json
{
  "id": "run-123",
  "status": "pending|running|completed|failed|cancelled",
  "progress": { "percentage": 75, "message": "Processing..." },
  "prUrl": "https://github.com/org/repo/pull/123"
}
```

**Methods:**
```go
func (c *Client) GetRun(ctx context.Context, id string) (*Run, error)
func (c *Client) GetRunWithRetry(ctx context.Context, id string) (*Run, error)
```

### List Runs
`GET /api/v1/runs` - List user's runs with pagination

**Query Parameters:** `page`, `limit` (max 1000), `status`, `repository`

**Method:**
```go
func (c *Client) ListRuns(ctx context.Context, opts ListOptions) (*RunList, error)
```

### Additional Endpoints
- `DELETE /api/v1/runs/{id}` - Cancel active run
- `GET /api/v1/runs/{id}/logs` - Stream run logs
- `GET /api/v1/user` - Get user info and quotas
- `GET /api/v1/repositories` - List accessible repositories

## Error Handling

### Error Types
| Code | HTTP Status | Description | Retryable |
|------|-------------|-------------|-----------|
| `AUTH_FAILED` | 401 | Invalid API key | No |
| `QUOTA_EXCEEDED` | 429 | Usage limit reached | No |
| `RATE_LIMITED` | 429 | Too many requests | Yes |
| `SERVER_ERROR` | 500 | Internal error | Yes |
| `SERVICE_UNAVAILABLE` | 503 | Service down | Yes |

### Error Response Format
```json
{
  "error": {
    "code": "AUTH_FAILED",
    "message": "Invalid API key",
    "retryable": false
  }
}
```

### Go Error Handling
```go
if err != nil {
    switch {
    case errors.IsAuthError(err):
        // Handle authentication error
    case errors.IsQuotaExceeded(err):
        // Handle quota error  
    case errors.IsRetryable(err):
        // Retry with exponential backoff
    }
}
```

## Resilience Patterns

### Retry Configuration
```go
type RetryConfig struct {
    MaxRetries     int           // Default: 5
    InitialDelay   time.Duration // Default: 1s
    MaxDelay       time.Duration // Default: 30s
    Multiplier     float64       // Default: 2.0
}
```

### Circuit Breaker
- **Failure Threshold:** 5 consecutive failures
- **Recovery Timeout:** 30 seconds
- **States:** Closed → Open → Half-Open → Closed

### Rate Limiting
**Client-side:** 10 req/s with burst of 20  
**Server-side:** 60 req/min, 1000 req/hour

**Response Headers:**
```
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 45
X-RateLimit-Reset: 1704067200
```

## Polling Operations

### Configuration
```go
poller := utils.NewPoller(PollerConfig{
    InitialInterval: 2 * time.Second,
    MaxInterval:     30 * time.Second,
    Timeout:         45 * time.Minute,
})
```

### Usage
```go
err := poller.Poll(ctx, func() (bool, error) {
    run, _ := client.GetRun(ctx, runID)
    return run.IsTerminal(), nil
})
```

## Client Usage

### Initialization
```go
// Simple
client := api.NewClient("your_api_key")

// With custom config
client := api.NewClient(api.Config{
    APIKey:  "your_api_key",
    BaseURL: "https://custom.api.url",
    Timeout: 30 * time.Second,
})
```

### Environment Variables
```bash
REPOBIRD_API_KEY=your_key
REPOBIRD_API_URL=https://custom.url  # Optional
REPOBIRD_DEBUG=true                  # Enable debug logging
```

### Example: Create and Poll Run
```go
// Create run
run, err := client.CreateRunWithRetry(ctx, &api.RunRequest{
    Prompt:     "Fix authentication bug",
    Repository: "org/repo",
    Source:     "main",
    RunType:    "run",
})

// Poll for completion
poller := utils.NewPoller(config)
err = poller.Poll(ctx, func() (bool, error) {
    run, err := client.GetRun(ctx, run.ID)
    fmt.Printf("Status: %s (%.0f%%)\n", run.Status, run.Progress.Percentage)
    return run.IsTerminal(), err
})
```

## Performance

### Connection Pooling
```go
transport := &http.Transport{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 10,
    IdleConnTimeout:     90 * time.Second,
}
```

### Optimization Tips
- Use pagination for large datasets
- Enable HTTP/2 for multiplexing
- Implement response caching
- Monitor retry rates and circuit breaker state

## Debug Mode

Enable verbose logging:
```bash
REPOBIRD_DEBUG_LOG=1 repobird status
# Logs written to /tmp/repobird_debug.log
```

See **[Troubleshooting Guide](troubleshooting.md)** for debugging techniques.
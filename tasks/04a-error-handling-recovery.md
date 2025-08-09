# Task 04a: Error Handling & Recovery

## Overview
Implement comprehensive error handling and recovery mechanisms for the RepoBird CLI, ensuring graceful degradation and user-friendly error messages.

## Background Research

### Best Practices for Go CLI Error Handling
Based on industry best practices:
- **Always check and handle returned errors explicitly** - Go's error handling philosophy
- **Wrap errors with context** using `fmt.Errorf` and the `%w` verb for better traceability
- **Resource cleanup with defer** - Ensures resources are released regardless of error paths
- **Interpret HTTP status codes** - Distinguish between retryable (503, 429, 504) and permanent failures
- **Implement exponential backoff** - Reduces server load and improves resilience

### Retry Strategy Guidelines
- **Maximum retries:** 3 attempts (configurable)
- **Backoff sequence:** 1s, 2s, 4s, 8s (with jitter)
- **Retryable errors:** Network timeouts, 503 Service Unavailable, 429 Too Many Requests, 504 Gateway Timeout
- **Non-retryable:** 401 Unauthorized, 404 Not Found, 400 Bad Request

## Implementation Tasks

### 1. Error Types & Classification
- [x] Create custom error types in `internal/errors/types.go`
  - `APIError` - For API-related errors with status codes
  - `NetworkError` - For network connectivity issues
  - `AuthError` - For authentication failures
  - `QuotaError` - For "No runs remaining" scenarios
  - `ValidationError` - For input validation failures
- [x] Implement error classification functions
  - `IsRetryable(error) bool`
  - `IsTemporary(error) bool`
  - `IsQuotaExceeded(error) bool`

### 2. API Error Mapping
- [x] Map API status enums to user-friendly messages
  ```go
  var statusMessages = map[string]string{
      "NO_RUNS_REMAINING": "You've used all your available runs. Upgrade your plan at https://repobird.ai/dashboard",
      "REPO_NOT_FOUND": "Repository not found or not connected. Please connect it at https://repobird.ai/repos",
      "INVALID_API_KEY": "Invalid API key. Get a new one at https://repobird.ai/settings/api",
  }
  ```
- [x] Include tier information in quota errors
- [x] Add contextual help links for each error type

### 3. Retry Logic Implementation
- [x] Create `internal/retry/client.go` with exponential backoff
  ```go
  type RetryConfig struct {
      MaxAttempts int
      InitialDelay time.Duration
      MaxDelay time.Duration
      Multiplier float64
      Jitter float64
  }
  ```
- [x] Implement circuit breaker pattern for repeated failures
- [x] Add request timeout handling (45 min max for long-running operations)
- [x] Log retry attempts with debug flag

### 4. Polling & Status Updates
- [x] Implement 5-second polling for status updates
- [x] Stop polling when status is DONE, FAILED, or CANCELLED
- [x] Show progress indicators during polling
- [x] Handle poll interruptions (Ctrl+C) gracefully
- [x] Display elapsed time and estimated completion

### 5. Network Resilience
- [x] Detect network connectivity issues
  ```go
  if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
      // Apply retry logic
  }
  ```
- [x] Implement connection pooling for API requests
- [x] Add timeout configuration per operation type
- [x] Handle partial responses and resume capability

### 6. User Experience
- [x] Display remaining runs from user tier
- [x] Show clear, actionable error messages
- [x] Provide fallback suggestions for common errors
- [ ] Add `--debug` flag for verbose error output
- [ ] Color-code errors by severity (red for critical, yellow for warnings)

## Error Message Templates

```go
const (
    ErrNoRunsRemaining = "You have no runs remaining (Tier: %s, Limit: %d/month). Upgrade at: %s"
    ErrRepoNotFound = "Repository '%s' not found or not connected. Connect it at: %s"
    ErrInvalidAPIKey = "Invalid API key. Get a new one at: %s"
    ErrRateLimited = "Rate limit exceeded. Please wait %s before retrying."
    ErrNetworkTimeout = "Network timeout after %s. Check your connection and try again."
    ErrServerUnavailable = "RepoBird servers are temporarily unavailable. Please try again in a few minutes."
)
```

## Testing Requirements

### Unit Tests
- [ ] Test all error type classifications
- [ ] Test retry logic with various scenarios
- [ ] Test exponential backoff calculations
- [ ] Test circuit breaker behavior
- [ ] Test error message formatting

### Integration Tests
- [ ] Simulate network failures
- [ ] Test API error responses
- [ ] Test polling interruption
- [ ] Test graceful degradation
- [ ] Test timeout handling

## Success Metrics
- Zero unhandled panics in production
- All errors have user-friendly messages
- Retry logic reduces failure rate by >50%
- Average recovery time <10 seconds for transient errors
- User satisfaction with error clarity >90%

## Code Examples

### Retry Client Implementation
```go
func (c *RetryClient) DoWithRetry(ctx context.Context, fn func() error) error {
    delay := c.config.InitialDelay
    
    for attempt := 1; attempt <= c.config.MaxAttempts; attempt++ {
        err := fn()
        if err == nil {
            return nil
        }
        
        if !IsRetryable(err) {
            return fmt.Errorf("permanent error: %w", err)
        }
        
        if attempt == c.config.MaxAttempts {
            return fmt.Errorf("giving up after %d attempts: %w", c.config.MaxAttempts, err)
        }
        
        // Add jitter to prevent thundering herd
        jitter := time.Duration(rand.Float64() * c.config.Jitter * float64(delay))
        select {
        case <-time.After(delay + jitter):
            delay = time.Duration(float64(delay) * c.config.Multiplier)
            if delay > c.config.MaxDelay {
                delay = c.config.MaxDelay
            }
        case <-ctx.Done():
            return ctx.Err()
        }
    }
    return nil
}
```

### Graceful Degradation Example
```go
func (c *Client) GetRunStatus(id string) (*RunStatus, error) {
    status, err := c.fetchFromAPI(id)
    if err != nil {
        // Try cached version
        if cached, cacheErr := c.cache.Get(id); cacheErr == nil {
            log.Debug("Using cached status due to API error: %v", err)
            return cached, nil
        }
        
        // Provide offline status if available
        if IsNetworkError(err) {
            return &RunStatus{
                Status: "UNKNOWN",
                Message: "Unable to fetch current status. Last known state may be outdated.",
            }, nil
        }
        
        return nil, err
    }
    
    // Update cache for future use
    c.cache.Set(id, status)
    return status, nil
}
```

## Dependencies
- Standard library: `net/http`, `time`, `context`
- Consider: `github.com/cenkalti/backoff` for advanced retry strategies
- Consider: `github.com/sony/gobreaker` for circuit breaker implementation

## References
- [Go Error Handling Best Practices](https://blog.logrocket.com/error-handling-golang-best-practices/)
- [Exponential Backoff and Jitter](https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/)
- [Circuit Breaker Pattern](https://martinfowler.com/bliki/CircuitBreaker.html)
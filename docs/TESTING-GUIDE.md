# Testing Guide

## Overview

Comprehensive testing strategies and patterns for RepoBird CLI.

## Related Documentation
- **[Development Guide](DEVELOPMENT-GUIDE.md)** - Development setup and workflow
- **[Architecture Overview](ARCHITECTURE.md)** - System design for testability
- **[TUI Guide](TUI-GUIDE.md)** - TUI testing patterns

## Quick Start

```bash
# Run all tests (REPOBIRD_API_KEY is set to empty by Makefile)
make test

# With coverage
make coverage

# Specific package (set API key to empty to avoid errors)
REPOBIRD_API_KEY="" go test ./internal/api

# With race detection
REPOBIRD_API_KEY="" go test -race ./...

# Verbose output
REPOBIRD_API_KEY="" go test -v ./...

# Note: Always set REPOBIRD_API_KEY="" when running tests directly
# to avoid environment pollution and API key validation errors
```

## Test Structure

### Test Pyramid
- **Unit Tests (70%)** - Fast, isolated, mock dependencies
- **Integration Tests (25%)** - Component interaction
- **E2E Tests (5%)** - Full workflow validation

### File Organization
```
package_test.go    # Unit tests
integration_test.go # Integration tests (build tag)
testdata/          # Test fixtures
mocks/            # Mock implementations
```

## Writing Tests

### Table-Driven Tests
```go
func TestProcessRun(t *testing.T) {
    tests := []struct {
        name    string
        input   Run
        want    Status
        wantErr bool
    }{
        {
            name:  "successful run",
            input: Run{ID: "123", Status: "pending"},
            want:  StatusRunning,
        },
        {
            name:    "invalid run",
            input:   Run{ID: ""},
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ProcessRun(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("got = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Testing with Mocks
```go
type MockAPIClient struct {
    mock.Mock
}

func (m *MockAPIClient) GetRun(ctx context.Context, id string) (*Run, error) {
    args := m.Called(ctx, id)
    return args.Get(0).(*Run), args.Error(1)
}

func TestRunPoller(t *testing.T) {
    mockClient := new(MockAPIClient)
    mockClient.On("GetRun", mock.Anything, "123").
        Return(&Run{Status: "completed"}, nil)
    
    poller := NewPoller(mockClient)
    err := poller.Poll(context.Background(), "123")
    
    assert.NoError(t, err)
    mockClient.AssertExpectations(t)
}
```

## TUI Testing

### Test Isolation
```go
func TestDashboardView(t *testing.T) {
    // Isolate cache directory
    tmpDir := t.TempDir()
    t.Setenv("XDG_CONFIG_HOME", tmpDir)
    
    cache := cache.NewSimpleCache()
    view := NewDashboardView(mockClient, cache)
    
    // Test view initialization
    cmd := view.Init()
    assert.NotNil(t, cmd)
}
```

### Navigation Testing
```go
func TestNavigation(t *testing.T) {
    app := NewApp(mockClient)
    
    // Test navigation message
    model, cmd := app.Update(NavigateToCreateMsg{})
    
    // Verify view transition
    _, isCreateView := model.(*CreateView)
    assert.True(t, isCreateView)
}
```

### Key Handling Tests
```go
func TestKeyHandling(t *testing.T) {
    view := NewDashboardView(mockClient, cache)
    
    // Test key press
    model, cmd := view.Update(tea.KeyMsg{
        Type: tea.KeyRunes,
        Runes: []rune{'n'},
    })
    
    // Verify navigation command returned
    assert.NotNil(t, cmd)
}
```

## Cache Testing

### Cache Isolation
```go
func TestCacheOperations(t *testing.T) {
    tmpDir := t.TempDir()
    t.Setenv("XDG_CONFIG_HOME", tmpDir)
    
    cache := cache.NewSimpleCache()
    
    // Test cache operations
    cache.SetRuns([]*Run{{ID: "123"}})
    runs := cache.GetRuns()
    
    assert.Len(t, runs, 1)
    assert.Equal(t, "123", runs[0].ID)
}
```

### Concurrent Access
```go
func TestCacheConcurrency(t *testing.T) {
    cache := cache.NewSimpleCache()
    var wg sync.WaitGroup
    
    // Concurrent writes
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            cache.SetRun(&Run{ID: fmt.Sprintf("%d", id)})
        }(i)
    }
    
    wg.Wait()
    runs := cache.GetRuns()
    assert.Len(t, runs, 100)
}
```

## API Client Testing

### HTTP Mocking
```go
func TestAPIClient(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(
        func(w http.ResponseWriter, r *http.Request) {
            assert.Equal(t, "Bearer test_key", 
                r.Header.Get("Authorization"))
            
            json.NewEncoder(w).Encode(Run{
                ID: "123",
                Status: "running",
            })
        }))
    defer server.Close()
    
    client := NewClient("test_key")
    client.BaseURL = server.URL
    
    run, err := client.GetRun(context.Background(), "123")
    assert.NoError(t, err)
    assert.Equal(t, "running", run.Status)
}
```

### Retry Logic Testing
```go
func TestRetryLogic(t *testing.T) {
    attempts := 0
    server := httptest.NewServer(http.HandlerFunc(
        func(w http.ResponseWriter, r *http.Request) {
            attempts++
            if attempts < 3 {
                w.WriteHeader(http.StatusInternalServerError)
                return
            }
            w.WriteHeader(http.StatusOK)
        }))
    defer server.Close()
    
    client := NewClientWithRetry("key", 3)
    client.BaseURL = server.URL
    
    err := client.CreateRun(context.Background(), &RunRequest{})
    assert.NoError(t, err)
    assert.Equal(t, 3, attempts)
}
```

## Integration Tests

### Build Tags
```go
//go:build integration
// +build integration

package api_test

func TestRealAPIIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    client := NewClient(os.Getenv("TEST_API_KEY"))
    runs, err := client.ListRuns(context.Background())
    
    assert.NoError(t, err)
    assert.NotEmpty(t, runs)
}
```

Run with:
```bash
go test -tags=integration ./...
```

## Test Coverage

### Generate Coverage Report
```bash
# Generate coverage
make coverage

# View HTML report
go tool cover -html=coverage.out

# Check coverage threshold
go test -cover ./... | grep -E "coverage: [7-9][0-9]\.[0-9]%|100\.0%"
```

### Coverage Requirements
- Minimum 70% for new code
- Critical paths must have 90%+
- Error handling must be covered

## Test Data Management

### Using testdata Directory
```go
func TestParseConfig(t *testing.T) {
    data, err := os.ReadFile("testdata/valid_config.json")
    require.NoError(t, err)
    
    config, err := ParseConfig(data)
    assert.NoError(t, err)
    assert.Equal(t, "expected_value", config.Field)
}
```

### Golden Files
```go
func TestGoldenOutput(t *testing.T) {
    got := GenerateOutput(input)
    golden := filepath.Join("testdata", "output.golden")
    
    if *update {
        os.WriteFile(golden, []byte(got), 0644)
    }
    
    want, _ := os.ReadFile(golden)
    assert.Equal(t, string(want), got)
}
```

## Performance Testing

### Benchmarks
```go
func BenchmarkCacheOperations(b *testing.B) {
    cache := NewSimpleCache()
    run := &Run{ID: "test"}
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        cache.SetRun(run)
        cache.GetRun("test")
    }
}
```

Run benchmarks:
```bash
go test -bench=. -benchmem ./...
```

## Best Practices

1. **Test Naming** - Use descriptive names: `TestFeature_Scenario_Expectation`
2. **Isolation** - Each test should be independent
3. **Cleanup** - Use `t.Cleanup()` for resource cleanup
4. **Assertions** - Use testify for clear assertions
5. **Parallel Tests** - Use `t.Parallel()` where safe
6. **Error Messages** - Provide context in failures

## Common Patterns

### Testing Contexts
```go
func TestWithTimeout(t *testing.T) {
    ctx, cancel := context.WithTimeout(
        context.Background(), 
        100*time.Millisecond,
    )
    defer cancel()
    
    err := SlowOperation(ctx)
    assert.ErrorIs(t, err, context.DeadlineExceeded)
}
```

### Testing Goroutines
```go
func TestConcurrentOperation(t *testing.T) {
    done := make(chan bool)
    
    go func() {
        // Async operation
        done <- true
    }()
    
    select {
    case <-done:
        // Success
    case <-time.After(1 * time.Second):
        t.Fatal("timeout waiting for goroutine")
    }
}
```

## Debugging Tests

```bash
# Run single test with verbose output
go test -v -run TestSpecificFunction ./package

# Debug with delve
dlv test ./package -- -test.run TestFunction

# Race detection
go test -race ./...

# CPU profiling
go test -cpuprofile=cpu.prof ./...
go tool pprof cpu.prof
```
# RepoBird CLI - Testing Guide

## Testing Philosophy

RepoBird CLI follows a comprehensive testing strategy that ensures reliability, maintainability, and confidence in code changes. Our approach emphasizes:

- **Test Pyramid**: More unit tests, fewer integration tests, minimal E2E tests
- **Fast Feedback**: Quick test execution for rapid development
- **Isolation**: Tests should not depend on external services unless necessary
- **Clarity**: Tests serve as documentation for expected behavior

## Test Organization

### Directory Structure
```
.
├── internal/
│   ├── api/
│   │   ├── client_test.go      # Unit tests
│   │   └── integration_test.go # Integration tests
│   ├── cache/
│   │   └── cache_test.go       # Comprehensive cache tests
│   └── ...
├── pkg/
│   └── utils/
│       └── git_test.go
└── test/
    ├── fixtures/               # Test data files
    ├── mocks/                 # Mock implementations
    └── e2e/                   # End-to-end tests
```

## Running Tests

### Quick Commands
```bash
# Run all tests
make test

# Run with coverage
make coverage

# Run unit tests only
make test-unit

# Run integration tests
make test-integration

# Run benchmarks
make benchmark

# Run with race detection
go test -race ./...
```

### Targeted Testing
```bash
# Test specific package
go test ./internal/api

# Test with verbose output
go test -v ./internal/cache

# Run specific test
go test -run TestClient_CreateRun ./internal/api

# Run tests matching pattern
go test -run ".*Cache.*" ./...

# Skip slow tests
go test -short ./...
```

## Writing Unit Tests

### Basic Test Structure
```go
package api_test

import (
    "testing"
    "github.com/repobird/cli/internal/api"
)

func TestClient_CreateRun(t *testing.T) {
    // Arrange
    client := api.NewClient(api.Config{
        APIKey: "test-key",
    })
    
    // Act
    run, err := client.CreateRun(ctx, request)
    
    // Assert
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if run.ID == "" {
        t.Error("expected run ID to be set")
    }
}
```

### Table-Driven Tests
```go
func TestValidateRunRequest(t *testing.T) {
    tests := []struct {
        name    string
        request RunRequest
        wantErr bool
        errMsg  string
    }{
        {
            name: "valid request",
            request: RunRequest{
                Prompt:     "Fix bug",
                Repository: "org/repo",
                Source:     "main",
            },
            wantErr: false,
        },
        {
            name: "missing prompt",
            request: RunRequest{
                Repository: "org/repo",
                Source:     "main",
            },
            wantErr: true,
            errMsg:  "prompt is required",
        },
        {
            name: "invalid repository format",
            request: RunRequest{
                Prompt:     "Fix bug",
                Repository: "invalid",
                Source:     "main",
            },
            wantErr: true,
            errMsg:  "repository must be in format org/repo",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateRunRequest(tt.request)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if err != nil && tt.errMsg != "" {
                if !strings.Contains(err.Error(), tt.errMsg) {
                    t.Errorf("error message = %v, want %v", err.Error(), tt.errMsg)
                }
            }
        })
    }
}
```

### Testing with Subtests
```go
func TestCache(t *testing.T) {
    cache := NewCache()
    
    t.Run("Set and Get", func(t *testing.T) {
        cache.Set("key", "value", time.Hour)
        val, found := cache.Get("key")
        if !found {
            t.Error("expected value to be found")
        }
        if val != "value" {
            t.Errorf("got %v, want %v", val, "value")
        }
    })
    
    t.Run("Expiration", func(t *testing.T) {
        cache.Set("temp", "value", time.Millisecond)
        time.Sleep(2 * time.Millisecond)
        _, found := cache.Get("temp")
        if found {
            t.Error("expected value to be expired")
        }
    })
    
    t.Run("Delete", func(t *testing.T) {
        cache.Set("delete-me", "value", time.Hour)
        cache.Delete("delete-me")
        _, found := cache.Get("delete-me")
        if found {
            t.Error("expected value to be deleted")
        }
    })
}
```

## Mocking and Test Doubles

### Interface-Based Mocking
```go
// Define interface
type APIClient interface {
    CreateRun(ctx context.Context, req *RunRequest) (*Run, error)
    GetRun(ctx context.Context, id string) (*Run, error)
}

// Mock implementation
type MockAPIClient struct {
    CreateRunFunc func(ctx context.Context, req *RunRequest) (*Run, error)
    GetRunFunc    func(ctx context.Context, id string) (*Run, error)
}

func (m *MockAPIClient) CreateRun(ctx context.Context, req *RunRequest) (*Run, error) {
    if m.CreateRunFunc != nil {
        return m.CreateRunFunc(ctx, req)
    }
    return &Run{ID: "mock-id"}, nil
}

// Use in tests
func TestService(t *testing.T) {
    mockClient := &MockAPIClient{
        CreateRunFunc: func(ctx context.Context, req *RunRequest) (*Run, error) {
            return &Run{ID: "test-123"}, nil
        },
    }
    
    service := NewService(mockClient)
    // Test service methods
}
```

### HTTP Mocking
```go
func TestHTTPClient(t *testing.T) {
    // Create test server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify request
        if r.URL.Path != "/api/v1/runs" {
            t.Errorf("unexpected path: %s", r.URL.Path)
        }
        
        // Send response
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "id": "test-123",
            "status": "pending",
        })
    }))
    defer server.Close()
    
    // Use test server
    client := NewClient(Config{
        BaseURL: server.URL,
        APIKey:  "test-key",
    })
    
    run, err := client.CreateRun(context.Background(), &RunRequest{})
    if err != nil {
        t.Fatal(err)
    }
    if run.ID != "test-123" {
        t.Errorf("got ID %s, want test-123", run.ID)
    }
}
```

## Testing Patterns

### Testing Error Conditions
```go
func TestErrorHandling(t *testing.T) {
    tests := []struct {
        name       string
        statusCode int
        response   string
        wantErr    error
    }{
        {
            name:       "unauthorized",
            statusCode: 401,
            response:   `{"error": "invalid api key"}`,
            wantErr:    ErrUnauthorized,
        },
        {
            name:       "rate limited",
            statusCode: 429,
            response:   `{"error": "rate limit exceeded"}`,
            wantErr:    ErrRateLimited,
        },
        {
            name:       "server error",
            statusCode: 500,
            response:   `{"error": "internal server error"}`,
            wantErr:    ErrServerError,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(tt.statusCode)
                w.Write([]byte(tt.response))
            }))
            defer server.Close()
            
            client := NewClient(Config{BaseURL: server.URL})
            _, err := client.GetRun(context.Background(), "test")
            
            if !errors.Is(err, tt.wantErr) {
                t.Errorf("got error %v, want %v", err, tt.wantErr)
            }
        })
    }
}
```

### Testing Concurrent Operations
```go
func TestConcurrentCache(t *testing.T) {
    cache := NewCache()
    
    // Run concurrent operations
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(n int) {
            defer wg.Done()
            key := fmt.Sprintf("key-%d", n)
            cache.Set(key, n, time.Hour)
            
            val, found := cache.Get(key)
            if !found {
                t.Errorf("key %s not found", key)
            }
            if val.(int) != n {
                t.Errorf("got %v, want %d", val, n)
            }
        }(i)
    }
    wg.Wait()
}
```

### Testing with Context
```go
func TestWithTimeout(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()
    
    client := NewClient(Config{})
    
    // Simulate slow server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        time.Sleep(200 * time.Millisecond)
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()
    
    client.BaseURL = server.URL
    _, err := client.GetRun(ctx, "test")
    
    if !errors.Is(err, context.DeadlineExceeded) {
        t.Errorf("expected timeout error, got %v", err)
    }
}
```

## Integration Testing

### Database Integration
```go
// +build integration

func TestDatabaseIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    
    // Setup test database
    db := setupTestDB(t)
    defer cleanupDB(db)
    
    repo := NewRepository(db)
    
    // Test operations
    run := &Run{
        ID:     "test-123",
        Status: "pending",
    }
    
    err := repo.Create(context.Background(), run)
    if err != nil {
        t.Fatal(err)
    }
    
    retrieved, err := repo.Get(context.Background(), "test-123")
    if err != nil {
        t.Fatal(err)
    }
    
    if retrieved.Status != "pending" {
        t.Errorf("got status %s, want pending", retrieved.Status)
    }
}
```

### API Integration
```go
// +build integration

func TestRealAPI(t *testing.T) {
    apiKey := os.Getenv("REPOBIRD_API_KEY")
    if apiKey == "" {
        t.Skip("REPOBIRD_API_KEY not set")
    }
    
    client := NewClient(Config{
        APIKey: apiKey,
    })
    
    // Test real API
    auth, err := client.VerifyAuth()
    if err != nil {
        t.Fatal(err)
    }
    
    if auth.User.Email == "" {
        t.Error("expected user email")
    }
}
```

## Benchmarking

### Writing Benchmarks
```go
func BenchmarkCache_Set(b *testing.B) {
    cache := NewCache()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        cache.Set(fmt.Sprintf("key-%d", i), i, time.Hour)
    }
}

func BenchmarkCache_Get(b *testing.B) {
    cache := NewCache()
    
    // Setup
    for i := 0; i < 1000; i++ {
        cache.Set(fmt.Sprintf("key-%d", i), i, time.Hour)
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        cache.Get(fmt.Sprintf("key-%d", i%1000))
    }
}

func BenchmarkCache_Parallel(b *testing.B) {
    cache := NewCache()
    
    b.RunParallel(func(pb *testing.PB) {
        i := 0
        for pb.Next() {
            key := fmt.Sprintf("key-%d", i)
            cache.Set(key, i, time.Hour)
            cache.Get(key)
            i++
        }
    })
}
```

### Running Benchmarks
```bash
# Run all benchmarks
go test -bench=.

# Run specific benchmark
go test -bench=BenchmarkCache

# Run with memory allocation stats
go test -bench=. -benchmem

# Run for specific duration
go test -bench=. -benchtime=10s

# Compare benchmarks
go test -bench=. -count=10 > old.txt
# Make changes
go test -bench=. -count=10 > new.txt
benchstat old.txt new.txt
```

## Test Coverage

### Generating Coverage Reports
```bash
# Generate coverage profile
go test -coverprofile=coverage.out ./...

# View coverage in terminal
go tool cover -func=coverage.out

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html

# View coverage by package
go test -cover ./...
```

### Coverage Guidelines
- Aim for **70%+ coverage** for new code
- Critical paths should have **90%+ coverage**
- Focus on behavior coverage, not line coverage
- Exclude generated code from coverage

### Improving Coverage
```go
// Identify uncovered lines
go test -coverprofile=coverage.out ./internal/api
go tool cover -html=coverage.out

// Add tests for edge cases
func TestEdgeCases(t *testing.T) {
    // Test nil inputs
    err := ProcessRun(nil)
    if err == nil {
        t.Error("expected error for nil input")
    }
    
    // Test empty values
    err = ProcessRun(&Run{})
    if err == nil {
        t.Error("expected error for empty run")
    }
    
    // Test boundary conditions
    run := &Run{
        Title: strings.Repeat("a", 1001), // Over max length
    }
    err = ProcessRun(run)
    if err == nil {
        t.Error("expected error for title too long")
    }
}
```

## Testing TUI Components

### TUI Model Testing
```go
func TestTUIModel(t *testing.T) {
    model := NewRunListView(mockClient)
    
    // Test initialization
    cmd := model.Init()
    if cmd == nil {
        t.Error("expected init command")
    }
    
    // Test key handling
    updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
    if updatedModel == nil {
        t.Error("expected updated model")
    }
    
    // Test view rendering
    view := model.View()
    if !strings.Contains(view, "RepoBird CLI") {
        t.Error("expected title in view")
    }
}
```

### Testing TUI Messages
```go
func TestTUIMessages(t *testing.T) {
    model := NewRunListView(mockClient)
    
    // Test data loaded message
    msg := runsLoadedMsg{
        runs: []Run{{ID: "test-123"}},
        err:  nil,
    }
    
    updatedModel, _ := model.Update(msg)
    listView := updatedModel.(*RunListView)
    
    if len(listView.runs) != 1 {
        t.Error("expected runs to be loaded")
    }
}
```

## Bubble Tea Layout Testing

RepoBird CLI uses Bubble Tea for its TUI. Testing layout rendering and preventing black screen issues requires specific patterns to simulate terminal dimensions and validate view output.

### Dependencies

Install the testing libraries:
```bash
go get github.com/charmbracelet/bubbletea/teatest
```

### Window Size Testing

Test that views handle different terminal sizes correctly:

```go
import (
    "strings"
    "testing"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/bubbletea/teatest"
)

func TestView_RendersAtDifferentSizes(t *testing.T) {
    tests := []struct {
        name   string
        width  int
        height int
        want   []string // Expected content in view
    }{
        {
            name:   "standard terminal",
            width:  80,
            height: 24,
            want:   []string{"RepoBird CLI", "Create New Run"},
        },
        {
            name:   "wide terminal", 
            width:  120,
            height: 30,
            want:   []string{"RepoBird CLI", "Create New Run"},
        },
        {
            name:   "narrow terminal",
            width:  40,
            height: 20,
            want:   []string{"RepoBird", "Create"},
        },
        {
            name:   "very small terminal",
            width:  20,
            height: 10,
            want:   []string{"CLI"}, // Should still render something
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            model := NewCreateRunView(mockClient)
            
            // Simulate window size message
            updatedModel, _ := model.Update(tea.WindowSizeMsg{
                Width:  tt.width,
                Height: tt.height,
            })
            
            view := updatedModel.View()
            
            // Check that view is not empty (prevents black screen)
            if strings.TrimSpace(view) == "" {
                t.Errorf("view is empty at size %dx%d", tt.width, tt.height)
            }
            
            // Check for expected content
            for _, want := range tt.want {
                if !strings.Contains(view, want) {
                    t.Errorf("view missing expected content %q at size %dx%d\nView:\n%s", 
                        want, tt.width, tt.height, view)
                }
            }
        })
    }
}
```

### Layout Dimension Testing

Test that components properly store and use dimensions:

```go
func TestView_StoreDimensions(t *testing.T) {
    model := NewCreateRunView(mockClient)
    
    // Send window size message
    updatedModel, _ := model.Update(tea.WindowSizeMsg{
        Width:  100,
        Height: 40,
    })
    
    createView := updatedModel.(*CreateRunView)
    
    // Verify dimensions are stored
    if createView.width != 100 {
        t.Errorf("width = %d, want 100", createView.width)
    }
    if createView.height != 40 {
        t.Errorf("height = %d, want 40", createView.height)
    }
    
    // Test that view renders with stored dimensions
    view := createView.View()
    if strings.TrimSpace(view) == "" {
        t.Error("view should not be empty after setting dimensions")
    }
}
```

### View Transition Testing

Test that dimensions are preserved when transitioning between views:

```go
func TestView_PreserveDimensionsOnTransition(t *testing.T) {
    // Start with create view
    createView := NewCreateRunView(mockClient)
    
    // Set dimensions
    updatedCreateView, _ := createView.Update(tea.WindowSizeMsg{
        Width:  80,
        Height: 24,
    })
    
    // Simulate successful run creation
    mockRun := models.RunResponse{ID: "test-123"}
    msg := runCreatedMsg{run: mockRun, err: nil}
    
    // Transition to details view
    detailsView, _ := updatedCreateView.Update(msg)
    
    // Verify details view received dimensions
    details := detailsView.(*RunDetailsView)
    if details.width != 80 {
        t.Errorf("details view width = %d, want 80", details.width)
    }
    if details.height != 24 {
        t.Errorf("details view height = %d, want 24", details.height)
    }
    
    // Verify details view renders properly
    view := details.View()
    if strings.TrimSpace(view) == "" {
        t.Error("details view should not be empty after transition")
    }
}
```

### Using teatest for Integration Testing

For more complex TUI testing scenarios:

```go
func TestTUI_FullFlow(t *testing.T) {
    model := NewCreateRunView(mockClient)
    
    tm := teatest.NewTestModel(
        t, model,
        teatest.WithInitialTermSize(80, 24),
    )
    
    // Send keystrokes to fill form
    tea.Send(tm, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Test run")})
    tea.Send(tm, tea.KeyMsg{Type: tea.KeyTab})
    tea.Send(tm, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("org/repo")})
    
    // Submit form
    tea.Send(tm, tea.KeyMsg{Type: tea.KeyCtrlS})
    
    // Get final model and verify state
    fm := tm.FinalModel(t)
    
    // Verify view output
    output := fm.View()
    if strings.TrimSpace(output) == "" {
        t.Error("final view should not be empty")
    }
}
```

### Black Screen Prevention Tests

Specifically test for conditions that could cause black screens:

```go
func TestView_PreventBlackScreen(t *testing.T) {
    tests := []struct {
        name      string
        setupFunc func(*CreateRunView) *CreateRunView
    }{
        {
            name: "zero dimensions",
            setupFunc: func(v *CreateRunView) *CreateRunView {
                // Don't send window size message - should still render
                return v
            },
        },
        {
            name: "minimal dimensions",
            setupFunc: func(v *CreateRunView) *CreateRunView {
                updatedView, _ := v.Update(tea.WindowSizeMsg{Width: 1, Height: 1})
                return updatedView.(*CreateRunView)
            },
        },
        {
            name: "after submission",
            setupFunc: func(v *CreateRunView) *CreateRunView {
                // Set dimensions first
                updatedView, _ := v.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
                v = updatedView.(*CreateRunView)
                
                // Start submission
                v.submitting = true
                return v
            },
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            model := NewCreateRunView(mockClient)
            model = tt.setupFunc(model)
            
            view := model.View()
            
            // Should always render something, never empty
            if strings.TrimSpace(view) == "" {
                t.Errorf("view is empty in scenario: %s", tt.name)
            }
            
            // Should contain some basic UI elements
            if !strings.Contains(view, "Create") && !strings.Contains(view, "Run") {
                t.Errorf("view missing basic UI elements in scenario: %s\nView:\n%s", 
                    tt.name, view)
            }
        })
    }
}
```

### Best Practices for TUI Testing

1. **Always Test Multiple Sizes**: Test your views at different terminal dimensions
2. **Test Edge Cases**: Zero dimensions, very small terminals, very large terminals
3. **Test State Transitions**: Ensure dimensions are preserved across view changes
4. **Mock External Dependencies**: Use mock API clients to control test scenarios
5. **Use Table-Driven Tests**: Test multiple scenarios systematically
6. **Check for Non-Empty Output**: Always verify views render something to prevent black screens
7. **Test Interactive Elements**: Verify keyboard navigation and input handling works correctly

### Testing Checklist for New Views

For every new TUI view, ensure tests cover:

- [ ] Handles `tea.WindowSizeMsg` correctly
- [ ] Stores width/height in model
- [ ] View() returns non-empty content at different sizes
- [ ] Dimensions are passed to child views
- [ ] Layout adapts to terminal size constraints
- [ ] No black screen at minimum terminal sizes
- [ ] Interactive elements work with different layouts

## Testing Best Practices

### 1. Test Naming
```go
// Good: Descriptive test names
func TestClient_CreateRun_WithValidInput_ReturnsRun(t *testing.T)
func TestCache_Get_AfterExpiration_ReturnsNotFound(t *testing.T)

// Bad: Vague names
func TestCreate(t *testing.T)
func TestCache1(t *testing.T)
```

### 2. Test Independence
```go
// Good: Each test is independent
func TestIndependent(t *testing.T) {
    t.Run("test1", func(t *testing.T) {
        cache := NewCache() // Fresh instance
        // test logic
    })
    
    t.Run("test2", func(t *testing.T) {
        cache := NewCache() // Fresh instance
        // test logic
    })
}

// Bad: Tests depend on shared state
var sharedCache = NewCache() // Shared state

func TestDependent1(t *testing.T) {
    sharedCache.Set("key", "value", time.Hour)
}

func TestDependent2(t *testing.T) {
    val, _ := sharedCache.Get("key") // Depends on TestDependent1
}
```

### 3. Clear Assertions
```go
// Good: Clear error messages
if got != want {
    t.Errorf("CreateRun() returned ID = %v, want %v", got, want)
}

// Bad: Generic messages
if got != want {
    t.Error("test failed")
}
```

### 4. Test Helpers
```go
// Define reusable test helpers
func setupTestClient(t *testing.T) *Client {
    t.Helper()
    return &Client{
        APIKey: "test-key",
        HTTPClient: &http.Client{
            Timeout: 5 * time.Second,
        },
    }
}

func assertRunEqual(t *testing.T, got, want *Run) {
    t.Helper()
    if got.ID != want.ID {
        t.Errorf("ID = %v, want %v", got.ID, want.ID)
    }
    if got.Status != want.Status {
        t.Errorf("Status = %v, want %v", got.Status, want.Status)
    }
}
```

### 5. Cleanup
```go
func TestWithCleanup(t *testing.T) {
    // Setup
    tmpDir := t.TempDir() // Automatically cleaned up
    
    file, err := os.Create(filepath.Join(tmpDir, "test.txt"))
    if err != nil {
        t.Fatal(err)
    }
    defer file.Close() // Ensure cleanup
    
    // Register cleanup function
    t.Cleanup(func() {
        // Additional cleanup if needed
    })
    
    // Test logic
}
```

## Continuous Integration

### GitHub Actions Workflow
```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - uses: actions/setup-go@v4
        with:
          go-version: '1.20'
      
      - name: Run tests
        run: |
          make test
          make coverage
      
      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out
```

## Test Data Management

### Fixtures
```go
// test/fixtures/fixtures.go
package fixtures

import _ "embed"

//go:embed testdata/valid_run.json
var ValidRunJSON []byte

//go:embed testdata/invalid_run.json
var InvalidRunJSON []byte

// Use in tests
func TestWithFixtures(t *testing.T) {
    var run Run
    err := json.Unmarshal(fixtures.ValidRunJSON, &run)
    if err != nil {
        t.Fatal(err)
    }
}
```

### Test Builders
```go
// Test data builder pattern
type RunBuilder struct {
    run Run
}

func NewRunBuilder() *RunBuilder {
    return &RunBuilder{
        run: Run{
            ID:     "test-123",
            Status: "pending",
        },
    }
}

func (b *RunBuilder) WithStatus(status string) *RunBuilder {
    b.run.Status = status
    return b
}

func (b *RunBuilder) WithRepository(repo string) *RunBuilder {
    b.run.Repository = repo
    return b
}

func (b *RunBuilder) Build() Run {
    return b.run
}

// Use in tests
func TestWithBuilder(t *testing.T) {
    run := NewRunBuilder().
        WithStatus("completed").
        WithRepository("org/repo").
        Build()
    
    // Test with custom run
}
```

## Debugging Tests

### Verbose Output
```bash
# Run with verbose flag
go test -v ./...

# Add debug logging in tests
func TestDebug(t *testing.T) {
    t.Logf("Debug: value = %v", value)
}
```

### Failed Test Investigation
```bash
# Run only failed test
go test -run TestThatFailed ./package

# Run with race detector
go test -race -run TestThatFailed ./package

# Get stack trace
GOTRACEBACK=all go test -run TestThatFailed ./package
```

### Test Isolation
```bash
# Run tests in random order
go test -shuffle=on ./...

# Run single test in isolation
go test -count=1 -run "^TestSpecific$" ./package
```

## Performance Testing

### Load Testing
```go
func TestUnderLoad(t *testing.T) {
    cache := NewCache()
    
    start := time.Now()
    for i := 0; i < 10000; i++ {
        cache.Set(fmt.Sprintf("key-%d", i), i, time.Hour)
    }
    elapsed := time.Since(start)
    
    if elapsed > time.Second {
        t.Errorf("Set operations took too long: %v", elapsed)
    }
}
```

### Memory Testing
```go
func TestMemoryUsage(t *testing.T) {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    before := m.Alloc
    
    // Run operation
    cache := NewCache()
    for i := 0; i < 1000; i++ {
        cache.Set(fmt.Sprintf("key-%d", i), make([]byte, 1024), time.Hour)
    }
    
    runtime.ReadMemStats(&m)
    after := m.Alloc
    
    used := after - before
    if used > 10*1024*1024 { // 10MB
        t.Errorf("Used too much memory: %d bytes", used)
    }
}
```

## Test Maintenance

### Keeping Tests Fast
1. Use `testing.Short()` for slow tests
2. Mock external dependencies
3. Use parallel test execution
4. Minimize file I/O
5. Cache expensive setup

### Test Documentation
```go
// TestCache_ConcurrentAccess verifies that the cache
// correctly handles concurrent read and write operations
// without data races or corruption.
//
// The test creates 100 goroutines that simultaneously:
// - Write unique values to the cache
// - Read and verify those values
// - Delete random entries
//
// Success criteria:
// - No panics or data races
// - All values are correctly stored and retrieved
// - Performance remains acceptable under load
func TestCache_ConcurrentAccess(t *testing.T) {
    // Test implementation
}
```
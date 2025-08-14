# RepoBird CLI - Integration Testing Guide

## Overview

The RepoBird CLI integration test suite ensures end-to-end functionality by executing the actual compiled binary and validating real command-line interactions. This guide covers the architecture, implementation, and best practices for integration testing.

## Table of Contents

1. [Architecture](#architecture)
2. [Test Infrastructure](#test-infrastructure)
3. [Running Tests](#running-tests)
4. [Writing Tests](#writing-tests)
5. [Mock Server](#mock-server)
6. [Test Patterns](#test-patterns)
7. [Safety Features](#safety-features)
8. [Troubleshooting](#troubleshooting)
9. [Best Practices](#best-practices)
10. [CI/CD Integration](#cicd-integration)

## Architecture

### Design Principles

- **Binary Testing**: Tests execute the actual CLI binary, not Go functions
- **Isolation**: Each test runs in an isolated environment with temporary directories
- **Safety First**: All destructive operations use `--dry-run` flags
- **Mock APIs**: Never calls production servers during testing
- **Fast Execution**: Complete suite runs in < 2 seconds

### Directory Structure

```
test/integration/
├── cli_test.go          # Main test suite
├── mock_server.go       # Mock API server implementation
├── helpers.go           # Test utilities and assertions
├── testdata/
│   ├── task.json        # Valid task configuration
│   ├── task.yaml        # YAML task configuration
│   ├── task.md          # Markdown task format
│   ├── invalid.json     # Malformed JSON for error testing
│   ├── bulk_config.json # Bulk run configuration
│   └── golden/          # Expected output snapshots
│       ├── version.txt  # Version command output
│       ├── help.txt     # Help command output
│       ├── status.json  # JSON status format
│       └── status.yaml  # YAML status format
└── README.md            # Quick reference guide
```

## Test Infrastructure

### Build Tags

Integration tests use build tags to prevent accidental execution:

```go
//go:build integration
// +build integration

package integration
```

### Helper Functions

The test suite provides comprehensive helper functions in `helpers.go`:

```go
// Execute CLI commands
func RunCommand(t *testing.T, args ...string) *CommandResult
func RunCommandWithEnv(t *testing.T, env map[string]string, args ...string) *CommandResult
func RunCommandWithInput(t *testing.T, input string, args ...string) *CommandResult

// Environment setup
func SetupTestConfig(t *testing.T) string
func SetupTestEnv(t *testing.T) (map[string]string, *MockServer)

// Assertions
func AssertSuccess(t *testing.T, result *CommandResult)
func AssertFailure(t *testing.T, result *CommandResult)
func AssertExitCode(t *testing.T, result *CommandResult, expected int)
func AssertContains(t *testing.T, output, expected string)
func AssertNotContains(t *testing.T, output, unexpected string)
func AssertJSONEquals(t *testing.T, actual, expected string)

// Golden files
func CompareGolden(t *testing.T, actual, goldenPath string, update bool)
func GetUpdateFlag() bool

// Test data
func CreateTestFile(t *testing.T, dir, name, content string) string
func CreateTestDirectory(t *testing.T, base string, structure map[string]string)
```

### Binary Management

The test suite builds the CLI binary once and reuses it across all tests:

```go
var (
    binaryPath string
    buildOnce  sync.Once
    buildErr   error
)

func BuildBinary(t *testing.T) string {
    t.Helper()
    buildOnce.Do(func() {
        // Build binary to /tmp/repobird-test
        cmd := exec.Command("go", "build", "-o", binaryPath, "../../cmd/repobird")
        // ...
    })
    return binaryPath
}
```

## Running Tests

### Make Targets

```bash
# Run integration tests only
make test-integration

# Run all tests (unit + integration)
make test-all

# Generate coverage report
make coverage-integration

# Run with race detection
go test -tags=integration -race ./test/integration
```

### Direct Execution

```bash
# Run all integration tests
go test -tags=integration -v ./test/integration

# Run specific test
go test -tags=integration -v ./test/integration -run TestVersionCommand

# Run with timeout
go test -tags=integration -v -timeout 30s ./test/integration

# Update golden files
UPDATE_GOLDEN=1 go test -tags=integration ./test/integration -run TestGoldenFiles
```

### Test Selection

```bash
# Run tests matching pattern
go test -tags=integration -v ./test/integration -run ".*Config.*"

# Skip slow tests
go test -tags=integration -short ./test/integration

# Run tests in random order
go test -tags=integration -shuffle=on ./test/integration
```

## Writing Tests

### Basic Test Structure

```go
func TestCommandName(t *testing.T) {
    // Setup test environment
    env, mockServer := SetupTestEnv(t)
    defer mockServer.Close()
    
    // Execute command
    result := RunCommandWithEnv(t, env, "command", "arg1", "arg2")
    
    // Assert results
    AssertSuccess(t, result)
    AssertContains(t, result.Stdout, "expected output")
}
```

### Table-Driven Tests

```go
func TestMultipleScenarios(t *testing.T) {
    tests := []struct {
        name     string
        args     []string
        env      map[string]string
        wantExit int
        contains []string
        wantErr  string
    }{
        {
            name:     "valid command",
            args:     []string{"status"},
            wantExit: 0,
            contains: []string{"ID", "Status", "Repository"},
        },
        {
            name:     "missing argument",
            args:     []string{"run"},
            wantExit: 1,
            wantErr:  "requires",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := RunCommandWithEnv(t, tt.env, tt.args...)
            AssertExitCode(t, result, tt.wantExit)
            
            for _, expected := range tt.contains {
                AssertContains(t, result.Stdout, expected)
            }
            
            if tt.wantErr != "" {
                AssertContains(t, result.Stderr, tt.wantErr)
            }
        })
    }
}
```

### Testing with Input

```go
func TestInteractiveCommand(t *testing.T) {
    input := "yes\n"
    result := RunCommandWithInput(t, input, "confirm", "action")
    
    AssertSuccess(t, result)
    AssertContains(t, result.Stdout, "Action confirmed")
}
```

### Golden File Testing

```go
func TestGoldenOutput(t *testing.T) {
    update := GetUpdateFlag() // Check UPDATE_GOLDEN env var
    
    result := RunCommand(t, "help")
    AssertSuccess(t, result)
    
    goldenPath := filepath.Join("testdata", "golden", "help.txt")
    CompareGolden(t, result.Stdout, goldenPath, update)
}
```

## Mock Server

### Overview

The mock server (`mock_server.go`) simulates the RepoBird API for testing:

```go
type MockServer struct {
    *httptest.Server
    mu           sync.RWMutex
    runs         map[string]*MockRun
    bulkRuns     map[string]*MockBulkRun
    apiKeys      map[string]bool
    rateLimits   map[string]int
    failNext     bool
    responseTime time.Duration
}
```

### Endpoints

The mock server implements all necessary API endpoints:

- `GET /api/v1/auth/verify` - Verify API key
- `GET /api/v1/users/me` - Get user information
- `POST /api/v1/runs` - Create run (respects --dry-run)
- `GET /api/v1/runs` - List all runs
- `GET /api/v1/runs/{id}` - Get specific run
- `POST /api/v1/bulk/runs` - Create bulk runs
- `GET /api/v1/bulk/{id}` - Get bulk status

### Configuration

```go
func TestWithMockServer(t *testing.T) {
    mockServer := NewMockServer(t)
    defer mockServer.Close()
    
    // Add custom API key
    mockServer.AddAPIKey("CUSTOM_KEY")
    
    // Simulate server error
    mockServer.SetFailNext(true)
    
    // Add response delay
    mockServer.SetResponseTime(100 * time.Millisecond)
    
    // Reset rate limits
    mockServer.ResetRateLimits()
}
```

### Mock Data

The server provides default mock data:

```go
// Default runs
ms.runs["12345"] = &MockRun{
    ID:             12345,
    Status:         "DONE",
    Repository:     "test/repo",
    RepositoryName: "test/repo",
    Title:          "Test Run",
    RunType:        "run",
    Source:         "main",
    Target:         "feature/test",
    CreatedAt:      time.Now().Add(-1 * time.Hour),
    UpdatedAt:      time.Now().Add(-30 * time.Minute),
    PrURL:          "https://github.com/test/repo/pull/1",
}
```

## Test Patterns

### Configuration Testing

```go
func TestConfigCommands(t *testing.T) {
    homeDir := SetupTestConfig(t)
    env := map[string]string{
        "HOME":            homeDir,
        "XDG_CONFIG_HOME": filepath.Join(homeDir, ".config"),
    }
    
    t.Run("set and get API key", func(t *testing.T) {
        // Set API key
        result := RunCommandWithEnv(t, env, "config", "set", "api-key", "TEST_KEY")
        AssertSuccess(t, result)
        
        // Get API key
        result = RunCommandWithEnv(t, env, "config", "get", "api-key")
        AssertSuccess(t, result)
        AssertContains(t, result.Stdout, "TEST")
    })
}
```

### Authentication Testing

```go
func TestAuthCommands(t *testing.T) {
    env, mockServer := SetupTestEnv(t)
    defer mockServer.Close()
    
    t.Run("verify with valid key", func(t *testing.T) {
        result := RunCommandWithEnv(t, env, "auth", "verify")
        AssertSuccess(t, result)
        AssertContains(t, result.Stdout, "valid")
    })
    
    t.Run("without API key", func(t *testing.T) {
        delete(env, "REPOBIRD_API_KEY")
        result := RunCommandWithEnv(t, env, "auth", "verify")
        AssertFailure(t, result)
        AssertContains(t, result.Stderr, "no API key")
    })
}
```

### Run Command Testing

```go
func TestRunCommand(t *testing.T) {
    env, mockServer := SetupTestEnv(t)
    defer mockServer.Close()
    
    tmpDir := t.TempDir()
    taskFile := CreateTestFile(t, tmpDir, "task.json", `{
        "prompt": "Test task",
        "repository": "test/repo",
        "source": "main",
        "target": "feature/test",
        "runType": "run",
        "title": "Test Run"
    }`)
    
    t.Run("with --dry-run", func(t *testing.T) {
        result := RunCommandWithEnv(t, env, "run", taskFile, "--dry-run")
        AssertSuccess(t, result)
        AssertContains(t, result.Stdout, "Validation successful")
    })
}
```

### Error Testing

```go
func TestErrorHandling(t *testing.T) {
    t.Run("invalid command", func(t *testing.T) {
        result := RunCommand(t, "invalidcommand")
        AssertFailure(t, result)
        AssertContains(t, result.Stderr, "unknown command")
    })
    
    t.Run("server error", func(t *testing.T) {
        env, mockServer := SetupTestEnv(t)
        defer mockServer.Close()
        
        mockServer.SetFailNext(true)
        result := RunCommandWithEnv(t, env, "auth", "verify")
        AssertFailure(t, result)
    })
}
```

### Performance Testing

```go
func TestPerformance(t *testing.T) {
    t.Run("version command speed", func(t *testing.T) {
        result := RunCommand(t, "version")
        AssertSuccess(t, result)
        
        if result.Duration > 1*time.Second {
            t.Errorf("Command took too long: %v", result.Duration)
        }
    })
}
```

## Safety Features

### 1. Dry Run Enforcement

All run and bulk commands MUST use `--dry-run`:

```go
// ✅ CORRECT - Always use --dry-run
result := RunCommandWithEnv(t, env, "run", taskFile, "--dry-run")
result := RunCommandWithEnv(t, env, "bulk", bulkFile, "--dry-run")

// ❌ WRONG - Never omit --dry-run
result := RunCommandWithEnv(t, env, "run", taskFile)  // DANGEROUS!
```

### 2. Environment Isolation

Tests use isolated temporary directories:

```go
func SetupTestConfig(t *testing.T) string {
    tmpDir := t.TempDir()  // Auto-cleaned after test
    configDir := filepath.Join(tmpDir, ".repobird")
    cacheDir := filepath.Join(tmpDir, ".config", "repobird", "cache")
    
    os.MkdirAll(configDir, 0755)
    os.MkdirAll(cacheDir, 0755)
    
    return tmpDir
}
```

### 3. Mock API Server

Never use production URLs:

```go
// ✅ CORRECT - Use mock server
env := map[string]string{
    "REPOBIRD_API_URL": mockServer.URL,  // http://127.0.0.1:xxxxx
    "REPOBIRD_API_KEY": "TEST_KEY",
}

// ❌ WRONG - Never use production
env := map[string]string{
    "REPOBIRD_API_URL": "https://api.repobird.ai",  // DANGEROUS!
}
```

### 4. Test Mode Flag

Set safety flags in environment:

```go
env := map[string]string{
    "REPOBIRD_TEST_MODE": "true",  // Safety flag
    "NO_COLOR":           "true",  // Easier output parsing
}
```

## Troubleshooting

### Common Issues

#### Tests Timeout

```bash
# Increase timeout
go test -tags=integration -timeout 5m ./test/integration

# Check for hanging commands
go test -tags=integration -v ./test/integration 2>&1 | grep -i "running"
```

#### Binary Build Fails

```bash
# Clean and rebuild
rm -f /tmp/repobird-test
go mod tidy
go build ./cmd/repobird

# Check build errors
go build -v ./cmd/repobird 2>&1
```

#### Mock Server Port Conflicts

```go
// The mock server uses random ports via httptest.NewServer()
// If issues persist, check for process leaks:
lsof -i :8080-9000 | grep repobird
```

#### Golden File Mismatches

```bash
# Update golden files
UPDATE_GOLDEN=1 go test -tags=integration ./test/integration -run TestGoldenFiles

# View differences
diff test/integration/testdata/golden/help.txt <(./build/repobird help)
```

#### Race Conditions

```bash
# Run with race detector
go test -tags=integration -race ./test/integration

# Check specific test
go test -tags=integration -race -run TestConcurrent ./test/integration
```

### Debug Techniques

#### Verbose Output

```go
func TestDebug(t *testing.T) {
    result := RunCommand(t, "status", "--debug")
    
    // Print full output for debugging
    t.Logf("Exit Code: %d", result.ExitCode)
    t.Logf("Stdout:\n%s", result.Stdout)
    t.Logf("Stderr:\n%s", result.Stderr)
    t.Logf("Duration: %v", result.Duration)
}
```

#### Capture Environment

```go
func TestEnvironment(t *testing.T) {
    // Log current environment
    for _, env := range os.Environ() {
        if strings.HasPrefix(env, "REPOBIRD") {
            t.Logf("Env: %s", env)
        }
    }
}
```

#### Mock Server Logs

```go
func (ms *MockServer) handler(w http.ResponseWriter, r *http.Request) {
    // Add debug logging
    log.Printf("Mock Server: %s %s", r.Method, r.URL.Path)
    log.Printf("Headers: %v", r.Header)
    
    // ... handle request
}
```

## Best Practices

### 1. Test Organization

- Group related tests using subtests
- Use descriptive test names
- Keep tests focused on single functionality
- Separate test data from test logic

### 2. Resource Management

```go
func TestWithResources(t *testing.T) {
    // Always cleanup resources
    env, mockServer := SetupTestEnv(t)
    defer mockServer.Close()
    
    tmpDir := t.TempDir()  // Auto-cleaned
    
    // Use t.Cleanup for additional cleanup
    t.Cleanup(func() {
        // Additional cleanup if needed
    })
}
```

### 3. Assertion Messages

```go
// Provide context in assertions
if result.ExitCode != 0 {
    t.Errorf("Command failed with exit code %d\nArgs: %v\nStderr: %s",
        result.ExitCode, args, result.Stderr)
}
```

### 4. Parallel Testing

```go
func TestParallel(t *testing.T) {
    t.Parallel()  // Enable parallel execution
    
    tests := []struct{...}
    
    for _, tt := range tests {
        tt := tt  // Capture range variable
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()  // Parallel subtests
            // ... test logic
        })
    }
}
```

### 5. Skip Conditions

```go
func TestConditional(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    if os.Getenv("CI") == "" {
        t.Skip("Skipping CI-only test")
    }
}
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Integration Tests

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  integration:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v3
    
    - uses: actions/setup-go@v4
      with:
        go-version: '1.20'
    
    - name: Build CLI
      run: make build
    
    - name: Run Integration Tests
      run: make test-integration
      env:
        REPOBIRD_TEST_MODE: true
    
    - name: Upload Coverage
      if: always()
      run: |
        make coverage-integration
        bash <(curl -s https://codecov.io/bash) -f coverage-integration.out
```

### Local CI Simulation

```bash
# Run full CI pipeline locally
make ci

# Run integration tests as in CI
REPOBIRD_TEST_MODE=true make test-integration

# Generate all reports
make test-all coverage-integration
```

## Test Coverage

### Measuring Coverage

```bash
# Generate coverage report
go test -tags=integration -coverprofile=coverage.out ./test/integration

# View coverage in browser
go tool cover -html=coverage.out

# Coverage summary
go tool cover -func=coverage.out | grep total
```

### Coverage Goals

- **Command Coverage**: All CLI commands should have at least one test
- **Error Paths**: Test both success and failure scenarios
- **Edge Cases**: Test boundary conditions and invalid inputs
- **Integration Points**: Test interaction between components

## Future Enhancements

### Planned Improvements

1. **PTY Testing**: Add pseudo-terminal tests for TUI interaction
2. **Interrupt Handling**: Test Ctrl+C behavior
3. **Concurrent Execution**: Test parallel command execution
4. **Fuzz Testing**: Add fuzzing for input parsing
5. **Cross-Platform**: Expand Windows and macOS testing
6. **Performance Benchmarks**: Add benchmark suite
7. **Load Testing**: Test behavior under high load
8. **Network Simulation**: Test timeout and retry behavior

### Contributing

When adding new integration tests:

1. Follow existing patterns and conventions
2. Ensure tests are idempotent
3. Add appropriate safety checks
4. Document any special requirements
5. Update this guide if adding new patterns

## Quick Reference

### Essential Commands

```bash
# Run all integration tests
make test-integration

# Run specific test
go test -tags=integration -v ./test/integration -run TestName

# Update golden files
UPDATE_GOLDEN=1 go test -tags=integration ./test/integration

# Debug failing test
go test -tags=integration -v -run TestName ./test/integration

# Check race conditions
go test -tags=integration -race ./test/integration
```

### Key Files

- `cli_test.go` - Main test implementations
- `mock_server.go` - Mock API server
- `helpers.go` - Test utilities
- `testdata/` - Test data files
- `Makefile` - Test targets

### Safety Checklist

- [ ] All run/bulk commands use `--dry-run`
- [ ] Tests use mock server URL
- [ ] Temporary directories for config/cache
- [ ] No production API keys
- [ ] Build tags prevent accidental execution
- [ ] Resources properly cleaned up

## Conclusion

The integration test suite provides comprehensive coverage of the RepoBird CLI's functionality while maintaining safety and isolation. By following the patterns and practices outlined in this guide, you can confidently add new tests and ensure the CLI remains reliable and bug-free.

For questions or improvements to the testing infrastructure, please refer to the main [Testing Guide](./testing-guide.md) or open an issue in the repository.

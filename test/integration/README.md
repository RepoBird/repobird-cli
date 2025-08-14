# Integration Tests for RepoBird CLI

This directory contains integration tests that execute the actual CLI binary to verify end-to-end functionality.

## Overview

Integration tests ensure that the CLI works correctly from a user's perspective by:
- Building and executing the actual binary
- Testing command-line argument parsing
- Verifying output formatting
- Testing environment variable handling
- Ensuring proper error messages and exit codes

## Safety Features

⚠️ **All integration tests are designed to be safe and isolated:**
- Use mock API server (never calls production)
- All `run` and `bulk run` commands use `--dry-run` flag
- Isolated configuration directories (temp dirs)
- Custom cache locations to avoid pollution
- `REPOBIRD_TEST_MODE` environment variable set

## Running Tests

### Run Integration Tests Only
```bash
# Using make
make test-integration

# Direct go test command
go test -tags=integration -v ./test/integration/...

# Run specific test
go test -tags=integration -v ./test/integration -run TestVersionCommand
```

### Run All Tests (Unit + Integration)
```bash
make test-all
```

### Update Golden Files
```bash
# Update golden files when output changes
go test -tags=integration ./test/integration -update
```

### Generate Coverage Report
```bash
make coverage-integration
```

## Test Structure

```
test/integration/
├── cli_test.go          # Main integration test suite
├── mock_server.go       # Mock API server implementation
├── helpers.go           # Test helper functions
├── testdata/
│   ├── task.json        # Valid task file
│   ├── task.yaml        # Valid YAML task
│   ├── task.md          # Valid Markdown task
│   ├── invalid.json     # Invalid JSON for error testing
│   ├── bulk_config.json # Bulk run configuration
│   └── golden/          # Expected output files
│       ├── version.txt  # Expected version output
│       ├── help.txt     # Expected help output
│       ├── status.json  # Expected JSON status
│       └── status.yaml  # Expected YAML status
└── README.md            # This file
```

## Test Categories

### 1. Basic Commands
- `version` - Version information
- `help` - Help text
- `completion` - Shell completions
- `docs` - Documentation generation

### 2. Configuration
- Setting API keys
- Getting configuration values
- Listing all configuration
- Deleting configuration items
- Config file path

### 3. Authentication
- API key verification
- User info retrieval
- Missing key handling

### 4. Run Commands (--dry-run only)
- JSON task files
- YAML task files
- Markdown task files
- Invalid file handling
- Missing file errors

### 5. Status Commands
- List all runs
- Get specific run
- JSON output format
- YAML output format
- Repository filtering

### 6. Bulk Operations (--dry-run only)
- Bulk run submission
- Batch status checking
- Error handling

### 7. Environment Variables
- `REPOBIRD_API_KEY`
- `REPOBIRD_API_URL`
- `REPOBIRD_DEBUG`
- `REPOBIRD_TIMEOUT`

### 8. Error Handling
- Invalid commands
- Missing arguments
- Rate limiting
- Network errors
- Server errors

## Writing New Tests

### Basic Test Template
```go
func TestNewFeature(t *testing.T) {
    // Setup test environment
    env, mockServer := SetupTestEnv(t)
    defer mockServer.Close()
    
    // Run command
    result := RunCommandWithEnv(t, env, "command", "arg1", "arg2")
    
    // Assert results
    AssertSuccess(t, result)
    AssertContains(t, result.Stdout, "expected output")
}
```

### Table-Driven Test Template
```go
func TestMultipleScenarios(t *testing.T) {
    tests := []struct {
        name     string
        args     []string
        wantExit int
        contains string
    }{
        {
            name:     "scenario 1",
            args:     []string{"cmd", "arg"},
            wantExit: 0,
            contains: "success",
        },
        // Add more test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := RunCommand(t, tt.args...)
            AssertExitCode(t, result, tt.wantExit)
            AssertContains(t, result.Stdout, tt.contains)
        })
    }
}
```

## Mock Server

The mock server (`mock_server.go`) simulates the RepoBird API:

### Endpoints
- `GET /api/v1/auth/verify` - Verify API key
- `GET /api/v1/users/me` - Get user info
- `POST /api/v1/runs` - Create run (checks for dry-run)
- `GET /api/v1/runs` - List runs
- `GET /api/v1/runs/{id}` - Get specific run
- `POST /api/v1/bulk/runs` - Create bulk runs
- `GET /api/v1/bulk/{id}` - Get bulk status

### Features
- Rate limiting simulation
- Error injection
- Response delay configuration
- API key validation

## Helper Functions

### Building Binary
```go
binary := BuildBinary(t)  // Builds once, cached for all tests
```

### Running Commands
```go
// Simple command
result := RunCommand(t, "version")

// With environment variables
env := map[string]string{"REPOBIRD_API_KEY": "test"}
result := RunCommandWithEnv(t, env, "status")
```

### Assertions
```go
AssertSuccess(t, result)                    // Exit code 0
AssertFailure(t, result)                    // Exit code != 0
AssertExitCode(t, result, 1)               // Specific exit code
AssertContains(t, result.Stdout, "text")   // Output contains text
AssertNotContains(t, result.Stderr, "err") // Output doesn't contain
```

### Golden Files
```go
// Compare with golden file
CompareGolden(t, actual, "testdata/golden/output.txt", false)

// Update golden file
CompareGolden(t, actual, "testdata/golden/output.txt", true)
```

## CI/CD Integration

Integration tests are excluded from normal test runs but should be included in CI:

```yaml
# GitHub Actions example
- name: Run Unit Tests
  run: make test

- name: Run Integration Tests
  run: make test-integration

- name: Run All Tests
  run: make test-all
```

## Troubleshooting

### Tests Timeout
- Increase timeout in Makefile: `-timeout 5m`
- Check for hanging commands or infinite loops

### Binary Build Fails
- Ensure Go modules are up to date: `go mod tidy`
- Check for compilation errors: `go build ./cmd/repobird`

### Mock Server Issues
- Verify port is not in use
- Check mock server logs for errors

### Golden File Mismatches
- Run with `-update` flag to regenerate
- Check for environment-specific output
- Normalize dynamic values (timestamps, versions)

## Best Practices

1. **Always use --dry-run** for run/bulk commands
2. **Isolate test environment** with temp directories
3. **Clean up resources** in defer statements
4. **Use table-driven tests** for similar scenarios
5. **Normalize output** before golden file comparison
6. **Test both success and failure** cases
7. **Mock external dependencies** (API, filesystem)
8. **Keep tests fast** (< 30 seconds total)

## Future Enhancements

- [ ] Add TUI interaction tests with PTY emulation
- [ ] Test interrupt handling (Ctrl+C)
- [ ] Add performance benchmarks
- [ ] Test concurrent command execution
- [ ] Add fuzz testing for input parsing
- [ ] Test shell completion scripts
- [ ] Add cross-platform tests (Windows, macOS)
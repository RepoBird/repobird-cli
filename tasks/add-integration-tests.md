# Task: Add Comprehensive Integration Tests for RepoBird CLI

## Overview
Implement a complete integration test suite that exercises the actual CLI binary with real command executions, while ensuring safety through mock APIs and --dry-run flags.

## Objectives
- Test all CLI commands through actual binary execution
- Ensure no interference with production API servers
- Validate user-facing behavior and output formatting
- Catch issues with command wiring, flag parsing, and environment handling
- Maintain fast test execution with proper isolation

## Safety Requirements
⚠️ **CRITICAL**: All integration tests MUST follow these safety rules:
1. **NEVER** make real API calls to production (api.repobird.ai)
2. **ALWAYS** use `--dry-run` flag for `run` and `bulk run` commands
3. **ALWAYS** use mock API server (localhost) for API interactions
4. **ALWAYS** use temporary directories for config/cache
5. **NEVER** use real API keys in tests

## Test Categories

### 1. Basic Commands (No API Required)
```bash
# Test these commands that don't need API
repobird version
repobird help
repobird help [command]
repobird completion bash/zsh/fish/powershell
repobird docs
```

### 2. Configuration Commands
```bash
# Test with isolated config directory
repobird config set api-key TEST_KEY
repobird config get api-key
repobird config list
repobird config delete api-key
repobird config reset
repobird config path
```

### 3. Authentication Commands (Mock API)
```bash
# Test with mock server
repobird auth verify
repobird auth info
```

### 4. Run Commands (--dry-run ONLY)
```bash
# MUST use --dry-run to prevent agent creation
repobird run task.json --dry-run
repobird run task.yaml --dry-run
repobird run task.md --dry-run
repobird run invalid.json --dry-run  # Error case
```

### 5. Status Commands (Mock API)
```bash
# Query mock server for status
repobird status
repobird status RUN_ID
repobird status --output json
repobird status --output yaml
repobird status --repo owner/repo
```

### 6. Bulk Commands (--dry-run ONLY)
```bash
# MUST use --dry-run for bulk runs
repobird bulk run config.json --dry-run
repobird bulk status BATCH_ID  # Mock API
repobird bulk cancel BATCH_ID  # Mock API
```

### 7. Environment Variable Tests
```bash
REPOBIRD_API_KEY=test_key repobird status
REPOBIRD_API_URL=http://localhost:8080 repobird status
REPOBIRD_DEBUG=true repobird version
REPOBIRD_TIMEOUT=5s repobird status
```

### 8. Error Cases
- Missing API key
- Invalid command syntax
- Malformed JSON/YAML files
- Network timeouts (simulated)
- Rate limiting (simulated)
- Invalid run IDs

## Implementation Plan

### Directory Structure
```
/test/integration/
├── cli_test.go          # Main test suite (with build tag)
├── mock_server.go       # Mock API implementation
├── helpers.go           # Test utilities
├── testdata/
│   ├── task.json        # Valid task file
│   ├── task.yaml        # Valid YAML task
│   ├── task.md          # Valid Markdown task
│   ├── invalid.json     # Malformed JSON
│   ├── bulk_config.json # Bulk configuration
│   └── golden/          # Expected outputs
│       ├── version.txt
│       ├── help.txt
│       ├── status.json
│       └── status.yaml
└── README.md            # How to run integration tests
```

### Mock API Server Endpoints
The mock server should implement:
- `GET /api/v1/auth/verify` - Verify API key
- `GET /api/v1/users/me` - User info
- `POST /api/v1/runs` - Create run (return mock ID)
- `GET /api/v1/runs` - List runs
- `GET /api/v1/runs/{id}` - Get run status
- `POST /api/v1/bulk/runs` - Create bulk runs
- `GET /api/v1/bulk/{id}` - Get bulk status

### Test Helper Functions
```go
// Build the CLI binary once
func buildBinary(t *testing.T) string

// Execute command and capture output
func runCommand(t *testing.T, args ...string) (stdout, stderr string, exitCode int)

// Execute with environment variables
func runCommandWithEnv(t *testing.T, env map[string]string, args ...string) (stdout, stderr string, exitCode int)

// Create temporary config directory
func setupTestConfig(t *testing.T) string

// Start mock API server
func startMockServer(t *testing.T) *httptest.Server

// Compare output with golden file
func compareGolden(t *testing.T, actual, goldenPath string)
```

### Build Tags for Isolation
Use build tags to exclude from normal test runs:
```go
//go:build integration
// +build integration

package integration
```

### Makefile Targets
```makefile
# Regular tests (no integration)
test:
	go test ./... -v

# Integration tests only
integration-test:
	go test -tags=integration ./test/integration -v

# All tests
test-all: test integration-test

# Integration with coverage
integration-coverage:
	go test -tags=integration -coverprofile=integration.out ./test/integration
```

## Test Examples

### Example 1: Version Command
```go
func TestVersionCommand(t *testing.T) {
    out, _, code := runCommand(t, "version")
    
    assert.Equal(t, 0, code)
    assert.Contains(t, out, "RepoBird CLI")
    assert.Contains(t, out, "Version:")
}
```

### Example 2: Config Commands
```go
func TestConfigCommands(t *testing.T) {
    configDir := setupTestConfig(t)
    defer os.RemoveAll(configDir)
    
    // Set API key
    _, _, code := runCommandWithEnv(t, 
        map[string]string{"HOME": configDir},
        "config", "set", "api-key", "TEST_KEY")
    assert.Equal(t, 0, code)
    
    // Get API key
    out, _, code := runCommandWithEnv(t,
        map[string]string{"HOME": configDir},
        "config", "get", "api-key")
    assert.Equal(t, 0, code)
    assert.Contains(t, out, "TEST_KEY")
}
```

### Example 3: Run Command with --dry-run
```go
func TestRunCommandDryRun(t *testing.T) {
    server := startMockServer(t)
    defer server.Close()
    
    env := map[string]string{
        "REPOBIRD_API_URL": server.URL,
        "REPOBIRD_API_KEY": "TEST_KEY",
    }
    
    // CRITICAL: Always use --dry-run
    out, _, code := runCommandWithEnv(t, env,
        "run", "testdata/task.json", "--dry-run")
    
    assert.Equal(t, 0, code)
    assert.Contains(t, out, "Dry run mode")
    assert.Contains(t, out, "Would create run")
}
```

### Example 4: Status with Mock API
```go
func TestStatusCommand(t *testing.T) {
    server := startMockServer(t)
    defer server.Close()
    
    env := map[string]string{
        "REPOBIRD_API_URL": server.URL,
        "REPOBIRD_API_KEY": "TEST_KEY",
    }
    
    out, _, code := runCommandWithEnv(t, env, "status")
    
    assert.Equal(t, 0, code)
    assert.Contains(t, out, "ID")
    assert.Contains(t, out, "Status")
    assert.Contains(t, out, "Repository")
}
```

## Testing Workflow

### 1. Initial Setup
```bash
# Create directory structure
mkdir -p test/integration/testdata/golden

# Build the binary for testing
go build -o test/integration/repobird ./cmd/repobird
```

### 2. Run Integration Tests
```bash
# Run only integration tests
make integration-test

# Run with verbose output
go test -tags=integration -v ./test/integration

# Run specific test
go test -tags=integration -v ./test/integration -run TestVersionCommand
```

### 3. Update Golden Files
```bash
# Run with update flag to regenerate golden files
go test -tags=integration ./test/integration -update
```

## Success Criteria
- [ ] All CLI commands have at least one integration test
- [ ] Tests run in < 30 seconds total
- [ ] No real API calls to production servers
- [ ] All run/bulk commands use --dry-run
- [ ] Tests are excluded from normal `make test`
- [ ] Mock server handles all necessary endpoints
- [ ] Golden files validate complex outputs
- [ ] Environment variable overrides work correctly
- [ ] Error cases are properly tested
- [ ] CI/CD pipeline runs integration tests

## Notes
- Integration tests complement but don't replace unit tests
- Focus on user-facing behavior and command integration
- Keep tests fast by reusing binary build and mock server
- Use table-driven tests for similar command variations
- Document any flaky tests and mitigation strategies

## References
- [Writing integration tests for Go CLI applications](https://lucapette.me/writing/writing-integration-tests-for-a-go-cli-application/)
- [CLI testing in Go](https://bitfieldconsulting.com/posts/cli-testing)
- [testscript package](https://github.com/rogpeppe/go-internal/tree/master/testscript)
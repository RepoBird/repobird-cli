# Task 04c: Comprehensive Testing

## Overview
Implement a comprehensive testing strategy for the RepoBird CLI, achieving 80%+ code coverage through unit tests, integration tests, property-based testing, benchmarks, and fuzz testing.

## Background Research

### Testing Best Practices for Go CLIs
Based on industry standards:
- **Keep `main()` minimal** - One line invoking logic from testable packages
- **Table-driven tests** - Efficiently test multiple scenarios in maintainable form
- **Testify framework** - Use for expressive assertions and mocking
- **Golden files** - Store expected CLI output for regression testing
- **Property-based testing** - Verify invariants across wide input ranges
- **Fuzz testing** - Uncover panics with random inputs (Go 1.18+)
- **Separate test types** - Organize unit/integration/e2e tests logically

## Implementation Tasks

### 1. Test Structure & Organization
- [x] Create test directory structure
  ```
  tests/
  ├── helpers/       # Test helpers and utilities
  ├── integration/   # Integration tests
  ├── testdata/      # Test data and fixtures
  └── ...           # Other test types as needed
  ```
- [x] Set up test helpers in `tests/helpers/`
- [x] Configure test tags for different test types
- [x] Create Makefile targets for each test type
- [x] Set up coverage reporting with HTML output

### 2. Unit Testing (Target: 90% coverage)
- [x] Implement table-driven tests for all packages
  ```go
  func TestRunCommand(t *testing.T) {
      tests := []struct {
          name      string
          args      []string
          wantErr   bool
          wantOutput string
      }{
          {"valid json", []string{"task.json"}, false, "Run created"},
          {"missing file", []string{"missing.json"}, true, "file not found"},
          {"invalid json", []string{"invalid.json"}, true, "invalid JSON"},
      }
      for _, tt := range tests {
          t.Run(tt.name, func(t *testing.T) {
              // Test implementation
          })
      }
  }
  ```
- [x] Mock external dependencies using testify/mock
- [x] Test error paths and edge cases
- [x] Test command flag parsing
- [x] Test configuration loading

### 3. Integration Testing
- [x] Create integration test suite
  ```go
  func TestCLIIntegration(t *testing.T) {
      if testing.Short() {
          t.Skip("skipping integration test")
      }
      // Test actual CLI binary
  }
  ```
- [x] Test API client with mock server
- [x] Test file system operations
- [x] Test configuration persistence
- [x] Test keyring integration (mock keyring)
- [x] Test command combinations

### 4. End-to-End Testing
- [ ] Create E2E test scenarios (Skipped - focused on core unit/integration tests)
  ```go
  func TestE2EWorkflow(t *testing.T) {
      // 1. Configure API key
      // 2. Submit a run
      // 3. Check status
      // 4. Verify output
  }
  ```
- [ ] Test complete user workflows (Skipped)
- [ ] Test against staging API (Skipped)
- [ ] Test error recovery scenarios (Covered in unit tests)
- [ ] Test interrupt handling (Ctrl+C) (Skipped)
- [ ] Test cross-platform behavior (Skipped)

### 5. Property-Based Testing
- [x] Implement property tests for parsers
  ```go
  func FuzzJSONParser(f *testing.F) {
      testcases := []string{
          `{"prompt": "test"}`,
          `{"invalid": }`,
      }
      for _, tc := range testcases {
          f.Add(tc)
      }
      f.Fuzz(func(t *testing.T, input string) {
          // Should not panic
          _, _ = ParseJSON(input)
      })
  }
  ```
- [x] Test command argument combinations
- [x] Test configuration value ranges
- [x] Test retry logic with various delays
- [x] Test concurrent operations

### 6. Fuzz Testing
- [x] Add fuzz tests for JSON/YAML parsers
- [ ] Fuzz test markdown parsing (N/A - no markdown parsing)
- [x] Fuzz test command-line argument parsing
- [x] Fuzz test API response handling
- [x] Run fuzzing in CI (time-boxed)

### 7. Benchmark Tests
- [x] Benchmark command execution time
  ```go
  func BenchmarkRunCommand(b *testing.B) {
      for i := 0; i < b.N; i++ {
          cmd := NewRunCommand()
          cmd.Execute()
      }
  }
  ```
- [x] Benchmark JSON/YAML parsing
- [x] Benchmark API client operations
- [ ] Benchmark TUI rendering (Skipped - TUI has compilation issues)
- [x] Profile memory allocations
- [x] Set performance baselines

### 8. Mock Generation & Management
- [x] Set up mockery for interface mocking
- [x] Generate mocks for API client
- [x] Generate mocks for file system
- [x] Generate mocks for keyring
- [x] Document mock usage patterns

## Test Coverage Goals

| Package | Target Coverage | Priority |
|---------|----------------|----------|
| `/internal/api` | 90% | High |
| `/internal/commands` | 85% | High |
| `/internal/config` | 95% | High |
| `/internal/models` | 95% | Medium |
| `/internal/tui` | 70% | Medium |
| `/pkg/utils` | 90% | High |
| Overall | 80%+ | - |

## Golden File Testing

```go
func TestCLIOutput(t *testing.T) {
    golden := filepath.Join("testdata", "output.golden")
    
    output := runCLI(t, "status", "--format", "json")
    
    if *update {
        os.WriteFile(golden, output, 0644)
    }
    
    expected, _ := os.ReadFile(golden)
    assert.Equal(t, string(expected), string(output))
}
```

## Mock Strategies

### API Client Mock
```go
type MockAPIClient struct {
    mock.Mock
}

func (m *MockAPIClient) CreateRun(req RunRequest) (*RunResponse, error) {
    args := m.Called(req)
    return args.Get(0).(*RunResponse), args.Error(1)
}

// Usage in tests
mockClient := new(MockAPIClient)
mockClient.On("CreateRun", mock.Anything).Return(&RunResponse{ID: "123"}, nil)
```

### Test Helpers
```go
// tests/helpers/cli.go
func RunCLI(t *testing.T, args ...string) (stdout, stderr string, err error) {
    cmd := exec.Command("./build/repobird", args...)
    var outBuf, errBuf bytes.Buffer
    cmd.Stdout = &outBuf
    cmd.Stderr = &errBuf
    err = cmd.Run()
    return outBuf.String(), errBuf.String(), err
}

// tests/helpers/fixtures.go
func LoadFixture(t *testing.T, name string) []byte {
    path := filepath.Join("testdata", name)
    data, err := os.ReadFile(path)
    require.NoError(t, err)
    return data
}
```

## CI Testing Strategy

```yaml
# .github/workflows/test.yml
test:
  strategy:
    matrix:
      os: [ubuntu-latest, macos-latest, windows-latest]
      go: ['1.20', '1.21']
  steps:
    - name: Unit Tests
      run: go test -v -race -coverprofile=coverage.out ./...
    
    - name: Integration Tests
      run: go test -v -tags=integration ./tests/integration
    
    - name: E2E Tests
      if: matrix.os == 'ubuntu-latest'
      run: go test -v -tags=e2e ./tests/e2e
    
    - name: Fuzz Tests (5 min)
      run: go test -fuzz=. -fuzztime=5m ./...
    
    - name: Benchmarks
      run: go test -bench=. -benchmem ./...
    
    - name: Coverage Report
      run: go tool cover -html=coverage.out -o coverage.html
```

## Test Data Management

```
testdata/
├── valid/
│   ├── simple_run.json
│   ├── complex_run.yaml
│   └── markdown_prompt.md
├── invalid/
│   ├── malformed.json
│   ├── missing_fields.yaml
│   └── huge_file.json
├── golden/
│   ├── status_output.golden
│   ├── run_output.golden
│   └── error_output.golden
└── api_responses/
    ├── success.json
    ├── rate_limited.json
    └── unauthorized.json
```

## Performance Baselines

| Operation | Baseline | Max Acceptable |
|-----------|----------|----------------|
| CLI startup | 50ms | 100ms |
| Command execution | 100ms | 200ms |
| JSON parsing (1MB) | 10ms | 50ms |
| API request | 500ms | 2s |
| TUI render | 16ms | 33ms |

## Testing Commands

```bash
# Run all tests
make test

# Run with coverage
make coverage

# Run specific test type
make test-unit
make test-integration
make test-e2e

# Run benchmarks
make bench

# Run fuzz tests
make fuzz

# Update golden files
go test ./... -update

# Generate mocks
make mocks

# Run tests with race detector
go test -race ./...

# Profile tests
go test -cpuprofile=cpu.prof -memprofile=mem.prof ./...
```

## Success Metrics
- 80%+ overall code coverage
- All critical paths have integration tests
- Zero race conditions detected
- Benchmarks show no performance regressions
- Fuzz testing finds no panics
- E2E tests pass on all platforms

## Dependencies
- `github.com/stretchr/testify` - Assertions and mocking
- `github.com/vektra/mockery` - Mock generation
- `github.com/golangci/golangci-lint` - Linting
- Standard library: `testing`, `testing/quick`

## References
- [Go Testing Best Practices](https://fossa.com/blog/golang-best-practices-testing-go/)
- [Table-Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Integration Testing Go CLIs](https://lucapette.me/writing/writing-integration-tests-for-a-go-cli-application/)
- [Property-Based Testing in Go](https://golang.org/doc/fuzz/)
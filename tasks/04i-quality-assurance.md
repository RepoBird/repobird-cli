# Task 04i: Quality Assurance

## Overview
Implement comprehensive quality assurance processes for RepoBird CLI, including automated linting, security scanning, code formatting, dependency management, and enforced quality gates.

## Background Research

### Quality Assurance Best Practices
Based on industry standards for Go:
- **golangci-lint** - Comprehensive linting with multiple analyzers
- **gosec** - Static security analysis
- **gofmt/goimports** - Enforced code formatting
- **Dependency scanning** - CVE monitoring and updates
- **License compliance** - Ensure compatible dependencies
- **Pre-commit hooks** - Local quality enforcement
- **Code review process** - Standardized checklists
- **Automated gates** - CI/CD quality requirements

## Implementation Tasks

### 1. golangci-lint Configuration
- [ ] Create `.golangci.yml`:
  ```yaml
  run:
    timeout: 5m
    issues-exit-code: 1
    tests: true
    skip-dirs:
      - vendor
      - testdata
      - build
    skip-files:
      - ".*\\.pb\\.go$"
      - ".*\\.gen\\.go$"
  
  output:
    format: colored-line-number
    print-issued-lines: true
    print-linter-name: true
    uniq-by-line: true
    sort-results: true
  
  linters:
    enable:
      # Enabled by default
      - govet
      - errcheck
      - staticcheck
      - unused
      - gosimple
      - ineffassign
      - typecheck
      # Additional linters
      - bodyclose
      - contextcheck
      - cyclop
      - deadcode
      - depguard
      - dogsled
      - dupl
      - durationcheck
      - errname
      - errorlint
      - exhaustive
      - exportloopref
      - forbidigo
      - gci
      - gochecknoinits
      - gocognit
      - goconst
      - gocritic
      - gocyclo
      - godot
      - godox
      - gofmt
      - gofumpt
      - goheader
      - goimports
      - gomnd
      - gomoddirectives
      - gomodguard
      - goprintffuncname
      - gosec
      - ifshort
      - importas
      - lll
      - makezero
      - misspell
      - nakedret
      - nestif
      - nilerr
      - nilnil
      - noctx
      - nolintlint
      - prealloc
      - predeclared
      - promlinter
      - revive
      - rowserrcheck
      - sqlclosecheck
      - structcheck
      - stylecheck
      - thelper
      - tparallel
      - unconvert
      - unparam
      - varcheck
      - wastedassign
      - whitespace
  
  linters-settings:
    cyclop:
      max-complexity: 15
    dupl:
      threshold: 150
    errcheck:
      check-type-assertions: true
      check-blank: true
    exhaustive:
      default-signifies-exhaustive: true
    forbidigo:
      forbid:
        - ^print.*$
        - 'fmt\.Print.*'
    gci:
      sections:
        - standard
        - default
        - prefix(github.com/repobird/repobird-cli)
    gocognit:
      min-complexity: 20
    goconst:
      min-len: 3
      min-occurrences: 3
    gocritic:
      enabled-tags:
        - diagnostic
        - experimental
        - opinionated
        - performance
        - style
    gocyclo:
      min-complexity: 15
    godot:
      scope: declarations
      capital: true
    gofmt:
      simplify: true
    goimports:
      local-prefixes: github.com/repobird/repobird-cli
    gomnd:
      checks:
        - argument
        - case
        - condition
        - operation
        - return
        - assign
    govet:
      check-shadowing: true
      enable-all: true
    lll:
      line-length: 120
    misspell:
      locale: US
    nakedret:
      max-func-lines: 30
    nestif:
      min-complexity: 4
    prealloc:
      simple: true
      range-loops: true
      for-loops: true
    revive:
      confidence: 0.8
      severity: warning
      rules:
        - name: blank-imports
        - name: context-as-argument
        - name: context-keys-type
        - name: dot-imports
        - name: error-return
        - name: error-strings
        - name: error-naming
        - name: exported
        - name: if-return
        - name: increment-decrement
        - name: var-naming
        - name: var-declaration
        - name: range
        - name: receiver-naming
        - name: time-naming
        - name: unexported-return
        - name: indent-error-flow
        - name: errorf
        - name: empty-block
        - name: superfluous-else
        - name: unused-parameter
        - name: unreachable-code
        - name: redefines-builtin-id
    unparam:
      check-exported: false
    unused:
      check-exported: false
  
  issues:
    exclude-rules:
      - path: _test\.go
        linters:
          - dupl
          - gomnd
          - goconst
      - path: cmd/
        linters:
          - forbidigo  # Allow fmt.Print in CLI commands
      - text: "weak cryptographic primitive"
        linters:
          - gosec
    max-issues-per-linter: 0
    max-same-issues: 0
    new: false
    fix: true
  ```
- [ ] Add linting to Makefile
- [ ] Configure IDE integration
- [ ] Set up custom rules
- [ ] Document linting exceptions

### 2. Security Scanning (gosec)
- [ ] Configure `.gosec.yml`:
  ```yaml
  global:
    audit: enabled
    nosec: false
    show-ignored: false
    confidence: medium
    severity: medium
    cap-warnings: -1
    exclude-generated: true
  
  rules:
    - G101  # Hardcoded credentials
    - G102  # Bind to all interfaces
    - G103  # Use of unsafe block
    - G104  # Audit errors not checked
    - G106  # Audit SSH
    - G107  # URL provided to HTTP request
    - G108  # Profiling endpoint
    - G109  # Integer overflow
    - G110  # Potential DoS vulnerability
    - G201  # SQL string formatting
    - G202  # SQL string concatenation
    - G203  # Use of unescaped data
    - G204  # Audit command execution
    - G301  # Poor file permissions
    - G302  # Poor file permissions (chmod)
    - G303  # Creating temp file using predictable path
    - G304  # File path provided as taint
    - G305  # File path traversal
    - G306  # Poor file permissions (WriteFile)
    - G307  # Deferring unsafe method
    - G401  # Use of weak crypto (DES)
    - G402  # Use of weak crypto (RC4)
    - G403  # Use of weak crypto (RSA key < 2048)
    - G404  # Use of weak random
    - G501  # Blacklisted imports (crypto/md5)
    - G502  # Blacklisted imports (crypto/des)
    - G503  # Blacklisted imports (crypto/rc4)
    - G504  # Blacklisted imports (net/http/cgi)
    - G505  # Blacklisted imports (crypto/sha1)
    - G601  # Implicit memory aliasing
  ```
- [ ] Add security scanning to CI
- [ ] Create security policy
- [ ] Set up vulnerability reporting
- [ ] Document security exceptions

### 3. Code Formatting
- [ ] Configure gofmt/goimports:
  ```makefile
  .PHONY: fmt
  fmt:
      @echo "Running gofmt..."
      @gofmt -s -w .
      @echo "Running goimports..."
      @goimports -w -local github.com/repobird/repobird-cli .
  
  .PHONY: fmt-check
  fmt-check:
      @echo "Checking formatting..."
      @test -z "$$(gofmt -s -l . | tee /dev/stderr)"
      @test -z "$$(goimports -l . | tee /dev/stderr)"
  ```
- [ ] Add gofumpt for stricter formatting
- [ ] Configure editor formatting
- [ ] Create format fixing script
- [ ] Document formatting rules

### 4. Dependency Management
- [ ] Set up dependency scanning:
  ```yaml
  # .github/workflows/dependency-check.yml
  name: Dependency Check
  
  on:
    schedule:
      - cron: '0 0 * * *'  # Daily
    pull_request:
      paths:
        - 'go.mod'
        - 'go.sum'
  
  jobs:
    scan:
      runs-on: ubuntu-latest
      steps:
        - uses: actions/checkout@v4
        
        - name: Run go mod verify
          run: go mod verify
        
        - name: Run go mod tidy check
          run: |
            go mod tidy
            git diff --exit-code go.mod go.sum
        
        - name: Run Nancy vulnerability scan
          run: |
            go list -json -m all | nancy sleuth
        
        - name: Run Trivy scan
          uses: aquasecurity/trivy-action@master
          with:
            scan-type: 'fs'
            scan-ref: '.'
            format: 'sarif'
            output: 'trivy-results.sarif'
        
        - name: Upload Trivy results
          uses: github/codeql-action/upload-sarif@v2
          with:
            sarif_file: 'trivy-results.sarif'
  ```
- [ ] Configure Dependabot
- [ ] Set up Renovate bot
- [ ] Create update policy
- [ ] Document dependency process

### 5. License Compliance
- [ ] Implement license checking:
  ```bash
  # Install go-licenses
  go install github.com/google/go-licenses@latest
  
  # Check licenses
  go-licenses check ./cmd/repobird
  
  # Save license report
  go-licenses save ./cmd/repobird --save_path=./licenses
  ```
- [ ] Create allowed licenses list:
  ```yaml
  # .license-policy.yml
  allowed:
    - MIT
    - Apache-2.0
    - BSD-3-Clause
    - BSD-2-Clause
    - ISC
    - MPL-2.0
  
  forbidden:
    - GPL-2.0
    - GPL-3.0
    - AGPL-3.0
    - LGPL-2.1
    - LGPL-3.0
  
  exceptions:
    - package: github.com/some/package
      reason: "Required for core functionality"
  ```
- [ ] Add license checking to CI
- [ ] Generate license attribution file
- [ ] Document license policy

### 6. Pre-commit Hooks
- [ ] Create `.pre-commit-config.yaml`:
  ```yaml
  repos:
    - repo: local
      hooks:
        - id: go-fmt
          name: go fmt
          entry: gofmt -s -w
          language: system
          types: [go]
          pass_filenames: true
        
        - id: go-imports
          name: go imports
          entry: goimports -w -local github.com/repobird/repobird-cli
          language: system
          types: [go]
          pass_filenames: true
        
        - id: go-vet
          name: go vet
          entry: go vet
          language: system
          types: [go]
          pass_filenames: false
        
        - id: golangci-lint
          name: golangci-lint
          entry: golangci-lint run --fix
          language: system
          types: [go]
          pass_filenames: false
        
        - id: go-test
          name: go test
          entry: go test -short ./...
          language: system
          types: [go]
          pass_filenames: false
        
        - id: go-mod-tidy
          name: go mod tidy
          entry: go mod tidy
          language: system
          types: [go]
          pass_filenames: false
        
        - id: gosec
          name: gosec
          entry: gosec -fmt json -out gosec-report.json ./...
          language: system
          types: [go]
          pass_filenames: false
  ```
- [ ] Create installation script
- [ ] Document hook usage
- [ ] Add bypass instructions
- [ ] Test hook performance

### 7. Code Review Process
- [ ] Create review checklist:
  ```markdown
  # Code Review Checklist
  
  ## Functionality
  - [ ] Code fulfills the requirements
  - [ ] Edge cases are handled
  - [ ] Error handling is appropriate
  - [ ] No regression in existing features
  
  ## Code Quality
  - [ ] Code is readable and self-documenting
  - [ ] Functions are focused and small
  - [ ] No code duplication
  - [ ] Naming is clear and consistent
  - [ ] Comments explain "why" not "what"
  
  ## Testing
  - [ ] Unit tests cover new code
  - [ ] Integration tests updated if needed
  - [ ] Tests are meaningful and maintainable
  - [ ] Edge cases are tested
  - [ ] Benchmarks added for performance-critical code
  
  ## Security
  - [ ] No hardcoded secrets or credentials
  - [ ] Input validation is proper
  - [ ] No SQL injection vulnerabilities
  - [ ] Proper authentication/authorization
  - [ ] Sensitive data is encrypted
  
  ## Performance
  - [ ] No obvious performance issues
  - [ ] Database queries are optimized
  - [ ] No memory leaks
  - [ ] Appropriate use of concurrency
  - [ ] Caching used where beneficial
  
  ## Documentation
  - [ ] Public APIs are documented
  - [ ] README updated if needed
  - [ ] Changelog entry added
  - [ ] Complex logic is explained
  - [ ] Examples provided for new features
  ```
- [ ] Set up PR templates
- [ ] Configure branch protection
- [ ] Create review guidelines
- [ ] Implement review metrics

### 8. Quality Gates
- [ ] Configure CI quality gates:
  ```yaml
  # .github/workflows/quality-gates.yml
  name: Quality Gates
  
  on:
    pull_request:
    push:
      branches: [main]
  
  jobs:
    quality:
      runs-on: ubuntu-latest
      steps:
        - uses: actions/checkout@v4
        
        - uses: actions/setup-go@v5
          with:
            go-version: '1.21'
        
        - name: Check formatting
          run: make fmt-check
        
        - name: Run linters
          run: |
            golangci-lint run --out-format=github-actions
        
        - name: Run security scan
          run: |
            gosec -fmt sarif -out gosec.sarif ./...
        
        - name: Run tests with coverage
          run: |
            go test -v -race -coverprofile=coverage.out ./...
            go tool cover -func=coverage.out
        
        - name: Check coverage threshold
          run: |
            COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
            echo "Coverage: $COVERAGE%"
            if (( $(echo "$COVERAGE < 80" | bc -l) )); then
              echo "Coverage is below 80%"
              exit 1
            fi
        
        - name: Check dependencies
          run: |
            go mod verify
            go mod tidy
            git diff --exit-code go.mod go.sum
        
        - name: License check
          run: |
            go-licenses check ./cmd/repobird
        
        - name: Build check
          run: |
            make build
  ```
- [ ] Set up SonarQube/CodeClimate
- [ ] Configure quality metrics
- [ ] Create quality dashboard
- [ ] Document quality standards

### 9. Static Analysis Tools
- [ ] Configure additional analyzers:
  ```makefile
  .PHONY: analyze
  analyze:
      @echo "Running static analysis..."
      @staticcheck ./...
      @ineffassign ./...
      @errcheck ./...
      @unconvert ./...
      @prealloc -set_exit_status ./...
  ```
- [ ] Add custom analyzers
- [ ] Configure analysis reports
- [ ] Set up trend tracking

### 10. Continuous Improvement
- [ ] Create metrics tracking:
  ```go
  // internal/metrics/quality.go
  type QualityMetrics struct {
      CoveragePercent   float64
      LinterIssues      int
      SecurityIssues    int
      TechnicalDebt     time.Duration
      CyclomaticComplexity int
      DuplicateLines    int
  }
  ```
- [ ] Set up quality dashboards
- [ ] Create improvement goals
- [ ] Regular quality reviews
- [ ] Document best practices

## Quality Standards

### Code Coverage Requirements
| Component | Minimum | Target |
|-----------|---------|--------|
| Core logic | 85% | 95% |
| API client | 80% | 90% |
| Commands | 75% | 85% |
| Utils | 90% | 95% |
| Overall | 80% | 90% |

### Complexity Limits
| Metric | Maximum |
|--------|---------|
| Cyclomatic complexity | 15 |
| Cognitive complexity | 20 |
| Function length | 50 lines |
| File length | 500 lines |
| Package size | 20 files |

### Performance Standards
| Check | Limit |
|-------|-------|
| Test execution | < 5 minutes |
| Linting | < 2 minutes |
| Build time | < 1 minute |
| PR checks | < 10 minutes |

## Success Metrics
- Zero critical security issues
- Code coverage > 80%
- All quality gates passing
- Linting issues < 10 per PR
- Review turnaround < 24 hours
- Build success rate > 99%

## Tools & Dependencies
- `golangci-lint` - Comprehensive linting
- `gosec` - Security scanning
- `go-licenses` - License checking
- `pre-commit` - Git hooks
- `nancy` - Vulnerability scanning
- `trivy` - Container scanning

## References
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [golangci-lint](https://golangci-lint.run/)
- [gosec](https://github.com/securego/gosec)
- [Go Best Practices](https://go.dev/doc/)
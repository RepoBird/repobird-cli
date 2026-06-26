# Local Build Scripts

These scripts replicate the GitHub Actions CI/CD workflows locally, allowing you to run the same build processes without GitHub Actions.

## Quick Start

```bash
# Make scripts executable (first time only)
chmod +x scripts/local-*.sh

# Run full CI pipeline
./scripts/local-build.sh ci

# Quick build and test
./scripts/local-build.sh quick
```

## Available Scripts

### 1. Main Build Script (`local-build.sh`)

The main entry point for all build operations:

```bash
./scripts/local-build.sh [command]
```

Commands:
- `ci` - Run full CI pipeline (tests, lint, build, coverage)
- `test` - Run tests only
- `build` - Build binary for current platform
- `release` - Build release artifacts
- `quick` - Quick build and test (no coverage)
- `lint` - Run linting checks only
- `clean` - Clean build artifacts
- `install` - Install binary to /usr/local/bin
- `help` - Show help message

### 2. CI Pipeline Script (`local-ci.sh`)

Replicates the full GitHub Actions CI workflow:

```bash
./scripts/local-ci.sh
```

This script:
- Downloads and verifies dependencies
- Checks code formatting
- Runs go vet
- Runs unit and integration tests
- Generates coverage reports
- Builds the binary
- Runs security scans (if tools installed)
- Checks for uncommitted go.mod changes

### 3. Test Runner Script (`local-test.sh`)

Focused testing with multiple Go version support:

```bash
./scripts/local-test.sh [options]
```

Options:
- `--go-version VERSION` - Test with specific Go version (can be used multiple times)
- `--skip-integration` - Skip integration tests
- `--coverage` - Generate coverage report
- `--verbose, -v` - Verbose output
- `--help, -h` - Show help message

Examples:
```bash
# Test with current Go version
./scripts/local-test.sh

# Test with specific Go version
./scripts/local-test.sh --go-version 1.22

# Test with multiple versions
./scripts/local-test.sh --go-version 1.22 --go-version 1.23

# Generate coverage report
./scripts/local-test.sh --coverage
```

### 4. Release Build Script (`local-release.sh`)

Build release artifacts without publishing:

```bash
./scripts/local-release.sh [options]
```

Options:
- `--version VERSION` - Set version for the build (default: from VERSION file)
- `--cross-compile` - Build for multiple platforms
- `--packages` - Build DEB and RPM packages
- `--sign` - Sign binaries (requires GPG key)
- `--help, -h` - Show help message

Examples:
```bash
# Build for current platform
./scripts/local-release.sh

# Build with specific version
./scripts/local-release.sh --version v1.2.3

# Cross-compile for all platforms
./scripts/local-release.sh --cross-compile

# Build everything including packages
./scripts/local-release.sh --cross-compile --packages
```

## Prerequisites

### Required
- Go 1.20 or later
- Git
- Make

### Optional Tools

For full CI functionality, install these optional tools:

```bash
# Linting
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Security scanning
go install github.com/securego/gosec/v2/cmd/gosec@latest

# Vulnerability checking
go install golang.org/x/vuln/cmd/govulncheck@latest

# Multiple Go versions (example for Go 1.22)
go install golang.org/dl/go1.22@latest
go1.22 download
```

For package building:
- **DEB packages**: `dpkg-deb` (install with `apt-get install dpkg-dev`)
- **RPM packages**: `rpmbuild` (install with `apt-get install rpm`)

## Comparison with GitHub Actions

| GitHub Actions Workflow | Local Script | Purpose |
|------------------------|--------------|---------|
| `.github/workflows/ci.yml` | `local-ci.sh` | Full CI pipeline |
| Test matrix (multiple Go versions) | `local-test.sh --go-version` | Multi-version testing |
| Build job | `local-build.sh build` | Binary compilation |
| Release workflow | `local-release.sh` | Release artifacts |
| Security scan | Part of `local-ci.sh` | Security checks |

## Environment Variables

The scripts use the same environment variables as the CI:

- `REPOBIRD_API_URL` - Set to `http://localhost:3000` for testing
- `XDG_CONFIG_HOME` - Set to `/tmp/test-config` for test isolation

## Typical Workflows

### Before Committing
```bash
# Run quick checks
./scripts/local-build.sh quick

# Or run full CI
./scripts/local-build.sh ci
```

### Testing Changes
```bash
# Test with coverage
./scripts/local-build.sh test --coverage

# Test with specific Go version
./scripts/local-test.sh --go-version 1.22
```

### Preparing a Release
```bash
# Build release artifacts
./scripts/local-release.sh --cross-compile

# With packages
./scripts/local-release.sh --cross-compile --packages
```

### Debugging Failed CI
```bash
# Run the exact same checks as CI
./scripts/local-ci.sh

# Run specific parts
./scripts/local-build.sh lint
./scripts/local-build.sh test
```

## Output Locations

- **Binaries**: `./build/repobird`
- **Coverage**: `./coverage.html`
- **Release artifacts**: `./dist/local-release/`
- **Test logs**: `/tmp/test-config/`

## Troubleshooting

### Permission Denied
```bash
chmod +x scripts/local-*.sh
```

### Missing Dependencies
```bash
go mod download
go mod verify
```

### Formatting Issues
```bash
make fmt
# or
gofmt -w .
```

### Test Failures
```bash
# Run with verbose output
./scripts/local-test.sh --verbose

# Check debug logs
tail -f /tmp/repobird_debug.log
```

## Notes

- These scripts are designed to closely mirror the GitHub Actions workflows
- They use the same Makefile targets where possible
- Color output is used for better readability in terminal
- Scripts exit on first error (`set -e`) for reliability
- All scripts support `--help` for usage information
#!/bin/bash
# local-ci.sh - Local CI build script that replicates GitHub Actions CI workflow
# This script runs the same steps as .github/workflows/ci.yml locally

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_step() {
    echo -e "${BLUE}==>${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

print_step "Running Local CI Build"
echo "OS: $OS"
echo "Architecture: $ARCH"
echo ""

# Check Go version
print_step "Checking Go version"
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo "Go version: $GO_VERSION"

# Set environment variables for testing
export REPOBIRD_API_URL="http://localhost:3000"
export XDG_CONFIG_HOME="/tmp/test-config"

# Clean previous build artifacts
print_step "Cleaning previous build artifacts"
rm -rf build/ coverage.* /tmp/test-config

# Download dependencies
print_step "Downloading dependencies"
go mod download

# Verify dependencies
print_step "Verifying dependencies"
go mod verify

# Check formatting
print_step "Checking code formatting"
if [ -n "$(gofmt -l .)" ]; then
    print_error "Go files are not formatted:"
    gofmt -l .
    print_warning "Run 'make fmt' or 'gofmt -w .' to fix formatting"
    exit 1
else
    print_success "Code formatting is correct"
fi

# Run go vet
print_step "Running go vet"
go vet ./...
print_success "go vet passed"

# Run tests
print_step "Running unit tests"
make test
print_success "Unit tests passed"

# Run integration tests (Linux/macOS only)
if [[ "$OS" != "windows" ]]; then
    print_step "Running integration tests"
    make test-integration
    print_success "Integration tests passed"
fi

# Generate coverage report
print_step "Generating coverage report"
make coverage
if [ -f coverage.out ]; then
    go tool cover -html=coverage.out -o coverage.html
    print_success "Coverage report generated: coverage.html"
    
    # Display coverage summary
    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
    echo "Total coverage: $COVERAGE"
fi

# Build binary
print_step "Building binary"
make build
print_success "Binary built successfully"

# Test binary execution
print_step "Testing binary execution"
./build/repobird version
./build/repobird --help > /dev/null
print_success "Binary execution test passed"

# Optional: Run linting if golangci-lint is installed
if command -v golangci-lint &> /dev/null; then
    print_step "Running golangci-lint"
    golangci-lint run --timeout=5m
    print_success "Linting passed"
else
    print_warning "golangci-lint not installed, skipping linting"
    echo "  Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
fi

# Optional: Run security checks if tools are installed
if command -v gosec &> /dev/null; then
    print_step "Running gosec security scanner"
    gosec -quiet ./...
    print_success "Security scan passed"
else
    print_warning "gosec not installed, skipping security scan"
    echo "  Install with: go install github.com/securego/gosec/v2/cmd/gosec@latest"
fi

if command -v govulncheck &> /dev/null; then
    print_step "Running vulnerability check"
    govulncheck ./...
    print_success "Vulnerability check passed"
else
    print_warning "govulncheck not installed, skipping vulnerability check"
    echo "  Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"
fi

# Check for uncommitted changes (useful for PR validation)
print_step "Checking for uncommitted changes"
go mod tidy
if git diff --exit-code go.mod go.sum; then
    print_success "No uncommitted changes in go.mod/go.sum"
else
    print_error "Uncommitted changes detected in go.mod/go.sum"
    print_warning "Run 'go mod tidy' and commit the changes"
    exit 1
fi

echo ""
print_success "Local CI build completed successfully!"
echo ""
echo "Build artifacts:"
echo "  - Binary: ./build/repobird"
echo "  - Coverage: ./coverage.html"
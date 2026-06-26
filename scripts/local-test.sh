#!/bin/bash
# local-test.sh - Local test runner with multiple Go version support
# Replicates the test matrix from GitHub Actions

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

# Parse command line arguments
GO_VERSIONS=()
RUN_INTEGRATION=true
RUN_COVERAGE=false
VERBOSE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --go-version)
            GO_VERSIONS+=("$2")
            shift 2
            ;;
        --skip-integration)
            RUN_INTEGRATION=false
            shift
            ;;
        --coverage)
            RUN_COVERAGE=true
            shift
            ;;
        --verbose|-v)
            VERBOSE=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --go-version VERSION   Test with specific Go version (can be used multiple times)"
            echo "  --skip-integration     Skip integration tests"
            echo "  --coverage            Generate coverage report"
            echo "  --verbose, -v         Verbose output"
            echo "  --help, -h            Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0                                      # Test with current Go version"
            echo "  $0 --go-version 1.22                   # Test with Go 1.22"
            echo "  $0 --go-version 1.22 --go-version 1.23 # Test with multiple versions"
            echo "  $0 --coverage                          # Run tests with coverage"
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# If no Go versions specified, use current version
if [ ${#GO_VERSIONS[@]} -eq 0 ]; then
    GO_VERSIONS=("current")
fi

# Set environment variables for testing
export REPOBIRD_API_URL="http://localhost:3000"
export XDG_CONFIG_HOME="/tmp/test-config"

# Function to run tests with a specific Go version
run_tests() {
    local version=$1
    
    if [ "$version" != "current" ]; then
        # Check if the specified Go version is installed
        if ! command -v go$version &> /dev/null; then
            print_warning "Go $version not installed, skipping"
            echo "  Install with: go install golang.org/dl/go${version}@latest && go${version} download"
            return 1
        fi
        GO_CMD="go$version"
    else
        GO_CMD="go"
        version=$(go version | awk '{print $3}' | sed 's/go//')
    fi
    
    echo ""
    print_step "Testing with Go $version"
    
    # Clean test cache
    rm -rf /tmp/test-config
    mkdir -p /tmp/test-config
    
    # Download dependencies
    print_step "Downloading dependencies"
    $GO_CMD mod download
    
    # Run unit tests
    print_step "Running unit tests"
    if [ "$VERBOSE" = true ]; then
        $GO_CMD test -v ./...
    else
        $GO_CMD test ./...
    fi
    print_success "Unit tests passed"
    
    # Run integration tests if not skipped
    if [ "$RUN_INTEGRATION" = true ]; then
        print_step "Running integration tests"
        if [ "$VERBOSE" = true ]; then
            $GO_CMD test -v -tags=integration ./tests/integration
        else
            $GO_CMD test -tags=integration ./tests/integration
        fi
        print_success "Integration tests passed"
    fi
    
    # Generate coverage if requested
    if [ "$RUN_COVERAGE" = true ]; then
        print_step "Generating coverage report"
        $GO_CMD test -coverprofile=coverage.out ./...
        $GO_CMD tool cover -html=coverage.out -o coverage-go${version}.html
        
        # Display coverage summary
        COVERAGE=$($GO_CMD tool cover -func=coverage.out | grep total | awk '{print $3}')
        print_success "Coverage report generated: coverage-go${version}.html"
        echo "Total coverage: $COVERAGE"
    fi
    
    return 0
}

# Main execution
print_step "Local Test Runner"
echo "Testing with Go versions: ${GO_VERSIONS[*]}"
echo "Integration tests: $([ "$RUN_INTEGRATION" = true ] && echo "enabled" || echo "disabled")"
echo "Coverage report: $([ "$RUN_COVERAGE" = true ] && echo "enabled" || echo "disabled")"

FAILED_VERSIONS=()

for version in "${GO_VERSIONS[@]}"; do
    if ! run_tests "$version"; then
        FAILED_VERSIONS+=("$version")
    fi
done

echo ""
if [ ${#FAILED_VERSIONS[@]} -eq 0 ]; then
    print_success "All tests passed successfully!"
else
    print_error "Tests failed for Go versions: ${FAILED_VERSIONS[*]}"
    exit 1
fi

# Run additional checks
echo ""
print_step "Running additional checks"

# Check formatting
print_step "Checking code formatting"
if [ -n "$(gofmt -l .)" ]; then
    print_error "Go files are not formatted:"
    gofmt -l .
    print_warning "Run 'make fmt' or 'gofmt -w .' to fix formatting"
else
    print_success "Code formatting is correct"
fi

# Run go vet
print_step "Running go vet"
go vet ./...
print_success "go vet passed"

echo ""
print_success "Test run completed!"
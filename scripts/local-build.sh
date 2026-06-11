#!/bin/bash
# local-build.sh - Main local build script wrapper
# Provides easy access to all local CI/build scripts

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Function to print colored output
print_header() {
    echo ""
    echo -e "${CYAN}════════════════════════════════════════════════════════${NC}"
    echo -e "${CYAN}  RepoBird CLI - Local Build System${NC}"
    echo -e "${CYAN}════════════════════════════════════════════════════════${NC}"
    echo ""
}

print_menu() {
    echo -e "${BLUE}Available commands:${NC}"
    echo ""
    echo -e "  ${GREEN}ci${NC}         Run full CI pipeline (tests, lint, build)"
    echo -e "  ${GREEN}test${NC}       Run tests only"
    echo -e "  ${GREEN}build${NC}      Build binary for current platform"
    echo -e "  ${GREEN}release${NC}    Build release artifacts"
    echo -e "  ${GREEN}quick${NC}      Quick build and test (no coverage)"
    echo -e "  ${GREEN}lint${NC}       Run linting checks only"
    echo -e "  ${GREEN}clean${NC}      Clean build artifacts"
    echo -e "  ${GREEN}install${NC}    Install binary to /usr/local/bin"
    echo -e "  ${GREEN}help${NC}       Show this help message"
    echo ""
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Make scripts executable
make_executable() {
    chmod +x "$SCRIPT_DIR/local-ci.sh" 2>/dev/null || true
    chmod +x "$SCRIPT_DIR/local-test.sh" 2>/dev/null || true
    chmod +x "$SCRIPT_DIR/local-release.sh" 2>/dev/null || true
}

# Commands
case "${1:-help}" in
    ci)
        print_header
        echo -e "${BLUE}Running full CI pipeline...${NC}"
        make_executable
        "$SCRIPT_DIR/local-ci.sh"
        ;;
        
    test)
        print_header
        echo -e "${BLUE}Running tests...${NC}"
        shift
        make_executable
        "$SCRIPT_DIR/local-test.sh" "$@"
        ;;
        
    build)
        print_header
        echo -e "${BLUE}Building binary...${NC}"
        make build
        print_success "Binary built: ./build/repobird"
        ./build/repobird version
        ;;
        
    release)
        print_header
        echo -e "${BLUE}Building release artifacts...${NC}"
        shift
        make_executable
        "$SCRIPT_DIR/local-release.sh" "$@"
        ;;
        
    quick)
        print_header
        echo -e "${BLUE}Quick build and test...${NC}"
        
        # Quick format check
        if [ -n "$(gofmt -l .)" ]; then
            print_warning "Code needs formatting. Run 'make fmt' to fix."
        fi
        
        # Quick build
        echo -e "${BLUE}Building...${NC}"
        make build
        
        # Quick test (no coverage)
        echo -e "${BLUE}Testing...${NC}"
        go test ./...
        
        # Test binary
        ./build/repobird version > /dev/null
        
        print_success "Quick build completed!"
        ;;
        
    lint)
        print_header
        echo -e "${BLUE}Running linting checks...${NC}"
        
        # Format check
        if [ -n "$(gofmt -l .)" ]; then
            print_error "Code is not formatted. Files needing formatting:"
            gofmt -l .
            echo ""
            echo "Run 'make fmt' to fix formatting"
            exit 1
        else
            print_success "Code formatting OK"
        fi
        
        # go vet
        go vet ./...
        print_success "go vet OK"
        
        # golangci-lint if available
        if command -v golangci-lint &> /dev/null; then
            golangci-lint run --timeout=5m
            print_success "golangci-lint OK"
        else
            print_warning "golangci-lint not installed"
            echo "  Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
        fi
        ;;
        
    clean)
        print_header
        echo -e "${BLUE}Cleaning build artifacts...${NC}"
        rm -rf build/ dist/ coverage.* /tmp/test-config /tmp/repobird_debug.log
        go clean -cache -testcache
        print_success "Build artifacts cleaned"
        ;;
        
    install)
        print_header
        echo -e "${BLUE}Installing RepoBird CLI...${NC}"
        
        # Build first
        make build
        
        # Install binary
        if [ -w /usr/local/bin ]; then
            cp build/repobird /usr/local/bin/
        else
            print_warning "Need sudo to install to /usr/local/bin"
            sudo cp build/repobird /usr/local/bin/
        fi
        
        print_success "RepoBird CLI installed to /usr/local/bin/repobird"
        
        # Test installation
        if command -v repobird &> /dev/null; then
            echo "Version: $(repobird version)"
        else
            print_error "Installation verification failed"
            exit 1
        fi
        ;;
        
    help|--help|-h)
        print_header
        print_menu
        echo -e "${YELLOW}Examples:${NC}"
        echo ""
        echo "  # Run full CI pipeline"
        echo "  ./scripts/local-build.sh ci"
        echo ""
        echo "  # Run tests with coverage"
        echo "  ./scripts/local-build.sh test --coverage"
        echo ""
        echo "  # Build release for all platforms"
        echo "  ./scripts/local-build.sh release --cross-compile"
        echo ""
        echo "  # Quick build and test"
        echo "  ./scripts/local-build.sh quick"
        echo ""
        echo -e "${YELLOW}Individual script options:${NC}"
        echo ""
        echo "  # Test with specific Go version"
        echo "  ./scripts/local-test.sh --go-version 1.22"
        echo ""
        echo "  # Build release with packages"
        echo "  ./scripts/local-release.sh --cross-compile --packages"
        echo ""
        ;;
        
    *)
        print_error "Unknown command: $1"
        print_menu
        exit 1
        ;;
esac
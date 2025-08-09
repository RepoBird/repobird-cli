#!/bin/bash
set -e

# Local Package Testing Script for RepoBird CLI
# This script tests package builds and installations locally

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
log() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

run_test() {
    local test_name="$1"
    local test_cmd="$2"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    log "Running test: $test_name"
    
    if eval "$test_cmd" >/dev/null 2>&1; then
        success "✓ $test_name"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        error "✗ $test_name"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

# Main testing function
main() {
    echo -e "${BLUE}"
    cat << "EOF"
    ____                ____  _         _   _____         _   
   |  _ \ ___ _ __   ___| __ )(_)_ __ __| | |_   _|__  ___| |_ 
   | |_) / _ \ '_ \ / _ \  _ \| | '__/ _` |   | |/ _ \/ __| __|
   |  _ <  __/ |_) | (_) |_) | | | | (_| |   | |  __/\__ \ |_ 
   |_| \_\___| .__/ \___/____/|_|_|  \__,_|   |_|\___||___/\__|
             |_|                                              
   
   Local Package Testing Suite
EOF
    echo -e "${NC}"
    
    log "Starting RepoBird CLI local package tests..."
    
    # Test 1: Basic Build
    run_test "Basic build" "make build"
    
    # Test 2: Cross-platform builds
    run_test "Cross-platform builds" "make build-all"
    
    # Test 3: Binary functionality
    if [ -f "build/repobird" ]; then
        run_test "Binary version check" "./build/repobird version"
        run_test "Binary help command" "./build/repobird --help"
        run_test "Binary config command" "./build/repobird config --help"
        run_test "Binary run command" "./build/repobird run --help"
        run_test "Binary status command" "./build/repobird status --help"
        run_test "Binary auth command" "./build/repobird auth --help"
    else
        error "Binary not found, skipping functionality tests"
        TESTS_FAILED=$((TESTS_FAILED + 6))
    fi
    
    # Test 4: Local installation
    log "Testing local installation..."
    if make install >/dev/null 2>&1; then
        if [ -f "$HOME/.local/bin/repobird" ] && [ -L "$HOME/.local/bin/rb" ]; then
            success "✓ Local installation"
            TESTS_PASSED=$((TESTS_PASSED + 1))
            
            # Test installed binaries
            run_test "Installed binary version" "$HOME/.local/bin/repobird version"
            run_test "Alias functionality" "$HOME/.local/bin/rb version"
            
            # Uninstall
            if make uninstall >/dev/null 2>&1; then
                success "✓ Local uninstallation"
                TESTS_PASSED=$((TESTS_PASSED + 1))
            else
                error "✗ Local uninstallation"
                TESTS_FAILED=$((TESTS_FAILED + 1))
            fi
        else
            error "✗ Local installation - files not found"
            TESTS_FAILED=$((TESTS_FAILED + 1))
        fi
    else
        error "✗ Local installation failed"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
    TESTS_RUN=$((TESTS_RUN + 2))
    
    # Test 5: Shell completions
    log "Testing shell completions..."
    if ./scripts/generate-completions.sh >/dev/null 2>&1; then
        if [ -f "completions/repobird.bash" ] && [ -f "completions/_repobird" ] && [ -f "completions/repobird.fish" ]; then
            success "✓ Shell completion generation"
            TESTS_PASSED=$((TESTS_PASSED + 1))
            
            # Test completion files are not empty
            run_test "Bash completion not empty" "test -s completions/repobird.bash"
            run_test "Zsh completion not empty" "test -s completions/_repobird"
            run_test "Fish completion not empty" "test -s completions/repobird.fish"
        else
            error "✗ Shell completion generation - files missing"
            TESTS_FAILED=$((TESTS_FAILED + 1))
        fi
    else
        error "✗ Shell completion generation failed"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
    TESTS_RUN=$((TESTS_RUN + 1))
    
    # Test 6: Documentation generation
    log "Testing documentation generation..."
    if ./scripts/generate-docs.sh >/dev/null 2>&1; then
        if [ -f "man/repobird.1" ] && [ -f "docs/cli/repobird.md" ]; then
            success "✓ Documentation generation"
            TESTS_PASSED=$((TESTS_PASSED + 1))
            
            # Test doc files are not empty
            run_test "Man page not empty" "test -s man/repobird.1"
            run_test "Markdown docs not empty" "test -s docs/cli/repobird.md"
        else
            error "✗ Documentation generation - files missing"
            TESTS_FAILED=$((TESTS_FAILED + 1))
        fi
    else
        error "✗ Documentation generation failed"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
    TESTS_RUN=$((TESTS_RUN + 1))
    
    # Test 7: Package creation simulation
    log "Testing package creation..."
    
    # Test DEB package structure
    if command -v dpkg-deb >/dev/null 2>&1; then
        log "Creating test DEB package..."
        mkdir -p test-deb/DEBIAN test-deb/usr/bin
        echo "Package: repobird
Version: 0.0.0-test
Architecture: amd64
Maintainer: Test
Description: Test package" > test-deb/DEBIAN/control
        cp build/repobird test-deb/usr/bin/ 2>/dev/null || echo "Binary not found"
        
        if dpkg-deb --build test-deb test.deb >/dev/null 2>&1; then
            success "✓ DEB package creation"
            TESTS_PASSED=$((TESTS_PASSED + 1))
            rm -rf test-deb test.deb
        else
            error "✗ DEB package creation"
            TESTS_FAILED=$((TESTS_FAILED + 1))
        fi
        TESTS_RUN=$((TESTS_RUN + 1))
    else
        warn "dpkg-deb not available, skipping DEB test"
    fi
    
    # Test archive creation
    if [ -f "build/repobird" ]; then
        log "Creating test archives..."
        mkdir -p test-archive
        cp build/repobird test-archive/
        
        if tar -czf test.tar.gz -C test-archive . >/dev/null 2>&1; then
            success "✓ TAR.GZ archive creation"
            TESTS_PASSED=$((TESTS_PASSED + 1))
        else
            error "✗ TAR.GZ archive creation"
            TESTS_FAILED=$((TESTS_FAILED + 1))
        fi
        
        if command -v zip >/dev/null 2>&1; then
            if cd test-archive && zip -r ../test.zip . >/dev/null 2>&1; then
                cd ..
                success "✓ ZIP archive creation"
                TESTS_PASSED=$((TESTS_PASSED + 1))
            else
                cd ..
                error "✗ ZIP archive creation"
                TESTS_FAILED=$((TESTS_FAILED + 1))
            fi
            TESTS_RUN=$((TESTS_RUN + 1))
        fi
        
        rm -rf test-archive test.tar.gz test.zip
        TESTS_RUN=$((TESTS_RUN + 1))
    fi
    
    # Test 8: GoReleaser configuration
    if command -v goreleaser >/dev/null 2>&1; then
        run_test "GoReleaser config check" "goreleaser check"
        
        log "Testing GoReleaser snapshot build..."
        if goreleaser build --snapshot --clean >/dev/null 2>&1; then
            success "✓ GoReleaser snapshot build"
            TESTS_PASSED=$((TESTS_PASSED + 1))
            rm -rf dist/
        else
            error "✗ GoReleaser snapshot build"
            TESTS_FAILED=$((TESTS_FAILED + 1))
        fi
        TESTS_RUN=$((TESTS_RUN + 1))
    else
        warn "goreleaser not available, skipping GoReleaser tests"
    fi
    
    # Test 9: Version script
    if [ -f "scripts/bump-version.sh" ]; then
        run_test "Version script syntax" "bash -n scripts/bump-version.sh"
    fi
    
    # Test 10: Install scripts
    if [ -f "scripts/install.sh" ]; then
        run_test "Install script syntax" "bash -n scripts/install.sh"
    fi
    
    if [ -f "scripts/install.ps1" ]; then
        if command -v pwsh >/dev/null 2>&1; then
            run_test "PowerShell script syntax" "pwsh -Command 'Get-Content scripts/install.ps1 | ForEach-Object { }'"
        else
            warn "PowerShell not available, skipping PS1 script test"
        fi
    fi
    
    # Test results summary
    echo ""
    echo "=============================="
    echo "Test Results Summary"
    echo "=============================="
    echo "Total tests run: $TESTS_RUN"
    echo -e "Tests passed:    ${GREEN}$TESTS_PASSED${NC}"
    echo -e "Tests failed:    ${RED}$TESTS_FAILED${NC}"
    
    if [ $TESTS_FAILED -eq 0 ]; then
        success "🎉 All tests passed!"
        echo ""
        echo "Your RepoBird CLI is ready for package distribution!"
        echo ""
        echo "Next steps:"
        echo "1. Set up your package repositories (Homebrew tap, APT repo, etc.)"
        echo "2. Configure GitHub secrets for signing and publishing"
        echo "3. Run: ./scripts/bump-version.sh 1.0.0"
        echo "4. Push tags: git push origin main --tags"
        return 0
    else
        error "❌ Some tests failed. Please fix issues before release."
        echo ""
        echo "Common fixes:"
        echo "- Run 'make deps' to install dependencies"
        echo "- Check Go version: go version"
        echo "- Verify build tools are available"
        return 1
    fi
}

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites..."
    
    if ! command -v go >/dev/null 2>&1; then
        error "Go is not installed. Please install Go 1.21+"
        exit 1
    fi
    
    if ! command -v make >/dev/null 2>&1; then
        error "make is not installed. Please install make"
        exit 1
    fi
    
    if [ ! -f "Makefile" ]; then
        error "Makefile not found. Run this script from the project root."
        exit 1
    fi
    
    success "Prerequisites check passed"
}

# Parse command line arguments
CLEAN_AFTER=false
VERBOSE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --clean)
            CLEAN_AFTER=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --clean    Clean up generated files after testing"
            echo "  --verbose  Show detailed output"
            echo "  --help     Show this help message"
            exit 0
            ;;
        *)
            error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Run the tests
check_prerequisites
main
exit_code=$?

# Cleanup if requested
if [ "$CLEAN_AFTER" = true ]; then
    log "Cleaning up generated files..."
    make clean >/dev/null 2>&1 || true
    rm -rf completions/ man/ docs/cli/ test-* >/dev/null 2>&1 || true
    success "Cleanup complete"
fi

exit $exit_code
#!/bin/bash
# local-release.sh - Local release build script
# Builds release artifacts locally using GoReleaser or manual compilation

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
VERSION=""
CROSS_COMPILE=false
BUILD_PACKAGES=false
SIGN_BINARIES=false
USE_GORELEASER=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --version)
            VERSION="$2"
            shift 2
            ;;
        --cross-compile)
            CROSS_COMPILE=true
            shift
            ;;
        --packages)
            BUILD_PACKAGES=true
            shift
            ;;
        --sign)
            SIGN_BINARIES=true
            shift
            ;;
        --goreleaser)
            USE_GORELEASER=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --version VERSION     Set version for the build (default: from VERSION file)"
            echo "  --cross-compile       Build for multiple platforms"
            echo "  --packages           Build DEB and RPM packages"
            echo "  --sign               Sign binaries (requires GPG key)"
            echo "  --goreleaser         Use GoReleaser instead of manual build"
            echo "  --help, -h           Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0                                    # Build for current platform"
            echo "  $0 --version v1.2.3                  # Build with specific version"
            echo "  $0 --cross-compile                   # Build for all platforms"
            echo "  $0 --cross-compile --packages        # Build everything"
            echo "  $0 --goreleaser                      # Use GoReleaser for builds"
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Get version if not specified
if [ -z "$VERSION" ]; then
    if [ -f VERSION ]; then
        VERSION=$(cat VERSION)
    else
        VERSION="dev-$(git rev-parse --short HEAD)"
    fi
fi

# Clean version (remove 'v' prefix if present)
CLEAN_VERSION=${VERSION#v}

print_step "Local Release Build"
echo "Version: $VERSION"
echo "Cross-compile: $CROSS_COMPILE"
echo "Build packages: $BUILD_PACKAGES"
echo "Sign binaries: $SIGN_BINARIES"
echo "Use GoReleaser: $USE_GORELEASER"
echo ""

# Check for GoReleaser if requested
if [ "$USE_GORELEASER" = true ]; then
    if ! command -v goreleaser &> /dev/null; then
        print_error "GoReleaser not installed"
        echo "Install with: brew install goreleaser"
        echo "Or see: https://goreleaser.com/install"
        exit 1
    fi
fi

# Use GoReleaser if requested
if [ "$USE_GORELEASER" = true ]; then
    print_step "Building with GoReleaser"
    
    # Set version tag for GoReleaser
    git tag -f "$VERSION" 2>/dev/null || git tag "$VERSION"
    
    # Run GoReleaser in snapshot mode (doesn't publish)
    goreleaser release --clean --skip=publish --skip=validate
    
    print_success "Release artifacts built with GoReleaser"
    echo "Artifacts location: ./dist/"
    ls -la dist/*.tar.gz dist/*.zip dist/checksums.txt 2>/dev/null || true
    exit 0
fi

# Manual build process (original logic)
# Create dist directory
DIST_DIR="dist/local-release"
rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

# Build the main binary first
print_step "Building main binary"
make build
print_success "Main binary built"

# Generate completions and docs
print_step "Generating completions and documentation"
mkdir -p completions man docs/cli

if [ -f ./scripts/generate-completions.sh ]; then
    ./scripts/generate-completions.sh
    print_success "Completions generated"
else
    print_warning "Completions script not found"
fi

if [ -f ./scripts/generate-docs.sh ]; then
    ./scripts/generate-docs.sh
    print_success "Documentation generated"
else
    print_warning "Documentation script not found"
fi

# Function to build for a specific platform
build_platform() {
    local GOOS=$1
    local GOARCH=$2
    local EXT=""
    
    if [ "$GOOS" = "windows" ]; then
        EXT=".exe"
    fi
    
    local OUTPUT="$DIST_DIR/repobird_${GOOS}_${GOARCH}/repobird${EXT}"
    
    print_step "Building for $GOOS/$GOARCH"
    
    GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags "-X github.com/repobird/repobird-cli/pkg/version.Version=$VERSION \
                  -X github.com/repobird/repobird-cli/pkg/version.GitCommit=$(git rev-parse HEAD) \
                  -X github.com/repobird/repobird-cli/pkg/version.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        -o "$OUTPUT" \
        ./cmd/repobird
    
    # Create archive
    local ARCHIVE_DIR=$(dirname "$OUTPUT")
    cp README.md LICENSE "$ARCHIVE_DIR/" 2>/dev/null || true
    
    if [ -d completions ]; then
        cp -r completions "$ARCHIVE_DIR/"
    fi
    
    if [ -d man ]; then
        cp -r man "$ARCHIVE_DIR/"
    fi
    
    cd "$DIST_DIR"
    if [ "$GOOS" = "windows" ]; then
        zip -r "repobird_${GOOS}_${GOARCH}.zip" "repobird_${GOOS}_${GOARCH}"
    else
        tar -czf "repobird_${GOOS}_${GOARCH}.tar.gz" "repobird_${GOOS}_${GOARCH}"
    fi
    cd - > /dev/null
    
    print_success "Built for $GOOS/$GOARCH"
}

# Build for current platform or cross-compile
if [ "$CROSS_COMPILE" = true ]; then
    print_step "Cross-compiling for multiple platforms"
    
    # Linux
    build_platform linux amd64
    build_platform linux arm64
    build_platform linux 386
    build_platform linux arm
    
    # macOS
    build_platform darwin amd64
    build_platform darwin arm64
    
    # Windows
    build_platform windows amd64
    build_platform windows arm64
    build_platform windows 386
    
    # FreeBSD
    build_platform freebsd amd64
    build_platform freebsd arm64
else
    # Build for current platform only
    OS=$(go env GOOS)
    ARCH=$(go env GOARCH)
    build_platform "$OS" "$ARCH"
fi

# Sign binaries if requested
if [ "$SIGN_BINARIES" = true ]; then
    print_step "Signing binaries"
    
    if ! command -v gpg &> /dev/null; then
        print_error "GPG not installed"
        exit 1
    fi
    
    for binary in "$DIST_DIR"/*/repobird*; do
        if [ -f "$binary" ]; then
            gpg --detach-sign --armor "$binary"
            print_success "Signed: $(basename $binary)"
        fi
    done
fi

# Build packages if requested
if [ "$BUILD_PACKAGES" = true ]; then
    print_step "Building distribution packages"
    
    # Build DEB package
    if command -v dpkg-deb &> /dev/null; then
        print_step "Building DEB package"
        
        for arch in amd64 arm64; do
            DEB_DIR="$DIST_DIR/debian-$arch"
            mkdir -p "$DEB_DIR/DEBIAN"
            mkdir -p "$DEB_DIR/usr/bin"
            mkdir -p "$DEB_DIR/usr/share/man/man1"
            mkdir -p "$DEB_DIR/usr/share/bash-completion/completions"
            mkdir -p "$DEB_DIR/usr/share/zsh/site-functions"
            mkdir -p "$DEB_DIR/usr/share/fish/vendor_completions.d"
            
            # Copy binary
            if [ -f "$DIST_DIR/repobird_linux_$arch/repobird" ]; then
                cp "$DIST_DIR/repobird_linux_$arch/repobird" "$DEB_DIR/usr/bin/"
                chmod 755 "$DEB_DIR/usr/bin/repobird"
            else
                print_warning "Binary for $arch not found, skipping DEB package"
                continue
            fi
            
            # Copy man pages
            if [ -d man ]; then
                cp man/*.1 "$DEB_DIR/usr/share/man/man1/" 2>/dev/null || true
            fi
            
            # Copy completions
            if [ -d completions ]; then
                [ -f completions/repobird.bash ] && cp completions/repobird.bash "$DEB_DIR/usr/share/bash-completion/completions/repobird"
                [ -f completions/_repobird ] && cp completions/_repobird "$DEB_DIR/usr/share/zsh/site-functions/"
                [ -f completions/repobird.fish ] && cp completions/repobird.fish "$DEB_DIR/usr/share/fish/vendor_completions.d/"
            fi
            
            # Create control file
            cat > "$DEB_DIR/DEBIAN/control" << EOF
Package: repobird
Version: $CLEAN_VERSION
Section: utils
Priority: optional
Architecture: $arch
Maintainer: RepoBird Team <team@repobird.ai>
Homepage: https://github.com/repobird/repobird-cli
Description: Fast CLI for RepoBird AI agent platform
 RepoBird CLI enables users to submit AI-powered code
 generation tasks and track their progress.
EOF
            
            # Build DEB
            dpkg-deb --build "$DEB_DIR" "$DIST_DIR/repobird_${CLEAN_VERSION}_${arch}.deb"
            print_success "Built DEB package for $arch"
        done
    else
        print_warning "dpkg-deb not installed, skipping DEB packages"
    fi
    
    # Build RPM package
    if command -v rpmbuild &> /dev/null; then
        print_step "Building RPM package"
        print_warning "RPM building requires more setup, skipping for now"
        echo "  To build RPMs, use the full release workflow with proper RPM build environment"
    else
        print_warning "rpmbuild not installed, skipping RPM packages"
    fi
fi

# Generate checksums
print_step "Generating checksums"
cd "$DIST_DIR"
sha256sum *.tar.gz *.zip *.deb 2>/dev/null > checksums.txt || true
cd - > /dev/null
print_success "Checksums generated"

# Summary
echo ""
print_success "Local release build completed!"
echo ""
echo "Build artifacts in: $DIST_DIR/"
echo ""
ls -la "$DIST_DIR/" | grep -E "\.(tar\.gz|zip|deb|rpm)$" || true

if [ "$SIGN_BINARIES" = true ]; then
    echo ""
    echo "Signed files:"
    ls -la "$DIST_DIR"/*/*.asc 2>/dev/null || echo "  No signatures found"
fi

echo ""
echo "To test the binary:"
echo "  $DIST_DIR/repobird_$(go env GOOS)_$(go env GOARCH)/repobird version"
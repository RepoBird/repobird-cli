#!/bin/bash
# local-github-release.sh - Create and publish GitHub releases locally
# This script handles the complete release process without GitHub Actions

set -e

# Cleanup function
cleanup() {
    # GoReleaser handles its own cleanup, but remove any accidental artifacts
    if [ -d completions ] || [ -d man ]; then
        echo "Cleaning up any accidental artifacts..."
        rm -rf completions/ man/ 2>/dev/null || true
    fi
}

# Set trap to cleanup on exit
trap cleanup EXIT

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
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

print_info() {
    echo -e "${CYAN}ℹ${NC} $1"
}

# Check for required tools
check_requirements() {
    local missing_tools=()
    
    if ! command -v gh &> /dev/null; then
        missing_tools+=("gh (GitHub CLI)")
    fi
    
    if ! command -v goreleaser &> /dev/null; then
        missing_tools+=("goreleaser")
    fi
    
    if ! command -v git &> /dev/null; then
        missing_tools+=("git")
    fi
    
    if [ ${#missing_tools[@]} -gt 0 ]; then
        print_error "Missing required tools:"
        for tool in "${missing_tools[@]}"; do
            echo "  - $tool"
        done
        echo ""
        echo "Installation instructions:"
        echo "  gh: brew install gh (or see https://cli.github.com)"
        echo "  goreleaser: brew install goreleaser (or see https://goreleaser.com/install)"
        exit 1
    fi
}

# Parse command line arguments
VERSION=""
DRAFT=false
PRERELEASE=false
SKIP_PUBLISH=false
SKIP_VALIDATION=false
LOCAL_ONLY=false
SIGN_RELEASE=false
NOTES_FILE=""
OVERRIDE_VERSION=""
GIT_REMOTE="gh"  # Default to 'gh' for GitHub

show_help() {
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  --draft             Create as draft release"
    echo "  --prerelease        Mark as pre-release"
    echo "  --skip-publish      Build but don't publish to GitHub"
    echo "  --skip-validation   Skip pre-release validation"
    echo "  --local-only        Build locally without any GitHub interaction"
    echo "  --sign              Sign release with GPG"
    echo "  --notes FILE        Release notes file (default: auto-generate)"
    echo "  --version VERSION   Override VERSION file (not recommended)"
    echo "  --remote REMOTE     Git remote for GitHub (default: gh)"
    echo "  --help, -h          Show this help message"
    echo ""
    echo "Examples:"
    echo "  # Create a new release using VERSION file"
    echo "  $0"
    echo ""
    echo "  # Create a draft release"
    echo "  $0 --draft"
    echo ""
    echo "  # Build locally without publishing"
    echo "  $0 --local-only"
    echo ""
    echo "  # Create signed release with custom notes"
    echo "  $0 --sign --notes RELEASE_NOTES.md"
    echo ""
    echo "  # Override version (not recommended)"
    echo "  $0 --version v1.2.3-hotfix"
}

while [[ $# -gt 0 ]]; do
    case $1 in
        --version)
            OVERRIDE_VERSION="$2"
            print_warning "Overriding VERSION file with $2"
            shift 2
            ;;
        --draft)
            DRAFT=true
            shift
            ;;
        --prerelease)
            PRERELEASE=true
            shift
            ;;
        --skip-publish)
            SKIP_PUBLISH=true
            shift
            ;;
        --skip-validation)
            SKIP_VALIDATION=true
            shift
            ;;
        --local-only)
            LOCAL_ONLY=true
            SKIP_PUBLISH=true
            shift
            ;;
        --sign)
            SIGN_RELEASE=true
            shift
            ;;
        --notes)
            NOTES_FILE="$2"
            shift 2
            ;;
        --remote)
            GIT_REMOTE="$2"
            shift 2
            ;;
        --help|-h)
            show_help
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Get version from VERSION file or override
if [ -n "$OVERRIDE_VERSION" ]; then
    VERSION="$OVERRIDE_VERSION"
elif [ -f VERSION ]; then
    VERSION=$(cat VERSION)
    print_info "Using version from VERSION file: $VERSION"
else
    print_error "VERSION file not found and no version specified"
    echo "Please create a VERSION file with the version number (e.g., v1.2.3)"
    echo "Or use --version to override"
    exit 1
fi

# Ensure version starts with 'v'
if [[ ! "$VERSION" =~ ^v ]]; then
    VERSION="v$VERSION"
fi

print_step "RepoBird CLI Release Builder"
echo "Version: $VERSION"
echo "Draft: $DRAFT"
echo "Prerelease: $PRERELEASE"
echo "Local only: $LOCAL_ONLY"
echo "Git remote: $GIT_REMOTE"
echo ""

# Verify git remote exists
if ! git remote | grep -q "^${GIT_REMOTE}$"; then
    print_error "Git remote '$GIT_REMOTE' not found"
    echo "Available remotes:"
    git remote -v
    echo ""
    echo "Use --remote to specify the correct GitHub remote"
    exit 1
fi

# Check requirements
check_requirements

# Validation checks (unless skipped)
if [ "$SKIP_VALIDATION" = false ]; then
    print_step "Running validation checks"
    
    # Check for uncommitted changes
    if ! git diff --quiet || ! git diff --cached --quiet; then
        print_error "Uncommitted changes detected"
        echo "Please commit or stash your changes before releasing"
        exit 1
    fi
    
    # Check if on main branch
    CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
    if [ "$CURRENT_BRANCH" != "main" ] && [ "$CURRENT_BRANCH" != "master" ]; then
        print_warning "Not on main branch (current: $CURRENT_BRANCH)"
        read -p "Continue anyway? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
    
    # Run tests
    print_step "Running tests"
    make test
    print_success "Tests passed"
    
    # Check formatting
    if [ -n "$(gofmt -l .)" ]; then
        print_error "Code is not formatted"
        echo "Run 'make fmt' to fix"
        exit 1
    fi
    print_success "Code formatting OK"
fi

# Verify VERSION file matches
if [ -f VERSION ]; then
    FILE_VERSION=$(cat VERSION)
    # Normalize versions (add 'v' if missing)
    [[ ! "$FILE_VERSION" =~ ^v ]] && FILE_VERSION="v$FILE_VERSION"
    
    if [ "$FILE_VERSION" != "$VERSION" ]; then
        print_warning "VERSION file contains $FILE_VERSION but using $VERSION"
        read -p "Update VERSION file to match? (y/N) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            echo "$VERSION" > VERSION
            git add VERSION
            git commit -m "chore: bump version to $VERSION" || true
        fi
    fi
fi

# Create and push tag
print_step "Creating git tag"
git tag -a "$VERSION" -m "Release $VERSION"

if [ "$LOCAL_ONLY" = false ]; then
    print_step "Pushing tag to $GIT_REMOTE"
    git push "$GIT_REMOTE" "$VERSION"
    print_success "Tag pushed to $GIT_REMOTE"
fi

# GoReleaser will handle generating completions and docs via hooks
print_step "Preparing for release build"
print_info "GoReleaser will generate completions and man pages automatically"

# Build release with GoReleaser
print_step "Building release with GoReleaser"

GORELEASER_ARGS="release --clean"

if [ "$LOCAL_ONLY" = true ] || [ "$SKIP_PUBLISH" = true ]; then
    GORELEASER_ARGS="$GORELEASER_ARGS --skip=publish"
fi

if [ "$VERSION" != "$(git describe --tags --abbrev=0)" ]; then
    GORELEASER_ARGS="$GORELEASER_ARGS --skip=validate"
fi

# Set up environment for GoReleaser
export GITHUB_TOKEN="${GITHUB_TOKEN:-$(gh auth token)}"

# Sign release if requested
if [ "$SIGN_RELEASE" = true ]; then
    print_step "Setting up GPG signing"
    
    # Check for GPG key
    if ! gpg --list-secret-keys | grep -q "sec"; then
        print_error "No GPG secret key found"
        echo "Generate one with: gpg --full-generate-key"
        exit 1
    fi
    
    # Get GPG fingerprint
    GPG_FINGERPRINT=$(gpg --list-secret-keys --keyid-format LONG | grep sec | head -1 | awk '{print $2}' | cut -d'/' -f2)
    export GPG_FINGERPRINT
    print_info "Using GPG key: $GPG_FINGERPRINT"
fi

# Run GoReleaser
goreleaser $GORELEASER_ARGS

print_success "Release artifacts built"

# If not publishing, show local artifacts
if [ "$LOCAL_ONLY" = true ] || [ "$SKIP_PUBLISH" = true ]; then
    print_step "Local release artifacts"
    echo "Location: ./dist/"
    ls -la dist/*.tar.gz dist/*.zip dist/checksums.txt 2>/dev/null || true
    
    if [ "$LOCAL_ONLY" = false ]; then
        print_info "To publish this release, run:"
        echo "  gh release create $VERSION ./dist/*.tar.gz ./dist/*.zip ./dist/checksums.txt"
    fi
    exit 0
fi

# Create GitHub release if not done by GoReleaser
if [ "$SKIP_PUBLISH" = false ] && [ "$LOCAL_ONLY" = false ]; then
    print_step "Creating GitHub release"
    
    GH_ARGS=""
    
    if [ "$DRAFT" = true ]; then
        GH_ARGS="$GH_ARGS --draft"
    fi
    
    if [ "$PRERELEASE" = true ]; then
        GH_ARGS="$GH_ARGS --prerelease"
    fi
    
    if [ -n "$NOTES_FILE" ] && [ -f "$NOTES_FILE" ]; then
        GH_ARGS="$GH_ARGS --notes-file $NOTES_FILE"
    else
        # Generate release notes from commits
        print_step "Generating release notes"
        PREV_TAG=$(git describe --tags --abbrev=0 HEAD^ 2>/dev/null || echo "")
        if [ -n "$PREV_TAG" ]; then
            GH_ARGS="$GH_ARGS --generate-notes"
        else
            GH_ARGS="$GH_ARGS --notes 'Initial release'"
        fi
    fi
    
    # Check if release already exists
    if gh release view "$VERSION" &>/dev/null; then
        print_warning "Release $VERSION already exists"
        read -p "Delete and recreate? (y/N) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            gh release delete "$VERSION" --yes
        else
            print_info "Uploading assets to existing release"
            gh release upload "$VERSION" dist/*.tar.gz dist/*.zip dist/checksums.txt --clobber
            print_success "Assets uploaded"
            exit 0
        fi
    fi
    
    # Create release with assets
    gh release create "$VERSION" \
        dist/*.tar.gz \
        dist/*.zip \
        dist/checksums.txt \
        $GH_ARGS
    
    print_success "GitHub release created"
fi

# Summary
echo ""
print_success "Release $VERSION completed successfully!"
echo ""
echo "Next steps:"
# Extract GitHub repo from the correct remote
GITHUB_REPO=$(git remote get-url "$GIT_REMOTE" | sed 's/.*github.com[^:]*[:\/]\(.*\)\.git/\1/')
echo "1. View release: https://github.com/$GITHUB_REPO/releases/tag/$VERSION"
echo "2. Update package managers (run ./scripts/local-package-publish.sh)"
echo "3. Announce the release"

# Show how to install
echo ""
echo "Installation commands for users:"
echo "  # Homebrew"
echo "  brew tap repobird/tap"
echo "  brew install repobird"
echo ""
echo "  # Direct download"
echo "  curl -L https://github.com/repobird/repobird-cli/releases/download/$VERSION/repobird_linux_amd64.tar.gz | tar xz"
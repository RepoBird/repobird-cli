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

    # Clean up temporary changelog file
    rm -f /tmp/repobird-changelog-*.md 2>/dev/null || true
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

# Generate completions and man pages BEFORE GoReleaser (best practice)
print_step "Generating release artifacts"

# Build binary first
make build

# Generate completions
print_info "Generating shell completions..."
mkdir -p completions
./build/repobird completion bash > completions/repobird.bash
./build/repobird completion zsh > completions/_repobird
./build/repobird completion fish > completions/repobird.fish
./build/repobird completion powershell > completions/repobird.ps1
print_success "Completions generated"

# Generate man pages
print_info "Generating man pages..."
mkdir -p man
./build/repobird docs man man
print_success "Man pages generated"

# Build release with GoReleaser
print_step "Building release with GoReleaser"

# Always use --clean flag to ensure dist directory is cleaned
GORELEASER_ARGS="release --clean"

if [ "$LOCAL_ONLY" = true ] || [ "$SKIP_PUBLISH" = true ]; then
    GORELEASER_ARGS="$GORELEASER_ARGS --skip=publish"
fi

if [ "$VERSION" != "$(git describe --tags --abbrev=0 2>/dev/null)" ]; then
    GORELEASER_ARGS="$GORELEASER_ARGS --skip=validate"
fi

# Add custom notes file if provided
if [ -n "$NOTES_FILE" ] && [ -f "$NOTES_FILE" ]; then
    GORELEASER_ARGS="$GORELEASER_ARGS --release-notes=$NOTES_FILE"
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

# First run GoReleaser in snapshot mode to test the build
print_step "Testing build with GoReleaser (snapshot mode)"
goreleaser build --snapshot --clean
print_success "Build test successful"

# Generate changelog preview
print_step "Generating changelog preview"
CHANGELOG_FILE="/tmp/repobird-changelog-$VERSION.md"

# Get the previous tag to generate changelog from
PREV_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
if [ -z "$PREV_TAG" ]; then
    print_warning "No previous tag found, showing all commits"
    GIT_RANGE="HEAD"
else
    GIT_RANGE="$PREV_TAG..HEAD"
    print_info "Generating changelog from $PREV_TAG to HEAD"
fi

# Generate changelog with filtering
cat > "$CHANGELOG_FILE" << 'CHANGELOG_HEADER'
# Release Notes

## Features
CHANGELOG_HEADER

# Add features - clean up verbose commit messages
git log $GIT_RANGE --pretty=format:"%s" --no-merges | grep "^feat" | while IFS= read -r line; do
    # Extract just the commit title (remove verbose explanations after first sentence/clause)
    # Look for patterns like "The addition of", "This change", etc. and remove them
    clean_line=$(echo "$line" | sed 's/ The .*//' | sed 's/ This .*//' | sed 's/ It .*//' | sed 's/\. .*//')
    echo "- $clean_line"
done >> "$CHANGELOG_FILE"

# Check if any features were added
if ! grep -q "^- feat" "$CHANGELOG_FILE"; then
    echo "(none)" >> "$CHANGELOG_FILE"
fi

cat >> "$CHANGELOG_FILE" << 'CHANGELOG_FIXES'

## Bug Fixes
CHANGELOG_FIXES

# Add fixes - clean up verbose commit messages
git log $GIT_RANGE --pretty=format:"%s" --no-merges | grep "^fix" | while IFS= read -r line; do
    # Extract just the commit title
    clean_line=$(echo "$line" | sed 's/ The .*//' | sed 's/ This .*//' | sed 's/ It .*//' | sed 's/\. .*//')
    echo "- $clean_line"
done >> "$CHANGELOG_FILE"

# Check if any fixes were added
if ! grep -q "^- fix" "$CHANGELOG_FILE"; then
    echo "(none)" >> "$CHANGELOG_FILE"
fi

cat >> "$CHANGELOG_FILE" << 'CHANGELOG_BREAKING'

## Breaking Changes
CHANGELOG_BREAKING

# Add breaking changes
git log $GIT_RANGE --pretty=format:"%s" --no-merges | grep -i "BREAKING CHANGE" | while IFS= read -r line; do
    clean_line=$(echo "$line" | sed 's/ The .*//' | sed 's/ This .*//' | sed 's/ It .*//' | sed 's/\. .*//')
    echo "- $clean_line"
done >> "$CHANGELOG_FILE"

# Check if any breaking changes were added
if ! grep -q "BREAKING CHANGE" "$CHANGELOG_FILE"; then
    echo "(none)" >> "$CHANGELOG_FILE"
fi

cat >> "$CHANGELOG_FILE" << 'CHANGELOG_FOOTER'

---
**Note:** Internal changes (chore, docs, test, style, refactor, perf, build, ci) are excluded from this changelog.
CHANGELOG_FOOTER

# Show the changelog to user
echo ""
print_info "Generated changelog preview:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
cat "$CHANGELOG_FILE"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Ask user for confirmation
if [ "$LOCAL_ONLY" = false ] && [ "$SKIP_PUBLISH" = false ]; then
    echo "Options:"
    echo "  [Y] Accept and continue with release"
    echo "  [e] Edit changelog in \$EDITOR (${EDITOR:-nano})"
    echo "  [N] Cancel release"
    echo ""
    read -p "Proceed with this changelog? (Y/e/N) " -n 1 -r
    echo

    if [[ $REPLY =~ ^[Ee]$ ]]; then
        # Open changelog in editor
        ${EDITOR:-nano} "$CHANGELOG_FILE"

        echo ""
        print_info "Edited changelog:"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        cat "$CHANGELOG_FILE"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo ""

        read -p "Proceed with release? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_warning "Release cancelled by user"
            rm -f "$CHANGELOG_FILE"
            exit 0
        fi
    elif [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_warning "Release cancelled by user"
        rm -f "$CHANGELOG_FILE"
        exit 0
    fi

    # Use the custom changelog file for the release
    NOTES_FILE="$CHANGELOG_FILE"
    print_success "Changelog approved"
fi

# Create and push the tag after successful test build
if [ "$LOCAL_ONLY" = false ]; then
    print_step "Creating and pushing git tag"
    # Check if tag already exists
    if git rev-parse "$VERSION" >/dev/null 2>&1; then
        print_warning "Tag $VERSION already exists locally"
        read -p "Delete and recreate? (y/N) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            git tag -d "$VERSION"
            git push "$GIT_REMOTE" --delete "$VERSION" 2>/dev/null || true
        else
            print_info "Using existing tag"
        fi
    fi
    
    # Create the tag if it doesn't exist
    if ! git rev-parse "$VERSION" >/dev/null 2>&1; then
        git tag -a "$VERSION" -m "Release $VERSION"
        print_success "Tag created: $VERSION"
    fi
    
    # Push the tag
    git push "$GIT_REMOTE" "$VERSION"
    print_success "Tag pushed to $GIT_REMOTE"
fi

# Now run the actual release with GoReleaser
print_step "Creating GitHub release with GoReleaser"
goreleaser $GORELEASER_ARGS

print_success "Release completed successfully"

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

# GoReleaser handles the release creation, so we don't need gh release create
# Only upload additional assets if needed and release already exists
if [ "$SKIP_PUBLISH" = false ] && [ "$LOCAL_ONLY" = false ]; then
    # Check if we need to upload any additional assets
    if gh release view "$VERSION" &>/dev/null; then
        print_info "Release $VERSION created by GoReleaser"
        # GoReleaser should have already uploaded all assets
    else
        print_warning "Release $VERSION was not created (check GoReleaser output above)"
    fi
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
echo "  curl -L https://github.com/RepoBird/repobird-cli/releases/download/$VERSION/repobird-cli_linux_amd64.tar.gz | tar xz"
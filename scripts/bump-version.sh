#!/bin/bash
set -e

# Version bumping script for RepoBird CLI
# Usage: ./scripts/bump-version.sh [major|minor|patch|<version>]

CURRENT_VERSION=""
NEW_VERSION=""

# Function to get current version from git tags
get_current_version() {
    git fetch --tags 2>/dev/null || true
    CURRENT_VERSION=$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "0.0.0")
    echo "Current version: $CURRENT_VERSION"
}

# Function to increment version
increment_version() {
    local version=$1
    local bump_type=$2
    
    IFS='.' read -ra VERSION_PARTS <<< "$version"
    local major=${VERSION_PARTS[0]}
    local minor=${VERSION_PARTS[1]}
    local patch=${VERSION_PARTS[2]}
    
    case $bump_type in
        major)
            major=$((major + 1))
            minor=0
            patch=0
            ;;
        minor)
            minor=$((minor + 1))
            patch=0
            ;;
        patch)
            patch=$((patch + 1))
            ;;
        *)
            echo "Invalid bump type: $bump_type"
            exit 1
            ;;
    esac
    
    echo "${major}.${minor}.${patch}"
}

# Function to validate version format
validate_version() {
    local version=$1
    if [[ ! $version =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        echo "Invalid version format: $version"
        echo "Expected format: X.Y.Z (e.g., 1.0.0)"
        exit 1
    fi
}

# Main logic
main() {
    if [ $# -eq 0 ]; then
        echo "Usage: $0 [major|minor|patch|<version>]"
        echo ""
        echo "Examples:"
        echo "  $0 patch      # Increment patch version"
        echo "  $0 minor      # Increment minor version"
        echo "  $0 major      # Increment major version"
        echo "  $0 1.2.3      # Set specific version"
        exit 1
    fi
    
    get_current_version
    
    local input=$1
    
    if [[ $input =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        # Specific version provided
        NEW_VERSION=$input
    elif [[ $input =~ ^(major|minor|patch)$ ]]; then
        # Bump type provided
        NEW_VERSION=$(increment_version "$CURRENT_VERSION" "$input")
    else
        echo "Invalid input: $input"
        echo "Use 'major', 'minor', 'patch', or a specific version like '1.0.0'"
        exit 1
    fi
    
    validate_version "$NEW_VERSION"
    
    echo "Bumping version from $CURRENT_VERSION to $NEW_VERSION"
    
    # Check if version already exists
    if git tag | grep -q "^v$NEW_VERSION$"; then
        echo "❌ Version v$NEW_VERSION already exists!"
        exit 1
    fi
    
    # Update version in files
    echo "📝 Updating version in files..."
    
    # Update goreleaser config (if version is hardcoded anywhere)
    if grep -q "version:" .goreleaser.yml; then
        sed -i.bak "s/version: .*/version: 2/" .goreleaser.yml
        rm -f .goreleaser.yml.bak
    fi
    
    # Update package.json if it exists (for npm compatibility)
    if [ -f package.json ]; then
        sed -i.bak "s/\"version\": \".*\"/\"version\": \"$NEW_VERSION\"/" package.json
        rm -f package.json.bak
        echo "  ✓ Updated package.json"
    fi
    
    # Update Dockerfile if it exists
    if [ -f Dockerfile ]; then
        sed -i.bak "s/LABEL version=.*/LABEL version=\"$NEW_VERSION\"/" Dockerfile
        rm -f Dockerfile.bak
        echo "  ✓ Updated Dockerfile"
    fi
    
    # Update Snap configuration
    if [ -f snapcraft.yaml ]; then
        sed -i.bak "s/version: .*/version: '$NEW_VERSION'/" snapcraft.yaml
        rm -f snapcraft.yaml.bak
        echo "  ✓ Updated snapcraft.yaml"
    fi
    
    # Update Helm chart if it exists
    if [ -f chart/Chart.yaml ]; then
        sed -i.bak "s/version: .*/version: $NEW_VERSION/" chart/Chart.yaml
        sed -i.bak "s/appVersion: .*/appVersion: $NEW_VERSION/" chart/Chart.yaml
        rm -f chart/Chart.yaml.bak
        echo "  ✓ Updated chart/Chart.yaml"
    fi
    
    echo "✓ Version updated in configuration files"
    
    # Commit changes
    echo "📦 Committing version bump..."
    git add -A
    git commit -m "chore: bump version to $NEW_VERSION

- Update version across all package configurations
- Prepare for release v$NEW_VERSION" || true
    
    # Create tag
    echo "🏷️  Creating tag v$NEW_VERSION..."
    git tag -a "v$NEW_VERSION" -m "Release version $NEW_VERSION

$(./scripts/generate-changelog.sh v$CURRENT_VERSION..HEAD 2>/dev/null || echo "See CHANGELOG.md for details")"
    
    echo ""
    echo "🎉 Version bump complete!"
    echo ""
    echo "Next steps:"
    echo "1. Review the changes: git show v$NEW_VERSION"
    echo "2. Push the tag: git push origin main --tags"
    echo "3. The release workflow will automatically:"
    echo "   - Build binaries for all platforms"
    echo "   - Create GitHub release"
    echo "   - Update package managers"
    echo "   - Sign and publish packages"
    echo ""
    echo "Or to undo: git tag -d v$NEW_VERSION && git reset HEAD~1"
}

main "$@"
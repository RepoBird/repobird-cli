#!/bin/bash
set -e

# Build the binary if it doesn't exist
if [ ! -f "build/repobird" ]; then
    echo "Building repobird binary..."
    make build
fi

echo "Generating documentation..."

# Generate man pages
echo "  - man pages"
./build/repobird docs man

# Generate markdown docs
echo "  - markdown documentation"
./build/repobird docs markdown docs/cli

# Generate YAML docs
echo "  - YAML documentation"
./build/repobird docs yaml docs/repobird.yaml

echo "✓ Documentation generated in docs/ and man/ directories"
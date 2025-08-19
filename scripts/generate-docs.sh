#!/bin/bash
set -e

# Determine output directory (default to temp if not specified)
OUTPUT_DIR="${1:-/tmp/repobird-docs-$$}"

# Build the binary if it doesn't exist
if [ ! -f "build/repobird" ]; then
    echo "Building repobird binary..."
    make build
fi

echo "Generating documentation in $OUTPUT_DIR..."

# Create output directories
mkdir -p "$OUTPUT_DIR/man"
mkdir -p "$OUTPUT_DIR/markdown"
mkdir -p "$OUTPUT_DIR/yaml"

# Generate man pages
echo "  - man pages"
./build/repobird docs man "$OUTPUT_DIR/man"

# Generate markdown docs
echo "  - markdown documentation"
./build/repobird docs markdown "$OUTPUT_DIR/markdown"

# Generate YAML docs
echo "  - YAML documentation"
./build/repobird docs yaml "$OUTPUT_DIR/yaml"

echo "✓ Documentation generated in $OUTPUT_DIR"
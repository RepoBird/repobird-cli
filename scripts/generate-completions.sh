#!/bin/bash
set -e

# Determine output directory (default to temp if not specified)
OUTPUT_DIR="${1:-/tmp/repobird-completions-$$}"

# Build the binary if it doesn't exist
if [ ! -f "build/repobird" ]; then
    echo "Building repobird binary..."
    make build
fi

echo "Generating shell completions in $OUTPUT_DIR..."

# Create completions directory
mkdir -p "$OUTPUT_DIR"

# Generate bash completion
echo "  - bash completion"
./build/repobird completion bash > "$OUTPUT_DIR/repobird.bash"

# Generate zsh completion
echo "  - zsh completion"
./build/repobird completion zsh > "$OUTPUT_DIR/_repobird"

# Generate fish completion
echo "  - fish completion"
./build/repobird completion fish > "$OUTPUT_DIR/repobird.fish"

# Generate PowerShell completion
echo "  - powershell completion"
./build/repobird completion powershell > "$OUTPUT_DIR/repobird.ps1"

echo "✓ Shell completions generated in $OUTPUT_DIR"
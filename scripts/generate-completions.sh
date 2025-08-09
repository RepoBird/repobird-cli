#!/bin/bash
set -e

# Create completions directory
mkdir -p completions

# Build the binary if it doesn't exist
if [ ! -f "build/repobird" ]; then
    echo "Building repobird binary..."
    make build
fi

echo "Generating shell completions..."

# Generate bash completion
echo "  - bash completion"
./build/repobird completion bash > completions/repobird.bash

# Generate zsh completion
echo "  - zsh completion"
./build/repobird completion zsh > completions/_repobird

# Generate fish completion
echo "  - fish completion"
./build/repobird completion fish > completions/repobird.fish

# Generate PowerShell completion
echo "  - powershell completion"
./build/repobird completion powershell > completions/repobird.ps1

echo "✓ Shell completions generated in completions/ directory"
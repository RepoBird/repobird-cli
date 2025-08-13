#!/bin/bash

# Script to clear test/debug cache files while preserving real user cache

echo "Clearing test/debug cache files..."

# Remove test user directories (these are fake)
if [ -d "$HOME/.cache/repobird/users/user-123" ]; then
    echo "Removing test cache: user-123"
    rm -rf "$HOME/.cache/repobird/users/user-123"
fi

if [ -d "$HOME/.cache/repobird/users/user-456" ]; then
    echo "Removing test cache: user-456"
    rm -rf "$HOME/.cache/repobird/users/user-456"
fi

if [ -d "$HOME/.cache/repobird/users/user-789" ]; then
    echo "Removing test cache: user-789"
    rm -rf "$HOME/.cache/repobird/users/user-789"
fi

# Remove any debug cache directories (negative user IDs)
if [ -d "$HOME/.cache/repobird/debug" ]; then
    echo "Removing debug cache directory"
    rm -rf "$HOME/.cache/repobird/debug"
fi

echo "Test cache cleared. Your real user cache is preserved."
echo ""
echo "You can now run 'make tui' to test with your real account."
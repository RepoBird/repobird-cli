#!/bin/bash

# Script to clear test/debug cache files while preserving real user cache

echo "Clearing test/debug cache files..."

# Support both old and new cache locations
OLD_CACHE="$HOME/.cache/repobird"
NEW_CACHE="$HOME/.config/repobird/cache"

# Clear old cache location test directories
if [ -d "$OLD_CACHE/users/user-123" ]; then
    echo "Removing old test cache: user-123"
    rm -rf "$OLD_CACHE/users/user-123"
fi

if [ -d "$OLD_CACHE/users/user-456" ]; then
    echo "Removing old test cache: user-456"
    rm -rf "$OLD_CACHE/users/user-456"
fi

if [ -d "$OLD_CACHE/users/user-789" ]; then
    echo "Removing old test cache: user-789"
    rm -rf "$OLD_CACHE/users/user-789"
fi

if [ -d "$OLD_CACHE/users/user--1" ]; then
    echo "Removing old debug user cache: user--1"
    rm -rf "$OLD_CACHE/users/user--1"
fi

if [ -d "$OLD_CACHE/debug" ]; then
    echo "Removing old debug cache directory"
    rm -rf "$OLD_CACHE/debug"
fi

# Clear new cache location test directories (without 'user-' prefix)
if [ -d "$NEW_CACHE/users/123" ]; then
    echo "Removing test cache: 123"
    rm -rf "$NEW_CACHE/users/123"
fi

if [ -d "$NEW_CACHE/users/456" ]; then
    echo "Removing test cache: 456"
    rm -rf "$NEW_CACHE/users/456"
fi

if [ -d "$NEW_CACHE/users/789" ]; then
    echo "Removing test cache: 789"
    rm -rf "$NEW_CACHE/users/789"
fi

# Remove negative user IDs (debug mode)
if [ -d "$NEW_CACHE/users/-1" ]; then
    echo "Removing debug user cache: -1"
    rm -rf "$NEW_CACHE/users/-1"
fi

# Remove anonymous cache directories
if [ -d "$NEW_CACHE/anonymous" ]; then
    echo "Removing anonymous cache"
    rm -rf "$NEW_CACHE/anonymous"
fi

if [ -d "$NEW_CACHE/users/anonymous" ]; then
    echo "Removing anonymous user cache"
    rm -rf "$NEW_CACHE/users/anonymous"
fi

echo "Test cache cleared. Your real user cache is preserved."
echo ""
echo "You can now run 'make tui' to test with your real account."
echo "Or run 'make tui-debug' to test with mock data."
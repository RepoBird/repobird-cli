#!/bin/bash

# Example Kitty terminal development setup for RepoBird CLI
# Copy this to kitty-dev.local.sh and customize for your environment
# 
# This script sets up a multi-window development environment with:
# - Multiple Claude CLI windows for AI assistance
# - Editor window with your preferred editor
# - Terminal windows for running commands

# Save the current working directory
current_dir=$(pwd)

kitty @ goto-layout tall
kitty @ set-window-title --match state:self "Main Window"

# First tab - Multiple windows for development tools
# Example: Claude CLI windows (adjust paths/commands as needed)
kitty @ launch --type=window --cwd="$current_dir" --title="Claude 1" --hold $SHELL -c "source ~/.zshrc && claude"

kitty @ launch --type=window --cwd="$current_dir" --title="Claude 2" --hold $SHELL -c "source ~/.zshrc && claude"

kitty @ launch --type=window --cwd="$current_dir" --title="Shell" $SHELL

kitty @ goto-layout tall

# Second tab - Editor
# Replace 'nvim' with your preferred editor (vim, emacs, code, etc.)
kitty @ launch --type=tab --cwd="$current_dir" --title="Editor" $SHELL -c "source ~/.zshrc && nvim; exec $SHELL"

# Add more tabs as needed for your workflow
# kitty @ launch --type=tab --cwd="$current_dir" --title="Tests" $SHELL
# kitty @ launch --type=tab --cwd="$current_dir" --title="Logs" $SHELL -c "tail -f /tmp/repobird_debug.log"
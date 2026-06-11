#!/bin/bash

# Personal development setup for RepoBird CLI with Kitty terminal
# This script sets up a multi-window development environment
# 
# NOTE: This is a personal development script that may contain user-specific paths.
# Feel free to modify for your own setup or create your own version.
# Consider copying this to kitty-dev.local.sh and customizing there.

# Save the current working directory
current_dir=$(pwd)

kitty @ goto-layout tall
kitty @ set-window-title --match state:self "Main Window"

# First tab - shell workspace
kitty @ launch --type=window --cwd="$current_dir" --title="Shell" $SHELL
kitty @ goto-layout tall

# Second tab - Editor
kitty @ launch --type=tab --cwd="$current_dir" --title="Editor" $SHELL -c "source ~/.zshrc && nvim; exec $SHELL"

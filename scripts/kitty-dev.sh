#!/bin/bash

# Personal development setup for RepoBird CLI with Kitty terminal
# This script sets up a multi-window development environment
# 
# TRUE VIBE CODING: 3 Claude windows, 1 editor, 0 Stack Overflow tabs
# Because who needs documentation when you have AI pair programmers everywhere?
# 
# NOTE: This is a personal development script that may contain user-specific paths.
# Feel free to modify for your own setup or create your own version.
# Consider copying this to kitty-dev.local.sh and customizing there.

# Save the current working directory
current_dir=$(pwd)

kitty @ goto-layout tall
kitty @ set-window-title --match state:self "Main Window"

# First tab - 4 windows (3 Claude, 1 shell)
# Launch Claude windows - source zshrc first to get PATH with bun
kitty @ launch --type=window --cwd="$current_dir" --title="Claude 1" --hold $SHELL -c "source ~/.zshrc && ANTHROPIC_API_KEY='' claude"

kitty @ launch --type=window --cwd="$current_dir" --title="Claude 2" --hold $SHELL -c "source ~/.zshrc && ANTHROPIC_API_KEY='' claude"

kitty @ launch --type=window --cwd="$current_dir" --title="Claude 3" --hold $SHELL -c "source ~/.zshrc && ANTHROPIC_API_KEY='' claude"
kitty @ goto-layout tall

# Second tab - Editor
kitty @ launch --type=tab --cwd="$current_dir" --title="Editor" $SHELL -c "source ~/.zshrc && nvim; exec $SHELL"

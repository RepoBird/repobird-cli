#!/bin/bash

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

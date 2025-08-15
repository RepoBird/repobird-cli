#!/bin/bash
set -e

# RepoBird CLI Shell Completion Setup Script
# This script helps set up shell completions for repobird and rb commands

BINARY_NAME="repobird"
ALIAS_NAME="rb"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}✓${NC} $1"
}

warn() {
    echo -e "${YELLOW}⚠${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1" >&2
}

info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

# Check if repobird is installed
if ! command -v "$BINARY_NAME" >/dev/null 2>&1; then
    error "RepoBird CLI not found. Please install it first."
    echo "Installation: curl -fsSL https://get.repobird.ai | sh"
    exit 1
fi

# Detect shell
detect_shell() {
    if [ -n "$SHELL" ]; then
        case "$SHELL" in
            */bash) echo "bash" ;;
            */zsh) echo "zsh" ;;
            */fish) echo "fish" ;;
            *) echo "unknown" ;;
        esac
    else
        echo "unknown"
    fi
}

setup_bash_completions() {
    local bashrc="$HOME/.bashrc"
    
    info "Setting up Bash completions..."
    
    # Check if completions already exist
    if grep -q "source <($BINARY_NAME completion bash)" "$bashrc" 2>/dev/null; then
        warn "Bash completions for $BINARY_NAME already configured"
    else
        echo "" >> "$bashrc"
        echo "# RepoBird CLI completions" >> "$bashrc"
        echo "source <($BINARY_NAME completion bash)" >> "$bashrc"
        echo "complete -o default -F __start_${BINARY_NAME} $ALIAS_NAME" >> "$bashrc"
        log "Added $BINARY_NAME completions to $bashrc"
    fi
    
    echo ""
    info "To activate completions now, run:"
    echo "  source $bashrc"
}

setup_zsh_completions() {
    local zshrc="$HOME/.zshrc"
    
    info "Setting up Zsh completions..."
    
    # Check if completions already exist
    if grep -q "source <($BINARY_NAME completion zsh)" "$zshrc" 2>/dev/null; then
        warn "Zsh completions for $BINARY_NAME already configured"
    else
        echo "" >> "$zshrc"
        echo "# RepoBird CLI completions" >> "$zshrc"
        echo "source <($BINARY_NAME completion zsh)" >> "$zshrc"
        echo "compdef _${BINARY_NAME} $ALIAS_NAME" >> "$zshrc"
        log "Added $BINARY_NAME completions to $zshrc"
    fi
    
    echo ""
    info "To activate completions now, run:"
    echo "  source $zshrc"
}

setup_fish_completions() {
    local fish_dir="$HOME/.config/fish/completions"
    
    info "Setting up Fish completions..."
    
    # Create completions directory if it doesn't exist
    mkdir -p "$fish_dir"
    
    # Generate completion files
    "$BINARY_NAME" completion fish > "$fish_dir/${BINARY_NAME}.fish"
    "$BINARY_NAME" completion fish | sed "s/$BINARY_NAME/$ALIAS_NAME/g" > "$fish_dir/${ALIAS_NAME}.fish"
    
    log "Added $BINARY_NAME completions to $fish_dir"
    info "Completions will be active in new Fish sessions"
}

# Interactive mode
interactive_setup() {
    echo -e "${BLUE}RepoBird CLI Completion Setup${NC}"
    echo ""
    
    local detected_shell=$(detect_shell)
    
    if [ "$detected_shell" != "unknown" ]; then
        info "Detected shell: $detected_shell"
        echo ""
        echo "Would you like to set up completions for $detected_shell? (y/n)"
        read -r response
        
        if [[ "$response" =~ ^[Yy]$ ]]; then
            case "$detected_shell" in
                bash) setup_bash_completions ;;
                zsh) setup_zsh_completions ;;
                fish) setup_fish_completions ;;
            esac
        fi
    else
        warn "Could not detect shell type"
        echo ""
        echo "Please select your shell:"
        echo "  1) Bash"
        echo "  2) Zsh"
        echo "  3) Fish"
        echo "  4) Exit"
        echo ""
        read -p "Enter choice [1-4]: " choice
        
        case $choice in
            1) setup_bash_completions ;;
            2) setup_zsh_completions ;;
            3) setup_fish_completions ;;
            4) exit 0 ;;
            *) error "Invalid choice" ;;
        esac
    fi
    
    echo ""
    log "Setup complete!"
    echo ""
    echo "Test completions by typing:"
    echo "  $BINARY_NAME [TAB][TAB]"
    echo "  $ALIAS_NAME [TAB][TAB]"
}

# Main logic
main() {
    # Check for command line arguments
    if [ $# -eq 0 ]; then
        interactive_setup
    else
        case "$1" in
            bash) setup_bash_completions ;;
            zsh) setup_zsh_completions ;;
            fish) setup_fish_completions ;;
            --help|-h)
                echo "Usage: $0 [bash|zsh|fish]"
                echo ""
                echo "Without arguments: Interactive setup"
                echo "With shell name: Setup for specific shell"
                ;;
            *)
                error "Unknown shell: $1"
                echo "Supported shells: bash, zsh, fish"
                exit 1
                ;;
        esac
    fi
}

main "$@"
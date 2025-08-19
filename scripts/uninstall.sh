#!/bin/bash
set -e

# RepoBird CLI Uninstaller Script
# This script removes the RepoBird CLI binary and associated data

APPNAME="repobird"
ALIAS_NAME="rb"
YELLOW='\033[1;33m'
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

echo -e "${GREEN}RepoBird CLI Uninstaller${NC}"
echo "========================="
echo ""

# Function to confirm action
confirm() {
    local prompt="$1"
    local response
    echo -en "${YELLOW}$prompt (y/N): ${NC}"
    read -r response
    case "$response" in
        [yY][eE][sS]|[yY]) 
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

# Function to find and remove binary
remove_binary() {
    local found=false
    
    # Check common installation locations for both repobird and rb
    local locations=(
        "/usr/local/bin/$APPNAME"
        "/usr/bin/$APPNAME"
        "$HOME/go/bin/$APPNAME"
        "$HOME/.local/bin/$APPNAME"
        "/usr/local/bin/$ALIAS_NAME"
        "/usr/bin/$ALIAS_NAME"
        "$HOME/go/bin/$ALIAS_NAME"
        "$HOME/.local/bin/$ALIAS_NAME"
    )
    
    # Check if repobird is in PATH
    if command -v "$APPNAME" &> /dev/null; then
        local bin_path=$(which "$APPNAME")
        echo -e "Found $APPNAME at: ${GREEN}$bin_path${NC}"
        
        if confirm "Remove binary at $bin_path?"; then
            # Check if we need sudo
            if [[ "$bin_path" == "/usr/local/bin/"* ]] || [[ "$bin_path" == "/usr/bin/"* ]]; then
                echo "Removing binary (requires sudo)..."
                sudo rm -f "$bin_path"
            else
                echo "Removing binary..."
                rm -f "$bin_path"
            fi
            echo -e "${GREEN}✓${NC} Binary removed"
            found=true
        else
            echo "Skipping binary removal"
        fi
    fi
    
    # Check if rb alias is in PATH
    if command -v "$ALIAS_NAME" &> /dev/null; then
        local alias_path=$(which "$ALIAS_NAME")
        echo -e "Found $ALIAS_NAME at: ${GREEN}$alias_path${NC}"
        
        if confirm "Remove alias at $alias_path?"; then
            # Check if we need sudo
            if [[ "$alias_path" == "/usr/local/bin/"* ]] || [[ "$alias_path" == "/usr/bin/"* ]]; then
                echo "Removing alias (requires sudo)..."
                sudo rm -f "$alias_path"
            else
                echo "Removing alias..."
                rm -f "$alias_path"
            fi
            echo -e "${GREEN}✓${NC} Alias removed"
            found=true
        else
            echo "Skipping alias removal"
        fi
    fi
    
    # If not found in PATH, check known locations
    if [ "$found" = false ]; then
        # Check known locations even if not in PATH
        for location in "${locations[@]}"; do
            if [ -f "$location" ]; then
                local name_to_display="$APPNAME"
                if [[ "$location" == *"/$ALIAS_NAME" ]]; then
                    name_to_display="$ALIAS_NAME (alias)"
                fi
                echo -e "Found $name_to_display at: ${GREEN}$location${NC}"
                
                if confirm "Remove file at $location?"; then
                    # Check if we need sudo
                    if [[ "$location" == "/usr/local/bin/"* ]] || [[ "$location" == "/usr/bin/"* ]]; then
                        echo "Removing binary (requires sudo)..."
                        sudo rm -f "$location"
                    else
                        echo "Removing binary..."
                        rm -f "$location"
                    fi
                    echo -e "${GREEN}✓${NC} Binary removed"
                    found=true
                fi
            fi
        done
        
        if [ "$found" = false ]; then
            echo -e "${YELLOW}No $APPNAME binary found in common locations${NC}"
        fi
    fi
}

# Function to remove config directory
remove_config() {
    local config_dirs=(
        "$HOME/.config/$APPNAME"
        "$HOME/.repobird"  # Legacy location
    )
    
    local found=false
    for config_dir in "${config_dirs[@]}"; do
        if [ -d "$config_dir" ]; then
            found=true
            echo -e "Found config directory: ${GREEN}$config_dir${NC}"
            
            # Check for API key
            if [ -f "$config_dir/config.yaml" ] || [ -f "$config_dir/config.json" ]; then
                echo -e "${YELLOW}⚠ This directory contains your API key and configuration${NC}"
            fi
            
            if confirm "Remove config directory at $config_dir?"; then
                echo "Removing config directory..."
                rm -rf "$config_dir"
                echo -e "${GREEN}✓${NC} Config directory removed"
            else
                echo "Skipping config directory removal"
            fi
        fi
    done
    
    if [ "$found" = false ]; then
        echo -e "${YELLOW}No config directories found${NC}"
    fi
}

# Function to remove cache directory
remove_cache() {
    local cache_dirs=(
        "$HOME/.config/$APPNAME/cache"  # Current location
        "$HOME/.cache/$APPNAME"         # Alternative location
    )
    
    local found=false
    for cache_dir in "${cache_dirs[@]}"; do
        if [ -d "$cache_dir" ]; then
            found=true
            echo -e "Found cache directory: ${GREEN}$cache_dir${NC}"
            
            # Show cache size
            if command -v du &> /dev/null; then
                local size=$(du -sh "$cache_dir" 2>/dev/null | cut -f1)
                echo -e "Cache size: ${YELLOW}$size${NC}"
            fi
            
            if confirm "Remove cache directory at $cache_dir?"; then
                echo "Removing cache directory..."
                rm -rf "$cache_dir"
                echo -e "${GREEN}✓${NC} Cache directory removed"
            else
                echo "Skipping cache directory removal"
            fi
        fi
    done
    
    if [ "$found" = false ]; then
        echo -e "${YELLOW}No cache directories found${NC}"
    fi
}

# Function to remove shell completions
remove_shell_completions() {
    local found=false
    local shell_configs=(
        "$HOME/.bashrc"
        "$HOME/.bash_profile"
        "$HOME/.zshrc"
        "$HOME/.config/fish/config.fish"
    )
    
    for config_file in "${shell_configs[@]}"; do
        if [ -f "$config_file" ]; then
            # Check for repobird completion entries
            if grep -q "repobird completion\|rb completion\|_repobird\|__start_repobird" "$config_file" 2>/dev/null; then
                found=true
                echo -e "Found RepoBird completions in: ${GREEN}$config_file${NC}"
                
                # Show the lines that will be removed
                echo -e "${YELLOW}Lines containing RepoBird completions:${NC}"
                grep --color=never "repobird completion\|rb completion\|_repobird\|__start_repobird" "$config_file" | head -5
                
                if confirm "Remove RepoBird completions from $config_file?"; then
                    # Create backup
                    cp "$config_file" "$config_file.repobird-backup"
                    echo "Created backup: $config_file.repobird-backup"
                    
                    # Remove completion lines
                    if [[ "$OSTYPE" == "darwin"* ]]; then
                        # macOS sed requires different syntax
                        sed -i '' '/repobird completion/d; /rb completion/d; /_repobird/d; /__start_repobird/d' "$config_file"
                    else
                        # Linux sed
                        sed -i '/repobird completion/d; /rb completion/d; /_repobird/d; /__start_repobird/d' "$config_file"
                    fi
                    
                    echo -e "${GREEN}✓${NC} Removed completions from $config_file"
                else
                    echo "Skipping completion removal from $config_file"
                fi
            fi
        fi
    done
    
    # Check for Fish completion files
    local fish_completions=(
        "$HOME/.config/fish/completions/repobird.fish"
        "$HOME/.config/fish/completions/rb.fish"
    )
    
    for completion_file in "${fish_completions[@]}"; do
        if [ -f "$completion_file" ]; then
            found=true
            echo -e "Found Fish completion file: ${GREEN}$completion_file${NC}"
            
            if confirm "Remove $completion_file?"; then
                rm -f "$completion_file"
                echo -e "${GREEN}✓${NC} Removed $completion_file"
            else
                echo "Skipping $completion_file"
            fi
        fi
    done
    
    if [ "$found" = false ]; then
        echo -e "${YELLOW}No shell completions found${NC}"
    else
        echo ""
        echo -e "${YELLOW}Note: Restart your shell or run 'source ~/.bashrc' (or ~/.zshrc) to apply changes${NC}"
    fi
}

# Main uninstall process
echo "This script will uninstall RepoBird CLI and optionally remove its data."
echo ""

if ! confirm "Proceed with uninstallation?"; then
    echo -e "${YELLOW}Uninstallation cancelled${NC}"
    exit 0
fi

echo ""
echo "Step 1: Remove RepoBird binary"
echo "-------------------------------"
remove_binary

echo ""
echo "Step 2: Remove configuration files"
echo "----------------------------------"
remove_config

echo ""
echo "Step 3: Remove cache files"
echo "--------------------------"
remove_cache

echo ""
echo "Step 4: Remove shell completions"
echo "--------------------------------"
remove_shell_completions

echo ""
echo "========================="
echo -e "${GREEN}Uninstallation complete!${NC}"
echo ""
echo "Thank you for using RepoBird CLI."
echo "To reinstall, visit: https://github.com/RepoBird/repobird-cli"
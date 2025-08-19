#!/bin/bash
set -e

# RepoBird CLI One-Line Installer
# Usage: curl -fsSL https://get.repobird.ai | sh
# Or: wget -qO- https://get.repobird.ai | sh

# Constants
GITHUB_REPO="RepoBird/repobird-cli"
BINARY_NAME="repobird"
INSTALL_DIR="$HOME/.local/bin"
ALIAS_NAME="rb"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
log() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Detect OS and architecture
detect_platform() {
    local os=""
    local arch=""
    
    # Detect OS
    case "$(uname -s)" in
        Linux*)     os="linux";;
        Darwin*)    os="darwin";;
        CYGWIN*|MINGW*|MSYS*) os="windows";;
        *)
            error "Unsupported operating system: $(uname -s)"
            exit 1
            ;;
    esac
    
    # Detect architecture
    case "$(uname -m)" in
        x86_64|amd64)   arch="amd64";;
        arm64|aarch64)  arch="arm64";;
        i386|i686)      arch="386";;
        armv6l)         arch="armv6";;
        armv7l)         arch="armv7";;
        *)
            error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac
    
    # Windows doesn't support arm64 in our builds
    if [ "$os" = "windows" ] && [ "$arch" = "arm64" ]; then
        warn "Windows ARM64 not supported, falling back to amd64"
        arch="amd64"
    fi
    
    echo "${os}_${arch}"
}

# Get latest version from GitHub API
get_latest_version() {
    local api_url="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
    
    # Try with curl first
    if command -v curl >/dev/null 2>&1; then
        curl -s "$api_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
    # Fall back to wget
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "$api_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
    else
        error "Neither curl nor wget found. Cannot download."
        exit 1
    fi
}

# Check if we have required tools
check_dependencies() {
    local missing_tools=()
    
    if ! command -v tar >/dev/null 2>&1; then
        missing_tools+=("tar")
    fi
    
    if ! command -v gzip >/dev/null 2>&1; then
        missing_tools+=("gzip")
    fi
    
    if [ ${#missing_tools[@]} -ne 0 ]; then
        error "Missing required tools: ${missing_tools[*]}"
        error "Please install these tools and try again."
        exit 1
    fi
}

# Download and extract binary
install_binary() {
    local platform=$1
    local version=$2
    local temp_dir=$(mktemp -d)
    
    log "Detected platform: $platform"
    log "Latest version: $version"
    
    # Construct download URL
    local filename=""
    local extract_cmd=""
    
    if [[ $platform == *"windows"* ]]; then
        filename="${BINARY_NAME}_${platform}.zip"
        extract_cmd="unzip -q"
        
        if ! command -v unzip >/dev/null 2>&1; then
            error "unzip not found. Please install unzip or download manually."
            exit 1
        fi
    else
        filename="${BINARY_NAME}_${platform}.tar.gz"
        extract_cmd="tar -xzf"
    fi
    
    local download_url="https://github.com/${GITHUB_REPO}/releases/download/${version}/${filename}"
    local archive_path="${temp_dir}/${filename}"
    
    log "Downloading from: $download_url"
    
    # Download
    if command -v curl >/dev/null 2>&1; then
        curl -L -o "$archive_path" "$download_url"
    elif command -v wget >/dev/null 2>&1; then
        wget -O "$archive_path" "$download_url"
    else
        error "Neither curl nor wget found. Cannot download."
        exit 1
    fi
    
    # Verify download
    if [ ! -f "$archive_path" ]; then
        error "Download failed: $archive_path not found"
        exit 1
    fi
    
    log "Download complete, extracting..."
    
    # Extract
    cd "$temp_dir"
    $extract_cmd "$archive_path"
    
    # Find the binary (handle different archive structures)
    local binary_path=""
    if [[ $platform == *"windows"* ]]; then
        binary_path=$(find . -name "${BINARY_NAME}.exe" | head -1)
    else
        binary_path=$(find . -name "$BINARY_NAME" | head -1)
    fi
    
    if [ -z "$binary_path" ]; then
        error "Binary not found in archive"
        exit 1
    fi
    
    # Create install directory
    mkdir -p "$INSTALL_DIR"
    
    # Install binary
    local installed_binary="$INSTALL_DIR/$BINARY_NAME"
    if [[ $platform == *"windows"* ]]; then
        installed_binary="${installed_binary}.exe"
    fi
    
    cp "$binary_path" "$installed_binary"
    chmod +x "$installed_binary"
    
    # Create alias
    if [[ ! $platform == *"windows"* ]]; then
        ln -sf "$installed_binary" "$INSTALL_DIR/$ALIAS_NAME"
    fi
    
    # Cleanup
    rm -rf "$temp_dir"
    
    log "✓ Installed $BINARY_NAME to $installed_binary"
    if [[ ! $platform == *"windows"* ]]; then
        log "✓ Created alias 'rb' -> 'repobird'"
    fi
}

# Setup shell completions
setup_completions() {
    local shell_type=""
    
    # Detect current shell
    if [ -n "$BASH_VERSION" ]; then
        shell_type="bash"
    elif [ -n "$ZSH_VERSION" ]; then
        shell_type="zsh"
    elif [ -n "$FISH_VERSION" ]; then
        shell_type="fish"
    fi
    
    echo ""
    echo -e "${BLUE}Shell Completions:${NC}"
    echo "To enable tab completions for $BINARY_NAME and $ALIAS_NAME:"
    echo ""
    
    # Provide shell-specific instructions
    echo "For Bash:"
    echo "  echo 'source <($BINARY_NAME completion bash)' >> ~/.bashrc"
    echo "  echo 'complete -o default -F __start_${BINARY_NAME} $ALIAS_NAME' >> ~/.bashrc"
    echo ""
    echo "For Zsh:"
    echo "  echo 'source <($BINARY_NAME completion zsh)' >> ~/.zshrc"
    echo "  echo 'compdef _${BINARY_NAME} $ALIAS_NAME' >> ~/.zshrc"
    echo ""
    echo "For Fish:"
    echo "  $BINARY_NAME completion fish > ~/.config/fish/completions/${BINARY_NAME}.fish"
    echo "  $BINARY_NAME completion fish | sed 's/${BINARY_NAME}/$ALIAS_NAME/g' > ~/.config/fish/completions/${ALIAS_NAME}.fish"
    echo ""
    
    if [ -n "$shell_type" ]; then
        echo -e "${GREEN}Detected shell: $shell_type${NC}"
        echo "Run the commands above for your shell, then restart your terminal or source your config."
    fi
}

# Check if install directory is in PATH
check_path() {
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        warn "$INSTALL_DIR is not in your PATH"
        echo ""
        echo "To use $BINARY_NAME from anywhere, add this to your shell profile:"
        echo "  echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.bashrc"
        echo "  source ~/.bashrc"
        echo ""
        echo "Or for zsh users:"
        echo "  echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.zshrc"
        echo "  source ~/.zshrc"
        echo ""
    fi
}

# Verify installation
verify_installation() {
    local installed_binary="$INSTALL_DIR/$BINARY_NAME"
    
    if [ -x "$installed_binary" ]; then
        log "Installation verified!"
        
        # Try to run version command
        if "$installed_binary" version >/dev/null 2>&1; then
            local version_output=$("$installed_binary" version)
            log "Version: $version_output"
        fi
        
        echo ""
        echo -e "${GREEN}🎉 RepoBird CLI installed successfully!${NC}"
        echo ""
        echo "Get started:"
        echo "  1. Configure your API key: $BINARY_NAME config set api-key YOUR_KEY"
        echo "  2. Run your first task: $BINARY_NAME run task.json"
        echo "  3. Check status: $BINARY_NAME status"
        echo ""
        echo "Documentation: https://docs.repobird.ai"
        echo "Issues: https://github.com/$GITHUB_REPO/issues"
        
    else
        error "Installation verification failed"
        exit 1
    fi
}

# Main installation flow
main() {
    echo -e "${BLUE}"
    cat << "EOF"
    ____                ____  _         _ 
   |  _ \ ___ _ __   ___| __ )(_)_ __ __| |
   | |_) / _ \ '_ \ / _ \  _ \| | '__/ _` |
   |  _ <  __/ |_) | (_) |_) | | | | (_| |
   |_| \_\___| .__/ \___/____/|_|_|  \__,_|
             |_|                          
   
   RepoBird CLI Installer
EOF
    echo -e "${NC}"
    
    log "Starting RepoBird CLI installation..."
    
    # Check for existing installation
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        local current_version=$("$BINARY_NAME" version 2>/dev/null | head -1 || echo "unknown")
        warn "RepoBird CLI is already installed: $current_version"
        echo "This will update your existing installation."
        echo ""
    fi
    
    check_dependencies
    
    local platform=$(detect_platform)
    local version=$(get_latest_version)
    
    if [ -z "$version" ]; then
        error "Could not determine latest version"
        exit 1
    fi
    
    install_binary "$platform" "$version"
    check_path
    verify_installation
    setup_completions
}

# Handle script being piped from curl
if [ -t 0 ]; then
    # Running interactively
    main "$@"
else
    # Being piped, run with error handling
    {
        main "$@"
    } || {
        error "Installation failed"
        echo ""
        echo "Manual installation options:"
        echo "  1. Download from: https://github.com/$GITHUB_REPO/releases"
        echo "  2. Package managers: https://docs.repobird.ai/installation"
        echo "  3. Build from source: git clone https://github.com/$GITHUB_REPO"
        exit 1
    }
fi
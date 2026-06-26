# RepoBird CLI Installation Guide

GitHub Releases are the canonical artifact host for RepoBird CLI. The repobird.ai install URLs are thin web entrypoints that should serve or redirect to the scripts in this repository.

## Quick Install

### Linux/macOS

```bash
curl -fsSL https://repobird.ai/install.sh | sh
```

Equivalent install page:

```text
https://repobird.ai/install
```

The script detects your OS and architecture, downloads the matching GitHub Release archive, verifies it against `checksums.txt`, installs `repobird` to `~/.local/bin`, and creates the `rb` alias.

Pin a specific release by setting `REPOBIRD_VERSION` to the tag:

```bash
curl -fsSL https://repobird.ai/install.sh | REPOBIRD_VERSION=v1.2.3 sh
```

Install somewhere else with `REPOBIRD_INSTALL_DIR`:

```bash
curl -fsSL https://repobird.ai/install.sh | REPOBIRD_INSTALL_DIR=/usr/local/bin sh
```

### Windows PowerShell

```powershell
iwr -useb https://repobird.ai/install.ps1 | iex
```

Install a specific release:

```powershell
& ([scriptblock]::Create((iwr -useb https://repobird.ai/install.ps1).Content)) -Version v1.2.3
```

## Release Artifacts

### Download Binary

1. Visit the [releases page](https://github.com/repobird/repobird-cli/releases)
2. Download the appropriate binary for your OS and architecture:
   - `repobird-cli_linux_amd64.tar.gz` - Linux 64-bit
   - `repobird-cli_linux_arm64.tar.gz` - Linux ARM64
   - `repobird-cli_darwin_amd64.tar.gz` - macOS Intel
   - `repobird-cli_darwin_arm64.tar.gz` - macOS Apple Silicon
   - `repobird-cli_windows_amd64.zip` - Windows 64-bit
   - `repobird-cli_windows_386.zip` - Windows 32-bit

### Linux/macOS Installation

```bash
# Download and extract
curl -L -o repobird.tar.gz "https://github.com/RepoBird/repobird-cli/releases/latest/download/repobird-cli_linux_amd64.tar.gz"
tar -xzf repobird.tar.gz

# Install to local bin
mkdir -p ~/.local/bin
cp repobird ~/.local/bin/
ln -s ~/.local/bin/repobird ~/.local/bin/rb

# Add to PATH (add this to ~/.bashrc or ~/.zshrc)
export PATH="$HOME/.local/bin:$PATH"
```

### Windows Installation

1. Download the Windows ZIP file
2. Extract `repobird.exe` to a directory like `C:\Program Files\RepoBird\`
3. Add the directory to your PATH environment variable

## Package manager follow-ups

Package manager distribution is not the canonical first-party path yet. Track these as follow-up work after GitHub Release installers are stable:

- Homebrew tap for macOS/Linux.
- Scoop or Winget distribution for Windows.
- Optional Chocolatey support if a Windows packaging workflow is maintained.
- Optional APT/RPM repositories if custom repository hosting becomes worth the operational cost.

## Development Installation

### From Source

```bash
# Prerequisites: Go 1.21+
git clone https://github.com/repobird/repobird-cli.git
cd repobird-cli

# Build and install
make build
make install
```

### Using Go Install

```bash
go install github.com/repobird/repobird-cli/cmd/repobird@latest
```

## Shell Completions

After installation, enable shell completions for better CLI experience:

### Recommended Setup

Use the built-in installer for your shell. Add `--dry-run` first if you want to preview which files will be changed.

```bash
repobird completion install bash
repobird completion install zsh
repobird completion install fish
repobird completion install powershell
```

The installer configures completions for both `repobird` and the `rb` alias where the shell supports aliases.

### Manual Setup

Use these commands when packaging RepoBird, installing completions system-wide, or managing shell startup files yourself.

### Bash
```bash
# Install completion
repobird completion bash | sudo tee /etc/bash_completion.d/repobird

# Or for current user only
repobird completion bash > ~/.bash_completions/repobird
```

### Zsh
```bash
# Add to ~/.zshrc
echo "autoload -U compinit; compinit" >> ~/.zshrc

# Install completion
repobird completion zsh > "${fpath[1]}/_repobird"
```

### Fish
```bash
repobird completion fish > ~/.config/fish/completions/repobird.fish
```

### PowerShell
```powershell
repobird completion powershell | Out-String | Invoke-Expression
```

## Verification

Verify your installation:

```bash
# Check version
repobird version

# Test basic functionality
repobird --help

# Verify alias (if installed)
rb version
```

## Configuration

Set up your RepoBird API key:

```bash
# Set API key
repobird config set api-key <your-api-key>

# Verify configuration
repobird config get api-key
```

## Updating

### Package Managers
Most package managers support updating:

```bash
# Homebrew
brew upgrade repobird

# APT
sudo apt update && sudo apt upgrade repobird

# Chocolatey
choco upgrade repobird

# Snap
sudo snap refresh repobird
```

### Manual Update
Use the one-liner install script to update to the latest version:

```bash
curl -fsSL https://get.repobird.ai | sh
```

## Uninstallation

### Using the Uninstall Script (Recommended)

The easiest way to completely remove RepoBird CLI and all its data:

```bash
# If you have the repository cloned
./scripts/uninstall.sh

# Or download and run directly
curl -sSL https://raw.githubusercontent.com/RepoBird/repobird-cli/main/scripts/uninstall.sh | bash
```

The uninstall script will:
- Detect and remove `repobird` binary from all common locations
- Remove the `rb` alias/symlink
- Delete configuration files (including API keys)
- Clean up cache directories
- Prompt for confirmation before each removal

### Package Manager Uninstallation

```bash
# Homebrew (macOS)
brew uninstall repobird

# APT (Ubuntu/Debian)
sudo apt remove repobird

# YUM/DNF (Fedora/RHEL)
sudo yum remove repobird
# or
sudo dnf remove repobird

# Chocolatey (Windows)
choco uninstall repobird

# Scoop (Windows)
scoop uninstall repobird

# Snap (Linux)
sudo snap remove repobird
```

### Manual Uninstallation

If you prefer to uninstall manually:

```bash
# Remove binaries (check all possible locations)
sudo rm -f /usr/local/bin/repobird /usr/local/bin/rb
rm -f ~/.local/bin/repobird ~/.local/bin/rb
rm -f ~/go/bin/repobird ~/go/bin/rb

# Remove configuration and cache
rm -rf ~/.config/repobird
rm -rf ~/.repobird  # Legacy location
```

## Troubleshooting

### Common Issues

1. **Command not found**: Ensure the installation directory is in your PATH
2. **Permission denied**: Check file permissions (`chmod +x repobird`)
3. **API errors**: Verify your API key is set correctly
4. **Network issues**: Check firewall and proxy settings

### Get Help

- GitHub Issues: https://github.com/RepoBird/repobird-cli/issues
- Documentation: https://docs.repobird.ai
- Community: https://discord.gg/repobird

### System Requirements

- **OS**: Linux, macOS, or Windows
- **Architecture**: amd64, arm64, or 386 (Windows)
- **Network**: Internet connection for API calls
- **Dependencies**: None (statically compiled binary)

## Security

All packages are cryptographically signed. Verify signatures:

```bash
# Download signature and verify
curl -L -o repobird.tar.gz.asc "https://github.com/RepoBird/repobird-cli/releases/latest/download/repobird_linux_amd64.tar.gz.asc"
gpg --verify repobird.tar.gz.asc repobird.tar.gz
```

Import our signing key:
```bash
curl -fsSL https://keys.repobird.ai/signing.asc | gpg --import
```

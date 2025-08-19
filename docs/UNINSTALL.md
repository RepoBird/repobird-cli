# Uninstallation Guide

## Using the Uninstall Script

The easiest way to completely remove RepoBird CLI and its data:

```bash
# If you have the repository cloned
./scripts/uninstall.sh

# Or download and run the script directly
curl -sSL https://raw.githubusercontent.com/RepoBird/repobird-cli/main/scripts/uninstall.sh | bash
```

The uninstall script will:
- Remove the `repobird` binary and `rb` alias from your system
- Delete configuration files (including API keys)
- Clean up cache directories
- Prompt for confirmation before each removal

## Manual Uninstallation

If you prefer to uninstall manually:

```bash
# Remove the binary (location depends on installation method)
sudo rm -f /usr/local/bin/repobird
sudo rm -f /usr/local/bin/rb
# Or if installed with go install
rm -f ~/go/bin/repobird
rm -f ~/go/bin/rb

# Remove configuration and cache
rm -rf ~/.config/repobird
rm -rf ~/.repobird  # Legacy location
```

## Package Manager Uninstallation

For package manager installations:

```bash
# Homebrew (macOS)
brew uninstall repobird-cli

# Scoop (Windows)
scoop uninstall repobird

# Arch Linux (AUR)
yay -R repobird

# Debian/Ubuntu
sudo apt remove repobird

# Red Hat/Fedora
sudo rpm -e repobird
```
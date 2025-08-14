# Troubleshooting Guide

## Overview

Common issues and solutions for RepoBird CLI.

## Related Documentation
- **[Configuration Guide](configuration-guide.md)** - Configuration setup
- **[API Reference](api-reference.md)** - API errors
- **[TUI Guide](tui-guide.md)** - TUI issues
- **[Development Guide](development-guide.md)** - Debug techniques

## Quick Diagnostics

```bash
# Check setup
repobird auth verify

# Test API connection
repobird status

# Enable debug mode
REPOBIRD_DEBUG_LOG=1 repobird tui
tail -f /tmp/repobird_debug.log
```

## Common Issues

### Authentication

#### API Key Not Found
```bash
# Check all sources
repobird config get api-key
echo $REPOBIRD_API_KEY

# Set API key
repobird config set api-key YOUR_KEY
```

#### Invalid API Key
- Verify key at https://app.repobird.ai/settings
- Check for extra spaces/newlines
- Ensure correct environment (prod/dev)

### Network Issues

#### Connection Refused
```bash
# Check API URL
repobird config get api-url

# Test connectivity
curl -I https://api.repobird.ai/health

# Use custom endpoint
export REPOBIRD_API_URL=https://custom.api.url
```

#### Timeout Errors
```bash
# Increase timeout
export REPOBIRD_TIMEOUT=60m

# Check for proxy
export HTTP_PROXY=http://proxy:8080
export HTTPS_PROXY=http://proxy:8080
```

### TUI Problems

#### Display Issues
```bash
# Check terminal
echo $TERM

# Try different terminal
export TERM=xterm-256color

# Reset terminal
reset
```

#### Navigation Not Working
- Check for tmux/screen interference
- Verify terminal supports ANSI escape codes
- Try different terminal emulator

#### Slow Performance
```bash
# Clear cache
rm -rf ~/.config/repobird/cache/

# Disable animations
export REPOBIRD_NO_ANIMATIONS=1
```

### Cache Issues

#### Stale Data
```bash
# Clear all cache
rm -rf ~/.config/repobird/cache/

# Clear specific user cache
rm -rf ~/.config/repobird/cache/users/*/
```

#### Permission Denied
```bash
# Fix permissions
chmod -R 700 ~/.config/repobird/
```

### Build/Installation

#### Command Not Found
```bash
# Check installation
which repobird

# Add to PATH
export PATH=$PATH:$HOME/go/bin

# Reinstall
go install github.com/repobird/cli/cmd/repobird@latest
```

#### Version Mismatch
```bash
# Check version
repobird version

# Update to latest
go install github.com/repobird/cli/cmd/repobird@latest
```

## Debug Techniques

### Enable Verbose Logging
```bash
# Debug environment variable
export REPOBIRD_DEBUG_LOG=1

# Run command
repobird tui

# View logs
tail -f /tmp/repobird_debug.log
```

### Trace API Calls
```bash
# Enable HTTP debugging
export REPOBIRD_HTTP_DEBUG=1

# See all API requests/responses
repobird status --debug
```

### Profile Performance
```bash
# CPU profiling
CPUPROFILE=cpu.prof repobird tui

# Analyze profile
go tool pprof cpu.prof
```

## Error Messages

### API Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `401 Unauthorized` | Invalid API key | Check API key configuration |
| `403 Forbidden` | No access to resource | Verify repository permissions |
| `429 Too Many Requests` | Rate limited | Wait and retry |
| `500 Internal Server Error` | Server issue | Retry later |
| `503 Service Unavailable` | Maintenance | Check status page |

### CLI Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `context deadline exceeded` | Timeout | Increase timeout setting |
| `no such file or directory` | Missing config | Run `repobird config init` |
| `permission denied` | File permissions | Fix with `chmod` |
| `invalid character` | Corrupted JSON | Check task file format |

## Platform-Specific

### macOS
```bash
# Keychain access issues
security unlock-keychain

# Code signing
xattr -d com.apple.quarantine repobird
```

### Linux
```bash
# Missing dependencies
sudo apt-get install ca-certificates

# Keyring issues
sudo apt-get install gnome-keyring
```

### Windows
```powershell
# Path issues
$env:Path += ";C:\Users\$env:USERNAME\go\bin"

# Execution policy
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned
```

## Recovery Steps

### Complete Reset
```bash
# Backup config
cp ~/.repobird/config.yaml ~/.repobird/config.yaml.bak

# Remove all data
rm -rf ~/.repobird/
rm -rf ~/.config/repobird/

# Reinitialize
repobird config init
repobird config set api-key YOUR_KEY
```

### Cache Recovery
```bash
# List cache contents
ls -la ~/.config/repobird/cache/

# Selective cleanup
find ~/.config/repobird/cache/ -name "*.tmp" -delete
```

## Getting Help

### Resources
- GitHub Issues: https://github.com/repobird/cli/issues
- Documentation: https://docs.repobird.ai
- Status Page: https://status.repobird.ai

### Diagnostic Information
When reporting issues, include:
```bash
repobird version
go version
echo $REPOBIRD_API_URL
uname -a
```
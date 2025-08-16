# Configuration Guide

## Overview

Multi-layered configuration system with secure API key management.

## Related Documentation
- **[Development Guide](development-guide.md)** - Environment setup
- **[API Reference](api-reference.md)** - API configuration
- **[Troubleshooting Guide](troubleshooting.md)** - Configuration issues

## Configuration Priority

1. **Command-line flags** - Override all settings
2. **Environment variables** - CI/CD friendly
3. **Configuration file** - Persistent settings
4. **Default values** - Built-in fallbacks

## API Key Setup

### Quick Setup
```bash
# Set via command
repobird config set api-key YOUR_KEY

# Or via environment
export REPOBIRD_API_KEY=YOUR_KEY

# Verify
repobird config get api-key
```

### Storage Methods

**System Keyring (Recommended):**
- Secure native storage
- macOS: Keychain
- Linux: Secret Service
- Windows: Credential Manager

**Encrypted File:**
- Fallback when keyring unavailable
- AES-256-GCM encryption
- Machine-specific key derivation

**Environment Variable:**
```bash
export REPOBIRD_API_KEY=your_key
```

## Configuration File

**Location:**
- Linux/macOS: `~/.repobird/config.yaml`
- Windows: `%USERPROFILE%\.repobird\config.yaml`

**Example:**
```yaml
api_url: https://repobird.ai
timeout: 45m
debug: false
output_format: table
cache:
  enabled: true
  ttl: 5m
tui:
  theme: dark
  refresh_interval: 5s
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `REPOBIRD_API_KEY` | API authentication key | - |
| `REPOBIRD_API_URL` | API endpoint | `https://repobird.ai` |
| `REPOBIRD_ENV` | Environment (prod/dev) | `prod` |
| `REPOBIRD_DEBUG_LOG` | Debug logging (0/1) | `0` |
| `REPOBIRD_TIMEOUT` | Request timeout | `45m` |

## CLI Commands

### Managing Configuration
```bash
# Set values
repobird config set api-key YOUR_KEY
repobird config set api-url https://custom.url

# Get values
repobird config get api-key
repobird config get api-url

# List all
repobird config list

# Reset to defaults
repobird config reset
```

### Validation
```bash
# Test configuration
repobird config test

# Verify API key
repobird verify
```

## Cache Configuration

**Location:**
- `~/.config/repobird/cache/` (Linux/macOS)
- `%LOCALAPPDATA%\repobird\cache\` (Windows)

**Settings:**
```yaml
cache:
  enabled: true
  ttl: 5m          # Memory cache TTL
  max_size: 100MB  # Max cache size
```

**Clear Cache:**
```bash
rm -rf ~/.config/repobird/cache/
```

## Security Best Practices

1. **Never commit API keys** to version control
2. **Use environment variables** in CI/CD
3. **Prefer system keyring** for local storage
4. **Rotate keys regularly**
5. **Use read-only keys** when possible

## Development Configuration

### Local API Server
```bash
export REPOBIRD_API_URL=http://localhost:8080
export REPOBIRD_ENV=dev
```

### Debug Mode
```bash
export REPOBIRD_DEBUG_LOG=1
repobird tui
# Logs to /tmp/repobird_debug.log
```

### Test Environment
```bash
# Use test config directory
export XDG_CONFIG_HOME=/tmp/test-config
repobird config set api-key test_key
```

## Profiles (Future)

```yaml
profiles:
  default:
    api_key: ${REPOBIRD_API_KEY}
    api_url: https://repobird.ai
  
  staging:
    api_key: ${STAGING_API_KEY}
    api_url: https://staging.api.repobird.ai
  
  local:
    api_key: test_key
    api_url: http://localhost:8080
```

Use profile:
```bash
repobird --profile staging status
```

## Troubleshooting

### API Key Not Found
```bash
# Check all sources
repobird config get api-key
echo $REPOBIRD_API_KEY
cat ~/.repobird/config.yaml | grep api_key
```

### Permission Denied
```bash
# Fix config directory permissions
chmod 700 ~/.repobird
chmod 600 ~/.repobird/config.yaml
```

### Keyring Issues
```bash
# Fall back to encrypted file
repobird config set --no-keyring api-key YOUR_KEY
```

See **[Troubleshooting Guide](troubleshooting.md)** for more solutions.
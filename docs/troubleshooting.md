# RepoBird CLI - Troubleshooting Guide

## Quick Diagnostics

Run the diagnostic command to check your setup:
```bash
repobird diagnose
```

This checks:
- API connectivity
- Authentication status
- Configuration validity
- Cache status
- System compatibility

## Common Issues and Solutions

### Authentication Issues

#### Error: "API key not found"
**Symptoms:**
- Commands fail with authentication error
- Cannot access RepoBird API

**Solutions:**
1. Check if API key is set:
```bash
repobird auth info
```

2. Set API key using secure method:
```bash
repobird auth login
# Or
repobird config set api-key
```

3. Verify environment variable:
```bash
echo $REPOBIRD_API_KEY
```

4. Check config file:
```bash
cat ~/.repobird/config.yaml | grep api_key
```

#### Error: "Invalid API key"
**Solutions:**
1. Verify API key is correct:
```bash
repobird auth verify
```

2. Re-login with correct key:
```bash
repobird auth logout
repobird auth login
```

3. Check for extra spaces or characters:
```bash
# Remove and re-add without spaces
repobird config delete api-key
repobird config set api-key
```

### Connection Issues

#### Error: "Connection refused" or "Network error"
**Symptoms:**
- Cannot connect to API
- Timeout errors

**Solutions:**
1. Check internet connectivity:
```bash
ping api.repobird.ai
curl -I https://api.repobird.ai/health
```

2. Check proxy settings:
```bash
echo $HTTP_PROXY
echo $HTTPS_PROXY
```

3. Test with custom endpoint:
```bash
REPOBIRD_API_URL=https://api.repobird.ai repobird status
```

4. Increase timeout:
```bash
repobird config set timeout 60s
# Or
REPOBIRD_TIMEOUT=60s repobird status
```

#### Error: "SSL certificate verification failed"
**Solutions:**
1. Update CA certificates:
```bash
# macOS
brew install ca-certificates

# Linux
sudo apt-get update && sudo apt-get install ca-certificates

# Alpine
apk add --update ca-certificates
```

2. For testing only (NOT for production):
```bash
export SSL_CERT_FILE=/path/to/cert.pem
```

### Rate Limiting

#### Error: "Rate limit exceeded"
**Symptoms:**
- HTTP 429 errors
- "Too many requests" messages

**Solutions:**
1. Check rate limit status:
```bash
repobird auth info
```

2. Wait for reset:
```bash
# Check retry-after header in debug mode
REPOBIRD_DEBUG=true repobird status
```

3. Reduce request frequency:
```bash
# Increase polling interval for TUI
repobird config set tui.refresh_interval 30s
```

### Run Creation Issues

#### Error: "Failed to create run"
**Symptoms:**
- Run creation fails
- Validation errors

**Solutions:**
1. Validate task file:
```bash
# Check JSON syntax
jq . task.json

# Dry run
repobird run task.json --dry-run
```

2. Check required fields:
```json
{
  "prompt": "Required: task description",
  "repository": "Required: org/repo format",
  "source": "Required: branch name",
  "runType": "Required: 'run' or 'approval'"
}
```

3. Verify repository access:
```bash
# Check if repo exists and is accessible
git ls-remote https://github.com/org/repo
```

#### Error: "Repository not found"
**Solutions:**
1. Check repository format:
```bash
# Correct format: org/repo
# Wrong: https://github.com/org/repo
# Wrong: org/repo.git
```

2. Verify repository exists:
```bash
gh repo view org/repo
```

3. Check permissions:
```bash
# Ensure API key has access to repository
repobird auth info
```

#### Duplicate Run Detection

**Understanding Duplicate Detection:**
When you load a task file (JSON, YAML, Markdown, etc.), RepoBird calculates a SHA-256 hash of the file content to detect duplicates.

**Visual Indicators:**
- ✓ **Green** next to Submit button: Ready to submit (not a duplicate)
- ⚠️ **Yellow** next to Submit button: Duplicate detected, but can still submit

**When Submitting a Duplicate:**
Instead of showing an error page, RepoBird shows a friendly prompt:
```
[DUPLICATE] ⚠️ DUPLICATE RUN DETECTED (ID: 123) - Override? [y] yes [n] no
```

**Actions:**
- Press `y` to override and submit the duplicate run
- Press `n` or `ESC` to cancel and return to the form
- The system automatically handles the retry with force flag

**Troubleshooting Duplicate Issues:**

1. **False Positives**: If you get duplicate warnings for different tasks:
```bash
# Clear the file hash cache
rm -rf ~/.cache/repobird/users/user-*/file_hashes.json
```

2. **Cache Out of Sync**: If duplicate detection seems inconsistent:
```bash
# Force cache refresh by restarting the TUI
repobird tui
```

3. **Intentional Duplicates**: To always allow duplicates without prompting:
```bash
# Edit your task file to make it unique (add a comment or timestamp)
# Or use different file names for similar tasks
```

### Cache Issues

#### Stale or Corrupted Cache
**Symptoms:**
- Outdated information displayed
- Inconsistent data
- TUI crashes

**Solutions:**
1. Clear cache:
```bash
repobird cache clear
```

2. Delete cache directory:
```bash
rm -rf ~/.cache/repobird
```

3. Disable cache temporarily:
```bash
repobird status --no-cache
```

4. Disable cache permanently:
```bash
repobird config set cache.enabled false
```

#### User-Specific Cache Issues
**Symptoms:**
- Seeing another user's run data
- Cache not being saved for current user
- Mixed data from different users

**New Cache Structure:**
RepoBird CLI now uses user-specific cache directories to prevent data mixing:
```
~/.cache/repobird/
├── users/
│   ├── user-123/  # User ID-based directories
│   │   ├── runs/
│   │   └── repository_history.json
│   └── user-456/
└── shared/        # Fallback for unknown users
    ├── runs/
    └── repository_history.json
```

**Solutions:**
1. Ensure you're authenticated:
```bash
repobird auth verify
```

2. Clear user-specific cache:
```bash
# Clear only your cache
rm -rf ~/.cache/repobird/users/user-YOUR_ID
```

3. Reset to shared cache:
```bash
# If user ID detection fails, manually clear shared cache
rm -rf ~/.cache/repobird/shared
```

4. Check cache location:
```bash
# Debug which cache directory is being used
REPOBIRD_DEBUG=true repobird status 2>&1 | grep cache
```

### TUI Issues

#### TUI Not Displaying Correctly
**Symptoms:**
- Garbled display
- Missing colors
- Layout issues

**Solutions:**
1. Check terminal compatibility:
```bash
echo $TERM
# Should be xterm-256color or similar
```

2. Set proper terminal:
```bash
export TERM=xterm-256color
```

3. Disable colors if needed:
```bash
export REPOBIRD_NO_COLOR=true
```

4. Try different terminal emulator:
- iTerm2 (macOS)
- Windows Terminal (Windows)
- Alacritty (Cross-platform)

#### TUI Keyboard Shortcuts Not Working
**Solutions:**
1. Check for conflicting terminal shortcuts
2. Try different terminal emulator
3. Disable vim mode if enabled:
```bash
repobird config set tui.vim_mode false
```

### File Permission Issues

#### Error: "Permission denied"
**Symptoms:**
- Cannot read/write config
- Cannot access cache

**Solutions:**
1. Fix directory permissions:
```bash
chmod 755 ~/.repobird
chmod 644 ~/.repobird/config.yaml
```

2. Check ownership:
```bash
ls -la ~/.repobird
# Fix if needed
chown -R $(whoami) ~/.repobird
```

3. Run with proper user:
```bash
# Don't use sudo unless necessary
repobird status  # Good
sudo repobird status  # Usually wrong
```

### Installation Issues

#### Command Not Found
**Solutions:**
1. Check if installed:
```bash
which repobird
```

2. Add to PATH:
```bash
# If installed in ~/go/bin
export PATH=$PATH:~/go/bin

# If installed in /usr/local/bin
export PATH=$PATH:/usr/local/bin
```

3. Reinstall:
```bash
go install github.com/repobird/cli/cmd/repobird@latest
```

#### Version Mismatch
**Solutions:**
1. Check version:
```bash
repobird version
```

2. Update to latest:
```bash
go install github.com/repobird/cli/cmd/repobird@latest
```

3. Clean install:
```bash
go clean -cache
go install github.com/repobird/cli/cmd/repobird@latest
```

### Performance Issues

#### Slow Response Times
**Solutions:**
1. Enable debug to see timing:
```bash
REPOBIRD_DEBUG=true repobird status
```

2. Check network latency:
```bash
ping api.repobird.ai
traceroute api.repobird.ai
```

3. Reduce timeout for faster failures:
```bash
repobird config set timeout 30s
```

4. Use caching:
```bash
repobird config set cache.enabled true
```

#### High Memory Usage
**Solutions:**
1. Clear cache:
```bash
repobird cache clear
```

2. Limit cache size:
```bash
repobird config set cache.max_size 50MB
```

3. Check for memory leaks:
```bash
# Run with profiling
GODEBUG=gctrace=1 repobird status
```

## Platform-Specific Issues

### macOS

#### Keychain Access Issues
**Problem:** Cannot store API key in keychain

**Solutions:**
1. Reset keychain access:
```bash
security unlock-keychain
```

2. Grant terminal access:
- System Preferences → Security & Privacy → Privacy → Full Disk Access
- Add Terminal.app or iTerm.app

3. Use alternative storage:
```bash
export REPOBIRD_API_KEY="your-key"
```

### Windows

#### Path Issues
**Problem:** Paths with spaces cause errors

**Solutions:**
1. Use quotes:
```cmd
repobird run "C:\My Documents\task.json"
```

2. Use short paths:
```cmd
repobird run C:\MYDOCU~1\task.json
```

#### PowerShell Execution Policy
**Problem:** Scripts blocked by execution policy

**Solutions:**
```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

### Linux

#### Library Dependencies
**Problem:** Missing shared libraries

**Solutions:**
1. Install dependencies:
```bash
# Debian/Ubuntu
sudo apt-get install libssl-dev ca-certificates

# RHEL/CentOS
sudo yum install openssl-devel ca-certificates

# Alpine
apk add openssl ca-certificates
```

## Docker Issues

### Container Networking
**Problem:** Cannot connect to API from container

**Solutions:**
1. Use host network:
```bash
docker run --network host repobird-cli status
```

2. Check DNS:
```bash
docker run repobird-cli nslookup api.repobird.ai
```

### Environment Variables
**Problem:** Environment variables not passed to container

**Solutions:**
```bash
docker run -e REPOBIRD_API_KEY=$REPOBIRD_API_KEY repobird-cli status
```

## CI/CD Issues

### GitHub Actions
**Problem:** Authentication fails in workflow

**Solutions:**
1. Check secret is set:
```yaml
- name: Check RepoBird
  env:
    REPOBIRD_API_KEY: ${{ secrets.REPOBIRD_API_KEY }}
  run: repobird status
```

2. Enable debug:
```yaml
- name: Debug RepoBird
  env:
    REPOBIRD_DEBUG: true
    REPOBIRD_API_KEY: ${{ secrets.REPOBIRD_API_KEY }}
  run: repobird auth verify
```

### Jenkins
**Problem:** Credentials not available

**Solutions:**
```groovy
withCredentials([string(credentialsId: 'repobird-api-key', variable: 'REPOBIRD_API_KEY')]) {
    sh 'repobird status'
}
```

## Debug Mode

### Enable Comprehensive Debugging
```bash
# Maximum debug output
export REPOBIRD_DEBUG=true
export REPOBIRD_DEBUG_FILE=/tmp/repobird.log
repobird status --debug

# View debug log
tail -f /tmp/repobird.log
```

### Debug Specific Components
```bash
# API requests only
REPOBIRD_DEBUG_API=true repobird status

# Cache operations only
REPOBIRD_DEBUG_CACHE=true repobird status

# TUI events
REPOBIRD_DEBUG_TUI=true repobird tui 2>tui.log
```

## Getting Help

### Self-Help Resources
1. Check documentation:
```bash
repobird help
repobird help [command]
```

2. View configuration:
```bash
repobird config list
repobird auth info
```

3. Run diagnostics:
```bash
repobird diagnose
```

### Community Support
- GitHub Issues: [github.com/repobird/cli/issues](https://github.com/repobird/cli/issues)
- Discord: [discord.gg/repobird](https://discord.gg/repobird)
- Email: support@repobird.ai

### Reporting Issues

When reporting issues, include:

1. **Version information:**
```bash
repobird version
go version
echo $OSTYPE
```

2. **Debug output:**
```bash
REPOBIRD_DEBUG=true repobird [command] 2>&1 | tee debug.log
```

3. **Configuration (sanitized):**
```bash
repobird config list | sed 's/api_key: .*/api_key: REDACTED/'
```

4. **Steps to reproduce:**
- Exact commands run
- Expected behavior
- Actual behavior
- Error messages

5. **Environment:**
- Operating system
- Terminal emulator
- Shell (bash/zsh/fish)
- Proxy settings (if applicable)

## Recovery Procedures

### Complete Reset
```bash
# Backup current config
cp -r ~/.repobird ~/.repobird.backup

# Remove all RepoBird data
rm -rf ~/.repobird

# Reinstall
go install github.com/repobird/cli/cmd/repobird@latest

# Reconfigure
repobird auth login
```

### Partial Reset
```bash
# Reset configuration only
repobird config reset

# Reset cache only
repobird cache clear

# Reset credentials only
repobird auth logout
repobird auth login
```

## Preventive Measures

### Regular Maintenance
```bash
# Weekly cache cleanup
repobird cache clear

# Update CLI monthly
go install github.com/repobird/cli/cmd/repobird@latest

# Verify setup quarterly
repobird diagnose
```

### Backup Configuration
```bash
# Create backup
tar -czf repobird-backup.tar.gz ~/.repobird

# Restore from backup
tar -xzf repobird-backup.tar.gz -C ~/
```

### Monitor Usage
```bash
# Check quota
repobird auth info

# Monitor rate limits
REPOBIRD_DEBUG=true repobird status 2>&1 | grep -i rate
```
# RepoBird CLI - Configuration Guide

## Configuration Overview

RepoBird CLI uses a layered configuration system that provides flexibility and security. Configuration values are resolved in the following priority order (highest to lowest):

1. **Command-line flags** - Override all other settings
2. **Environment variables** - Useful for CI/CD and containers
3. **Configuration file** - Persistent user settings
4. **Default values** - Built-in fallbacks

## Configuration File

### Location
The configuration file is stored at:
- **Linux/macOS**: `~/.repobird/config.yaml`
- **Windows**: `%USERPROFILE%\.repobird\config.yaml`

### File Format
```yaml
# ~/.repobird/config.yaml
api_key: your-api-key-here
api_url: https://api.repobird.ai
timeout: 45m
debug: false
output_format: table
auto_update: true
cache:
  enabled: true
  ttl: 30s
  max_size: 100MB
tui:
  theme: dark
  vim_mode: true
  refresh_interval: 5s
```

## Environment Variables

All configuration options can be set via environment variables with the `REPOBIRD_` prefix:

| Variable | Description | Default |
|----------|-------------|---------|
| `REPOBIRD_API_KEY` | API authentication key | - |
| `REPOBIRD_API_URL` | API endpoint URL | `https://api.repobird.ai` |
| `REPOBIRD_ENV` | Environment (prod/dev) - affects frontend URL generation | `prod` |
| `REPOBIRD_DEBUG` | Enable debug logging | `false` |
| `REPOBIRD_TIMEOUT` | Request timeout | `45m` |
| `REPOBIRD_OUTPUT_FORMAT` | Output format (table/json/yaml) | `table` |
| `REPOBIRD_CACHE_ENABLED` | Enable caching | `true` |
| `REPOBIRD_CACHE_TTL` | Cache time-to-live | `30s` |
| `REPOBIRD_NO_COLOR` | Disable colored output | `false` |
| `REPOBIRD_CONFIG_PATH` | Custom config file path | `~/.repobird/config.yaml` |

### Setting Environment Variables

#### Linux/macOS
```bash
# Temporary (current session)
export REPOBIRD_API_KEY="your-key-here"
export REPOBIRD_ENV=dev
export REPOBIRD_DEBUG=true

# Permanent (add to ~/.bashrc or ~/.zshrc)
echo 'export REPOBIRD_API_KEY="your-key-here"' >> ~/.bashrc
echo 'export REPOBIRD_ENV=dev' >> ~/.bashrc
source ~/.bashrc
```

#### Windows
```powershell
# Temporary (current session)
$env:REPOBIRD_API_KEY = "your-key-here"
$env:REPOBIRD_ENV = "dev"
$env:REPOBIRD_DEBUG = "true"

# Permanent (user level)
[System.Environment]::SetEnvironmentVariable("REPOBIRD_API_KEY", "your-key-here", "User")
[System.Environment]::SetEnvironmentVariable("REPOBIRD_ENV", "dev", "User")
```

## Environment-Specific Configuration

### REPOBIRD_ENV Variable

The `REPOBIRD_ENV` environment variable controls environment-specific behavior:

#### Production (Default)
```bash
export REPOBIRD_ENV=prod
# or leave unset (defaults to production)
```
- **Frontend URLs**: Generated URLs point to `https://repobird.ai`
- **Example**: Run ID 927 → `https://repobird.ai/repos/issue-runs/927`

#### Development
```bash
export REPOBIRD_ENV=dev
```
- **Frontend URLs**: Generated URLs point to `http://localhost:3000`
- **Example**: Run ID 927 → `http://localhost:3000/repos/issue-runs/927`

### Makefile Integration

The project Makefile automatically sets the appropriate environment:
```bash
# Development commands (set REPOBIRD_ENV=dev automatically)
make build          # Development build
make test           # Run tests in dev mode
make run            # Run application in dev mode
make tui            # Launch TUI in dev mode

# Production commands (set REPOBIRD_ENV=prod)
make build-prod     # Production build
make build-all-prod # Production build for all platforms
```

### URL Generation Behavior

When using the CLI's URL generation features (such as the 'o' key in the dashboard to open run URLs), the environment setting affects which frontend URLs are generated:

- **Production**: URLs open to the live RepoBird web application
- **Development**: URLs open to your local development server

This allows developers to seamlessly work with local frontend instances while maintaining production behavior for end users.

## API Key Management

### Secure Storage Methods

RepoBird CLI supports multiple secure storage methods for API keys, automatically selecting the most secure option available:

#### 1. System Keyring (Most Secure)
The system's native credential storage:
- **macOS**: Keychain Access
- **Windows**: Windows Credential Manager
- **Linux**: Secret Service (GNOME Keyring, KWallet)

```bash
# Set API key (stored securely)
repobird config set api-key

# Check storage method
repobird auth info
```

#### 2. Encrypted File Storage
When keyring is unavailable, uses AES-256-GCM encryption:
- Key derivation from machine-specific data
- Stored in `~/.repobird/.secure/`
- Automatic migration from plain text

#### 3. Environment Variable
Best for CI/CD and containerized environments:
```bash
export REPOBIRD_API_KEY="your-key-here"
```

### API Key Commands

#### Setting API Key
```bash
# Interactive secure input
repobird config set api-key
Enter API key: [hidden input]

# From environment
REPOBIRD_API_KEY="key" repobird status

# Via login command
repobird auth login
```

#### Verifying API Key
```bash
# Check if key is valid
repobird auth verify

# Show account info
repobird auth info
```

#### Removing API Key
```bash
# Remove from all storage locations
repobird auth logout

# Remove specific config value
repobird config delete api-key
```

## Configuration Commands

### View Configuration
```bash
# Show all configuration
repobird config list

# Get specific value
repobird config get api-key
repobird config get timeout

# Show configuration file location
repobird config path
```

### Set Configuration
```bash
# Set values
repobird config set timeout 30m
repobird config set debug true
repobird config set output-format json

# Set with validation
repobird config set api-url https://custom.api.url
```

### Delete Configuration
```bash
# Remove specific value
repobird config delete debug

# Reset to defaults
repobird config reset

# Clear all configuration
repobird config clear --confirm
```

## Output Formats

### Table Format (Default)
```bash
repobird status
# ┌──────────┬───────────┬────────────┬──────────┐
# │ ID       │ Status    │ Repository │ Time     │
# ├──────────┼───────────┼────────────┼──────────┤
# │ abc123   │ ✓ success │ org/repo   │ 2m ago   │
# └──────────┴───────────┴────────────┴──────────┘
```

### JSON Format
```bash
repobird status --output json
# or
export REPOBIRD_OUTPUT_FORMAT=json
repobird status
```

Output:
```json
{
  "runs": [
    {
      "id": "abc123",
      "status": "success",
      "repository": "org/repo",
      "created_at": "2024-01-01T12:00:00Z"
    }
  ]
}
```

### YAML Format
```bash
repobird status --output yaml
```

Output:
```yaml
runs:
  - id: abc123
    status: success
    repository: org/repo
    created_at: 2024-01-01T12:00:00Z
```

### Plain Format
```bash
repobird status --output plain
# abc123 success org/repo 2024-01-01T12:00:00Z
```

## Cache Configuration

### Cache Settings
```yaml
# ~/.repobird/config.yaml
cache:
  enabled: true          # Enable/disable caching
  ttl: 30s              # Time-to-live for cache entries
  max_size: 100MB       # Maximum cache size
  location: ~/.repobird/cache  # Cache directory
  persistent: true      # Keep cache between sessions
```

### Cache Management
```bash
# Clear cache
repobird cache clear

# Show cache statistics
repobird cache stats

# Disable cache temporarily
repobird status --no-cache

# Disable cache globally
repobird config set cache.enabled false
```

## TUI Configuration

### TUI Settings
```yaml
# ~/.repobird/config.yaml
tui:
  theme: dark           # dark, light, auto
  vim_mode: true       # Enable vim keybindings
  refresh_interval: 5s  # Auto-refresh interval
  show_help: true      # Show help bar
  compact: false       # Compact view mode
  colors:
    primary: "#007ACC"
    success: "#00FF00"
    error: "#FF0000"
    warning: "#FFA500"
```

### TUI Keybindings
```yaml
# ~/.repobird/config.yaml
tui:
  keybindings:
    quit: "q"
    help: "?"
    refresh: "r"
    search: "/"
    new_run: "n"
    navigate_up: "k"
    navigate_down: "j"
    page_up: "ctrl+u"
    page_down: "ctrl+d"
```

## Debug Configuration

### Enable Debug Mode
```bash
# Via environment
export REPOBIRD_DEBUG=true

# Via config file
repobird config set debug true

# Via command flag
repobird status --debug
```

### Debug Output Location
```bash
# Set custom debug log file
export REPOBIRD_DEBUG_FILE=/tmp/repobird.log

# Or in config
repobird config set debug_file /tmp/repobird.log
```

### Debug Levels
```yaml
# ~/.repobird/config.yaml
debug:
  enabled: true
  level: trace  # trace, debug, info, warn, error
  file: /tmp/repobird.log
  include_timestamps: true
  include_caller: true
```

## Proxy Configuration

### HTTP/HTTPS Proxy
```bash
# Via environment variables
export HTTP_PROXY=http://proxy.company.com:8080
export HTTPS_PROXY=http://proxy.company.com:8080
export NO_PROXY=localhost,127.0.0.1

# Or in config file
```

```yaml
# ~/.repobird/config.yaml
proxy:
  http: http://proxy.company.com:8080
  https: http://proxy.company.com:8080
  no_proxy: localhost,127.0.0.1
```

### SOCKS Proxy
```bash
export ALL_PROXY=socks5://localhost:1080
```

## Profile Management

### Multiple Profiles
```bash
# Create profiles directory
mkdir -p ~/.repobird/profiles

# Create work profile
cat > ~/.repobird/profiles/work.yaml << EOF
api_key: work-api-key
api_url: https://work.repobird.ai
EOF

# Create personal profile
cat > ~/.repobird/profiles/personal.yaml << EOF
api_key: personal-api-key
api_url: https://api.repobird.ai
EOF
```

### Using Profiles
```bash
# Use specific profile
repobird --profile work status
repobird --profile personal run task.json

# Set default profile
export REPOBIRD_PROFILE=work

# Or in config
repobird config set default_profile work
```

## Advanced Configuration

### Custom API Endpoints
```yaml
# ~/.repobird/config.yaml
api:
  base_url: https://custom.api.url
  version: v2
  endpoints:
    runs: /custom/runs
    auth: /custom/auth
```

### Retry Configuration
```yaml
# ~/.repobird/config.yaml
retry:
  max_attempts: 5
  initial_delay: 1s
  max_delay: 30s
  multiplier: 2
  jitter: 0.1
```

### Timeout Configuration
```yaml
# ~/.repobird/config.yaml
timeouts:
  default: 45m
  create_run: 5m
  get_status: 30s
  list_runs: 1m
  auth_verify: 10s
```

### Rate Limiting
```yaml
# ~/.repobird/config.yaml
rate_limit:
  requests_per_second: 10
  burst: 20
  wait_on_limit: true
```

## Configuration for CI/CD

### GitHub Actions
```yaml
# .github/workflows/repobird.yml
env:
  REPOBIRD_API_KEY: ${{ secrets.REPOBIRD_API_KEY }}
  REPOBIRD_OUTPUT_FORMAT: json
  REPOBIRD_NO_COLOR: true
  REPOBIRD_CACHE_ENABLED: false
```

### GitLab CI
```yaml
# .gitlab-ci.yml
variables:
  REPOBIRD_API_KEY: ${REPOBIRD_API_KEY}
  REPOBIRD_OUTPUT_FORMAT: json
  REPOBIRD_NO_COLOR: "true"
```

### Jenkins
```groovy
// Jenkinsfile
environment {
    REPOBIRD_API_KEY = credentials('repobird-api-key')
    REPOBIRD_OUTPUT_FORMAT = 'json'
    REPOBIRD_NO_COLOR = 'true'
}
```

### Docker
```dockerfile
# Dockerfile
ENV REPOBIRD_API_KEY=""
ENV REPOBIRD_API_URL="https://api.repobird.ai"
ENV REPOBIRD_CACHE_ENABLED="false"
ENV REPOBIRD_NO_COLOR="true"
```

```bash
# Docker run
docker run -e REPOBIRD_API_KEY=$API_KEY repobird-cli status
```

## Configuration Validation

### Validate Configuration File
```bash
# Check syntax and values
repobird config validate

# Validate specific file
repobird config validate --file custom-config.yaml
```

### Test Configuration
```bash
# Test API connection
repobird config test

# Verbose test
repobird config test --verbose
```

## Migration and Backup

### Backup Configuration
```bash
# Backup current configuration
repobird config backup

# Backup to specific location
repobird config backup --output ~/backup/repobird-config.tar.gz
```

### Restore Configuration
```bash
# Restore from backup
repobird config restore ~/backup/repobird-config.tar.gz

# Restore with merge
repobird config restore --merge backup.tar.gz
```

### Migrate Configuration
```bash
# Migrate from old version
repobird config migrate

# Migrate from different tool
repobird config import --from other-tool-config.json
```

## Security Best Practices

### 1. API Key Security
- Never commit API keys to version control
- Use environment variables in CI/CD
- Rotate keys regularly
- Use read-only keys when possible

### 2. File Permissions
```bash
# Secure config directory
chmod 700 ~/.repobird
chmod 600 ~/.repobird/config.yaml
```

### 3. Encryption
- API keys are automatically encrypted when stored
- Use system keyring when available
- Enable transport encryption (HTTPS)

### 4. Audit
```bash
# Show configuration access log
repobird config audit

# Clear sensitive data
repobird config clear-sensitive
```

## Troubleshooting Configuration

### Common Issues

#### API Key Not Found
```bash
# Check all locations
repobird auth info

# Verify environment
echo $REPOBIRD_API_KEY

# Check config file
cat ~/.repobird/config.yaml | grep api_key
```

#### Permission Denied
```bash
# Fix permissions
chmod 755 ~/.repobird
chmod 644 ~/.repobird/config.yaml
```

#### Invalid Configuration
```bash
# Validate config
repobird config validate

# Reset to defaults
repobird config reset

# Recreate config
rm -rf ~/.repobird
repobird config init
```

### Debug Configuration Loading
```bash
# Show configuration source
REPOBIRD_DEBUG=true repobird config list

# Trace configuration resolution
repobird --trace config get api-key
```

## Configuration Schema

### Complete Configuration Reference
```yaml
# Complete ~/.repobird/config.yaml example
api_key: string                    # API authentication key
api_url: string                    # API base URL
timeout: duration                  # Request timeout (e.g., 45m, 30s)
debug: boolean                     # Enable debug mode
output_format: enum                # table|json|yaml|plain
no_color: boolean                  # Disable colored output
auto_update: boolean               # Auto-update CLI

cache:
  enabled: boolean                 # Enable caching
  ttl: duration                    # Cache time-to-live
  max_size: string                 # Max cache size (e.g., 100MB)
  location: string                 # Cache directory path
  persistent: boolean              # Persist cache between sessions

tui:
  theme: enum                      # dark|light|auto
  vim_mode: boolean                # Enable vim keybindings
  refresh_interval: duration       # Auto-refresh interval
  show_help: boolean               # Show help bar
  compact: boolean                 # Compact view mode

proxy:
  http: string                     # HTTP proxy URL
  https: string                    # HTTPS proxy URL
  no_proxy: string                 # Comma-separated no-proxy hosts

retry:
  max_attempts: integer            # Maximum retry attempts
  initial_delay: duration          # Initial retry delay
  max_delay: duration              # Maximum retry delay
  multiplier: float                # Backoff multiplier
  jitter: float                    # Jitter factor (0-1)

rate_limit:
  requests_per_second: integer     # Rate limit
  burst: integer                   # Burst capacity
  wait_on_limit: boolean           # Wait when rate limited

timeouts:
  default: duration                # Default timeout
  create_run: duration             # Create run timeout
  get_status: duration             # Get status timeout
  list_runs: duration              # List runs timeout
  auth_verify: duration            # Auth verify timeout
```
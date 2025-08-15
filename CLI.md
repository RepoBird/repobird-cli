# RepoBird CLI Tool - Design & Implementation Plan

## Executive Summary

The RepoBird CLI will be a fast, cross-platform command-line tool built in **Go** that enables users to trigger AI agent runs in the cloud directly from their terminal. The tool will support multiple input formats (JSON, JSONL, YAML, TOML, Markdown) and provide both CLI and TUI modes with vim keybindings for efficient navigation.

## Language Choice: Go

### Why Go?

After thorough analysis, **Go** is the optimal choice for this CLI tool based on the following factors:

1. **Performance**: Compiled language with fast execution and minimal runtime overhead
2. **Cross-Platform Distribution**: Produces single static binaries that work across all major platforms (Windows, macOS, Linux)
3. **Ecosystem Maturity**: Extensive libraries for CLI/TUI development, file format parsing, and HTTP API integration
4. **Open Source Friendly**: Clear, maintainable code that's easy for contributors to understand
5. **Native Support**: Built-in support for JSON, excellent third-party libraries for YAML, TOML, and Markdown
6. **Production Ready**: Used by major CLI tools (Docker, Kubernetes, GitHub CLI)

### Language Comparison

| Language | Pros | Cons | Decision |
|----------|------|------|----------|
| **Go** | Fast, cross-platform binaries, mature ecosystem, excellent CLI frameworks | Verbose syntax | ✅ **Selected** |
| Ruby | Elegant syntax, enjoyable to write | Slower execution, complex cross-platform distribution | ❌ |
| Nim/Crystal | Fast, produces binaries | Limited ecosystem, smaller community, less mature TUI libraries | ❌ |
| Zig | Fast, powerful | Young ecosystem for CLI/TUI tools | ❌ |
| Python | Large ecosystem | Slower, complex distribution | ❌ |
| Rust | Fast, safe | Too complex for requirements | ❌ |

## Core Requirements

### Functional Requirements

1. **Input Format Support**
   - JSON task files with configuration
   - JSONL batch files for bulk operations
   - YAML/TOML for better readability with markdown-formatted prompts
   - Plain Markdown/TXT files for simple prompts (configuration via flags)

2. **Agent Run Management**
   - Trigger new agent runs via RepoBird API
   - Display real-time status of current/past runs
   - Support for canceling active runs
   - Retrieve run logs and outputs

3. **Authentication**
   - API key-based authentication
   - Secure storage of API keys (keyring integration)
   - Support for multiple profiles/accounts

4. **User Interface**
   - CLI mode for scripting and automation
   - TUI mode for interactive use
   - Vim keybindings (h,j,k,l, /, :, etc.) with arrow key fallback
   - Real-time status updates in TUI mode

### Non-Functional Requirements

- **Performance**: Sub-100ms startup time, responsive UI
- **Size**: Binary under 20MB
- **Platform Support**: Windows, macOS (Intel/ARM), Linux (x64/ARM)
- **Installation**: Single binary download, Homebrew, apt/yum packages
- **Documentation**: Comprehensive help text, man pages, online docs

## Technical Architecture

### Framework Selection

#### CLI Framework: Cobra + Viper
**Cobra** is chosen for its:
- Sophisticated command and subcommand structure
- Automatic help generation and documentation
- Native integration with Viper for configuration
- Used by kubectl, GitHub CLI, and other major tools

#### TUI Framework: Bubbletea + Lipgloss
**Bubbletea** for TUI because it:
- Provides modern, reactive terminal UI
- Supports custom keybindings (vim-style)
- Has excellent documentation and community
- Pairs well with Lipgloss for styling

#### Configuration: Viper
**Viper** handles:
- Multiple config format support (JSON, YAML, TOML)
- Environment variable binding
- Config file precedence
- Live config reloading

### Project Structure

```
repobird-cli/
├── cmd/
│   └── repobird/
│       ├── main.go           # Entry point
│       ├── root.go           # Root command setup
│       ├── run.go            # Run command
│       ├── status.go         # Status command
│       ├── config.go         # Config management
│       └── tui.go            # TUI mode entry
├── internal/
│   ├── api/
│   │   ├── client.go         # RepoBird API client
│   │   ├── models.go         # API data models
│   │   └── auth.go           # Authentication handling
│   ├── config/
│   │   ├── loader.go         # Config file loading
│   │   ├── parser.go         # Format-specific parsers
│   │   └── validator.go      # Config validation
│   ├── tui/
│   │   ├── app.go            # Main TUI application
│   │   ├── views/            # TUI views/screens
│   │   ├── components/       # Reusable TUI components
│   │   └── keybindings.go    # Vim keybinding setup
│   ├── core/
│   │   ├── runner.go         # Core run logic
│   │   ├── processor.go      # Input processing
│   │   └── status.go         # Status tracking
│   └── utils/
│       ├── format.go         # Output formatting
│       └── logger.go         # Logging utilities
├── pkg/
│   └── version/              # Version information
├── go.mod
├── go.sum
├── Makefile                   # Build automation
└── README.md

```

## Command Structure

### Primary Commands

```bash
# Run a new agent task
repobird run [flags] <input-file>
  -f, --format string     Input format (auto-detect by default)
  -c, --config string     Additional config file
  -w, --watch            Watch for status updates
  --repo string          Target repository (owner/name)
  --branch string        Target branch
  --dry-run              Validate without executing

# Check status of runs
repobird status [run-id]
  -a, --all              Show all runs
  -l, --limit int        Number of runs to show (default: 10)
  -f, --follow           Follow log output

# Launch TUI mode
repobird tui

# Configuration management
repobird config
  config init            Initialize configuration
  config set <key> <value>
  config get <key>
  config list

# Authentication
repobird auth
  auth login             Interactive login
  auth logout
  auth status            Show current auth status
```

### Input File Examples

#### JSON Task File
```json
{
  "prompt": "Implement user authentication with JWT",
  "repository": "acme/webapp",
  "branch": "feature/auth",
  "config": {
    "timeout": 3600,
    "model": "claude-sonnet-4"
  }
}
```

#### YAML Configuration
```yaml
prompt: |
  Implement user authentication with JWT tokens.
  Include refresh token mechanism and secure storage.
  
repository: acme/webapp
branch: feature/auth
config:
  timeout: 3600
  model: claude-sonnet-4
  notifications:
    email: true
    slack: false
```

#### JSONL Batch File
```jsonl
{"prompt": "Fix login bug", "repository": "acme/webapp", "priority": "high"}
{"prompt": "Add password reset", "repository": "acme/webapp", "priority": "medium"}
{"prompt": "Implement 2FA", "repository": "acme/webapp", "priority": "low"}
```

## API Integration

### Authentication Flow

```go
// API client initialization
type Client struct {
    baseURL    string
    apiKey     string
    httpClient *http.Client
}

func NewClient(apiKey string) *Client {
    return &Client{
        baseURL:    "https://repobird.ai/api/v1",
        apiKey:     apiKey,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

// Request with authentication
func (c *Client) doRequest(method, path string, body interface{}) (*http.Response, error) {
    req, err := http.NewRequest(method, c.baseURL+path, body)
    if err != nil {
        return nil, err
    }
    req.Header.Set("Authorization", "Bearer "+c.apiKey)
    req.Header.Set("Content-Type", "application/json")
    return c.httpClient.Do(req)
}
```

### API Endpoints

```
POST   /api/v1/runs               Create new agent run
GET    /api/v1/runs               List all runs
GET    /api/v1/runs/{id}          Get run details
GET    /api/v1/runs/{id}/logs     Stream run logs
DELETE /api/v1/runs/{id}          Cancel run
GET    /api/v1/user               Get user info
GET    /api/v1/repositories       List available repositories
```

## TUI Design

### Main View Components

1. **Run List View**
   - Table of recent runs with status
   - Vim navigation (j/k to move, Enter to view details)
   - Real-time status updates

2. **Run Details View**
   - Full run information
   - Log streaming
   - Actions (cancel, retry, view PR)

3. **New Run View**
   - File selector or text input
   - Repository/branch selection
   - Configuration options

### Keybindings

```
Navigation:
  j/↓     Move down
  k/↑     Move up
  h/←     Go back
  l/→     Go forward/select
  g       Go to top
  G       Go to bottom
  /       Search
  n       Next search result
  N       Previous search result

Actions:
  Enter   Select/confirm
  Esc     Cancel/back
  r       Refresh
  n       New run
  s       View status
  q       Quit
  ?       Help

Command mode:
  :       Enter command mode
  :q      Quit
  :w      Save (in editor)
  :run    Execute run
```

## Implementation Roadmap

### Phase 1: Core CLI (Week 1-2)
- [ ] Project setup with Go modules
- [ ] Cobra command structure
- [ ] Basic run command with JSON support
- [ ] API client implementation
- [ ] Status command

### Phase 2: Multi-Format Support (Week 3)
- [ ] YAML/TOML parsing with Viper
- [ ] JSONL batch processing
- [ ] Markdown/text file support
- [ ] Configuration management

### Phase 3: TUI Implementation (Week 4-5)
- [ ] Bubbletea setup
- [ ] Run list view
- [ ] Run details with log streaming
- [ ] Vim keybindings
- [ ] Interactive run creation

### Phase 4: Polish & Distribution (Week 6)
- [ ] Error handling and recovery
- [ ] Comprehensive testing
- [ ] Documentation and help text
- [ ] Cross-platform builds
- [ ] Package managers (Homebrew, apt, yum)
- [ ] GitHub releases automation

## Dependencies

### Core Dependencies
```go
// go.mod
module github.com/repobird/repobird-cli

go 1.21

require (
    github.com/spf13/cobra v1.8.0
    github.com/spf13/viper v1.18.0
    github.com/charmbracelet/bubbletea v0.25.0
    github.com/charmbracelet/lipgloss v0.9.1
    github.com/charmbracelet/bubbles v0.17.1
)

// File parsing
require (
    gopkg.in/yaml.v3 v3.0.1
    github.com/BurntSushi/toml v1.3.2
    github.com/russross/blackfriday/v2 v2.1.0
)

// Utilities
require (
    github.com/fatih/color v1.16.0
    github.com/olekukonko/tablewriter v0.0.5
    github.com/briandowns/spinner v1.23.0
)
```

## Security Considerations

1. **API Key Storage**
   - Use OS keyring for secure storage (keyring library)
   - Never log or display full API keys
   - Support environment variable override

2. **Input Validation**
   - Sanitize all user inputs
   - Validate file formats before processing
   - Size limits on input files

3. **Network Security**
   - TLS 1.3 for all API communications
   - Certificate pinning option
   - Proxy support with authentication

## Performance Targets

- **Startup Time**: < 100ms
- **API Response**: < 500ms for standard operations
- **TUI Refresh Rate**: 60 FPS
- **Memory Usage**: < 50MB for typical operations
- **Binary Size**: < 20MB compressed

## Testing Strategy

### Unit Tests
- Core logic (input parsing, API client)
- Configuration management
- Format converters

### Integration Tests
- API communication
- File I/O operations
- Multi-format processing

### E2E Tests
- Full command workflows
- TUI interaction testing
- Cross-platform validation

## Documentation Plan

1. **README.md**: Quick start, installation, basic usage
2. **Man Pages**: Comprehensive command documentation
3. **Online Docs**: Detailed guides and examples
4. **In-app Help**: Context-sensitive help in TUI
5. **Video Tutorials**: Screen recordings of common workflows

## Open Source Strategy

1. **License**: MIT License for maximum adoption
2. **Contributing Guidelines**: Clear contribution process
3. **Issue Templates**: Bug reports, feature requests
4. **CI/CD**: GitHub Actions for testing and releases
5. **Community**: Discord/Slack channel for support

## Success Metrics

- **Adoption**: 1000+ downloads in first month
- **Performance**: All operations under target times
- **Reliability**: < 0.1% crash rate
- **User Satisfaction**: > 4.5 star average on GitHub
- **Cross-Platform**: Works on 95% of target systems

## Conclusion

The RepoBird CLI will be built with Go using Cobra for CLI structure and Bubbletea for TUI functionality. This combination provides the perfect balance of performance, developer experience, and user functionality. The tool will support multiple input formats, provide both CLI and TUI modes, and integrate seamlessly with the RepoBird API to enable efficient AI agent management from the terminal.
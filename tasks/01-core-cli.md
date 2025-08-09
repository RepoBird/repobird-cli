# Phase 1: Core CLI Implementation (Week 1-2)

## Overview
Establish the foundational CLI structure and basic functionality for RepoBird CLI.

## Tasks

### Project Setup
- [ ] Initialize Go module (`go mod init github.com/repobird/repobird-cli`)
- [ ] Set up basic project structure (cmd/, internal/, pkg/)
- [ ] Configure .gitignore for Go projects
- [ ] Set up GitHub repository with README
- [ ] Create initial CI/CD workflow (GitHub Actions)

### Cobra Command Structure
- [ ] Install and configure Cobra framework
- [ ] Create main.go entry point
- [ ] Implement root command with global flags
- [ ] Set up command hierarchy structure
- [ ] Add version command with build info
- [ ] Implement help system and documentation

### Basic Run Command
- [ ] Create `run` command structure
- [ ] Implement JSON input file parsing
- [ ] Add basic validation for JSON schema
- [ ] Handle file reading errors gracefully
- [ ] Support stdin input for JSON
- [ ] Add --dry-run flag for validation only

### API Client Implementation
- [ ] Create HTTP client wrapper
- [ ] Implement Bearer token authentication (from dashboard)
- [ ] Support REPOBIRD_API_URL env var for dev override
- [ ] Add request/response models matching issueRunSchema
- [ ] Implement POST /api/v1/runs endpoint
- [ ] Implement GET /api/v1/auth/verify for API key validation
- [ ] Add proper error handling (no runs remaining, repo not found)
- [ ] Support timeout configuration (default 45 min)
- [ ] Add request logging for debugging
- [ ] Auto-detect repository from git config if not specified

### Status Command
- [ ] Create `status` command structure
- [ ] Implement GET /api/v1/runs/{id} endpoint
- [ ] Map status enums (QUEUED, INITIALIZING, PROCESSING, POST_PROCESS, DONE, FAILED)
- [ ] Add list all runs functionality with pagination
- [ ] Format output in table view with human-readable status
- [ ] Support JSON output format
- [ ] Add --follow flag with 5-second polling interval
- [ ] Stop polling when status is DONE or FAILED
- [ ] Display remaining runs from user tier info

### Configuration Foundation
- [ ] Set up Viper for config management
- [ ] Create default config structure
- [ ] Support environment variables (REPOBIRD_API_KEY, REPOBIRD_API_URL)
- [ ] Implement config file locations (~/.repobird/)
- [ ] Add API endpoint configuration (default: https://api.repobird.ai)
- [ ] Store API key securely (never in plain text)
- [ ] Create config validation
- [ ] Cache user tier info for offline usage checks

## Input File Example (Phase 1)

```json
{
  "prompt": "Fix the login authentication bug",
  "repository": "acme/webapp",
  "source": "main",
  "target": "fix/login-bug",
  "runType": "run",
  "title": "Fix login authentication",
  "context": "Additional context about the bug",
  "files": ["src/auth.js", "src/login.js"]
}
```

## Command Examples (Phase 1)

```bash
# Run a task
repobird run task.json
repobird run task.json --dry-run

# Check status
repobird status
repobird status abc123
repobird status --all --limit 20

# Version info
repobird version
```

## Testing Requirements

### Unit Tests
- [ ] Command parsing tests
- [ ] JSON validation tests
- [ ] API client mocking
- [ ] Error handling scenarios

### Integration Tests
- [ ] End-to-end command flow
- [ ] File I/O operations
- [ ] Configuration loading

## Deliverables

1. Working CLI with basic commands
2. JSON input support
3. API integration for run creation and status
4. Basic error handling and validation
5. Unit test coverage > 70%

## Success Criteria

- [ ] Can create a run from JSON file
- [ ] Can check status of runs
- [ ] Proper error messages for common failures
- [ ] Commands complete in < 1 second
- [ ] Clean, documented code structure
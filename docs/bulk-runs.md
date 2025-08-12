# Bulk Runs

Bulk runs allow you to execute multiple AI-powered tasks simultaneously, streamlining workflows where you need to process many similar requests or apply multiple changes to a repository.

## Overview

The RepoBird CLI supports creating and managing bulk runs through both the command-line interface and the Terminal User Interface (TUI). Bulk runs are useful for:

- Processing multiple bug fixes or feature requests at once
- Applying consistent changes across multiple files or components
- Running batch operations during maintenance windows
- Executing related tasks in parallel to save time

## Quick Start

### CLI Usage

```bash
# Create bulk runs from a JSON configuration file
repobird bulk run config.json

# Follow progress in real-time
repobird bulk run config.json --follow

# Check status of a bulk run batch
repobird bulk status BATCH_ID

# Cancel all runs in a batch
repobird bulk cancel BATCH_ID
```

### TUI Usage

```bash
# Launch the TUI and navigate to bulk runs
repobird tui

# Or directly start bulk run creation
repobird bulk tui
```

## Configuration File Format

Bulk runs are defined using JSON configuration files with the following structure:

```json
{
  "repositoryName": "owner/repo-name",
  "batchTitle": "Q1 2024 Bug Fixes",
  "runType": "run",
  "sourceBranch": "main",
  "force": false,
  "runs": [
    {
      "prompt": "Fix authentication timeout bug in login.js",
      "title": "Auth timeout fix",
      "context": "Users report getting logged out after 5 minutes",
      "target": "fix/auth-timeout"
    },
    {
      "prompt": "Update user profile validation to handle special characters",
      "title": "Profile validation update",
      "target": "fix/profile-validation"
    }
  ]
}
```

### Configuration Fields

#### Root Level Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repositoryName` | string | Yes | Repository in `owner/repo` format |
| `batchTitle` | string | No | Descriptive title for the entire batch |
| `runType` | string | No | Either `run` (default) or `plan` |
| `sourceBranch` | string | No | Source branch (defaults to repo default) |
| `force` | boolean | No | Override duplicate detection (default: false) |
| `runs` | array | Yes | Array of individual run configurations |

#### Run Item Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `prompt` | string | Yes | The task description for the AI agent |
| `title` | string | No | Custom title (auto-generated from prompt if omitted) |
| `context` | string | No | Additional context for the task |
| `target` | string | No | Target branch name for this specific run |
| `fileHash` | string | No | SHA-256 hash for duplicate detection |

## API Integration

The bulk runs feature integrates with the RepoBird API using the `/api/v1/runs/bulk` endpoint. The API specification is documented in `docs/CLI_API_SPECIFICATION.yaml`.

### Request Structure

The CLI transforms your configuration into an API request with the following structure:

```json
{
  "repositoryName": "owner/repo",
  "batchTitle": "Optional batch title",
  "runType": "run",
  "sourceBranch": "main",
  "force": false,
  "runs": [
    {
      "prompt": "Task description",
      "title": "Optional title",
      "target": "optional-target-branch",
      "context": "Additional context",
      "fileHash": "optional-sha256-hash"
    }
  ]
}
```

### Response Handling

The API returns a structured response with successful and failed runs:

```json
{
  "data": {
    "batchId": "batch_20240120_abc123",
    "batchTitle": "Optional batch title",
    "successful": [
      {
        "id": 12345,
        "status": "QUEUED",
        "repositoryName": "owner/repo",
        "title": "Task title",
        "requestIndex": 0
      }
    ],
    "failed": [
      {
        "requestIndex": 1,
        "prompt": "Failed task prompt",
        "error": "DUPLICATE_RUN",
        "message": "Duplicate detected (ID 12300). Use force=true to override.",
        "existingRunId": 12300
      }
    ],
    "metadata": {
      "totalRequested": 2,
      "totalSuccessful": 1,
      "totalFailed": 1
    }
  }
}
```

## Terminal User Interface (TUI)

The TUI provides an interactive way to create and manage bulk runs with the following features:

### Navigation Flow

1. **File Selection Mode**: Choose configuration files from your filesystem
2. **Run List Mode**: Review and select which runs to execute
3. **Submission Mode**: Shows progress while submitting to the API
4. **Results Mode**: Displays the outcome of bulk run creation

### Key Bindings

| Key | Action | Context |
|-----|--------|---------|
| `f` | Activate file fuzzy search | File selection |
| `Enter` | Select file/confirm selection | Any mode |
| `Space` | Toggle run selection | Run list |
| `Ctrl+A` | Select all runs | Run list |
| `Ctrl+D` | Deselect all runs | Run list |
| `Ctrl+S` | Submit selected runs | Run list |
| `q` | Go back/quit | Any mode |
| `?` | Show help | Any mode |

### File Discovery

The TUI automatically discovers bulk run configuration files by:

- Scanning current directory and subdirectories (up to 3 levels deep)
- Looking for `.json` files
- Filtering for files with bulk run structure
- Limiting to 500 files maximum for performance

### Fuzzy Search Integration

Built-in fuzzy search helps you quickly find files and navigate options:

- **Real-time filtering**: Type to filter results instantly
- **Smart matching**: Uses fuzzy string matching for flexible searches
- **Keyboard navigation**: Arrow keys or `Ctrl+J/K` for navigation
- **Visual indicators**: Clear marking of current selection

## Status and Monitoring

### Check Batch Status

```bash
# Get current status of all runs in a batch
repobird bulk status BATCH_ID

# Follow progress with real-time updates
repobird bulk status BATCH_ID --follow
```

### Batch Status Information

The status command provides:

- **Aggregate Status**: Overall batch status (QUEUED, PROCESSING, COMPLETED, etc.)
- **Individual Run Status**: Status of each run in the batch
- **Progress Information**: Completion percentage for active runs
- **Timing Information**: Start time and estimated completion
- **Statistics**: Counts of queued, processing, completed, and failed runs

### Status Examples

```bash
$ repobird bulk status batch_20240120_abc123

Batch: batch_20240120_abc123
Title: Authentication module refactoring
Status: PROCESSING
Started: 2024-01-20 10:00:00

Runs:
  ✓ Fix auth issue (ID: 12345) - DONE
    Completed: 2024-01-20 10:30:00
    PR: https://github.com/owner/repo/pull/123
    
  ⏳ Password reset feature (ID: 12346) - PROCESSING (45%)
    Started: 2024-01-20 10:15:00
    
  📋 Profile validation (ID: 12347) - QUEUED

Statistics:
  Total: 3 runs
  Completed: 1
  Processing: 1  
  Queued: 1
  Failed: 0

Estimated completion: 2024-01-20 10:45:00
```

## Best Practices

### Configuration Management

- **Organize by purpose**: Group related tasks in separate configuration files
- **Use descriptive titles**: Both batch titles and individual run titles should be clear
- **Version control**: Store configuration files in your repository for reproducibility
- **Test with plans**: Use `"runType": "plan"` to preview changes before execution

### Performance Considerations

- **Batch size limits**: Maximum of 10 runs per batch
- **Resource usage**: Consider repository size and complexity when batching
- **Parallel execution**: The API runs tasks with configurable parallelism (default: 5)
- **Rate limiting**: Be aware of API rate limits (100 requests/minute)

### Error Handling

- **Duplicate detection**: Use file hashes to prevent accidental re-runs
- **Force flag**: Override duplicate detection when intentional
- **Partial success**: Handle scenarios where some runs succeed and others fail
- **Retry strategy**: Failed runs can be retried individually

### Branch Management

- **Source branches**: Specify appropriate source branches for your changes
- **Target branches**: Use descriptive target branch names for tracking
- **Naming conventions**: Consider using prefixes like `bulk/`, `fix/`, or `feature/`

## Troubleshooting

### Common Issues

**Empty Results Screen**
- Check debug logs: `tail -f /tmp/repobird_debug.log`
- Verify API connectivity and authentication
- Ensure configuration file format is correct

**Duplicate Run Errors**
- Review existing runs to identify duplicates
- Use `force: true` in configuration to override
- Check file hash uniqueness if using duplicate detection

**Authentication Issues**
- Verify API key: `repobird config get api-key`
- Test authentication: `repobird status`
- Check repository access permissions

### Debug Logging

Enable debug logging for detailed troubleshooting:

```bash
# Set debug log location (optional)
export REPOBIRD_DEBUG_LOG=/path/to/debug.log

# Run with debug output
repobird bulk run config.json --debug

# Monitor debug log in real-time
tail -f /tmp/repobird_debug.log
```

### Configuration Validation

Validate your configuration before submission:

```bash
# Use plan mode to preview without execution
repobird bulk run config.json --type plan

# Check repository and branch access
repobird status --repo owner/repo
```

## Integration Examples

### CI/CD Pipeline Integration

```yaml
# GitHub Actions example
- name: Run bulk fixes
  run: |
    repobird bulk run .github/bulk-fixes.json --follow
    
- name: Check bulk status
  run: |
    repobird bulk status $BATCH_ID
```

### Automated Maintenance

```bash
#!/bin/bash
# Weekly maintenance script

# Run security updates
repobird bulk run configs/security-updates.json --follow

# Apply coding standard fixes
repobird bulk run configs/linting-fixes.json --follow

# Update documentation
repobird bulk run configs/doc-updates.json --follow
```

## Related Documentation

- [CLI Reference](cli-reference.md) - Complete command-line reference
- [TUI Guide](interactive-mode.md) - Terminal interface usage
- [API Specification](CLI_API_SPECIFICATION.yaml) - Complete API documentation
- [Configuration Guide](configuration-guide.md) - Advanced configuration options
- [Troubleshooting](troubleshooting.md) - Common issues and solutions

For implementation details, see:
- [Bulk Runs Implementation Plan](../tasks/bulk-runs-cli-implementation-plan.md)
- [Server API Requirements](../tasks/bulk-runs-server-api-requirements.md)
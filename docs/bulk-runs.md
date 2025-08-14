# Bulk Runs Guide

## Overview

Execute multiple AI-powered tasks simultaneously for efficient batch processing.

## Related Documentation
- **[TUI Guide](tui-guide.md)** - Bulk view in TUI
- **[API Reference](api-reference.md)** - Bulk API endpoints
- **[Configuration Guide](configuration-guide.md)** - Bulk configuration

## Quick Start

### CLI Usage
```bash
# Submit bulk runs
repobird bulk config.json

# With progress tracking
repobird bulk config.json --follow

# Dry run (validate only)
repobird bulk config.json --dry-run
```

### TUI Usage
Press `B` in dashboard to open bulk view:
1. Select configuration file
2. Review and toggle runs
3. Submit and track progress

## Configuration Format

### Basic Structure
```json
{
  "repository": "org/repo",
  "source": "main",
  "runType": "run",
  "runs": [
    {
      "title": "Fix auth bug",
      "prompt": "Fix authentication issue in login flow",
      "target": "fix/auth-bug"
    },
    {
      "title": "Add logging",
      "prompt": "Add comprehensive logging to API endpoints",
      "target": "feature/logging"
    }
  ]
}
```

### Advanced Configuration
```json
{
  "repository": "org/repo",
  "source": "main",
  "runType": "approval",
  "parallel": true,
  "maxConcurrent": 5,
  "defaults": {
    "context": "Follow existing code patterns",
    "files": ["src/"],
    "modelOverride": "claude-3"
  },
  "runs": [
    {
      "title": "Update dependencies",
      "prompt": "Update all npm dependencies to latest stable versions",
      "target": "chore/update-deps",
      "priority": "high"
    },
    {
      "title": "Refactor auth module",
      "prompt": "Refactor authentication to use JWT tokens",
      "target": "refactor/auth-jwt",
      "context": "Maintain backward compatibility",
      "files": ["src/auth/", "src/middleware/"]
    }
  ]
}
```

## Configuration Options

### Global Settings
| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `repository` | string | Target repository (org/repo) | Required |
| `source` | string | Source branch | Required |
| `runType` | string | "run" or "approval" | "run" |
| `parallel` | bool | Execute runs in parallel | true |
| `maxConcurrent` | int | Max parallel runs | 5 |
| `defaults` | object | Default values for all runs | {} |

### Run Settings
| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `title` | string | Run title | Yes |
| `prompt` | string | Task description | Yes |
| `target` | string | Target branch | No |
| `context` | string | Additional context | No |
| `files` | array | Specific files/dirs | No |
| `priority` | string | Run priority | No |
| `skip` | bool | Skip this run | No |

## Bulk View Features

### File Selection Mode
- Browse configuration files
- Preview file contents
- Validate JSON structure
- Recent files history

### Run List Mode
- Toggle individual runs (space)
- Select/deselect all (a/A)
- Preview run details
- Validate before submission

### Progress Mode
- Real-time status updates
- Success/failure tracking
- Error messages
- Cancel option (Ctrl+C)

### Results Mode
- Summary statistics
- Failed run details
- PR URLs for successful runs
- Export results

## Best Practices

### Configuration Management
```bash
# Organize configs
bulk-configs/
├── bugfixes/
│   ├── critical-fixes.json
│   └── minor-fixes.json
├── features/
│   └── q4-features.json
└── maintenance/
    └── dependency-updates.json
```

### Validation
```bash
# Always dry-run first
repobird bulk config.json --dry-run

# Check JSON validity
jq . config.json

# Validate with schema
ajv validate -s bulk-schema.json -d config.json
```

### Error Handling
```json
{
  "onError": "continue",  // continue, stop, or rollback
  "retryFailed": true,
  "maxRetries": 3,
  "runs": [...]
}
```

## Examples

### Bug Fix Batch
```json
{
  "repository": "myorg/app",
  "source": "main",
  "runType": "run",
  "runs": [
    {
      "title": "Fix null pointer in user service",
      "prompt": "Fix NPE when user.email is null",
      "target": "fix/user-npe",
      "files": ["src/services/UserService.java"]
    },
    {
      "title": "Fix race condition in cache",
      "prompt": "Add proper locking to prevent cache corruption",
      "target": "fix/cache-race",
      "files": ["src/cache/"]
    }
  ]
}
```

### Feature Implementation
```json
{
  "repository": "myorg/frontend",
  "source": "develop",
  "runType": "approval",
  "defaults": {
    "context": "Use React hooks and TypeScript"
  },
  "runs": [
    {
      "title": "Add dark mode",
      "prompt": "Implement dark mode toggle with system preference detection",
      "target": "feature/dark-mode"
    },
    {
      "title": "Add export functionality",
      "prompt": "Add CSV and PDF export for data tables",
      "target": "feature/export"
    }
  ]
}
```

### Refactoring Tasks
```json
{
  "repository": "myorg/backend",
  "source": "main",
  "parallel": false,
  "runs": [
    {
      "title": "Extract service layer",
      "prompt": "Refactor business logic from controllers to service layer",
      "target": "refactor/service-layer"
    },
    {
      "title": "Add dependency injection",
      "prompt": "Replace manual instantiation with DI container",
      "target": "refactor/di"
    }
  ]
}
```

## Monitoring

### CLI Progress
```bash
# Follow all runs
repobird bulk config.json --follow

# Check specific batch
repobird bulk status batch-123

# List recent batches
repobird bulk list
```

### TUI Navigation
- `B` - Open bulk view
- `Tab` - Navigate sections
- `Space` - Toggle selection
- `Enter` - Submit/Continue
- `q` - Back/Cancel

## Troubleshooting

### Common Issues

**Invalid JSON:**
```bash
# Validate JSON
jq . config.json || echo "Invalid JSON"
```

**Authentication Errors:**
```bash
# Verify API key
repobird auth verify
```

**Rate Limiting:**
```json
{
  "maxConcurrent": 2,
  "delayBetweenRuns": 5000
}
```

### Debug Mode
```bash
REPOBIRD_DEBUG_LOG=1 repobird bulk config.json
tail -f /tmp/repobird_debug.log
```

## Performance Tips

1. **Batch Similar Tasks** - Group related changes
2. **Use Parallel Execution** - For independent tasks
3. **Set Reasonable Limits** - maxConcurrent based on API limits
4. **Monitor Progress** - Use --follow or TUI
5. **Validate First** - Always use --dry-run

## API Integration

### Programmatic Usage
```go
client := api.NewClient(apiKey)
batch := &api.BulkRunRequest{
    Repository: "org/repo",
    Source:     "main",
    Runs:       runs,
}
results, err := client.CreateBulkRuns(ctx, batch)
```
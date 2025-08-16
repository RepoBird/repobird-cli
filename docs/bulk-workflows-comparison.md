# Bulk Workflows Comparison: TUI vs CLI

This document details the technical differences between running bulk operations through the TUI (Terminal User Interface) and the CLI `run` command, particularly focusing on how they process YAML configuration files and handle API requests.

## Overview

Both the TUI bulk view and the `repobird run` command support bulk operations from YAML/JSON files. While they appear to behave differently, the core processing logic is identical - the key difference lies in which API server they connect to by default.

## Workflow Comparison

### TUI Bulk View Workflow

1. **Initialization**
   - User navigates to bulk view from dashboard (press 'b')
   - `NewBulkView()` creates view with API client and cache
   - No repository information is passed at creation

2. **File Loading**
   - User selects files through file browser (press 'f')
   - `bulk.LoadBulkConfig(files)` parses the configuration
   - Extracts: `repository`, `repoID`, `source`, `runType`, `batchTitle`, and `runs[]`

3. **API Request Creation**
   ```go
   req := &dto.BulkRunRequest{
       RepositoryName: v.repository,  // From YAML
       RepoID:         v.repoID,      // From YAML (0 if not specified)
       RunType:        v.runType,
       SourceBranch:   v.sourceBranch,
       BatchTitle:     v.batchTitle,
       Force:          v.force,
       Runs:           runItems,
   }
   ```

4. **API Server Selection**
   - When run with `make tui`: Uses `REPOBIRD_API_URL` from `.env` file
   - Typically points to development server
   - Example: `https://localhost:3000`

### CLI Run Command Workflow

1. **Command Execution**
   - User runs: `repobird run bulk-config.yaml`
   - Checks if file is bulk config via `bulk.IsBulkConfig()`

2. **File Loading**
   - `bulk.ParseBulkConfig(filename)` parses the configuration
   - Uses same parsing logic as TUI
   - Extracts identical fields: `repository`, `repoID`, etc.

3. **API Request Creation**
   ```go
   bulkRequest := &dto.BulkRunRequest{
       RepositoryName: bulkConfig.Repository,  // From YAML
       RepoID:         bulkConfig.RepoID,      // From YAML (0 if not specified)
       RunType:        bulkConfig.RunType,
       SourceBranch:   bulkConfig.Source,
       BatchTitle:     bulkConfig.BatchTitle,
       Force:          false,
       Runs:           make([]dto.RunItem, len(bulkConfig.Runs)),
   }
   ```

4. **API Server Selection**
   - Checks `REPOBIRD_API_URL` environment variable
   - If not set: defaults to `https://repobird.ai` (production)
   - This is the critical difference from TUI

## Key Findings

### No Cache-Based Repository Resolution
- **TUI does NOT retrieve repoID from cache**
- Both workflows get repoID directly from the YAML file
- If repoID is not in YAML, it's sent as 0 (zero)
- No automatic repository ID lookup occurs

### Identical Request Structure
Both TUI and CLI generate the exact same API request structure:
- Same fields populated from YAML
- Same validation logic
- Same bulk configuration parsing (`bulk.LoadBulkConfig`)

### The Critical Difference: API Server
The only significant difference is the default API server:

| Aspect | TUI (make tui) | CLI (repobird run) |
|--------|----------------|-------------------|
| Default Server | Dev server from .env | Production (repobird.ai) |
| Repository Data | Dev repositories | Production repositories |
| Typical Usage | Development/Testing | Production runs |

## Common Issues and Solutions

### "Invalid Repository" Error
**Symptom**: YAML works in TUI but fails with `repobird run`

**Cause**: Repository exists on dev server but not on production

**Solutions**:
1. Use dev server with CLI:
   ```bash
   REPOBIRD_API_URL=https://your-dev-server.ngrok-free.app repobird run config.yaml
   ```

2. Use a production repository in YAML:
   ```yaml
   repository: production-org/production-repo  # Must exist on repobird.ai
   ```

3. Add repoID if known:
   ```yaml
   repository: support-rb/test-ruby
   repoID: 123  # Specific repository ID
   ```

### Environment-Specific Configuration

For consistent behavior between TUI and CLI:

```bash
# Set API URL for CLI to match TUI
export REPOBIRD_API_URL=https://your-dev-server.ngrok-free.app

# Now both will use the same server
make tui                    # Uses dev server
repobird run config.yaml    # Also uses dev server
```

## YAML Configuration Format

Both workflows support the same YAML structure:

```yaml
repository: owner/repo-name    # Required (unless repoId provided)
repoId: 123                    # Optional (overrides repository name)
source: main                   # Source branch
runType: run                   # or "plan"
batchTitle: Batch Title        # Optional
runs:
  - prompt: Task description
    title: Task title
    target: feature/branch
    context: Additional context
```

## Technical Implementation Files

Key files involved in bulk processing:

- **Bulk Config Parsing**: `internal/bulk/config.go`
  - `LoadBulkConfig()` - Used by both TUI and CLI
  - `ParseBulkConfig()` - Parses YAML/JSON files
  - `IsBulkConfig()` - Determines if file is bulk config

- **TUI Bulk View**: `internal/tui/views/bulk.go`
  - `NewBulkView()` - Creates bulk view
  - `submitBulkRuns()` - Submits to API

- **CLI Run Command**: `internal/commands/run.go`
  - `processBulkRuns()` - Handles bulk config
  - `executeBulkRuns()` - Submits to API

- **API Client**: `internal/api/client.go`
  - `CreateBulkRuns()` - API call used by both

## Debugging Tips

To understand which server is being used:

```bash
# Check TUI server
REPOBIRD_DEBUG_LOG=1 make tui
# Look for API URL in debug logs

# Check CLI server
REPOBIRD_DEBUG_LOG=1 repobird run --dry-run config.yaml
# Displays configuration without submission

# Force specific server
REPOBIRD_API_URL=https://repobird.ai repobird run config.yaml
```

## Conclusion

The TUI bulk view and CLI `run` command use identical logic for processing bulk configurations. The perceived differences in behavior stem from connecting to different API servers by default, not from different request handling or cache lookups. Understanding this distinction is crucial for troubleshooting bulk operation issues.
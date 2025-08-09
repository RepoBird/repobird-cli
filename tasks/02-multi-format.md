# Phase 2: Multi-Format Support (Week 3)

## Overview
Extend the CLI to support multiple input formats beyond JSON, enabling more flexible and user-friendly task definitions.

## Tasks

### YAML Support
- [ ] Add YAML parser dependency (gopkg.in/yaml.v3)
- [ ] Implement YAML file detection and parsing
- [ ] Support multi-line strings for prompts
- [ ] Add YAML schema validation
- [ ] Create YAML-specific error messages
- [ ] Document YAML format examples

### TOML Support
- [ ] Add TOML parser dependency (BurntSushi/toml)
- [ ] Implement TOML file detection and parsing
- [ ] Handle TOML-specific data types
- [ ] Add TOML validation
- [ ] Support inline tables and arrays
- [ ] Document TOML format examples

### JSONL Batch Processing
- [ ] Implement JSONL file reader
- [ ] Add streaming parser for large files
- [ ] Support parallel batch execution
- [ ] Implement progress tracking for batches
- [ ] Add batch result aggregation
- [ ] Support batch cancellation
- [ ] Add --max-concurrent flag for rate limiting

### Markdown/Text Support
- [ ] Add Markdown parser (blackfriday/v2)
- [ ] Extract prompts from Markdown files
- [ ] Support frontmatter for configuration
- [ ] Handle plain text files
- [ ] Implement configuration via CLI flags
- [ ] Support code block extraction
- [ ] Add template variable substitution

### Format Auto-Detection
- [ ] Implement file extension detection
- [ ] Add content-based format detection
- [ ] Support explicit --format flag override
- [ ] Handle ambiguous formats gracefully
- [ ] Add format conversion utilities
- [ ] Implement format validation command

### Enhanced Configuration Management
- [ ] Extend Viper for multi-format configs
- [ ] Support cascading configuration
- [ ] Add profile support (dev, prod, etc.)
- [ ] Implement config merge strategies
- [ ] Add interactive config initialization
- [ ] Support config encryption for sensitive data

## Input Format Examples

### YAML Configuration
```yaml
prompt: |
  Implement user authentication with JWT tokens.
  Include refresh token mechanism and secure storage.
  
  Requirements:
  - Use bcrypt for password hashing
  - Implement rate limiting
  - Add password reset functionality
  
repository: acme/webapp  # Can be "owner/repo" or just "repo"
title: Implement JWT authentication
source: main
target: feature/auth
runType: run  # 'run' for implementation, 'plan' for planning only
context: |
  Additional context about existing auth system
  Current using session-based auth
files:
  - src/auth/
  - src/middleware/
excludeFiles:
  - src/auth/legacy.js
issueNumber: 123  # Optional: link to GitHub issue
```

### TOML Configuration
```toml
prompt = """
Refactor the database connection pool
to improve performance and reliability.
"""

repository = "acme/webapp"
title = "Database connection pool refactor"
source = "main"
target = "perf/db-optimization"
runType = "run"

[context]
text = "Current using pg-pool with 10 connections"

files = ["src/db/", "src/models/"]
excludeFiles = ["src/db/migrations/"]

issueNumber = 456
```

### JSONL Batch File
```jsonl
{"prompt": "Fix login bug", "repository": "acme/webapp", "title": "Fix login authentication", "runType": "run"}
{"prompt": "Add password reset", "repository": "acme/webapp", "title": "Password reset feature", "runType": "run"}
{"prompt": "Implement 2FA", "repository": "acme/webapp", "title": "Two-factor auth", "runType": "plan"}
{"prompt": "Optimize database queries", "repoId": 12345, "source": "main", "target": "perf/db"}
{"prompt": "Add caching layer", "repository": "acme/backend", "context": "Use Redis for caching"}
```

### Markdown with Frontmatter
```markdown
---
repository: acme/webapp
title: Implement New Dashboard UI
source: main
target: feature/new-ui
runType: run
context: Using React with TypeScript
files:
  - src/components/dashboard/
  - src/styles/
---

# Task: Implement New Dashboard UI

## Requirements

The dashboard should include:
- Real-time data visualization
- Responsive design for mobile
- Dark mode support

## Technical Details

```javascript
// Use React with TypeScript
// Implement using Chart.js for graphs
```

Please ensure all components are properly tested.
```

## Command Examples (Phase 2)

```bash
# Auto-detect format
repobird run task.yaml
repobird run config.toml
repobird run batch.jsonl

# Explicit format
repobird run task.md --format markdown
repobird run data.txt --format text --repo acme/webapp

# Batch processing (sequential for now)
repobird run batch.jsonl --progress
repobird run batch.jsonl --dry-run  # Validate all entries

# Configuration
repobird config init --format yaml
repobird config set api.endpoint https://api.repobird.ai/v2
repobird config profile create production
```

## Testing Requirements

### Unit Tests
- [ ] YAML parser tests with edge cases
- [ ] TOML parser tests with complex structures
- [ ] JSONL streaming tests
- [ ] Markdown extraction tests
- [ ] Format detection accuracy tests

### Integration Tests
- [ ] Multi-format processing pipeline
- [ ] Batch execution flow
- [ ] Configuration cascading
- [ ] Format conversion accuracy

## Deliverables

1. Support for YAML, TOML, JSONL, and Markdown formats
2. Automatic format detection
3. Batch processing capabilities
4. Enhanced configuration management
5. Comprehensive format documentation

## Success Criteria

- [ ] All formats parse correctly
- [ ] Auto-detection accuracy > 95%
- [ ] Batch processing handles 1000+ items
- [ ] Format conversion preserves all data
- [ ] Performance remains under 100ms startup
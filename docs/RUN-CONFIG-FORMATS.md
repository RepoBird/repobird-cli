# RepoBird Run Configuration Formats

The `repobird run` command supports multiple configuration file formats to define AI coding tasks. This guide covers all supported formats with complete examples. The command automatically detects whether a configuration file contains a single run or multiple bulk runs.

## Supported Formats

- **Command-line flags** - Direct run creation using flags (single run only)
- **JSON** (`.json`) - Standard JSON configuration (single or bulk)
- **YAML** (`.yaml`, `.yml`) - Human-friendly YAML format (single or bulk)
- **Markdown** (`.md`, `.markdown`) - Markdown with YAML frontmatter for documentation (single or bulk)
- **JSONL** (`.jsonl`) - JSON Lines format for bulk runs
- **Stdin** - Pipe JSON directly without a file

## Configuration Fields

### Single Run Configuration

#### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `prompt` | string | The main task description/instructions for the AI agent |
| `repository` | string | Repository name in format `owner/repo` |

#### Optional Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `target` | string | auto-generated | Target branch name for the changes |
| `title` | string | auto-generated | Human-readable title for the run |
| `source` | string | `main` | Source branch to work from |
| `runType` | string | `run` | Type of run: `run` or `plan` |
| `context` | string | - | Additional context or instructions for the AI |
| `files` | array | - | List of specific files to include in the context |

### Bulk Run Configuration

#### Top-Level Fields

| Field | Type | Description |
|-------|------|-------------|
| `repository` | string | Repository for all runs (required) |
| `source` | string | Source branch for all runs (defaults to repository's default branch if not specified) |
| `runType` | string | Type for all runs: `run` or `plan` (defaults to `run`) |
| `runs` | array | Array of run configurations (required) |

#### Run Item Fields

| Field | Type | Description |
|-------|------|-------------|
| `prompt` | string | Task description for this run (required) |
| `title` | string | Human-readable title for this run |
| `target` | string | Target branch for this run |
| `context` | string | Additional context for this run |

## Format Examples

### Single Run Examples

#### Command-Line Flags

Create a run directly using command-line flags without any configuration file:

```bash
# Minimal example (required flags only)
repobird run -r myorg/webapp -p "Fix the login bug"

# Read prompt from a file using @ prefix
repobird run -r myorg/webapp -p @task.txt

# Read prompt from stdin
echo "Fix the login bug" | repobird run -r myorg/webapp -p -

# With additional options
repobird run --repo myorg/webapp \
  --prompt "Fix the login bug where users cannot authenticate after 5 failed attempts" \
  --source main \
  --target fix/login-rate-limit \
  --title "Fix authentication rate limiting issue" \
  --context "Users report being permanently locked out. Should reset after 15 minutes." \
  --follow

# Using files for complex prompts and context
repobird run -r myorg/webapp -p @detailed-task.md --context @requirements.txt

# Escape @ at the beginning with @@
repobird run -r myorg/webapp -p "@@mentions are preserved"

# Short form flags
repobird run -r myorg/webapp -p "Add unit tests for auth module" --follow
```

**Available flags:**
- `-r, --repo` (required) - Repository name (owner/repo or numeric ID)
- `-p, --prompt` (required) - The task description/instructions
  - Use `@filename` to read from a file
  - Use `-` to read from stdin
  - Use `@@` to escape a literal `@` at the beginning
- `--source` - Source branch (optional, defaults to repository's default branch)
- `--target` - Target branch (optional, auto-generated if not specified)
- `--title` - Human-readable title (optional, auto-generated if not specified)
- `--run-type` - Type of run: 'run' or 'plan' (optional, defaults to 'run')
- `--context` - Additional context (optional, also supports `@filename` and `-`)
- `--follow` - Follow the run status after creation
- `--dry-run` - Validate without creating the run

#### JSON Format

Create a file `task.json`:

```json
{
  "prompt": "Fix the login bug where users cannot authenticate after 5 failed attempts",
  "repository": "myorg/webapp",
  "source": "main",
  "target": "fix/login-rate-limit",
  "title": "Fix authentication rate limiting issue",
  "runType": "run",
  "context": "Users report being permanently locked out after 5 failed login attempts. The rate limiting should reset after 15 minutes.",
  "files": [
    "src/auth/login.js",
    "src/auth/rateLimit.js",
    "src/utils/validation.js"
  ]
}
```

Run with:
```bash
repobird run task.json
repobird run task.json --follow  # Follow the run status
repobird run task.json --dry-run # Validate without creating
```

#### YAML Format

Create a file `task.yaml`:

```yaml
prompt: Fix the login bug where users cannot authenticate after 5 failed attempts
repository: myorg/webapp
source: main
target: fix/login-rate-limit
title: Fix authentication rate limiting issue
runType: run
context: |
  Users report being permanently locked out after 5 failed login attempts.
  The rate limiting should reset after 15 minutes.
files:
  - src/auth/login.js
  - src/auth/rateLimit.js
  - src/utils/validation.js
```

Run with:
```bash
repobird run task.yaml
```

#### Markdown with YAML Frontmatter

Create a file `task.md`:

```markdown
---
prompt: Fix the login bug where users cannot authenticate after 5 failed attempts
repository: myorg/webapp
source: main
target: fix/login-rate-limit
title: Fix authentication rate limiting issue
runType: run
files:
  - src/auth/login.js
  - src/auth/rateLimit.js
---

# Additional Context

## Problem Description

Users are experiencing a critical issue with our authentication system. After 5 failed login attempts, they are permanently locked out instead of being temporarily rate-limited.

## Expected Behavior

- After 5 failed attempts, users should be temporarily locked for 15 minutes
- The lockout should automatically reset after the timeout period
- Users should see a clear message indicating when they can try again

## Technical Details

The issue appears to be in the `rateLimit.js` module where the reset logic is not properly implemented. The timestamp comparison might be using the wrong timezone or the reset function might not be called.

## Testing Requirements

- Test with multiple failed attempts
- Verify the 15-minute reset works
- Ensure proper error messages are shown
```

The markdown content after the frontmatter is automatically appended to the `context` field.

Run with:
```bash
repobird run task.md
```

#### Stdin (Piped JSON)

You can pipe JSON directly without creating a file:

```bash
# Simple example
echo '{"prompt":"Fix the login bug","repository":"myorg/webapp","target":"fix/login","title":"Fix auth issue"}' | repobird run

# From another command
cat task.json | repobird run --follow

# With jq to build JSON dynamically
jq -n '{
  prompt: "Fix the bug",
  repository: "myorg/webapp",
  target: "fix/bug",
  title: "Bug fix"
}' | repobird run
```

### Bulk Run Examples

#### Bulk JSON Format

Create a file `tasks.json`:

```json
{
  "repository": "myorg/webapp",
  "source": "main",
  "runType": "run",
  "runs": [
    {
      "prompt": "Fix the login bug where users cannot authenticate",
      "title": "Fix authentication issue",
      "target": "fix/auth-bug",
      "context": "Users report being locked out after 5 failed attempts"
    },
    {
      "prompt": "Add comprehensive logging to all API endpoints",
      "title": "Add API logging",
      "target": "feature/api-logging",
      "context": "Need request/response logging with timestamps"
    },
    {
      "prompt": "Optimize database queries in the user service",
      "title": "Optimize user queries",
      "target": "perf/user-queries"
    }
  ]
}
```

Run with:
```bash
repobird run tasks.json          # Process all runs
repobird run tasks.json --follow  # Follow batch progress
repobird run tasks.json --dry-run # Validate without running
```

#### Bulk YAML Format

Create a file `tasks.yaml`:

```yaml
repository: myorg/webapp
source: main
runType: run
runs:
  - prompt: Fix the login bug where users cannot authenticate
    title: Fix authentication issue
    target: fix/auth-bug
    context: Users report being locked out after 5 failed attempts
  
  - prompt: Add comprehensive logging to all API endpoints
    title: Add API logging
    target: feature/api-logging
    context: Need request/response logging with timestamps
  
  - prompt: Optimize database queries in the user service
    title: Optimize user queries
    target: perf/user-queries
```

#### JSONL Format (Bulk)

Create a file `tasks.jsonl`:

```jsonl
{"prompt": "Fix login bug", "title": "Fix auth", "target": "fix/auth", "repository": "myorg/webapp"}
{"prompt": "Add logging", "title": "Add logs", "target": "feature/logs", "repository": "myorg/webapp"}
{"prompt": "Optimize queries", "title": "DB optimization", "target": "perf/db", "repository": "myorg/webapp"}
```

Run with:
```bash
repobird run tasks.jsonl
```

## Minimal Examples

### Minimal JSON
```json
{
  "prompt": "Add dark mode support to the settings page",
  "repository": "myorg/webapp"
}
```

### Minimal YAML
```yaml
prompt: Add dark mode support to the settings page
repository: myorg/webapp
```

### Minimal Markdown
```markdown
---
prompt: Add dark mode support to the settings page
repository: myorg/webapp
---
```

## Run Types Explained

### `run` (Default)
- AI agent makes changes directly
- Creates a pull request automatically
- Best for straightforward tasks

### `plan`
- AI agent creates a detailed plan
- No code changes are made
- Useful for complex tasks requiring review


## Validation and Error Handling

### Dry Run Mode

Test your configuration without creating a run:

```bash
repobird run task.json --dry-run
```

This will:
- Validate all required fields
- Check field formats
- Show the final configuration that would be sent
- Report any errors without consuming API credits

### Common Validation Errors

1. **Missing Required Fields**
   ```
   Error: validation failed: missing required field: repository
   ```

2. **Invalid Repository Format**
   ```
   Error: validation failed: repository must be in format 'owner/repo'
   ```

3. **Empty Prompt**
   ```
   Error: validation failed: prompt cannot be empty
   ```

## Best Practices

1. **Use Descriptive Prompts**: Be specific about what you want the AI to do
2. **Include Context**: Add background information in the `context` field
3. **Specify Files**: When working on specific files, list them to provide focus
4. **Test with Dry Run**: Always validate complex configurations with `--dry-run`
5. **Use Markdown for Documentation**: For complex tasks, use Markdown format to include detailed documentation
6. **Follow Status**: Use `--follow` flag to monitor long-running tasks

## Example Workflows

### Bug Fix Workflow
```bash
# 1. Create configuration
cat > bugfix.yaml << EOF
prompt: Fix null pointer exception in user profile
repository: myorg/webapp
target: fix/npe-user-profile
title: Fix NPE in user profile loading
context: Exception occurs when user has no profile picture
files:
  - src/components/UserProfile.js
EOF

# 2. Validate
repobird run bugfix.yaml --dry-run

# 3. Run and follow
repobird run bugfix.yaml --follow
```

### Feature Development Workflow
```bash
# 1. Plan the feature first
cat > feature-plan.yaml << EOF
prompt: Plan implementation for user notifications system
repository: myorg/webapp
target: feature/notifications
title: User notifications system
runType: plan
EOF

repobird run feature-plan.yaml

# 2. After reviewing plan, implement
cat > feature-impl.yaml << EOF
prompt: Implement user notifications as planned
repository: myorg/webapp
target: feature/notifications
title: Implement user notifications
runType: run
context: Follow the previously created plan for notifications
EOF

repobird run feature-impl.yaml --follow
```

## See Also

- [Bulk Runs Guide](BULK-RUNS.md) - Running multiple tasks in batch
- [Configuration Guide](CONFIGURATION-GUIDE.md) - Setting up RepoBird CLI
- [API Reference](API-REFERENCE.md) - Complete API documentation
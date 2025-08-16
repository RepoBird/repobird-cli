# RepoBird Run Configuration Formats

The `repobird run` command supports multiple configuration file formats to define AI coding tasks. This guide covers all supported formats with complete examples.

## Supported Formats

- **JSON** (`.json`) - Standard JSON configuration
- **YAML** (`.yaml`, `.yml`) - Human-friendly YAML format
- **Markdown** (`.md`, `.markdown`) - Markdown with YAML frontmatter for documentation
- **Stdin** - Pipe JSON directly without a file

## Configuration Fields

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `prompt` | string | The main task description/instructions for the AI agent |
| `repository` | string | Repository name in format `owner/repo` (auto-detected if in git repo) |
| `target` | string | Target branch name for the changes |
| `title` | string | Human-readable title for the run |

### Optional Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `source` | string | `main` | Source branch to work from (auto-detected if in git repo) |
| `runType` | string | `run` | Type of run: `run`, `plan`, or `approval` |
| `context` | string | - | Additional context or instructions for the AI |
| `files` | array | - | List of specific files to include in the context |

## Format Examples

### JSON Format

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

### YAML Format

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

### Markdown with YAML Frontmatter

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

### Stdin (Piped JSON)

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

## Minimal Examples

### Minimal JSON
```json
{
  "prompt": "Add dark mode support to the settings page",
  "repository": "myorg/webapp",
  "target": "feature/dark-mode",
  "title": "Add dark mode feature"
}
```

### Minimal YAML
```yaml
prompt: Add dark mode support to the settings page
repository: myorg/webapp
target: feature/dark-mode
title: Add dark mode feature
```

### Minimal Markdown
```markdown
---
prompt: Add dark mode support to the settings page
repository: myorg/webapp
target: feature/dark-mode
title: Add dark mode feature
---
```

## Auto-Detection Features

When running `repobird run` from within a git repository:

1. **Repository Auto-Detection**: If `repository` field is omitted, RepoBird attempts to detect it from git remote
2. **Source Branch Auto-Detection**: If `source` field is omitted, RepoBird uses the current git branch

Example with auto-detection:
```yaml
# When run from inside the 'myorg/webapp' git repository on 'develop' branch
prompt: Fix the login bug
target: fix/login-bug
title: Fix authentication issue
# repository: automatically detected as 'myorg/webapp'
# source: automatically detected as 'develop'
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

### `approval`
- AI agent makes changes but waits for approval
- Changes are staged but not committed
- Good for sensitive modifications

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

- [Bulk Runs Guide](bulk-runs.md) - Running multiple tasks in batch
- [Configuration Guide](configuration-guide.md) - Setting up RepoBird CLI
- [API Reference](api-reference.md) - Complete API documentation
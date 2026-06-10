# RepoBird Run Configuration Formats

The `repobird run` command supports multiple configuration file formats to define AI coding tasks. This guide covers all supported single-run formats with complete examples.

## Supported Formats

- **Command-line flags** - Direct run creation using flags (single run only)
- **JSON** (`.json`) - Standard JSON configuration
- **YAML** (`.yaml`, `.yml`) - Human-friendly YAML format
- **Markdown** (`.md`, `.markdown`) - Markdown with YAML frontmatter for documentation
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
| `baseBranch` | string | repository default branch | Branch to start work from |
| `outputMode` | string | `pull_request` | Output mode: `pull_request` to create a pull request, `branch` to push without a PR. `pr` is accepted as a CLI alias. |
| `outputBranch` | string | auto-generated | Branch to push generated commits to |
| `prTargetBranch` | string | `baseBranch` | Branch the pull request targets when `outputMode` is `pull_request` |
| `outputBranchPolicy` | string | `create` | Output branch policy: `create` or `reuse` |
| `title` | string | auto-generated | Human-readable title for the run |
| `source` | string | repository default branch | Legacy alias for `baseBranch` |
| `target` | string | auto-generated | Legacy alias; in branch-only runs it maps to `outputBranch` |
| `runType` | string | `run` | Type of run: `run`; `plan` is development-only during the OpenCode migration |
| `context` | string | - | Additional context or instructions for the AI |
| `files` | array | - | List of specific files to include in the context |
| `branchOnly` | boolean | `false` | Legacy alias for `outputMode: branch` |
| `acknowledgePromptRisk` | boolean | `false` | Explicitly acknowledge a `PROMPT_RISK_ACK_REQUIRED` response after reviewing the prompt |
| `idempotencyKey` | string | auto-derived | Stable key for safely retrying run creation |

## Format Examples

### Single Run Examples

#### Command-Line Flags

Create a run directly using command-line flags without any configuration file:

```bash
# Minimal example (required flags only)
repobird run -r myorg/webapp -p "Fix the login bug"

# Basic and Pro cloud-agent presets
repobird run --basic -r myorg/webapp -p "Fix a small bug"     # DeepSeek V4 Flash
repobird run --pro -r myorg/webapp -p "Implement OAuth"       # Kimi K2.6
repobird basic -r myorg/webapp "Fix a small bug"
repobird pro -r myorg/webapp "Implement OAuth"

# Inside a git repo with an origin remote, Basic/Pro commands can auto-detect the repository
repobird pro "Fix the login bug"

# Read prompt from a file using @ prefix
repobird run -r myorg/webapp -p @task.txt

# Read prompt from stdin
echo "Fix the login bug" | repobird run -r myorg/webapp -p -

# Push commits to an output branch without opening a PR
repobird run -r myorg/webapp -p "Update generated docs" --output-branch automation/docs --branch-only

# Retry safely with an explicit key, or bypass the local duplicate guard after review
repobird run -r myorg/webapp -p @task.txt --idempotency-key task-2026-06-10-auth
repobird run -r myorg/webapp -p @task.txt --force

# With additional options
repobird run --repo myorg/webapp \
  --prompt "Fix the login bug where users cannot authenticate after 5 failed attempts" \
  --base-branch main \
  --output-branch fix/login-rate-limit \
  --pr-target-branch main \
  --title "Fix authentication rate limiting issue" \
  --context "Users report being permanently locked out. Should reset after 15 minutes." \
  --follow

# Using files for complex prompts and context
repobird run -r myorg/webapp -p @detailed-task.md --context @requirements.txt

# Escape @ at the beginning with @@
repobird run -r myorg/webapp -p "@@mentions are preserved"

# Short form flags
repobird run -r myorg/webapp -p "Add unit tests for auth module" --follow

# Script-friendly wait mode
repobird run -r myorg/webapp -p "Add unit tests for auth module" --wait --json --timeout 45m
```

**Available flags:**
- `-r, --repo` - Repository name (owner/repo or numeric ID); required unless a Basic/Pro preset can auto-detect it from git
- `-p, --prompt` (required) - The task description/instructions
  - Use `@filename` to read from a file
  - Use `-` to read from stdin
  - Use `@@` to escape a literal `@` at the beginning
- `--base-branch` - Branch to start work from (optional, defaults to repository's default branch)
- `--output-mode` - Output mode: `pull_request` or `branch` (optional, defaults to `pull_request`; `pr` is accepted as an alias)
- `--output-branch` - Branch to push generated commits to (optional, auto-generated if not specified)
- `--pr-target-branch` - Branch the pull request targets (optional, defaults to `baseBranch`)
- `--output-branch-policy` - Output branch policy: `create` or `reuse` (optional, defaults to `create`)
- `--source` - Legacy alias for `--base-branch`
- `--target` - Legacy target alias; with `--branch-only`, maps to `--output-branch`
- `--title` - Human-readable title (optional, auto-generated if not specified)
- `--run-type` - Type of run: `run` (optional, defaults to `run`); `plan` is development-only during the OpenCode migration
- `--basic` - Use the Basic cloud-agent preset (DeepSeek V4 Flash)
- `--pro` - Use the Pro cloud-agent preset (Kimi K2.6)
- `--branch-only`, `--no-pr` - Push commits to the output branch without creating a PR
- `--acknowledge-prompt-risk` - Resend after reviewing a prompt-risk acknowledgement error
- `--idempotency-key` - Stable key for safely retrying run creation; also sent as the `Idempotency-Key` header
- `--force` - Bypass the local 30-second duplicate-submission guard after reviewing the duplicate
- `--context` - Additional context (optional, also supports `@filename` and `-`)
- `--follow` - Follow the run status after creation
- `--wait` - Wait for the created run to reach a terminal state
- `--timeout` - Maximum time to wait with `--wait` (default `1h30m`; examples: `45m`, `2h`)
- `--json` - Output machine-readable JSON. With `--wait`, stdout contains one final JSON object.
- `--dry-run` - Validate without creating the run

For single-run creation, the CLI records a local submission key before the API request. If the same repository, prompt, and run type are submitted again within 30 seconds, the CLI stops before sending another POST. Use `--force` only when you intend to create another run.

#### JSON Format

Create a file `task.json`:

```json
{
  "prompt": "Fix the login bug where users cannot authenticate after 5 failed attempts",
  "repository": "myorg/webapp",
  "baseBranch": "main",
  "outputMode": "pull_request",
  "outputBranch": "fix/login-rate-limit",
  "prTargetBranch": "main",
  "outputBranchPolicy": "create",
  "title": "Fix authentication rate limiting issue",
  "runType": "run",
  "acknowledgePromptRisk": false,
  "idempotencyKey": "task-2026-06-10-auth",
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
repobird run task.json --wait --json --timeout 45m # Wait for scripts
repobird run task.json --dry-run # Validate without creating
```

#### YAML Format

Create a file `task.yaml`:

```yaml
prompt: Fix the login bug where users cannot authenticate after 5 failed attempts
repository: myorg/webapp
baseBranch: main
outputMode: pull_request
outputBranch: fix/login-rate-limit
prTargetBranch: main
outputBranchPolicy: create
title: Fix authentication rate limiting issue
runType: run
acknowledgePromptRisk: false
idempotencyKey: task-2026-06-10-auth
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
baseBranch: main
outputMode: pull_request
outputBranch: fix/login-rate-limit
prTargetBranch: main
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

### Wait Mode Exit Codes

`--follow` is intended for humans. Use `--wait` for scripts that need to block until the created run is terminal.

```bash
repobird run task.json --wait --json --timeout 45m
```

With `--wait --json`, stdout contains exactly one final JSON object:

```json
{
  "run": {
    "ID": "123",
    "PublicID": "run_abc",
    "Status": "completed"
  },
  "exitCode": 0,
  "status": "completed",
  "timedOut": false
}
```

If the wait times out, the JSON object includes the last observed run when available, `timedOut: true`, `exitCode: 5`, and an `error` message.

Exit-code contract:

| Code | Meaning |
|---:|---|
| `0` | Run reached `completed` |
| `1` | Generic CLI, validation, network, or unexpected error |
| `2` | Authentication/API key error |
| `3` | Quota or credits error |
| `4` | Run reached a non-success terminal state such as `failed` or `cancelled` |
| `5` | `--wait` timed out before a terminal state |

## Best Practices

1. **Use Descriptive Prompts**: Be specific about what you want the AI to do
2. **Include Context**: Add background information in the `context` field
3. **Specify Files**: When working on specific files, list them to provide focus
4. **Test with Dry Run**: Always validate complex configurations with `--dry-run`
5. **Use Markdown for Documentation**: For complex tasks, use Markdown format to include detailed documentation
6. **Follow Status**: Use `--follow` for human monitoring and `--wait --json` for scripts

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

Plan runs are temporarily development-only during the OpenCode migration. Use
`runType: run` for normal CLI submissions.

```bash
# 1. Capture planning context locally
cat > feature-plan.yaml << EOF
prompt: Plan implementation for user notifications system
repository: myorg/webapp
target: feature/notifications
title: User notifications system
runType: run
EOF

repobird run feature-plan.yaml

# 2. Submit implementation context
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

- [Configuration Guide](CONFIGURATION-GUIDE.md) - Setting up RepoBird CLI
- [API Reference](API-REFERENCE.md) - Complete API documentation

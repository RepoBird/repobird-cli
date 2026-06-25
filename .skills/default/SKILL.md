---
name: repobird-cli
description: >
  Use when installing, configuring, using, troubleshooting, or developing with
  the RepoBird CLI. Covers authentication, run submission, Basic/Pro cloud agent
  presets, branch-only and PR workflows, repository defaults, status monitoring,
  config files, examples, completions, and local development commands. Agents
  should avoid the experimental human-only TUI and use non-interactive commands.
---

# RepoBird CLI

RepoBird CLI submits one-shot OpenCode-powered cloud agent runs to RepoBird.ai
and tracks progress through non-interactive CLI commands.

Use `repobird` for normal commands. Some installations also provide the `rb`
alias, especially when shell completions are installed.

## Quick Commands

```bash
repobird login
repobird verify
repobird run -r owner/repo -p "Fix the login bug" --follow
repobird basic -r owner/repo "Fix a small bug"
repobird pro -r owner/repo "Implement OAuth"
repobird run task.json --dry-run
repobird status
repobird status RUN_ID --follow
repobird repo list
repobird repo show repo_123
repobird examples schema
repobird examples generate minimal -o task.json
```

## Installation

Install the CLI:

```bash
curl -sSL https://raw.githubusercontent.com/RepoBird/repobird-cli/main/scripts/install.sh | bash
```

Or build locally from source:

```bash
git clone https://github.com/RepoBird/repobird-cli.git
cd repobird-cli
make build
sudo cp build/repobird /usr/local/bin/
```

## Authentication

Prefer interactive login for local use:

```bash
repobird login
repobird verify
repobird info
```

For automation, use the environment variable instead of writing secrets into
files, prompts, shell history, or logs:

```bash
export REPOBIRD_API_KEY=...
```

Config commands:

```bash
repobird config get
repobird config get api-key
repobird config set api-key YOUR_KEY
repobird config set api-url https://repobird.ai
repobird config set color never
repobird config delete api-key
repobird logout
```

Important environment variables:

```bash
REPOBIRD_API_KEY      # API authentication key
REPOBIRD_API_URL      # API endpoint override
REPOBIRD_COLOR        # auto|always|never
REPOBIRD_ENV          # prod|dev; dev selects localhost defaults
REPOBIRD_DEBUG_LOG=1  # debug logging
NO_COLOR              # disable ANSI color when set
```

Never print or commit API keys. Redact keys from command output, docs, examples,
and bug reports.

## Creating Runs

The main command is `repobird run`. It accepts direct flags, JSON, YAML,
Markdown frontmatter, or JSON from stdin.

```bash
repobird run -r owner/repo -p "Fix auth rate limiting"
repobird run --repo owner/repo --prompt "Add tests" --follow
repobird run task.json
repobird run task.yaml --follow
repobird run task.md --dry-run
cat task.json | repobird run
```

Prompt and context input support file and stdin shorthands:

```bash
repobird run -r owner/repo -p @task.md
repobird run -r owner/repo -p - < task.md
repobird run -r owner/repo -p "Refactor auth" --context @requirements.md
repobird run -r owner/repo -p "@@literal-at-prefix"
```

Run creation flags:

```text
-r, --repo owner/repo                 Repository name or numeric ID
-p, --prompt text|@file|-             Task prompt
--context text|@file|-                Additional context
--title text                          Human-readable run title
--base-branch branch                  Branch to start work from
--output-mode pr|branch               PR or branch-only output
--output-branch branch                Branch to push generated commits to
--pr-target-branch branch             Pull request target branch
--output-branch-policy create|reuse   Output branch policy
--branch-only, --no-pr                Push commits without opening a PR
--provider-mode bundled|byok-user|enterprise-gateway
                                      Optional provider routing mode
--provider-credential-id id           Optional provider credential reference
--gitlab-token-reference-id id        Stored GitLab token reference for self-managed GitLab
--source branch                       Legacy alias for --base-branch
--target branch                       Legacy target/output branch alias
--run-type run                        Default public run type
--basic                               Use Basic preset
--pro                                 Use Pro preset
--acknowledge-prompt-risk             Resend after reviewing prompt-risk error
--follow                              Follow run status after creation
--dry-run                             Validate input without creating a run
```

`plan` run type is development-only during the OpenCode migration. Do not
advertise it as a public workflow.

## Presets

Use preset commands when the task fits a default cloud agent model:

```bash
repobird basic -r owner/repo "Fix a small bug"
repobird pro -r owner/repo "Implement OAuth"
repobird run --basic -r owner/repo -p "Fix a small bug"
repobird run --pro -r owner/repo -p "Implement OAuth"
```

Inside a git repository with an `origin` remote, `basic` and `pro` can
auto-detect the repository:

```bash
repobird pro "Fix the failing checkout test"
```

Current presets:

```text
Basic: DeepSeek V4 Flash
Pro:   GLM 5.2
```

RepoBird uses credits as the customer-facing unit for cloud agent work. Do not
describe availability as fixed Basic/Pro monthly run counts unless the current
API response explicitly requires compatibility language.

## Task Files

Minimal JSON:

```json
{
  "repository": "owner/repo",
  "prompt": "Fix the login bug"
}
```

Common fields:

```json
{
  "repository": "owner/repo",
  "prompt": "Implement OAuth2 authentication",
  "baseBranch": "main",
  "outputMode": "pr",
  "outputBranch": "feature/oauth",
  "prTargetBranch": "main",
  "outputBranchPolicy": "create",
  "title": "Add OAuth2 support",
  "runType": "run",
  "context": "Use Google and GitHub providers.",
  "files": ["src/auth/", "config/oauth.json"],
  "branchOnly": false,
  "acknowledgePromptRisk": false
}
```

GitLab repositories use the same run command and task file shape. Managed
GitLab.com repositories may not need an explicit credential reference. For
self-managed GitLab, pass a stored token reference ID, never a raw token:

```bash
repobird run -r group/project -p @task.md --gitlab-token-reference-id glref_123
```

Equivalent task-file fields:

```json
{
  "repository": "group/project",
  "prompt": "Fix the GitLab CI failure",
  "gitlabCredential": {
    "mode": "stored_token_reference",
    "tokenReferenceId": "glref_123"
  }
}
```

Do not put raw GitLab PATs, project tokens, deploy tokens, API keys, or provider
secrets in task files, command arguments, logs, or prompts.

Markdown files use YAML frontmatter for fields; Markdown body content is
appended to `context`.

Generate examples and schema from the CLI:

```bash
repobird examples
repobird examples schema
repobird examples generate run -f yaml -o task.yaml
repobird examples generate minimal -o task.json
```

## Branch And PR Workflows

Default output mode is PR creation. Use branch-only when the agent should push
commits to a branch without opening a PR:

```bash
repobird run -r owner/repo -p "Update generated docs" \
  --output-branch automation/docs \
  --branch-only
```

Repository defaults can persist branch choices server-side. Per-run flags still
override repository defaults.

```bash
repobird repo list
repobird repo list --json
repobird repo show repo_123
repobird repo show repo_123 --json
repobird repo defaults repo_123 --base develop --pr-target release
repobird repo defaults repo_123 --output automation/docs
repobird repo defaults repo_123 --clear-base
repobird repo defaults repo_123 --clear-pr-target
repobird repo defaults repo_123 --clear-output
```

## Monitoring

```bash
repobird status
repobird status --limit 25
repobird status --json
repobird status RUN_ID
repobird status RUN_ID --json
repobird status RUN_ID --follow
repobird st RUN_ID
repobird logs RUN_ID
repobird logs RUN_ID --json
repobird logs RUN_ID --follow
```

Use `--follow` on `run` or `status` when the user wants live polling.
Use `logs --follow` for NDJSON log polling in automation. The public API
documents a run diff endpoint when a diff is available, but this CLI build does
not expose a dedicated `diff` command.

## Human-Only TUI

`repobird tui` is an interactive, experimental dashboard for humans. Agents
should not use it for automation, validation, monitoring, or troubleshooting
because it depends on terminal interaction and key-driven navigation.

Use these non-interactive commands instead:

```bash
repobird status --json
repobird status RUN_ID --json
repobird status RUN_ID --follow
repobird repo list --json
repobird repo show repo_123 --json
repobird run task.json --dry-run
```

## Completion And Help

```bash
repobird --help
repobird COMMAND --help
repobird completion install zsh
repobird completion install bash
repobird completion install fish
repobird completion install powershell
repobird completion install zsh --dry-run
repobird completion zsh
```

`repobird completion install` configures completion for both `repobird` and the
`rb` alias where supported.

## Bulk Runs

Bulk run creation is legacy/development-only while the API migration is in
progress. Prefer creating individual runs with `repobird run`.

## Development

Common local commands:

```bash
make deps
make build
make test
make test-integration
make lint
make fmt
make fmt-check
make check
make install
```

For command changes under `internal/commands/`, run focused Go tests first, then
`make test`; run `make test-integration` when the behavior touches CLI command
contracts or command execution.

Keep CLI behavior backward compatible unless the task explicitly changes it.
Never log sensitive data such as API keys or tokens.

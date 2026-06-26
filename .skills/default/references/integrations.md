# RepoBird Integration Overview

Use these references when creating or reviewing RepoBird-powered automation:
CI jobs, Claude Code hooks, Codex hooks, scheduled maintenance, local wrappers,
or direct API clients.

## Core Rules

- Prefer `repobird run --wait --json` for scripts.
- Prefer `repobird run --dry-run --json` while developing task files.
- Use deterministic `--idempotency-key` values in CI and scheduled jobs.
- Use `--follow` only for humans watching a terminal.
- Use branch-only output when another workflow should inspect generated changes before a PR.
- Store `REPOBIRD_API_KEY` in CI secrets or a secret manager.
- Never place API keys, raw GitLab tokens, provider credentials, or secrets in prompts, task files, logs, or shell history.
- For self-managed GitLab repositories, use `--gitlab-token-reference-id` or `gitlabCredential.tokenReferenceId`.

## CLI Automation Pattern

```bash
repobird run task.md \
  --idempotency-key "task-${GITHUB_RUN_ID:-local}" \
  --wait \
  --json \
  --timeout 45m
```

## Task File Pattern

```markdown
---
repository: owner/repo
baseBranch: main
outputMode: pull_request
prTargetBranch: main
title: Fix repository maintenance checks
files:
  - README.md
  - docs/
---

Fix the requested maintenance issue.

Constraints:
- Keep the PR focused.
- Preserve public behavior unless a failing test proves a bug.
- Run focused validation first.
- Include changed files, commands run, and residual risk in the PR description.
```

## Reference Routing

- CI automation: `references/ci.md`
- Claude/Codex/local hooks: `references/agent-hooks.md`
- Language and framework starter prompts: `references/language-recipes.md`
- Direct REST API clients: `references/api.md`

Human-facing guides live under `docs/use-cases/`.

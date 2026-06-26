# RepoBird Common Use Cases

This directory collects starter workflows for using RepoBird CLI and the RepoBird API in automation.

Use these guides when you want RepoBird to handle repeatable engineering maintenance such as CI-triggered repairs, documentation updates, formatting fixes, lint cleanup, test generation, or framework-specific chores.

## Guides

| Guide | Use it for |
|---|---|
| [CI Automation](CI.md) | GitHub Actions, GitLab CI, and local scripts that submit RepoBird runs and wait for terminal status |
| [Agent Hooks](AGENT-HOOKS.md) | Claude Code hooks, Codex wrappers, pre-commit hooks, and scheduled agent tasks |
| [Language and Framework Recipes](LANGUAGE-FRAMEWORK-RECIPES.md) | Starter prompts for Go, Node, Python, Ruby, Java, .NET, Rust, PHP, mobile, and common frontend stacks |
| [API Integration](API-INTEGRATION.md) | Direct REST API examples when a script or service should call RepoBird without shelling out to the CLI |

## Baseline Pattern

For scripts, prefer `--wait --json` over `--follow`. `--wait --json` returns one final JSON object and a stable exit code.

```bash
repobird run -r owner/repo \
  -p @task.md \
  --idempotency-key "docs-${GITHUB_SHA:-local}" \
  --wait \
  --json \
  --timeout 45m
```

Use `--dry-run --json` while developing a task file:

```bash
repobird run task.yaml --dry-run --json
```

For automation that should push a branch without opening a pull request, use branch-only output:

```bash
repobird run -r owner/repo \
  -p @task.md \
  --output-branch automation/repobird-docs \
  --branch-only \
  --wait \
  --json
```

## Task File Template

Markdown task files work well because the frontmatter carries routing fields and the body carries detailed instructions.

```markdown
---
repository: owner/repo
baseBranch: main
outputMode: pull_request
prTargetBranch: main
title: Update generated documentation
idempotencyKey: docs-refresh-2026-06-26
files:
  - README.md
  - docs/
---

# Task

Update documentation that is stale after the latest code changes.

## Constraints

- Preserve existing tone and structure.
- Do not change public behavior.
- Run the repository documentation or test checks if they exist.
- Open a focused PR with a concise summary and validation notes.
```

## Automation Safety

- Store `REPOBIRD_API_KEY` in CI secrets or a local secret manager. Do not put API keys in task files, prompts, logs, or shell history.
- Use a deterministic `--idempotency-key` for retries from CI or scheduled jobs.
- Use `--force` only after reviewing that a duplicate run is intentional.
- Use `--json` for machine parsing and keep human progress output out of scripts.
- Use branch-only runs for generated content that another workflow should review or aggregate before opening a PR.
- For self-managed GitLab repositories, use `--gitlab-token-reference-id` or `gitlabCredential.tokenReferenceId`; never pass raw GitLab tokens.

## Agent Skill

The default RepoBird skill at [.skills/default/SKILL.md](../../.skills/default/SKILL.md) includes load-on-demand integration references under `.skills/default/references/`. Install one skill and ask your agent for CI, hook, framework, or API integration help when needed.

# Agent Hooks

Agent hooks let local coding agents or developer tooling submit bounded RepoBird tasks without hand-writing the same command every time.

Prefer explicit user-triggered commands over automatic submission from every local failure.

## Claude Code Or Codex Wrapper

```bash
#!/usr/bin/env bash
set -euo pipefail

repo="${REPOBIRD_REPOSITORY:-owner/repo}"
branch="${REPOBIRD_BASE_BRANCH:-main}"
kind="${1:-docs}"

case "$kind" in
  docs)
    title="Update repository documentation"
    prompt="Update README and docs to match the current code. Preserve tone and structure. Run docs checks if available."
    ;;
  lint)
    title="Fix lint and formatting"
    prompt="Fix lint and formatting failures using the repository's existing tools. Preserve behavior."
    ;;
  tests)
    title="Fix failing unit tests"
    prompt="Fix failing unit tests. Determine whether code or tests are stale. Run focused tests first."
    ;;
  *)
    echo "unknown task kind: $kind" >&2
    exit 64
    ;;
esac

task="$(mktemp)"
cat > "$task" <<TASK
---
repository: $repo
baseBranch: $branch
outputMode: pull_request
prTargetBranch: $branch
title: $title
---

$prompt
TASK

repobird run "$task" \
  --idempotency-key "hook-${kind}-$(date +%Y%m%d%H%M)" \
  --wait \
  --json \
  --timeout "${REPOBIRD_TIMEOUT:-45m}"
```

## Pre-Commit Or Pre-Push Repair

Avoid submitting a cloud run on every commit. Use an opt-in environment variable.

```bash
#!/usr/bin/env bash
set -euo pipefail

if [ "${REPOBIRD_AUTO_REPAIR:-0}" != "1" ]; then
  exit 0
fi

if ! npm run lint; then
  repobird run -r owner/repo \
    -p "Fix lint failures from the current branch. Preserve behavior and run npm run lint." \
    --output-branch "automation/lint-${USER:-local}" \
    --branch-only \
    --idempotency-key "prepush-lint-$(git rev-parse --short HEAD)" \
    --wait \
    --json
fi
```

## Hook Design Rules

- Require an explicit trigger for expensive or broad tasks.
- Include only bounded failure output or file lists.
- Keep secrets in environment variables or secret stores.
- Use `--dry-run --json` while developing the hook.
- Use deterministic idempotency keys so retries do not create duplicate runs.
- Prefer branch-only mode when another local or CI workflow should inspect generated changes before opening a PR.

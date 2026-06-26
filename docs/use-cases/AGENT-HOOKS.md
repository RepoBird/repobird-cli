# Agent Hooks With RepoBird

Agent hooks let local coding agents or developer tooling submit bounded RepoBird tasks without hand-writing the same command every time.

The safest default is to generate a task file, run `repobird run --dry-run --json`, then submit with `--wait --json` only when the hook is explicitly invoked or the failure condition is clear.

## Claude Code Hook Pattern

Use a shell wrapper that creates the task and calls RepoBird. Keep the hook small and put policy in the task text.

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

Example hook invocations:

```bash
./scripts/repobird-hook.sh docs
./scripts/repobird-hook.sh lint
./scripts/repobird-hook.sh tests
```

## Codex Hook Pattern

Codex and other local agents can use the same non-interactive wrapper. Prefer explicit user-triggered commands over automatic submission from every local failure.

```bash
#!/usr/bin/env bash
set -euo pipefail

changed_files="$(git diff --name-only HEAD | sed -n '1,80p')"

cat > /tmp/repobird-codex-task.md <<TASK
---
repository: ${REPOBIRD_REPOSITORY:-owner/repo}
baseBranch: ${REPOBIRD_BASE_BRANCH:-main}
outputMode: pull_request
prTargetBranch: ${REPOBIRD_PR_TARGET:-main}
title: ${REPOBIRD_TITLE:-Codex-requested maintenance}
---

Perform this maintenance task:

${REPOBIRD_PROMPT:-Update docs, tests, lint, or formatting as requested by the local agent.}

Changed files visible locally:

$changed_files

Constraints:
- Keep the PR focused.
- Do not include secrets.
- Run the relevant repository checks.
TASK

repobird run /tmp/repobird-codex-task.md --dry-run --json
repobird run /tmp/repobird-codex-task.md \
  --idempotency-key "codex-${USER:-user}-$(date +%Y%m%d%H%M)" \
  --wait \
  --json
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

## Scheduled Local Maintenance

Use a cron job or launchd/systemd timer for recurring maintenance that is safe to run without developer interaction.

```bash
#!/usr/bin/env bash
set -euo pipefail

export REPOBIRD_API_KEY="$(security find-generic-password -a "$USER" -s repobird-api-key -w)"

repobird run -r owner/repo \
  -p "Review dependency, docs, and generated-file drift. Open a PR only for concrete, validated maintenance updates." \
  --idempotency-key "weekly-maintenance-$(date +%G-W%V)" \
  --wait \
  --json \
  --timeout 90m
```

## Hook Design Rules

- Require an explicit trigger for expensive or broad tasks.
- Include only bounded failure output or file lists.
- Keep secrets in environment variables or secret stores.
- Use `--dry-run --json` while developing the hook.
- Use deterministic idempotency keys so retries do not create duplicate runs.
- Prefer branch-only mode when another local or CI workflow should inspect generated changes before opening a PR.

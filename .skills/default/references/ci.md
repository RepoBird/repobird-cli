# CI Automation

Use RepoBird in CI when you want a cloud agent to create a PR or branch from an automated signal: failing tests, stale generated docs, lint drift, missing coverage, or periodic maintenance.

## Good Triggers

- Scheduled docs refreshes.
- Mechanical lint or formatting failures.
- Unit-test failures with clear bounded output.
- Dependency update fallout after Dependabot/Renovate opens a PR.
- Generated files that are stale after schema, API, or docs changes.

Avoid automatic submission when logs may contain secrets, production credentials are needed, or product judgment is required.

## GitHub Actions Pattern

```yaml
name: RepoBird Maintenance

on:
  workflow_dispatch:

jobs:
  repobird:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install RepoBird
        run: curl -fsSL https://repobird.ai/install.sh | sh

      - name: Create task
        run: |
          cat > repobird-task.md <<'TASK'
          ---
          repository: owner/repo
          baseBranch: main
          outputMode: pull_request
          prTargetBranch: main
          title: Fix repository maintenance checks
          ---

          Fix the requested maintenance issue.
          Preserve behavior. Run focused checks first, then broader checks when practical.
          TASK

      - name: Submit RepoBird run
        env:
          REPOBIRD_API_KEY: ${{ secrets.REPOBIRD_API_KEY }}
        run: |
          repobird run repobird-task.md \
            --idempotency-key "maintenance-${GITHUB_RUN_ID}" \
            --wait \
            --json \
            --timeout 45m
```

## Failure Repair Pattern

Capture bounded failure output and embed it in the task body.

```bash
tail -n 250 test-output.txt > failure-summary.txt
cat > task.md <<'TASK'
---
repository: owner/repo
baseBranch: main
outputMode: pull_request
prTargetBranch: main
title: Fix failing unit tests
---

Fix the failing unit tests shown below.
Identify whether the product code or test expectation is wrong.

## Failure Summary
TASK
cat failure-summary.txt >> task.md

repobird run task.md \
  --idempotency-key "tests-${GITHUB_RUN_ID:-local}" \
  --wait \
  --json \
  --timeout 60m
```

## Branch-Only Generated Updates

```bash
repobird run -r owner/repo \
  -p "Regenerate OpenAPI docs and commit the generated output." \
  --output-branch automation/generated-docs \
  --output-branch-policy reuse \
  --branch-only \
  --idempotency-key "generated-docs-${GITHUB_RUN_ID:-local}" \
  --wait \
  --json
```

## Exit Codes

| Code | Meaning |
|---:|---|
| 0 | Run completed |
| 1 | Generic CLI, validation, network, or unexpected error |
| 2 | Authentication or API key error |
| 3 | Credits or quota error |
| 4 | Run reached a failed or cancelled terminal state |
| 5 | Wait timed out |

Treat code `5` as inconclusive. The run may still continue in RepoBird after the CI job times out.

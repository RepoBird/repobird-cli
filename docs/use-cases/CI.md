# CI Automation With RepoBird

Use RepoBird in CI when you want a cloud agent to create a PR or branch from an automated signal: failing tests, stale generated docs, lint drift, missing coverage, or periodic maintenance.

## When CI Should Submit A Run

Good triggers:

- A scheduled docs refresh.
- A failing lint or formatting check where the fix is mechanical.
- Unit-test failures with clear output and a bounded package or module.
- Dependency update fallout after Dependabot/Renovate opens a PR.
- Generated files are stale after a schema, API, or documentation change.

Avoid automatic submission when:

- The failure may expose secrets in logs.
- The task requires production credentials.
- The repository is already under an active incident response.
- The task needs product judgment rather than code or docs changes.

## GitHub Actions: Documentation Refresh

```yaml
name: RepoBird Docs Refresh

on:
  workflow_dispatch:
  schedule:
    - cron: "17 9 * * 1"

jobs:
  docs:
    runs-on: ubuntu-latest
    permissions:
      contents: read
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
          title: Refresh repository documentation
          files:
            - README.md
            - docs/
          ---

          Update documentation that is stale or inconsistent with the current code.
          Keep edits focused. Run available documentation checks, markdown lint,
          and relevant tests if this repository defines them.
          TASK

      - name: Submit RepoBird run
        env:
          REPOBIRD_API_KEY: ${{ secrets.REPOBIRD_API_KEY }}
        run: |
          repobird run repobird-task.md \
            --idempotency-key "docs-refresh-${GITHUB_RUN_ID}" \
            --wait \
            --json \
            --timeout 45m
```

## GitHub Actions: Fix Lint Or Formatting

Run your normal lint gate first. Submit RepoBird only when it fails.

```yaml
name: Lint With RepoBird Repair

on:
  workflow_dispatch:
  pull_request:

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Run lint
        id: lint
        continue-on-error: true
        run: |
          npm ci
          npm run lint

      - name: Install RepoBird
        if: steps.lint.outcome == 'failure'
        run: curl -fsSL https://repobird.ai/install.sh | sh

      - name: Ask RepoBird to fix lint
        if: steps.lint.outcome == 'failure'
        env:
          REPOBIRD_API_KEY: ${{ secrets.REPOBIRD_API_KEY }}
        run: |
          cat > task.md <<'TASK'
          ---
          repository: owner/repo
          baseBranch: main
          outputMode: pull_request
          prTargetBranch: main
          title: Fix lint and formatting failures
          ---

          Fix the current lint and formatting failures.
          Preserve behavior. Prefer the repository's existing formatter and lint scripts.
          Include validation output in the PR description.
          TASK

          repobird run task.md \
            --idempotency-key "lint-${GITHUB_RUN_ID}" \
            --wait \
            --json \
            --timeout 45m
```

## GitHub Actions: Unit Test Repair

Capture enough failure context for the agent, but avoid dumping secrets or huge logs.

```yaml
name: Test With RepoBird Repair

on:
  workflow_dispatch:
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Run tests
        id: test
        continue-on-error: true
        run: |
          set +e
          go test ./... 2>&1 | tee test-output.txt
          echo "exit_code=${PIPESTATUS[0]}" >> "$GITHUB_OUTPUT"
          exit 0

      - name: Install RepoBird
        if: steps.test.outputs.exit_code != '0'
        run: curl -fsSL https://repobird.ai/install.sh | sh

      - name: Submit failing tests to RepoBird
        if: steps.test.outputs.exit_code != '0'
        env:
          REPOBIRD_API_KEY: ${{ secrets.REPOBIRD_API_KEY }}
        run: |
          tail -n 250 test-output.txt > test-failure-summary.txt
          cat > task.md <<'TASK'
          ---
          repository: owner/repo
          baseBranch: main
          outputMode: pull_request
          prTargetBranch: main
          title: Fix failing unit tests
          ---

          Fix the failing unit tests shown below.
          Identify whether the failure is a product bug or stale test expectation.
          Keep changes minimal and run the focused failing tests before broader tests.

          ## Failure Summary
          TASK
          cat test-failure-summary.txt >> task.md

          repobird run task.md \
            --idempotency-key "tests-${GITHUB_RUN_ID}" \
            --wait \
            --json \
            --timeout 60m
```

## GitLab CI

```yaml
repobird_docs:
  image: alpine:3.20
  rules:
    - if: $CI_PIPELINE_SOURCE == "schedule"
    - if: $CI_PIPELINE_SOURCE == "web"
  before_script:
    - apk add --no-cache curl
    - curl -fsSL https://repobird.ai/install.sh | sh
    - export PATH="$HOME/.local/bin:$PATH"
  script:
    - |
      cat > task.yaml <<'TASK'
      repository: group/project
      baseBranch: main
      outputMode: pull_request
      prTargetBranch: main
      title: Refresh project documentation
      prompt: |
        Update stale README and docs content. Preserve public behavior.
        Run available docs and test checks.
      TASK
    - repobird run task.yaml --idempotency-key "docs-${CI_PIPELINE_ID}" --wait --json --timeout 45m
```

For self-managed GitLab repositories, add a stored token reference:

```bash
repobird run task.yaml \
  --gitlab-token-reference-id "$REPOBIRD_GITLAB_TOKEN_REFERENCE_ID" \
  --wait \
  --json
```

## Branch-Only Generated Updates

Use this when CI should produce a branch and a separate workflow should decide whether to open or update a PR.

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

`repobird run --wait --json` returns:

| Code | Meaning |
|---:|---|
| 0 | Run completed |
| 1 | Generic CLI, validation, network, or unexpected error |
| 2 | Authentication or API key error |
| 3 | Credits or quota error |
| 4 | Run reached a failed or cancelled terminal state |
| 5 | Wait timed out |

Treat code `5` as inconclusive. The run may still continue in RepoBird after the CI job times out.

# Reasoning Effort CLI Validation

Date: 2026-08-15
Task: T031
Scope: local, credential-free validation only

## Red-green evidence

Behavior-changing work was preceded by focused contract tests in branch history:

| Behavior | Red test commit(s) | Green implementation commit(s) |
|---|---|---|
| API DTO request/response policy fields | `6d15c7a` | `04d91de` |
| Client model policy fields | `2d85b84`, `4630687`, `54d64c0` | `6d914dc` |
| Opaque configuration validation | `7bbc494` | `db257fc` |
| Request repository mapping and server authority | `b8c4983` | `b476abd` |
| Run flag and request preparation | `5a186bc` | `b16191f` |
| Repository response propagation | `7e9fd4a` | `2d7a2f5`, `ee56f31` |
| CLI status rendering | `745addf` | `5cff9fe` |
| CLI JSON response rendering | `745addf`, `03136b4` | `1978a48` |
| Dedicated TUI details | `abc23d6` | `92817ff` |
| Dashboard details and copy output | `d914fde` | `b022030` |

The request-domain and request-output fields covered by T011 and T014 already existed at the branch base (`22e52ac6` and `5f6b472b`). Those task executions did not introduce new runtime behavior. Their Autospec contract tests (`b8c4983`, `2180947`, `3ba0bb5`, and `03136b4`) provide passing compatibility coverage.

## Validation results

| Check | Result |
|---|---|
| Changed-package tests with `REPOBIRD_API_KEY=''` | PASS, 1,043 individual test cases across 8 packages |
| `TMPDIR=/var/tmp REPOBIRD_API_KEY='' make test` | PASS, 654 individual test cases passed and 2 skipped |
| `TMPDIR=/var/tmp REPOBIRD_API_KEY='' make test-integration` | PASS, 22 integration test cases passed |
| `TMPDIR=/var/tmp REPOBIRD_API_KEY='' make check` | PASS, formatting, vet, lint, and tests |
| `TMPDIR=/tmp REPOBIRD_API_KEY='' go test -race -tags=integration ./tests/integration` | PASS, race-enabled integration test suite |
| `autospec artifact specs/001-reasoning-effort-cli/tasks.yaml` | PASS |
| `git diff --check` | PASS |
| Added live or paid endpoint scan | PASS, zero suspicious production URL additions |
| Secret-safety diff inspection | PASS, matches were policy text and test-only placeholder credentials; no secret values found |

The first plain `REPOBIRD_API_KEY='' make test` attempt exposed an environment-only failure: the harness set `TMPDIR` to a scratch directory nested inside a separate Git repository. `TestDetectRepository_NotGitRepo` therefore discovered that parent repository. The focused test passed with an isolated temporary directory, and all canonical targets then passed with that setting.

No live RepoBird service, production-adjacent system, paid provider, or billable infrastructure was contacted. Expected tests attempted localhost-only failure paths and local mock or `httptest` endpoints.

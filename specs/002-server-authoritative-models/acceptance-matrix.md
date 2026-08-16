# Server-authoritative model workflow acceptance matrix

All evidence below is local, deterministic, credential-cleared, and cost-free unless explicitly marked otherwise. No live RepoBird, provider, deployment, release, Bead, or sibling-repository mutation was performed.

## Autospec and workflow receipts

| Requirement | Direct check | Receipt |
|---|---|---|
| Specify artifact | `autospec artifact specs/002-server-authoritative-models/spec.yaml` | PASS: valid, 5 user stories, 13 functional requirements, 5 non-functional requirements |
| Plan artifact | `autospec artifact specs/002-server-authoritative-models/plan.yaml` | PASS: valid, 5 implementation phases, 0 open questions |
| Tasks artifact | `autospec artifact specs/002-server-authoritative-models/tasks.yaml` | PASS: valid, 38 completed tasks, 8 phases |
| Generated plan runner | `autospec --profile oauth-cheap plan ...` | BLOCKED before artifact generation: connected jcode rejected Autospec's unsupported `-c` argument. Manual plan and tasks were written in the validated repository artifact shape instead of retrying unchanged. |
| TDD red evidence | Focused model/discovery test command recorded before production edits | PASS: failed as expected on missing DTO, endpoint, client, model fields, repository mapping, and command registration before implementation commit `cab4bd3` |

## Contract mapping

| Contract | Direct local check | Receipt |
|---|---|---|
| T061 discovery client tests | `go test ./internal/api -run 'TestClientGetModels'` | PASS: GET path, anonymous-compatible auth, ETag, 304, 429, 503, and 500 behavior |
| T062 public model DTOs | `go test ./internal/api/dto` | PASS: documented fields, safe additive fields, opaque future values, and nested internal-field filtering |
| T063 discovery and selected-model API client | Focused API tests plus `make check` | PASS: typed GET client, conditional request metadata, stable errors, protected-request auth preserved |
| T064 built Cobra models command | `go test ./internal/commands -run 'TestModelsCommand'` | PASS: command registration, human output, JSON output, and local fixture behavior |
| T065 built Cobra models command | `make build` and integration command tests | PASS: binary exposes `repobird models` and additive `repobird models list` |
| T066 selected-model submission tests | Focused command, model, and repository tests | PASS: exact explicit values, omission, Basic/Pro behavior, and server rejection without fallback |
| T067 selected-model pass-through | Local POST serialization assertions | PASS: `opencodeModel` and `opencodeProvider` cross flags, config, domain, repository, and request DTO unchanged |
| T072 built CLI discovery contract | `make test-integration` and explicit built-binary test | PASS: local httptest human/JSON discovery and run submission acceptance |

## Feature requirements

| Requirement | Direct check | Receipt |
|---|---|---|
| Human and stable JSON `repobird models` | `TestBuiltBinaryModelsAndRunSubmissionContracts` | PASS: both modes render fixture revision and descriptors, JSON is one parseable value |
| Safe additive fields in human output | focused `TestModelsCommandOutputsHumanAndStableJSONFromPublicCatalog` plus built-binary acceptance | PASS after lifecycle review repair: catalog-level and descriptor-level safe fields render deterministically while forbidden nested fields remain absent |
| Safe additive DTO fields and opaque future values | `internal/api/dto/models_test.go` | PASS: safe additive fields and unfamiliar provider/capability values survive round-trip |
| No internal policy data | DTO and integration forbidden-field assertions | PASS: policy, worker, credential, tenant, dispatch, secret, token, and hash-looking fields omitted |
| Basic/Pro omit CLI defaults | local Basic/Pro POST assertions and existing run tests | PASS: no `opencodeModel` or `opencodeProvider` sent when omitted |
| Explicit model/provider exact pass-through | flags, JSON/YAML/Markdown, domain, repository, and POST tests | PASS: values remain exact and optional fields use omission semantics |
| Server rejection has no fallback | `TestAPIRunRepositoryCreateDoesNotFallbackAfterModelRejection` | PASS: one POST only, stable rejection propagated, no substitution |
| Existing auth/provider credentials and errors | focused repository/command tests, full suite, and integration suite | PASS: fake bearer auth and credential fields preserved, stable errors retained |
| No cache-based admission | code review plus omission/rejection tests | PASS: no catalog cache or local admission gate added, discovery is display-only |

## Required validation

| Check | Receipt |
|---|---|
| Focused Go package tests | PASS: `REPOBIRD_API_KEY= go test ./internal/api/dto ./internal/api ./internal/models ./internal/repository ./internal/utils ./internal/commands` |
| Affected-package race tests | PASS: `REPOBIRD_API_KEY= go test -race ./internal/api/dto ./internal/api ./internal/models ./internal/repository ./internal/utils ./internal/commands` |
| `REPOBIRD_API_KEY= make test` | PASS with `TMPDIR=/tmp` to avoid the harness's parent `.jcode` Git config. 664/666 passed, 2 skipped. The first unisolated run exposed a pre-existing temp-directory/global Git-config interaction in `TestDetectRepository_NotGitRepo`, then the same suite passed outside that environment. |
| `REPOBIRD_API_KEY= make test-integration` | PASS with `TMPDIR=/tmp`, 23/23 passed, one keyring-related test skipped by repository fixture. An initial golden-help mismatch from the new command was repaired by updating the expected fixture, then the complete suite passed. |
| `REPOBIRD_API_KEY= make check` | PASS with `TMPDIR=/tmp`: `go vet`, `golangci-lint` with 0 issues, format check, and full race test suite |
| Built binary human/JSON and submission acceptance | PASS: `make build`; `go test -tags=integration ./tests/integration -run TestBuiltBinaryModelsAndRunSubmissionContracts -count=1` |
| `git diff --check` | PASS |
| Explicit-file staged conventional commits | PASS: implementation commit `e2ec6b1` and the final documentation/artifact commit use explicit file lists and conventional messages |
| Clean worktree | PASS: verified with `git status --short` after the final documentation/artifact commit |

## Compatibility, risk, and safety

- Compatibility evidence: current-base tests preserve Basic/Pro flags, run types, config formats, provider credentials, authentication, JSON omission, stable errors, and legacy command aliases. The older non-ancestor branch `agent/repobird-cli-4y6` at `6a1aa8c` was read-only compatibility evidence only and was not merged, rebased, reset, edited, or pushed.
- Risk: low to medium. Blast radius is limited to model discovery, run request field propagation, command registration, documentation, and integration help fixtures. Existing run paths retain their prior defaults except that CLI-owned model IDs are no longer sent.
- Mitigation and rollback: server admission remains authoritative, explicit values are opaque, omitted values stay omitted, no live cache affects submission, and rollback is a normal revert of the feature commits before landing.
- Unresolved issues: the configured Autospec plan runner remains incompatible with this Jcode CLI because of its unsupported `-c` option. No product behavior is blocked because the hand-authored plan and tasks validate successfully.
- Next action after lifecycle-owner review: hand the committed worktree to `merge-wt` for current-base review and landing. This worktree itself is not merged, pushed, or cleaned up here.

## Scope and safety

- Sibling `/home/ari/repos/repobird-next/.worktrees/agent/repobird-next-d9ax-central-model-policy` sources and commit `6a1aa8c` were read-only evidence only.
- No Bead/provider mutation, worktree creation/removal, merge, push, release, live call, preview call, or provider spend was performed.

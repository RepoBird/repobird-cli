# Server-authoritative model workflow acceptance matrix

All evidence below is local, deterministic, credential-cleared, and cost-free unless explicitly marked otherwise. No live RepoBird, provider, deployment, release, Bead, or sibling-repository mutation is permitted.

## Autospec and workflow receipts

| Requirement | Direct check | Receipt |
|---|---|---|
| Specify artifact | `autospec artifact specs/002-server-authoritative-models/spec.yaml` | PASS: 2026-08-15 |
| Plan artifact | `autospec artifact specs/002-server-authoritative-models/plan.yaml` | PASS: 2026-08-15 |
| Tasks artifact | `autospec artifact specs/002-server-authoritative-models/tasks.yaml` | PASS: 2026-08-15 |
| Generated plan runner | `autospec --profile oauth-cheap plan ...` | BLOCKED before artifact generation: connected jcode rejected Autospec's unsupported `-c` argument; manual plan/tasks were written and schema-validated instead |
| TDD red evidence | `rtk env REPOBIRD_API_KEY= go test ./internal/api/dto ./internal/api ./internal/models ./internal/repository ./internal/commands -run 'TestPublicModels|TestClientGetModels|TestRunConfigPreservesExplicitOpenCodeSelection|TestRunConfigOmitsUnspecifiedOpenCodeSelection|TestAPIRunRepositoryCreatePassesExplicitOpenCodeSelection|TestModelsCommandIsRegistered'` | PASS: 2026-08-15, failed before production edits as expected: missing public DTO/endpoint/client/model fields, omitted repository mapping, and unregistered command |

## Contract mapping

| Contract | Direct local check | Receipt |
|---|---|---|
| T061 discovery client tests | `go test ./internal/api` focused test | PENDING |
| T062 public model DTOs | `go test ./internal/api/dto` focused test | PENDING |
| T063 discovery and selected-model API client | `go test ./internal/api` focused test | PENDING |
| T064 built Cobra models command tests | `go test ./internal/commands` focused test | PENDING |
| T065 built Cobra models command | `go test ./internal/commands` and built binary | PENDING |
| T066 selected-model submission tests | `go test ./internal/commands ./internal/repository` focused tests | PENDING |
| T067 selected-model pass-through | request serialization and command tests | PENDING |
| T072 built CLI discovery contract | `REPOBIRD_API_KEY= make test-integration` plus explicit local built-binary test | PENDING |

## Feature requirements

| Requirement | Direct check | Receipt |
|---|---|---|
| Human and stable JSON `repobird models` | built binary with local httptest server | PENDING |
| Safe additive DTO fields and opaque future values | DTO round-trip/redaction tests | PENDING |
| No internal policy data | DTO/output forbidden-field assertions | PENDING |
| Basic/Pro omit CLI defaults | dry-run and POST request assertions | PENDING |
| Explicit model/provider exact pass-through | flags/config/domain/repository POST assertions | PENDING |
| Server rejection has no fallback | local 400/unsupported fixture and request count | PENDING |
| Existing auth/provider credentials and errors | compatibility tests | PENDING |
| No cache-based admission | request construction tests and no cache implementation | PENDING |

## Required validation

| Check | Receipt |
|---|---|
| Focused Go package tests | PENDING |
| Race tests for affected packages | PENDING |
| `REPOBIRD_API_KEY= make test` | PENDING |
| `REPOBIRD_API_KEY= make test-integration` | PENDING |
| `REPOBIRD_API_KEY= make check` | PENDING |
| Built binary human/JSON and submission acceptance | PENDING |
| `git diff --check` | PENDING |
| Explicit-file staged conventional commits | PENDING |
| Clean worktree | PENDING |

## Scope and safety

- Sibling `/home/ari/repos/repobird-next/.worktrees/agent/repobird-next-d9ax-central-model-policy` sources and commit `6a1aa8c` are read-only evidence only.
- The older branch is not an ancestor and will not be merged, rebased, reset onto, edited, or pushed.
- No Bead/provider mutation, worktree creation/removal, merge, push, release, or live/costly validation is performed here.

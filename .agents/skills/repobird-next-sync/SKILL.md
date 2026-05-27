---
name: repobird-next-sync
description: "Compare RepoBird CLI with the sibling ../repobird-next app when updating API integrations, DTOs, schemas, endpoints, authentication, run/template/repository behavior, or CLI docs that must match the live web app."
---

# RepoBird Next Sync

Use this repo-local skill before changing CLI API integrations, DTOs, endpoint paths, request/response mapping, auth behavior, template APIs, repository APIs, run creation/status/log/diff behavior, credits/usage display, or docs that claim API behavior.

The sibling app at `../repobird-next` is the implementation source of truth. CLI docs and OpenAPI files are useful, but verify them against route handlers, schemas, utilities, and tests before changing Go code.

## First Checks

From `repobird-cli`:

```bash
test -d ../repobird-next
git status --short --branch
git -C ../repobird-next status --short --branch
sed -n '1,220p' ../repobird-next/AGENTS.md
```

If `../repobird-next` is missing, stop and report that the sync source is unavailable.

Preserve local work in both repos. Do not use `git stash`, broad checkout/reset commands, `git add -A`, or `git add .`.

## Source Map

Use these files first, then expand only as needed:

- App route handlers: `../repobird-next/src/app/api/**/route.ts`
- CLI-facing routes: `../repobird-next/src/app/api/v1/**/route.ts`
- Public template API contracts: `../repobird-next/specs/007-cli-template-api/contracts/openapi.yaml`
- Other feature contracts: `../repobird-next/specs/*/contracts/**`
- Current database schema: `../repobird-next/src/models/Schema.ts`
- API helpers and validation: `../repobird-next/src/utils/apiUtils.ts`, `../repobird-next/src/utils/apiWrapper.ts`, `../repobird-next/src/utils/apiKeyUtils.ts`
- API base/config: `../repobird-next/src/config/api.ts`
- Relevant app tests: `../repobird-next/src/app/api/**/*.test.ts`, `../repobird-next/tests/integration/*api*.test.ts`
- CLI client and DTOs: `internal/api/client.go`, `internal/api/endpoints.go`, `internal/api/dto/*.go`
- CLI models/services: `internal/models/*.go`, `internal/services/*.go`, `internal/repository/*.go`
- CLI command surfaces: `internal/commands/*.go`, `internal/tui/**`
- CLI API docs: `docs/CLI_API_SPECIFICATION.yaml`, `docs/API-REFERENCE.md`

## Discovery Commands

Find matching server endpoints:

```bash
rg --files ../repobird-next/src/app/api -g 'route.ts' -g '!node_modules/**' -g '!.next/**' | sort
rg -n "export async function (GET|POST|PUT|PATCH|DELETE)|NextRequest|NextResponse|validate|safeParse|z\\." ../repobird-next/src/app/api/v1 ../repobird-next/src/utils -g '*.ts'
```

Find schemas and contracts:

```bash
rg -n "pgTable|relations\\(|z\\.object|interface .*Request|type .*Response|credits|runType|apiKey|repository|template" ../repobird-next/src/models ../repobird-next/src/contracts ../repobird-next/specs -g '*.ts' -g '*.yaml' -g '*.md'
```

Find CLI integration points:

```bash
rg -n "api/v1|RunRequest|RunResponse|ListRuns|CreateRun|Repository|Template|Credits|Usage|runType|Authorization|Bearer|json:" internal docs -g '*.go' -g '*.md' -g '*.yaml'
```

Convert Next route paths mechanically:

- `src/app/api/v1/runs/route.ts` -> `/api/v1/runs`
- `src/app/api/v1/runs/[id]/route.ts` -> `/api/v1/runs/{id}`
- `src/app/api/v1/templates/[id]/execute/route.ts` -> `/api/v1/templates/{id}/execute`

## Comparison Workflow

1. Identify the user-facing CLI behavior or endpoint being changed.
2. Read the matching `../repobird-next` route handler and any helpers it calls.
3. Read the route test or integration test for expected request, response, error, and auth behavior.
4. Read the relevant schema/contracts if the handler maps database fields or validates with Zod.
5. Compare with CLI endpoint constants, DTO JSON tags, command flags, config parsing, and docs.
6. Update the CLI code and tests so the wire contract matches server behavior while preserving backward-compatible CLI flags when practical.
7. Update `docs/CLI_API_SPECIFICATION.yaml` and `docs/API-REFERENCE.md` only after source verification.

Prefer current server behavior over stale docs. If server source, contracts, and docs disagree, state the disagreement in the handoff and update only the artifacts required by the task.

## Integration Rules

- Treat OpenCode as the forward-looking run workflow unless existing API compatibility requires Claude-era naming.
- Product behavior is credits-based. Do not add Basic/Pro run-count assumptions unless the server endpoint still returns those fields for compatibility.
- Treat Basic and Pro in CLI run creation as cloud-agent/model presets, not fixed run-count plan limits. Current CLI shortcuts are `repobird basic "prompt"`, `repobird pro "prompt"`, `repobird run --basic ...`, and `repobird run --pro ...`.
- Keep Basic/Pro preset defaults synced with `../repobird-next/src/services/opencodeModelPolicy.ts`: Basic uses `openrouter/deepseek/deepseek-v4-flash`; Pro uses `openrouter/moonshotai/kimi-k2.6`.
- For Basic/Pro preset requests, preserve the OpenCode wire contract by sending `agent: "opencode"`, `opencodeModel`, and `opencodeProvider` when the CLI chooses the preset model.
- Match JSON field names exactly; Go struct names can differ, JSON tags cannot.
- Preserve unknown response fields where compatibility matters; avoid narrowing DTOs if the CLI only needs a subset.
- Keep auth as API-key bearer auth unless the matching route explicitly requires another mechanism.
- Never log API keys, bearer tokens, provider credentials, or scoped environment values.
- For error handling, inspect server error shapes instead of assuming a single `{error,message}` format.
- For nullable server fields, use pointer or explicit optional handling in Go instead of relying on zero values when the distinction affects behavior.

## Validation

Run focused tests first:

```bash
go test ./internal/api/... ./internal/models/... ./internal/commands/...
```

For command/API behavior changes, also run when feasible:

```bash
make test-integration
make test
```

For significant API sync work, run:

```bash
make check
```

If a validation failure appears unrelated or pre-existing, isolate it with the smallest failing test and report the evidence.

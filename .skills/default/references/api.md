# Direct API Integration

Use the API directly when a service, internal tool, or non-shell automation needs to create and monitor RepoBird runs. Use the CLI when shelling out is acceptable; it already handles config files, validation, idempotency headers, retries, and script-friendly wait output.

Live API references:

- REST docs: `https://repobird.ai/docs/api-reference`
- OpenAPI YAML: `https://api.repobird.ai/api/v1/openapi.yaml`

## Authentication

Send the API key as a bearer token:

```http
Authorization: Bearer ${REPOBIRD_API_KEY}
```

Do not store API keys in prompts, task files, logs, browser-visible config, or source code.

## Create A Run

For direct API calls, use `runType: "basic"` or `runType: "pro"` explicitly.
The CLI's task-file `runType: "run"` is a CLI compatibility value and should
not be copied into new direct API integrations.

```bash
curl -sS https://api.repobird.ai/api/v1/runs \
  -H "Authorization: Bearer $REPOBIRD_API_KEY" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: docs-refresh-2026-06-26" \
  -d '{
    "repositoryName": "owner/repo",
    "prompt": "Update stale documentation and open a focused PR.",
    "baseBranch": "main",
    "outputMode": "pull_request",
    "prTargetBranch": "main",
    "runType": "pro",
    "agent": "opencode"
  }'
```

Branch-only run:

```json
{
  "repositoryName": "owner/repo",
  "prompt": "Regenerate API docs and push the generated files.",
  "baseBranch": "main",
  "outputMode": "branch",
  "outputBranch": "automation/generated-docs",
  "outputBranchPolicy": "reuse",
  "runType": "pro",
  "agent": "opencode"
}
```

Self-managed GitLab repositories should use a stored token reference:

```json
{
  "repositoryName": "group/project",
  "prompt": "Fix the GitLab CI failure.",
  "gitlabCredential": {
    "mode": "stored_token_reference",
    "tokenReferenceId": "glref_123"
  }
}
```

## Poll A Run

```bash
run_id="run_123"

while true; do
  response="$(curl -sS "https://api.repobird.ai/api/v1/runs/$run_id" \
    -H "Authorization: Bearer $REPOBIRD_API_KEY")"

  status="$(printf '%s' "$response" | jq -r '.status // .data.status')"
  case "$status" in
    completed)
      exit 0
      ;;
    failed|cancelled)
      printf '%s\n' "$response" >&2
      exit 4
      ;;
  esac

  sleep 15
done
```

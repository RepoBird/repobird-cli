# RepoBird API Integration

Use the API directly when a service, internal tool, or non-shell automation needs to create and monitor RepoBird runs. Use the CLI when shelling out is acceptable; it already handles config files, validation, idempotency headers, retries, and script-friendly wait output.

For the live API contract, use:

- REST docs: <https://repobird.ai/docs/api-reference>
- OpenAPI YAML: <https://api.repobird.ai/api/v1/openapi.yaml>

## Authentication

Send the API key as a bearer token:

```http
Authorization: Bearer ${REPOBIRD_API_KEY}
```

Do not store API keys in prompts, task files, logs, browser-visible config, or source code.

## Create A Run

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

## Node Fetch Example

```ts
type CreateRunResponse = {
  data?: {
    id?: string | number;
    publicId?: string;
    status?: string;
  };
};

export async function createRepoBirdRun() {
  const apiKey = process.env.REPOBIRD_API_KEY;
  if (!apiKey) throw new Error("REPOBIRD_API_KEY is required");

  const response = await fetch("https://api.repobird.ai/api/v1/runs", {
    method: "POST",
    headers: {
      authorization: `Bearer ${apiKey}`,
      "content-type": "application/json",
      "idempotency-key": `docs-${process.env.GITHUB_RUN_ID ?? Date.now()}`
    },
    body: JSON.stringify({
      repositoryName: "owner/repo",
      prompt: "Fix failing unit tests and open a focused PR.",
      baseBranch: "main",
      outputMode: "pull_request",
      prTargetBranch: "main",
      runType: "pro",
      agent: "opencode"
    })
  });

  if (!response.ok) {
    throw new Error(`RepoBird create run failed: ${response.status} ${await response.text()}`);
  }

  return (await response.json()) as CreateRunResponse;
}
```

## Python Requests Example

```python
import os
import time
import requests

API_URL = "https://api.repobird.ai/api/v1"


def create_run() -> str:
    api_key = os.environ["REPOBIRD_API_KEY"]
    response = requests.post(
        f"{API_URL}/runs",
        headers={
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
            "Idempotency-Key": f"tests-{int(time.time())}",
        },
        json={
            "repositoryName": "owner/repo",
            "prompt": "Fix failing Python tests and open a focused PR.",
            "baseBranch": "main",
            "outputMode": "pull_request",
            "prTargetBranch": "main",
            "runType": "pro",
            "agent": "opencode",
        },
        timeout=30,
    )
    response.raise_for_status()
    data = response.json().get("data", response.json())
    return data.get("publicId") or str(data["id"])


def wait_for_run(run_id: str) -> None:
    api_key = os.environ["REPOBIRD_API_KEY"]
    while True:
        response = requests.get(
            f"{API_URL}/runs/{run_id}",
            headers={"Authorization": f"Bearer {api_key}"},
            timeout=30,
        )
        response.raise_for_status()
        data = response.json().get("data", response.json())
        status = str(data.get("status", "")).lower()
        if status == "completed":
            return
        if status in {"failed", "cancelled"}:
            raise RuntimeError(f"RepoBird run ended with {status}: {data}")
        time.sleep(15)
```

## Prefer The CLI For Local Scripts

The CLI is usually better for shell automation:

```bash
repobird run task.md --wait --json --timeout 45m
repobird status RUN_ID --json
repobird logs RUN_ID --json
```

Use the API when your integration already owns HTTP clients, retries, telemetry, and secret handling.

For direct API calls, use `runType: "basic"` or `runType: "pro"` explicitly. The CLI's public task-file `runType: "run"` remains supported by the CLI compatibility path, but direct API integrations should avoid depending on compatibility normalization.

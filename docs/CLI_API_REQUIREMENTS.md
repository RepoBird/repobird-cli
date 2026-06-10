# API Requirements from RepoBird-Next Server

## Critical Information Needed

### 1. Authentication & API Keys

#### API Key Management
- **API Key Format**: Bearer token (obtained from https://repobird.ai/dashboard/user-profile)
- **API Key Generation**: Via RepoBird web dashboard only
- **API Key Scopes/Permissions**: Full access to user's repositories and runs
- **API Key Expiration**: Check if active via verify endpoint
- **Rate Limiting**: Based on user's tier (free/pro/enterprise)
- **Multiple Keys**: One active key per user

#### Authentication Endpoints
```typescript
// Simple API key authentication - no login/logout needed
GET    /api/v1/auth/verify      // Verify API key is valid and active
GET    /api/v1/user              // Get user info with tier details
```

### 2. Agent Run Schema

#### Create Run Request Schema (Based on issueRunSchema)
```typescript
interface CreateRunRequest {
  // Required fields
  prompt: string;              // The task/prompt for the agent
  
  // Repository identification (one required)
  repository?: string;         // Format: "owner/repo" OR just "repo" name
  repoId?: number;            // Direct repo ID from dashboard
  // Note: If neither provided, CLI can detect from git config in CWD
  
  // Optional fields
  title?: string;              // Title for the run (auto-generated if not provided, defaults to "No Title")
  baseBranch?: string;         // Branch to start work from (default: repo's defaultBranch)
  outputMode?: 'pull_request' | 'branch'; // 'pull_request' creates a pull request; 'branch' pushes commits without a PR. CLI accepts 'pr' as an alias.
  outputBranch?: string;       // Branch to push generated commits to
  prTargetBranch?: string;     // PR target branch (defaults to baseBranch)
  outputBranchPolicy?: 'create' | 'reuse'; // Whether to create or reuse outputBranch
  source?: string;             // Legacy alias for baseBranch
  target?: string;             // Legacy target alias; branch-only maps it to outputBranch
  issueNumber?: number;        // GitHub issue number to work on
  pullRequestNumber?: number;  // Existing PR to modify
  branchOnly?: boolean;        // Legacy alias for outputMode='branch'
  acknowledgePromptRisk?: boolean; // Resend only after reviewing PROMPT_RISK_ACK_REQUIRED
  
  // Agent configuration
  runType?: 'run' | 'plan' | 'basic' | 'pro'; // CLI presets use 'basic'/'pro'; legacy plan is development-only during the OpenCode migration
  agent?: 'opencode';          // CLI sends OpenCode for cloud-agent runs
  opencodeModel?: string;      // Basic default: openrouter/deepseek/deepseek-v4-flash; Pro default: openrouter/moonshotai/kimi-k2.6
  opencodeProvider?: 'openrouter';
  context?: string;            // Additional context for the agent
  
  // Execution options
  triggerSource?: 'ui' | 'cli' | 'api';  // Source of trigger (default: 'cli')
  timeout?: number;            // Execution timeout in seconds (default: 45*60)
  
  // Additional context
  files?: string[];           // Specific files to focus on
  excludeFiles?: string[];    // Files to ignore
  
  // Notifications  
  emailNotification?: boolean; // Use user's notification preferences
```

#### Run Response Schema (Based on issueRunSchema)
```typescript
interface RunResponse {
  id: number;                 // Unique run ID
  publicId?: string;          // Public run ID used in dashboard URLs
  status: 'QUEUED' | 'INITIALIZING' | 'PROCESSING' | 'POST_PROCESS' | 'DONE' | 'FAILED';
  createdAt: string;          // ISO timestamp
  updatedAt: string;          // Last update timestamp
  
  // Execution details
  repoId: number;
  repository: string;         // Full repo name (owner/repo)
  baseBranch?: string;        // Branch started from
  outputMode?: 'pull_request' | 'branch';
  outputBranch?: string;      // Branch generated commits were pushed to
  prTargetBranch?: string;    // PR target branch
  source?: string;            // Legacy branch started from
  target?: string;            // Legacy target branch
  agent: 'opencode';          // Default cloud agent
  runType?: 'basic' | 'pro' | 'pro-plan';  // Run type returned by issue-run APIs
  
  // GitHub integration
  issueNumber?: number;
  issueTitle: string;
  issueDescription: string;
  pullRequestId?: string;
  pullRequestUrl?: string;
  
  // Results
  plan?: string;              // Generated plan (for plan runs)
  researchNotes?: string;     // Research notes
  diffString?: string;        // Git diff of changes
  filesModified?: string[];   // List of modified files
  errors?: string[];          // Any errors encountered
  
  // Metrics
  totalDuration?: number;     // Total execution time in seconds
  agentRunDuration?: number;  // Agent-specific run time
  
  // Logs
  commandLogUrl?: string;     // URL to command logs (if available)
}

interface FileChange {
  path: string;
  action: 'created' | 'modified' | 'deleted';
  diff?: string;              // File diff
```

### 3. API Endpoints

#### Core Endpoints Needed
```typescript
// Runs (issueRunSchema)
POST   /api/v1/runs                    // Create new run
GET    /api/v1/runs                    // List user's runs with pagination
GET    /api/v1/runs/{id}               // Get run details
DELETE /api/v1/runs/{id}               // Cancel run (if QUEUED/INITIALIZING)
GET    /api/issue-runs/{id}/agent-logs // Get agent logs as NDJSON (current compatibility endpoint)
GET    /api/v1/runs/{id}/diff          // Get diff of changes

// Repositories (repoSchema)
GET    /api/v1/repositories            // List user's accessible repos
GET    /api/v1/repositories/{id}       // Get repo details
GET    /api/v1/repositories/search     // Search repos by name

// User & Tier Info (userSchema + tiersSchema)
GET    /api/v1/user                    // Current user info with tier
GET    /api/v1/user/usage              // Remaining runs (basic/pro/plan)
GET    /api/v1/user/tier               // Tier details and limits

// GitHub Integration
GET    /api/v1/github/issues/{owner}/{repo}     // List repo issues
GET    /api/v1/github/pulls/{owner}/{repo}      // List repo PRs
```

### 4. Status Polling

#### Polling Strategy
- **Polling Interval**: 5 seconds recommended
- **Stop Conditions**: Stop polling when status is 'DONE' or 'FAILED'
- **Timeout**: Stop after 60 minutes (max run time)

```typescript
// CLI polling logic
const pollStatus = async (runId: number) => {
  while (true) {
    const status = await getRunStatus(runId);
    if (status === 'DONE' || status === 'FAILED') {
      break;
    }
    await sleep(5000); // 5 second interval
  }
};
```

### 5. Error Responses

#### Standard Error Format
```typescript
interface ErrorResponse {
  error: {
    code: string;           // Error code
    message: string;        // Human-readable message
    details?: any;          // Additional error details
    statusCode: number;     // HTTP status code
    requestId?: string;     // Request ID for debugging
  };
}

// Common error messages
- "No runs remaining" - User has exhausted their run quota
- "Repository not found or not connected" - Repo doesn't exist or no GitHub App installation
- "Invalid API key" - API key is invalid, expired, or revoked
- "Rate limit exceeded" - Too many requests (future implementation)
```

### 6. Pagination & Filtering

#### List Endpoints Parameters
```typescript
interface ListParams {
  page?: number;            // Page number (1-based?)
  limit?: number;           // Items per page (max?)
  sort?: string;            // Sort field (createdAt, status?)
  order?: 'asc' | 'desc';   // Sort order
  
  // Filters
  status?: string;          // Filter by status
  repository?: string;      // Filter by repository
  branch?: string;          // Filter by branch
  since?: string;           // Filter by date (ISO)
  until?: string;           // Filter by date (ISO)
}

interface ListResponse<T> {
  items: T[];
  total: number;
  page: number;
  limit: number;
  hasMore: boolean;
}
```

### 7. Configuration & Limits

#### System Configuration
- **API Base URL**: 
  - Production (default): https://repobird.ai
  - Development: Set via .env file (e.g., REPOBIRD_API_URL=http://localhost:3000)
  - CLI should check .env for development override, otherwise use production URL
- **API Version**: Version in path (/v1/)
- **Max Request Size**: 10MB for prompts/context
- **Timeout Limits**: Default 45 minutes, max 60 minutes
- **Rate Limits**: Based on tier (tiersSchema):
  - Free: 3 runs/month (basicRunsPerPeriod)
  - Pro: 30 runs/month (proRunsPerPeriod)
  - Additional runs: $10 each (pricePerAdditionalRun)
- **Concurrent Run Limits**: Unlimited (isolated cloud environments)

### 8. Additional Features

#### Confirmed Features
1. **GitHub Integration**:
   - GitHub App installation required (via web dashboard)
   - Support for GitHub only (gitProviderEnum)
   - No direct GitHub token support (uses installation)

2. **File Handling**:
   - No direct file upload in initial version
   - Files specified via paths in repository
   - Context provided as text string

3. **Run Types**:
   - Run execution uses the current OpenCode-backed agent contract.
   - Plan mode is temporarily unavailable during the OpenCode migration.

4. **Batch Operations**:
   - Bulk/batch run creation is development-gated and unavailable in current public CLI builds.
   - CLI support requires `REPOBIRD_ENV=development` and `REPOBIRD_DEV_ENABLE_BULK_RUNS=1`.
   - Public CLI docs should direct users to `repobird run` for individual runs.

5. **Usage Metrics**:
   - Credit balance (creditBalance from the credit wallet)
   - Remaining pro/plan runs are legacy compatibility fields only
   - Period reset date (lastPeriodResetDate from userSchema)
   - Note: Basic runs (remainingBasicRuns) not available currently

## Example API Call Flow

```bash
# CLI uses https://repobird.ai by default
# For development, set REPOBIRD_API_URL in .env file

# 1. Verify API key (obtained from https://repobird.ai/dashboard/user-profile)
curl -X GET https://repobird.ai/api/v1/auth/verify \
  -H "Authorization: Bearer <your-api-key>"

# 2. Create a run (minimal)
curl -X POST https://repobird.ai/api/v1/runs \
  -H "Authorization: Bearer <your-api-key>" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Fix the login bug in auth.js",
    "repository": "acme/webapp"
  }'

# 2b. Create a run (with options)
curl -X POST https://repobird.ai/api/v1/runs \
  -H "Authorization: Bearer <your-api-key>" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Fix the login bug in auth.js",
    "repoId": 12345,
    "source": "main",
    "target": "fix/login-bug",
    "runType": "run",
    "triggerSource": "cli"
  }'

# 3. Check status
curl -X GET https://repobird.ai/api/v1/runs/12345 \
  -H "Authorization: Bearer <your-api-key>"

# 4. Get logs
curl -X GET https://repobird.ai/api/issue-runs/12345/agent-logs \
  -H "Authorization: Bearer <your-api-key>"

# 5. List all runs
curl -X GET https://repobird.ai/api/v1/runs?page=1&limit=10 \
  -H "Authorization: Bearer <your-api-key>"
```

## Required Documentation

### API Documentation Needs
1. **OpenAPI/Swagger Spec**: Complete API specification
2. **Authentication Guide**: Simple Bearer token with API key from dashboard
3. **Rate Limiting Details**: Based on user tier (free/pro/enterprise)
4. **Status Codes**: Standard HTTP codes + custom error messages
5. **Error Code Reference**: Matching issueRunSchema status values
6. **Tier Information**: Limits per tier from tiersSchema
7. **CLI Examples**: Go CLI tool examples

## Testing Requirements

### API Testing Needs
1. **Test Environment**: Sandbox/staging API endpoint?
2. **Test API Keys**: How to get test credentials?
3. **Test Repositories**: Sample repos for testing?
4. **Mock Data**: Can we get mock responses?
5. **Load Testing**: Allowed on staging?

## Security Considerations

### Security Questions
1. **API Key Storage**: Best practices for key storage?
2. **Encryption**: TLS version requirements?
3. **IP Whitelisting**: Supported/required?
4. **Audit Logging**: What's logged on API calls?
5. **Compliance**: SOC2, GDPR considerations?

---

## Next Steps for CLI Implementation

1. **API Client Development**:
   - Implement Bearer token authentication
   - Support .env file for development API URL override
   - Repository resolution: name → repoId via API
   - Auto-detect repo from git config if not specified
   - Preserve legacy CLI runType values for compatibility, but keep plan submissions development-only during the OpenCode migration while exposing Basic/Pro presets
   - Basic preset uses DeepSeek V4 Flash; Pro preset uses Kimi K2.6
   - Handle status enum values correctly

2. **Core Features**:
   - Run creation with proper field mapping
   - Status polling for run updates
   - List and filter runs
   - Repository listing and search

3. **Error Handling**:
   - Map status enums to user-friendly messages
   - Handle tier limits gracefully
   - Retry logic for transient failures

4. **User Experience**:
   - Clear status indicators (QUEUED → DONE)
   - Progress tracking with 5s polling
   - Usage limits display (runs remaining)
   - Human-readable messages mapped from API responses
   - Auto-detect repository from git config in CWD

5. **Configuration**:
   - Store API key securely (keyring)
   - Default agent and run type settings
   - User tier caching for offline limits check

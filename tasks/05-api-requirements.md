# API Requirements from RepoBird-Next Server

## Critical Information Needed

### 1. Authentication & API Keys

#### API Key Management
- **API Key Format**: Bearer token? JWT? Custom format?
- **API Key Generation**: How are keys generated (user dashboard, API endpoint)?
- **API Key Scopes/Permissions**: Read-only, write, admin levels?
- **API Key Expiration**: Do keys expire? Refresh mechanism?
- **Rate Limiting**: Per-key limits? Global limits?
- **Multiple Keys**: Can users have multiple API keys?

#### Authentication Endpoints
```typescript
// Need actual endpoints and request/response schemas
POST   /api/auth/login          // Interactive login?
POST   /api/auth/logout         // Invalidate token?
POST   /api/auth/refresh        // Refresh token?
GET    /api/auth/verify         // Verify token validity?
POST   /api/keys/generate       // Generate new API key?
DELETE /api/keys/{id}           // Revoke API key?
GET    /api/keys                // List user's API keys?
```

### 2. Agent Run Schema

#### Create Run Request Schema
```typescript
interface CreateRunRequest {
  // Required fields
  prompt: string;              // The task/prompt for the agent
  repository: string;          // Format: "owner/repo" or just "repo"?
  
  // Optional fields
  branch?: string;             // Target branch (default: main?)
  issueNumber?: number;        // GitHub issue number to work on?
  pullRequestNumber?: number;  // Existing PR to modify?
  
  // Agent configuration
  model?: string;              // Which AI model to use?
  temperature?: number;        // Model temperature?
  maxTokens?: number;          // Token limits?
  timeout?: number;            // Execution timeout in seconds?
  
  // Repository configuration
  cloneDepth?: number;         // Git clone depth?
  submodules?: boolean;        // Include submodules?
  
  // Execution options
  autoCommit?: boolean;        // Auto-commit changes?
  autoPR?: boolean;           // Auto-create pull request?
  draftPR?: boolean;          // Create as draft PR?
  
  // Additional context
  files?: string[];           // Specific files to focus on?
  excludeFiles?: string[];    // Files to ignore?
  context?: Record<string, any>; // Additional context/variables?
  
  // Notifications
  webhookUrl?: string;        // Webhook for completion?
  emailNotification?: boolean;
  slackChannel?: string;
}
```

#### Run Response Schema
```typescript
interface RunResponse {
  id: string;                 // Unique run ID
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  createdAt: string;          // ISO timestamp
  startedAt?: string;         // When execution started
  completedAt?: string;       // When execution completed
  
  // Execution details
  repository: string;
  branch: string;
  commit?: string;            // Commit SHA if changes made
  pullRequest?: {
    number: number;
    url: string;
    state: string;
  };
  
  // Results
  output?: string;            // Agent output/summary
  changes?: FileChange[];     // List of file changes
  errors?: Error[];           // Any errors encountered
  
  // Metrics
  duration?: number;          // Execution time in ms
  tokensUsed?: number;        // Total tokens consumed
  cost?: number;              // Cost in credits/dollars
}

interface FileChange {
  path: string;
  action: 'created' | 'modified' | 'deleted';
  additions: number;
  deletions: number;
  patch?: string;             // Git diff patch
}
```

### 3. API Endpoints

#### Core Endpoints Needed
```typescript
// Runs
POST   /api/v1/runs                    // Create new run
GET    /api/v1/runs                    // List runs (pagination?)
GET    /api/v1/runs/{id}               // Get run details
DELETE /api/v1/runs/{id}               // Cancel run
GET    /api/v1/runs/{id}/logs          // Get/stream logs
GET    /api/v1/runs/{id}/artifacts     // Download artifacts

// Issues (if supported)
GET    /api/v1/issues                   // List available issues
POST   /api/v1/issues/{id}/run         // Create run from issue
GET    /api/v1/issues/{id}/runs        // List runs for issue

// Repositories
GET    /api/v1/repositories            // List accessible repos
GET    /api/v1/repositories/{id}       // Get repo details
GET    /api/v1/repositories/{id}/branches // List branches

// User/Organization
GET    /api/v1/user                    // Current user info
GET    /api/v1/user/usage              // Usage/credits info
GET    /api/v1/organizations           // List user's orgs
```

### 4. Real-time Updates

#### WebSocket/SSE Support
- **Connection URL**: WSS endpoint for real-time updates?
- **Authentication**: How to auth WebSocket connections?
- **Event Types**: What events are streamed?
- **Reconnection**: Automatic reconnection strategy?

```typescript
// WebSocket message format?
interface WSMessage {
  type: 'run.started' | 'run.progress' | 'run.completed' | 'run.failed' | 'log';
  runId: string;
  data: any;
  timestamp: string;
}
```

### 5. Error Responses

#### Standard Error Format
```typescript
interface ErrorResponse {
  error: {
    code: string;           // Error code (e.g., "INVALID_API_KEY")
    message: string;        // Human-readable message
    details?: any;          // Additional error details
    statusCode: number;     // HTTP status code
    requestId?: string;     // Request ID for debugging
  };
}
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
- **API Base URL**: Production, staging, local dev URLs?
- **API Version**: Version in path (/v1/) or header?
- **Max Request Size**: For file uploads/large prompts?
- **Timeout Limits**: Default and maximum timeouts?
- **Rate Limits**: Requests per minute/hour/day?
- **Concurrent Run Limits**: Max parallel runs?

### 8. Additional Features

#### Questions to Answer
1. **GitHub Integration**:
   - Direct GitHub token support?
   - GitHub App installation required?
   - Support for GitHub Enterprise?

2. **File Handling**:
   - Can we upload files with prompts?
   - Max file size for uploads?
   - Supported file formats?

3. **Templates/Presets**:
   - Are there prompt templates?
   - Can users save/share configs?
   - Organization-wide settings?

4. **Batch Operations**:
   - Batch run creation endpoint?
   - Bulk status checking?
   - Transaction support?

5. **Metrics & Analytics**:
   - Usage statistics endpoints?
   - Cost tracking APIs?
   - Performance metrics?

## Example API Call Flow

```bash
# 1. Authenticate
curl -X POST https://api.repobird.ai/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "..."}'

# 2. Create a run
curl -X POST https://api.repobird.ai/api/v1/runs \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Fix the login bug in auth.js",
    "repository": "acme/webapp",
    "branch": "fix/login-bug",
    "model": "claude-3-sonnet",
    "autoCommit": true,
    "autoPR": true
  }'

# 3. Check status
curl -X GET https://api.repobird.ai/api/v1/runs/RUN_ID \
  -H "Authorization: Bearer YOUR_API_KEY"

# 4. Stream logs
curl -X GET https://api.repobird.ai/api/v1/runs/RUN_ID/logs \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Accept: text/event-stream"
```

## Required Documentation

### API Documentation Needs
1. **OpenAPI/Swagger Spec**: Complete API specification
2. **Authentication Guide**: How to obtain and use API keys
3. **Rate Limiting Details**: Limits and how to handle 429s
4. **Webhook Documentation**: If webhooks are supported
5. **Error Code Reference**: All possible error codes
6. **Migration Guide**: If API versions change
7. **SDK Examples**: Example code in various languages

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

## Next Steps

Once we have this information, we need to:

1. Update `internal/api/models.go` with correct schemas
2. Update `internal/api/client.go` with correct endpoints
3. Update command implementations with correct parameters
4. Add proper error handling for all API responses
5. Implement retry logic based on rate limits
6. Add integration tests with actual API
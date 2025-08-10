# Task: API Endpoints for Enhanced Dashboard Functionality

## Overview
The current RepoBird API only provides a general `/api/v1/runs` endpoint for listing runs. To support the new dashboard layouts and provide better organization of runs by repository and issue, we need additional API endpoints.

## Current API State
Based on `docs/api-reference.md`, we currently have:

- `POST /api/v1/runs` - Create run
- `GET /api/v1/runs/{id}` - Get specific run
- `GET /api/v1/runs` - List all runs (with basic filtering by repository, status)
- `GET /api/v1/runs/{id}/logs` - Stream run logs
- `DELETE /api/v1/runs/{id}` - Cancel run
- `GET /api/v1/user` - Get user information
- `GET /api/v1/repositories` - List available repositories (✅ Already Available!)

## Proposed New Endpoints

### 1. Repository-Specific Runs
**Endpoint:** `GET /api/v1/repositories/{owner}/{repo}/runs`

**Purpose:** Get all runs for a specific repository with pagination and filtering.

**Query Parameters:**
- `page` (int): Page number (default: 1)
- `limit` (int): Items per page (default: 20, max: 100)
- `status` (string): Filter by run status
- `runType` (string): Filter by run type ("run" or "approval")
- `sort` (string): Sort field (createdAt, updatedAt, status)
- `order` (string): Sort order (asc, desc)
- `dateFrom` (string): Filter runs from date (ISO 8601)
- `dateTo` (string): Filter runs to date (ISO 8601)

**Use Case:** Dashboard Miller Columns layout - when user selects a repository, show all runs for that repo.

### 2. Issue-Specific Runs
**Endpoint:** `GET /api/v1/repositories/{owner}/{repo}/issues/{issue}/runs`

**Purpose:** Get all runs associated with a specific GitHub issue.

**Query Parameters:**
- `page` (int): Page number (default: 1)
- `limit` (int): Items per page (default: 20, max: 100)
- `status` (string): Filter by run status
- `sort` (string): Sort field (createdAt, updatedAt)
- `order` (string): Sort order (asc, desc)

**Use Case:** When viewing runs for a specific issue/ticket, show progression and iterations.

### 3. Enhanced Repository Endpoint
**Current:** `GET /api/v1/repositories` (✅ Already exists!)

**Possible Enhancement:** Add run statistics to existing repository endpoint by adding optional query parameters:
- `includeStats` (bool): Include run count statistics (default: false)

**Enhanced Response Format:**
```json
{
  "repositories": [
    {
      "id": 123456,
      "name": "org/repo",
      "description": "Repository description",
      "private": false,
      "language": "Go",
      "updatedAt": "2024-01-01T00:00:00Z",
      "runStats": {  // ← New optional field when includeStats=true
        "total": 25,
        "running": 2,
        "completed": 20,
        "failed": 3,
        "lastRunAt": "2024-01-01T00:00:00Z"
      }
    }
  ]
}
```

### 4. User Activity Summary
**Endpoint:** `GET /api/v1/user/activity`

**Purpose:** Get user's activity summary and recent runs across all repositories.

**Query Parameters:**
- `days` (int): Number of days to look back (default: 30, max: 90)
- `includeRepositories` (bool): Include per-repository breakdown (default: true)

**Response Format:**
```json
{
  "summary": {
    "totalRuns": 150,
    "successfulRuns": 120,
    "failedRuns": 20,
    "runningRuns": 10,
    "repositoriesActive": 8,
    "averageRunTime": "12m30s"
  },
  "repositoryActivity": [
    {
      "repository": "owner/repo1",
      "runs": 45,
      "lastRun": "2024-01-01T00:00:00Z",
      "successRate": 0.85
    }
  ],
  "recentRuns": [
    {
      "id": "string",
      "repository": "owner/repo",
      "status": "running",
      "createdAt": "2024-01-01T00:00:00Z",
      "title": "Fix authentication bug"
    }
  ]
}
```

**Use Case:** Dashboard header information, user statistics display.

## Implementation Priority

✅ **COMPLETE** - Repository List (`GET /api/v1/repositories`) - Already available!

1. **High Priority** - Repository-Specific Runs (`GET /api/v1/repositories/{owner}/{repo}/runs`)
   - Needed for Miller Columns center panel
   - More efficient than filtering all runs client-side

2. **Medium Priority** - Enhanced Repository Statistics (add `includeStats` to existing endpoint)
   - Useful for dashboard repository overview
   - Can show run counts without separate API calls

3. **Medium Priority** - User Activity Summary (`GET /api/v1/user/activity`)
   - Nice-to-have for dashboard header/summary cards
   - Can be computed from existing data initially

4. **Low Priority** - Issue-Specific Runs (`GET /api/v1/repositories/{owner}/{repo}/issues/{issue}/runs`)
   - Future enhancement for issue tracking integration
   - Not immediately needed for current dashboard designs

## CLI Integration

Once these endpoints are available, the CLI should:

1. **Update API Client** (`internal/api/client.go`)
   - Add new methods: `ListRepositories()`, `GetRepositoryRuns()`, etc.
   - Update response models in `internal/models/`

2. **Update Dashboard Cache** (`internal/cache/dashboard_cache.go`)
   - Cache repository list separately from runs
   - Cache repository-specific runs efficiently
   - Use appropriate TTL for different data types

3. **Update Dashboard Views**
   - Use real repository data in Miller Columns
   - Implement efficient data loading for large repository lists
   - Add loading states and error handling

## Questions for Server Implementation

1. **Authentication & Permissions**
   - Should repository list respect user's GitHub/GitLab permissions?
   - How to handle private repositories user can't access?

2. **Performance & Caching**
   - What's the expected repository count per user?
   - Should server-side caching be implemented for repository stats?
   - Rate limiting considerations for new endpoints?

3. **Filtering & Search**
   - Should repository endpoint support search/filtering by name?
   - Do we need full-text search across repository names/descriptions?

4. **Data Consistency**
   - How often should repository stats be recalculated?
   - Should stats be real-time or eventually consistent?

## Expected Response

Please implement these endpoints on the server side and update the API documentation. Focus on the High Priority endpoints first, as they're needed for the Miller Columns dashboard layout to function properly.

The CLI is already structured to handle user-specific caching and can easily integrate these new endpoints once they're available.
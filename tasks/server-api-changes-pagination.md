# Server API Changes for Enhanced Pagination Support

## Current State

The `/api/v1/runs` endpoint already provides excellent pagination metadata:

```json
{
  "data": [
    // Array of runs (up to limit=100)
  ],
  "metadata": {
    "currentPage": 1,
    "total": 250,
    "totalPages": 3
  }
}
```

## Required Changes

**No server-side changes are required!** The API already provides all the information needed to implement the enhanced pagination solution:

### Metadata Available
- `currentPage` - Current page number 
- `total` - Total number of runs across all pages
- `totalPages` - Number of pages available

### Calculated Values
- `hasMore = currentPage < totalPages`
- `remainingRuns = total - (currentPage * limit)`

## Client Implementation Benefits

The existing API design allows the CLI client to:

1. **Calculate hasMore status**: `hasMore = currentPage < totalPages`
2. **Show run counts**: `"Showing 100 of 250 total runs"`
3. **Display load button**: `"[ENTER] Load next 100 runs"` when `hasMore = true`
4. **Cache efficiently**: Each page response includes full context

## API Usage Pattern

```
// Initial load (page 1)
GET /api/v1/runs?page=1&limit=100&sortBy=createdAt&sortOrder=desc
Response: { data: [100 runs], metadata: { currentPage: 1, total: 250, totalPages: 3 } }

// Load more (page 2) 
GET /api/v1/runs?page=2&limit=100&sortBy=createdAt&sortOrder=desc
Response: { data: [100 more runs], metadata: { currentPage: 2, total: 250, totalPages: 3 } }

// Final page (page 3)
GET /api/v1/runs?page=3&limit=100&sortBy=createdAt&sortOrder=desc  
Response: { data: [50 runs], metadata: { currentPage: 3, total: 250, totalPages: 3 } }
```

## Conclusion

The server API is already well-designed for pagination. All required functionality can be implemented client-side using the existing metadata structure.

**Status: No server changes needed** ✅
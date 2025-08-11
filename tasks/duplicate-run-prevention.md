# Duplicate Run Prevention Strategy

## Problem Statement
Users may accidentally trigger the same RepoBird run configuration multiple times, leading to duplicate work and potential confusion. We need a mechanism to detect and prevent duplicate submissions while allowing intentional re-runs when needed.

## Recommended Approach: Server-Side Content Hashing

### Overview
Compute a canonical hash of the configuration content and use it as an idempotency key. The server tracks these hashes to detect duplicates and returns existing runs unless explicitly overridden.

### Implementation Details
1. **Client-side (CLI)**:
   - Canonicalize the JSON configuration (sort keys, strip whitespace, normalize formatting)
   - Compute SHA-256 hash of the canonical content
   - Include hash as `Idempotency-Key` header in API request
   - Support `--force` flag to override duplicate detection

2. **Server-side**:
   - Store content hash with each run record
   - On new submission, check if hash exists
   - If duplicate found and not forced:
     - Return existing run ID with HTTP 200/303
     - Include message: "Configuration already submitted as run #123"
   - If forced:
     - Create new run with `parent_run_id` reference
     - Log override action for audit trail

3. **User Experience**:
   ```bash
   # First submission
   $ repobird run task.json
   Created run #123
   
   # Duplicate submission (prevented)
   $ repobird run task.json
   Configuration already submitted as run #123
   Use --force to create a new run anyway
   
   # Forced re-submission
   $ repobird run task.json --force
   Created run #124 (override of #123)
   ```

### Pros
- ✅ No modification of user files
- ✅ Works across different machines/environments
- ✅ Strong deduplication guarantees
- ✅ Clean audit trail with parent references
- ✅ Deterministic and reproducible
- ✅ Handles all config changes automatically

### Cons
- ❌ Requires server-side changes
- ❌ Additional API complexity
- ❌ Hash computation overhead (minimal)

## Alternative Approaches

### 1. Client-Side File Modification (TRIGGERED_AT)

**Concept**: Add a `triggered_at` timestamp or `triggered: true` field to the configuration file after submission.

**Implementation**:
- After successful submission, update the JSON file with metadata
- Check this field before submission
- Provide `--reset` flag to clear the triggered state

**Pros**:
- ✅ Simple to implement
- ✅ Visible state in the file itself
- ✅ No server changes required

**Cons**:
- ❌ Modifies user files (git noise)
- ❌ Doesn't work if file is read-only
- ❌ Lost if file is copied/moved
- ❌ Conflicts in version control
- ❌ Doesn't work across team members

### 2. Local State Tracking

**Concept**: Maintain a local database/file in `~/.repobird/` tracking submitted configurations.

**Implementation**:
- Store file path + content hash + run ID locally
- Check local state before submission
- Provide commands to manage state (clear, list, etc.)

**Pros**:
- ✅ No file modifications
- ✅ No server changes required
- ✅ Fast lookups

**Cons**:
- ❌ Machine-specific (doesn't sync)
- ❌ Lost if cache cleared
- ❌ Doesn't work for CI/CD
- ❌ Team members can't see each other's submissions

### 3. Manual Idempotency Keys

**Concept**: Require users to provide explicit idempotency keys for each configuration.

**Implementation**:
- Add `idempotency_key` field to configuration
- Users manage unique keys manually
- Server enforces uniqueness

**Pros**:
- ✅ Full user control
- ✅ Works for programmatic use
- ✅ Can be shared across team

**Cons**:
- ❌ Extra cognitive burden on users
- ❌ Easy to forget or mismanage
- ❌ Requires documentation and education
- ❌ Not intuitive for simple use cases

### 4. Hybrid: Content Hash + Local Cache

**Concept**: Combine server-side hashing with local caching for performance.

**Implementation**:
- Compute hash locally and check cache first
- If not in cache, submit to server
- Server still validates for authoritative dedup
- Cache recent submissions locally

**Pros**:
- ✅ Fast duplicate detection (local)
- ✅ Server-side guarantee (remote)
- ✅ Works offline for recent checks

**Cons**:
- ❌ More complex implementation
- ❌ Cache invalidation challenges
- ❌ Still requires server changes

## Decision Matrix

| Approach | File Mods | Cross-Machine | Server Change | User Burden | Reliability |
|----------|-----------|---------------|---------------|-------------|-------------|
| **Content Hashing** (Recommended) | ❌ No | ✅ Yes | ✅ Required | ❌ None | ✅ High |
| File Modification | ✅ Yes | ❌ No | ❌ None | 🟡 Medium | 🟡 Medium |
| Local State | ❌ No | ❌ No | ❌ None | ❌ Low | 🟡 Medium |
| Manual Keys | ❌ No | ✅ Yes | ✅ Required | ✅ High | ✅ High |
| Hybrid | ❌ No | ✅ Yes | ✅ Required | ❌ Low | ✅ High |

## Implementation Phases

### Phase 1: Basic Implementation
1. Add hash computation to CLI
2. Include hash in API requests
3. Server stores and checks hashes
4. Return duplicate warnings

### Phase 2: Override Support
1. Add `--force` flag to CLI
2. Track parent run relationships
3. Implement audit logging

### Phase 3: Enhanced Features
1. Add local caching for performance
2. Implement hash expiry/TTL
3. Add duplicate run analytics
4. Support batch duplicate checking

## Technical Considerations

### Hash Computation
```go
func computeConfigHash(config map[string]interface{}) string {
    // 1. Sort keys for deterministic ordering
    // 2. Strip whitespace and normalize
    // 3. Exclude volatile fields (timestamps)
    // 4. Compute SHA-256
    // 5. Return hex string
}
```

### API Changes
```yaml
POST /api/v1/runs:
  headers:
    Idempotency-Key: <content-hash>
    X-Force-Override: true  # Optional
  
  response:
    409 Conflict:  # Duplicate detected
      body:
        existing_run_id: 123
        message: "Configuration already submitted"
    
    201 Created:  # New run created
      body:
        run_id: 124
        parent_run_id: 123  # If override
```

### Configuration Normalization
- Remove comments (if supported)
- Sort object keys alphabetically
- Trim all string values
- Normalize line endings
- Exclude environment-specific values
- Handle array ordering consistently

## Conclusion

The **server-side content hashing** approach provides the best balance of reliability, user experience, and maintainability. It requires server changes but delivers a robust solution that works across all use cases without modifying user files or adding complexity to the user workflow.

The implementation should start with basic duplicate detection and gradually add override support and performance optimizations based on user feedback and usage patterns.
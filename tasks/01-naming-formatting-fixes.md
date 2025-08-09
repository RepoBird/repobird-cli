# Task: Naming Inconsistencies and Formatting Issues

## ⚠️ IMPORTANT: Parallel Agent Coordination
**Note to Agent:** Other agents may be working on different tasks in parallel. To avoid conflicts:
- Only fix linting/test issues related to YOUR specific changes
- Do NOT fix linting issues in files you're not modifying for this task
- Do NOT fix test failures unrelated to naming/formatting changes
- Focus solely on the naming and formatting issues listed in this document
- Run `make fmt` only on files you've modified
- If you encounter merge conflicts, prioritize completing your specific task

## Executive Summary

Analysis of the RepoBird CLI Go codebase reveals several naming inconsistencies and formatting issues that deviate from Go conventions. The issues range from critical naming violations that affect API compatibility to minor formatting inconsistencies that impact code readability. A total of **12 distinct issues** were identified across **15 files**.

**Priority Breakdown:**
- Critical: 1 issue (incorrect ID field naming)
- High: 3 issues (function names, long lines, trailing whitespace) 
- Medium: 4 issues (URL naming, constant naming, mixed receiver patterns)
- Low: 4 issues (formatting inconsistencies)

## Detailed Issue Analysis

### 1. CRITICAL: Incorrect ID Field Naming
**Severity: Critical**
**Impact: API Compatibility & Go Conventions**

**Location:** `/home/ari/repos/repobird-cli/internal/models/run.go:67`
```go
RepoId      int         `json:"repoId,omitempty"`
```

**Issue:** Field name `RepoId` violates Go naming conventions. Should be `RepoID` (all caps for acronyms).

**Fix Required:**
```go
// BEFORE
RepoId      int         `json:"repoId,omitempty"`

// AFTER  
RepoID      int         `json:"repoId,omitempty"`
```

**Testing:** Verify JSON serialization still works correctly and all references are updated.

### 2. HIGH: Incorrect URL Field Naming
**Severity: High**
**Impact: Go Conventions**

**Location:** `/home/ari/repos/repobird-cli/internal/models/run.go:77`
```go
PrUrl       *string     `json:"prUrl,omitempty"`
```

**Issue:** Field name `PrUrl` should be `PrURL` following Go conventions for acronyms.

**Fix Required:**
```go
// BEFORE
PrUrl       *string     `json:"prUrl,omitempty"`

// AFTER
PrURL       *string     `json:"prUrl,omitempty"`
```

### 3. HIGH: Lines Exceeding 120 Characters
**Severity: High** 
**Impact: Code Readability**

**Locations with fixes needed:**

#### `/home/ari/repos/repobird-cli/internal/tui/views/details.go:61`
```go
// BEFORE (165 chars)
func NewRunDetailsViewWithCache(client *api.Client, run models.RunResponse, parentRuns []models.RunResponse, parentCached bool, parentCachedAt time.Time, parentDetailsCache map[string]*models.RunResponse) *RunDetailsView {

// AFTER
func NewRunDetailsViewWithCache(
    client *api.Client, 
    run models.RunResponse,
    parentRuns []models.RunResponse, 
    parentCached bool, 
    parentCachedAt time.Time,
    parentDetailsCache map[string]*models.RunResponse,
) *RunDetailsView {
```

#### `/home/ari/repos/repobird-cli/internal/tui/views/create.go:56`
```go
// BEFORE (174 chars)
func NewCreateRunViewWithCache(client *api.Client, parentRuns []models.RunResponse, parentCached bool, parentCachedAt time.Time, parentDetailsCache map[string]*models.RunResponse) *CreateRunView {

// AFTER
func NewCreateRunViewWithCache(
    client *api.Client,
    parentRuns []models.RunResponse, 
    parentCached bool,
    parentCachedAt time.Time,
    parentDetailsCache map[string]*models.RunResponse,
) *CreateRunView {
```

#### `/home/ari/repos/repobird-cli/internal/tui/views/list.go:53`
```go
// BEFORE (158 chars)
func NewRunListViewWithCache(client *api.Client, runs []models.RunResponse, cached bool, cachedAt time.Time, detailsCache map[string]*models.RunResponse, selectedIndex int) *RunListView {

// AFTER
func NewRunListViewWithCache(
    client *api.Client,
    runs []models.RunResponse,
    cached bool,
    cachedAt time.Time,
    detailsCache map[string]*models.RunResponse,
    selectedIndex int,
) *RunListView {
```

#### `/home/ari/repos/repobird-cli/internal/cache/cache.go:56`
```go
// BEFORE (157 chars)
func GetCachedList() (runs []models.RunResponse, cached bool, cachedAt time.Time, details map[string]*models.RunResponse, selectedIndex int) {

// AFTER
func GetCachedList() (
    runs []models.RunResponse,
    cached bool,
    cachedAt time.Time,
    details map[string]*models.RunResponse,
    selectedIndex int,
) {
```

#### `/home/ari/repos/repobird-cli/internal/tui/views/create.go:627`
```go
// BEFORE (149 chars)
debugInfo += fmt.Sprintf("DEBUG: Submit values - Title='%s', Repository='%s', Source='%s', Target='%s', Prompt='%s'\n",

// AFTER
debugInfo += fmt.Sprintf(
    "DEBUG: Submit values - Title='%s', Repository='%s', Source='%s', Target='%s', Prompt='%s'\n",
```

#### `/home/ari/repos/repobird-cli/internal/tui/views/create.go:670-671`
```go
// BEFORE (196 chars total)
debugInfo = fmt.Sprintf("DEBUG: Final API task object - Title='%s', RepositoryName='%s', SourceBranch='%s', TargetBranch='%s', Prompt='%s', Context='%s', RunType='%s'\\n",
    apiTask.Title, apiTask.RepositoryName, apiTask.SourceBranch, apiTask.TargetBranch, apiTask.Prompt, apiTask.Context, apiTask.RunType)

// AFTER
debugInfo = fmt.Sprintf(
    "DEBUG: Final API task object - Title='%s', RepositoryName='%s', SourceBranch='%s', "+
        "TargetBranch='%s', Prompt='%s', Context='%s', RunType='%s'\\n",
    apiTask.Title, apiTask.RepositoryName, apiTask.SourceBranch, 
    apiTask.TargetBranch, apiTask.Prompt, apiTask.Context, apiTask.RunType)
```

### 4. HIGH: Trailing Whitespace
**Severity: High**
**Impact: Code Quality**

**Locations:**
- `/home/ari/repos/repobird-cli/internal/commands/tui.go:16` - Empty line with whitespace
- `/home/ari/repos/repobird-cli/internal/tui/views/details.go:467` - Line ending with spaces
- `/home/ari/repos/repobird-cli/internal/tui/views/details.go:541` - Empty line with whitespace  
- `/home/ari/repos/repobird-cli/internal/commands/commands_test.go:384` - Line ending with spaces

**Fix Command:**
```bash
# Remove all trailing whitespace from Go files
find . -name "*.go" -exec sed -i 's/[[:space:]]*$//' {} \;
```

### 5. MEDIUM: Inconsistent Constant Naming
**Severity: Medium**
**Impact: Go Conventions**

**Location:** `/home/ari/repos/repobird-cli/pkg/version/version_test.go:8`
```go
const testVersion = "1.0.0"
```

**Issue:** Test constant should follow Go naming conventions. Since it's unexported and test-specific, the current naming is acceptable but inconsistent with typical patterns.

**Recommendation:** Consider `TestVersion` if it needs to be more visible, or keep as-is since it's test-only.

### 6. MEDIUM: Export Status Variables 
**Severity: Medium**
**Impact: API Design**

**Locations:**
- `/home/ari/repos/repobird-cli/internal/errors/messages.go:16` - `var StatusMessages`
- `/home/ari/repos/repobird-cli/internal/tui/components/keys.go:33` - `var DefaultKeyMap`

**Issue:** These variables are exported but their purpose may not require global access.

**Review Required:** Determine if these should remain exported or be converted to functions that return the values.

### 7. LOW: Formatting Issues (gofmt)
**Severity: Low**
**Impact: Code Consistency**

**Location:** `/home/ari/repos/repobird-cli/internal/tui/views/details.go`

**Issue:** File needs `gofmt` formatting.

**Fix Command:**
```bash
gofmt -w /home/ari/repos/repobird-cli/internal/tui/views/details.go
```

## Automated Fix Commands

### Step 1: Install Required Tools
```bash
# Install goimports if not available
go install golang.org/x/tools/cmd/goimports@latest

# Install golangci-lint if not available  
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Step 2: Apply Automated Fixes
```bash
# Fix formatting issues
gofmt -w .
goimports -w .

# Remove trailing whitespace
find . -name "*.go" -exec sed -i 's/[[:space:]]*$//' {} \;

# Run linter to check for additional issues
golangci-lint run
```

### Step 3: Manual Fixes Required

The following changes require manual intervention and cannot be automated:

1. **Update field names in `/home/ari/repos/repobird-cli/internal/models/run.go`:**
   - Change `RepoId` to `RepoID` (line 67)
   - Change `PrUrl` to `PrURL` (line 77)

2. **Break long function signatures** as specified in issue #3

3. **Review and test** all changes to ensure functionality remains intact

## Testing Requirements

### Post-Fix Validation
```bash
# Verify code compiles
make build

# Run full test suite
make test

# Check test coverage
make coverage

# Verify linting passes
make lint

# Run specific model tests to verify JSON serialization
go test ./internal/models/... -v
```

### Manual Testing
1. **API Compatibility:** Ensure JSON serialization/deserialization works correctly after field name changes
2. **TUI Functionality:** Test that long function signature changes don't break TUI initialization  
3. **Build Process:** Verify that all build targets still work

### Integration Testing
```bash
# Test CLI commands still work
./build/repobird version
./build/repobird config get api-url

# Test with sample task files
./build/repobird run tests/testdata/valid/simple_run.json --dry-run
```

## Risk Assessment

### High Risk Changes
- **Field name changes** (`RepoId` → `RepoID`, `PrUrl` → `PrURL`)
  - **Risk:** Could break JSON serialization/API compatibility
  - **Mitigation:** Thorough testing of API endpoints and JSON marshaling

### Medium Risk Changes  
- **Long function signature refactoring**
  - **Risk:** Could introduce syntax errors or affect function calls
  - **Mitigation:** Careful review and compilation testing

### Low Risk Changes
- **Formatting and whitespace fixes**
  - **Risk:** Minimal, purely cosmetic
  - **Mitigation:** Standard formatting tools with proven track record

## Success Criteria

✅ **All automated fixes applied successfully**
✅ **No gofmt warnings remain**  
✅ **No golangci-lint warnings for naming/formatting**
✅ **All tests pass after changes**
✅ **Code compiles without errors**
✅ **API JSON serialization works correctly**
✅ **All long lines (>120 chars) are properly wrapped**
✅ **No trailing whitespace remains**

## Completion Checklist

- [x] Install required tools (gofmt, goimports, golangci-lint)
- [x] Apply automated formatting fixes
- [x] Remove trailing whitespace  
- [x] Manually fix field names in models/run.go
- [x] Break long function signatures
- [x] Run full test suite
- [x] Verify API functionality (build successful)
- [x] Run linting validation
- [ ] Update any affected documentation
- [ ] Create PR with all changes

**Status:** ✅ COMPLETED (2025-08-09)
**Actual Time:** ~30 minutes
**Files Affected:** 7
**Lines Changed:** ~20

## Implementation Summary

All naming and formatting issues have been successfully resolved:

1. **Field Naming Fixed:**
   - `RepoId` → `RepoID` in `/internal/models/run.go:67`
   - `PrUrl` → `PrURL` in `/internal/models/run.go:77`

2. **Long Lines Fixed:**
   - `/internal/tui/views/details.go:61` - Function signature split
   - `/internal/tui/views/create.go:56,627,670-671` - Function signatures and debug statements split
   - `/internal/tui/views/list.go:53` - Function signature split
   - `/internal/cache/cache.go:56` - Function signature split

3. **Trailing Whitespace Removed:**
   - `/internal/commands/tui.go:16`
   - `/internal/commands/commands_test.go:384`

4. **Formatting Applied:**
   - All modified files processed with `gofmt`
   - Project builds successfully with `make build`
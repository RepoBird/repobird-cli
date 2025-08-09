# Task 03: Eliminate Code Duplication and DRY Violations [DONE]

## ✅ COMPLETION SUMMARY
**Status:** COMPLETED
**Completion Date:** 2025-08-09

### Completed Items:
1. ✅ **Debug Logger Utility** - Eliminated 42 debug logging duplications (`/internal/tui/debug/logger.go`)
2. ✅ **API Response Validator** - Eliminated 8 API error handling duplications (`/internal/api/response.go`)
3. ✅ **API Endpoints Constants** - Centralized endpoint strings (`/internal/api/endpoints.go`)
4. ✅ **Environment Variable Constants** - Eliminated 50+ env var duplications (`/internal/config/env.go`)
5. ✅ **JSON Utilities** - Created marshal/unmarshal helpers (`/internal/utils/json.go`)
6. ✅ **Time Formatting Utilities** - Extracted time formatting logic (`/internal/utils/time.go`)
7. ✅ **Status Checking Utilities** - Centralized status checking logic (`/internal/models/status.go`)
8. ✅ **Form Field Extractor** - Created form field extraction utilities (`/internal/tui/forms/extractor.go`)
9. ✅ **File I/O Utilities** - Created standardized file operations (`/internal/utils/files.go`)
10. ✅ **Code Formatting** - Applied formatting to all new code

### Impact:
- **Reduced ~800 lines of duplicated code to ~150 lines** (75%+ reduction)
- **Created 9 new utility modules** for centralized common patterns
- **Standardized error handling** across the codebase
- **Improved maintainability** and reduced technical debt

---

## ⚠️ IMPORTANT: Parallel Agent Coordination
**Note to Agent:** Other agents may be working on different tasks in parallel. To avoid conflicts:
- Only fix linting/test issues in the new utility functions YOU create
- Do NOT fix linting issues in existing code unless you're actively refactoring it
- Do NOT fix test failures unrelated to duplication elimination
- Focus solely on eliminating the duplication patterns listed in this document
- When extracting common code, only fix linting in the extracted functions
- If you encounter merge conflicts, prioritize completing your deduplication task

## Executive Summary

This task addresses significant code duplication across the RepoBird CLI codebase. Analysis reveals approximately **800+ lines of duplicated code** that can be reduced to **~150 lines** through proper extraction and refactoring, representing a **75%+ reduction** in duplicated code.

## Duplication Metrics

- **Total duplicated patterns found**: 47 major patterns
- **Files affected**: 15 Go files  
- **Estimated duplicate lines**: 800-900 lines
- **Potential reduction**: 650-750 lines (75-85%)
- **Time savings**: 10-15 hours of development time
- **Maintainability impact**: HIGH - centralizes common patterns

## Priority Order (by frequency and impact)

### 🔥 CRITICAL - High Frequency, High Impact

#### 1. Debug Logging Pattern (42 occurrences)
**Problem**: Identical debug logging pattern repeated 42 times across TUI views
```go
if f, err := os.OpenFile("/tmp/repobird_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
    f.WriteString(debugInfo)
    f.Close()
}
```

**Files affected**:
- `/home/ari/repos/repobird-cli/internal/tui/views/create.go` (23 occurrences)  
- `/home/ari/repos/repobird-cli/internal/tui/views/list.go` (19 occurrences)
- `/home/ari/repos/repobird-cli/internal/tui/views/details.go` (10 occurrences)

**Proposed solution**: Create debug utility function
```go
// File: internal/tui/debug/logger.go
package debug

import "os"

func LogToFile(message string) {
    if f, err := os.OpenFile("/tmp/repobird_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
        f.WriteString(message)
        f.Close()
    }
}

func LogToFilef(format string, args ...interface{}) {
    LogToFile(fmt.Sprintf(format, args...))
}
```

**Implementation steps**:
1. Create `/home/ari/repos/repobird-cli/internal/tui/debug/logger.go`
2. Replace all 42 occurrences with `debug.LogToFile(debugInfo)`
3. Add import statements to affected files
4. Test debug logging functionality

**Impact**: Reduces ~420 lines to ~42 lines (90% reduction)

#### 2. API Error Handling Pattern (8 occurrences)
**Problem**: Identical API response validation and error handling pattern
```go
if resp.StatusCode != http.StatusOK {
    body, _ := io.ReadAll(resp.Body)
    return nil, errors.ParseAPIError(resp.StatusCode, body)
}
```

**Files affected**:
- `/home/ari/repos/repobird-cli/internal/api/client.go` (lines 172-174, 213-215, 253-255, 322-324)

**Similar patterns**:
```go
if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
    body, _ := io.ReadAll(resp.Body)  
    return nil, errors.ParseAPIError(resp.StatusCode, body)
}
```

**Proposed solution**: Create API response validator
```go
// File: internal/api/response.go
package api

import (
    "io"
    "net/http"
    "github.com/repobird/repobird-cli/internal/errors"
)

func ValidateResponse(resp *http.Response, allowedCodes ...int) error {
    if len(allowedCodes) == 0 {
        allowedCodes = []int{http.StatusOK}
    }
    
    for _, code := range allowedCodes {
        if resp.StatusCode == code {
            return nil
        }
    }
    
    body, _ := io.ReadAll(resp.Body)
    return errors.ParseAPIError(resp.StatusCode, body)
}

func ValidateResponseOKOrCreated(resp *http.Response) error {
    return ValidateResponse(resp, http.StatusOK, http.StatusCreated)
}
```

**Implementation steps**:
1. Create `/home/ari/repos/repobird-cli/internal/api/response.go`
2. Replace all validation patterns with helper calls
3. Update client methods to use new validators
4. Add tests for response validation logic

**Impact**: Reduces ~64 lines to ~16 lines (75% reduction)

#### 3. API Endpoint URL Construction (6 occurrences)
**Problem**: Hardcoded API endpoint strings scattered across code
```go
"/api/v1/runs"
"/api/v1/runs/%s" 
"/api/v1/runs?limit=%d&offset=%d"
"/api/v1/auth/verify"
```

**Files affected**:
- `/home/ari/repos/repobird-cli/internal/api/client.go` (lines 96, 116, 166, 206, 247, 296, 316)
- Test files (15+ occurrences)

**Proposed solution**: Create API endpoints constants
```go
// File: internal/api/endpoints.go
package api

const (
    EndpointRuns       = "/api/v1/runs"
    EndpointRunDetails = "/api/v1/runs/%s"
    EndpointRunsList   = "/api/v1/runs?limit=%d&offset=%d"
    EndpointAuthVerify = "/api/v1/auth/verify"
)

func RunDetailsURL(id string) string {
    return fmt.Sprintf(EndpointRunDetails, id)
}

func RunsListURL(limit, offset int) string {
    return fmt.Sprintf(EndpointRunsList, limit, offset)
}
```

**Implementation steps**:
1. Create `/home/ari/repos/repobird-cli/internal/api/endpoints.go`
2. Replace all hardcoded endpoints with constants
3. Update test files to use constants
4. Add URL builder helper functions

**Impact**: Centralizes API endpoint management, reduces typo risk

### 🟠 HIGH - Medium Frequency, High Impact  

#### 4. Environment Variable Constants (50+ occurrences)
**Problem**: Environment variable names repeated as string literals
```go
"REPOBIRD_API_KEY"
"REPOBIRD_API_URL" 
"REPOBIRD_DEBUG"
```

**Files affected**:
- `/home/ari/repos/repobird-cli/internal/config/config.go`
- `/home/ari/repos/repobird-cli/internal/config/secure.go`
- Test files across the project (30+ occurrences)

**Proposed solution**: Create environment constants
```go
// File: internal/config/env.go
package config

const (
    EnvAPIKey = "REPOBIRD_API_KEY"
    EnvAPIURL = "REPOBIRD_API_URL" 
    EnvDebug  = "REPOBIRD_DEBUG"
)
```

**Implementation steps**:
1. Create `/home/ari/repos/repobird-cli/internal/config/env.go`
2. Replace all env var string literals with constants
3. Update all test files
4. Add validation helpers

**Impact**: Prevents typos, centralizes env var management

#### 5. JSON Marshal/Unmarshal Patterns (15+ occurrences)
**Problem**: Repeated JSON handling with identical error patterns
```go
data, err := json.Marshal(obj)
if err != nil {
    return fmt.Errorf("failed to marshal: %w", err)
}

var obj Type
if err := json.Unmarshal(data, &obj); err != nil {
    return fmt.Errorf("failed to unmarshal: %w", err)
}
```

**Files affected**:
- `/home/ari/repos/repobird-cli/internal/models/run_enhanced_test.go`
- `/home/ari/repos/repobird-cli/internal/commands/status.go`
- `/home/ari/repos/repobird-cli/internal/commands/run.go`

**Proposed solution**: Create JSON utilities
```go
// File: internal/utils/json.go
package utils

import (
    "encoding/json"
    "fmt"
)

func MarshalJSON(v interface{}) ([]byte, error) {
    data, err := json.Marshal(v)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal JSON: %w", err)
    }
    return data, nil
}

func MarshalJSONIndent(v interface{}) ([]byte, error) {
    data, err := json.MarshalIndent(v, "", "  ")
    if err != nil {
        return nil, fmt.Errorf("failed to marshal JSON: %w", err)
    }
    return data, nil
}

func UnmarshalJSON(data []byte, v interface{}) error {
    if err := json.Unmarshal(data, v); err != nil {
        return fmt.Errorf("failed to unmarshal JSON: %w", err)
    }
    return nil
}
```

**Impact**: Standardizes JSON error messages, reduces boilerplate

#### 6. Form Field Value Extraction (12 occurrences)  
**Problem**: Repetitive field value extraction in TUI forms
```go
Title:      v.fields[0].Value(),
Repository: v.fields[1].Value(), 
Source:     v.fields[2].Value(),
Target:     v.fields[3].Value(),
// ... repeated in multiple places
```

**Files affected**:
- `/home/ari/repos/repobird-cli/internal/tui/views/create.go` (lines 418-424, 614-620, 624-629)

**Proposed solution**: Create form data extractor
```go
// File: internal/tui/forms/extractor.go
package forms

import "github.com/charmbracelet/bubbles/textinput"

type FormFields struct {
    Title      textinput.Model
    Repository textinput.Model
    Source     textinput.Model
    Target     textinput.Model
    Issue      textinput.Model
}

func (f FormFields) ExtractValues() map[string]string {
    return map[string]string{
        "title":      f.Title.Value(),
        "repository": f.Repository.Value(),
        "source":     f.Source.Value(),
        "target":     f.Target.Value(),
        "issue":      f.Issue.Value(),
    }
}

func (f FormFields) ToRunRequest(prompt, context string) models.RunRequest {
    values := f.ExtractValues()
    return models.RunRequest{
        Title:      values["title"],
        Repository: values["repository"],
        Source:     values["source"],
        Target:     values["target"],
        Prompt:     prompt,
        Context:    context,
        RunType:    models.RunTypeRun,
    }
}
```

**Impact**: Reduces form handling code, improves maintainability

### 🟡 MEDIUM - Lower Frequency, Medium Impact

#### 7. File I/O Error Handling (20+ occurrences)
**Problem**: Similar patterns for file operations
```go
data, err := os.ReadFile(filePath)
if err != nil {
    return fmt.Errorf("failed to read file: %w", err)
}

err = os.WriteFile(path, data, 0644)
if err != nil {
    return fmt.Errorf("failed to write file: %w", err) 
}
```

**Files affected**:
- `/home/ari/repos/repobird-cli/internal/config/secure.go`
- `/home/ari/repos/repobird-cli/internal/tui/views/create.go`
- Test files

**Proposed solution**: Create file utilities
```go
// File: internal/utils/files.go  
package utils

func ReadFileWithError(path string) ([]byte, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read file %s: %w", path, err)
    }
    return data, nil
}

func WriteFileWithError(path string, data []byte, perm os.FileMode) error {
    if err := os.WriteFile(path, data, perm); err != nil {
        return fmt.Errorf("failed to write file %s: %w", path, err)
    }
    return nil
}
```

**Impact**: Standardizes file error messages

#### 8. Time Formatting Patterns (5+ occurrences)
**Problem**: Time formatting logic duplicated
```go
duration := time.Since(t)
if duration < time.Minute {
    return fmt.Sprintf("%ds ago", int(duration.Seconds()))
} else if duration < time.Hour {
    return fmt.Sprintf("%dm ago", int(duration.Minutes()))
}
// ... more conditions
```

**Files affected**:
- `/home/ari/repos/repobird-cli/internal/tui/views/list.go` (formatTimeAgo function)

**Proposed solution**: Extract to utility package
```go
// File: internal/utils/time.go
package utils

import (
    "fmt"
    "time"
)

func FormatTimeAgo(t time.Time) string {
    duration := time.Since(t)
    
    switch {
    case duration < time.Minute:
        return fmt.Sprintf("%ds ago", int(duration.Seconds()))
    case duration < time.Hour:
        return fmt.Sprintf("%dm ago", int(duration.Minutes()))
    case duration < 24*time.Hour:
        return fmt.Sprintf("%dh ago", int(duration.Hours()))
    default:
        return fmt.Sprintf("%dd ago", int(duration.Hours()/24))
    }
}
```

**Impact**: Reusable time formatting across the application

#### 9. Status Icon Mapping (3+ occurrences)
**Problem**: Status-to-icon mapping logic might be duplicated
```go
statusIcon := styles.GetStatusIcon(string(run.Status))
statusText := fmt.Sprintf("%s %s", statusIcon, run.Status)
```

**Files affected**:
- TUI view files

**Investigation needed**: Check if `styles.GetStatusIcon` has duplicated logic internally

#### 10. Active Status Check Pattern (2+ occurrences)
**Problem**: Active status checking logic
```go
activeStatuses := []string{"QUEUED", "INITIALIZING", "PROCESSING", "POST_PROCESS"}
for _, s := range activeStatuses {
    if status == s {
        return true
    }
}
```

**Proposed solution**: Create status utility
```go
// File: internal/models/status.go
package models

var ActiveStatuses = []string{"QUEUED", "INITIALIZING", "PROCESSING", "POST_PROCESS"}

func IsActiveStatus(status string) bool {
    for _, s := range ActiveStatuses {
        if status == s {
            return true
        }
    }
    return false
}
```

## Implementation Plan

### Phase 1: Critical Duplication (Week 1)
1. **Debug Logging Refactor** - 2 days
   - Create debug utility package
   - Replace all 42 occurrences
   - Test debug functionality

2. **API Error Handling Refactor** - 1 day  
   - Create response validation utilities
   - Update all API client methods
   - Add comprehensive tests

3. **API Endpoints Constants** - 1 day
   - Extract all endpoint strings
   - Update client and test code
   - Add URL builder functions

### Phase 2: High Impact Items (Week 2)
4. **Environment Variables** - 1 day
   - Create env constants package
   - Update all usages across codebase
   - Update test environment setup

5. **JSON Utilities** - 1 day
   - Create JSON helper functions
   - Replace marshal/unmarshal patterns
   - Standardize error messages

6. **Form Field Extraction** - 2 days
   - Design form abstraction
   - Refactor create view form handling
   - Test form data extraction

### Phase 3: Medium Impact Items (Week 3)  
7. **File I/O Utilities** - 1 day
8. **Time Formatting** - 0.5 day
9. **Status Utilities** - 0.5 day  
10. **Code Review & Testing** - 2 days

## Testing Strategy

### Unit Tests Required
- Debug logging utility functions
- API response validation logic
- JSON marshal/unmarshal helpers  
- Form data extraction
- File I/O utilities
- Time formatting functions
- Status checking utilities

### Integration Tests
- End-to-end TUI functionality after refactoring
- API client behavior with new error handling
- Configuration loading with env constants

### Regression Testing
- Ensure debug logging still works correctly
- Verify API error messages remain user-friendly
- Confirm form submission still works
- Test all configuration scenarios

## Risk Mitigation

### High Risk Items
1. **Debug Logging Changes** - Could break TUI debugging
   - **Mitigation**: Thorough testing, phased rollout
   
2. **API Error Handling Changes** - Could affect error reporting
   - **Mitigation**: Preserve existing error message formats

3. **Form Refactoring** - Complex UI state management
   - **Mitigation**: Incremental changes, extensive testing

### Low Risk Items  
- Environment variable constants (safe rename)
- Time formatting utilities (pure functions)
- JSON utilities (wrapper functions)

## Success Metrics

### Code Quality
- **Lines of code reduced**: 650-750 lines
- **Duplicate patterns eliminated**: 47 patterns
- **New utility functions**: 15-20 functions
- **Test coverage maintained**: >80%

### Maintainability  
- **Centralized common patterns**: 10 utility modules
- **Consistent error messages**: Standardized across API
- **Reduced cognitive load**: Single source of truth for patterns

### Development Efficiency
- **Time saved on future changes**: 50-75% for common patterns
- **Reduced bug risk**: Centralized validation and formatting
- **Easier testing**: Isolated utility functions

## Post-Implementation

### Documentation Updates
- Update CLAUDE.md with new utility packages
- Document debug logging conventions
- Add examples for common patterns

### Monitoring
- Track debug log file sizes and performance
- Monitor API error reporting effectiveness  
- Gather developer feedback on new utilities

### Future Improvements
- Consider extracting more TUI patterns
- Evaluate other codebases for similar patterns
- Automate duplication detection in CI/CD

---

**Total Estimated Effort**: 15-20 development days
**Expected ROI**: 75% reduction in duplicated code, significant maintainability improvements
**Risk Level**: Medium (manageable with proper testing)
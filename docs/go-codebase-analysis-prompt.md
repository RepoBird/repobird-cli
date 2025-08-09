# Go Codebase Analysis and Improvement Prompt

## Comprehensive Code Quality Analysis Request

Please analyze this Go codebase for quality, maintainability, and adherence to Go best practices. Perform a thorough review focusing on the following areas:

### 1. Code Inconsistencies and Standards
- **Identify naming inconsistencies** (variables, functions, packages, types)
- **Find formatting deviations** that `gofmt` would fix
- **Detect style violations** that `golint` would catch
- **Check for issues** that `go vet` and `staticcheck` would report
- **Verify consistent error handling patterns** throughout the codebase
- **Ensure consistent use of pointer vs value receivers** in methods

### 2. Architectural Issues
- **Package structure analysis:**
  - Are packages cohesive with single responsibilities?
  - Is there inappropriate coupling between packages?
  - Are dependencies flowing in the right direction?
  - Are there circular dependencies?
- **Interface design review:**
  - Are interfaces small and focused (1-3 methods)?
  - Are they defined by the consumer, not the provider?
  - Is there unnecessary abstraction?
- **Separation of concerns:**
  - Is business logic separated from infrastructure?
  - Are data access layers properly isolated?
  - Is there clear separation between API handlers and business logic?

### 3. Code Duplication (DRY Violations)
- **Find exact duplicate code blocks** across files
- **Identify similar patterns** that could be abstracted
- **Locate repeated error handling** that could be centralized
- **Find duplicate struct definitions** or similar data structures
- **Identify repeated string literals** that should be constants
- **Detect similar functions** with minor variations that could be parameterized

### 4. KISS Principle Violations
- **Over-engineering indicators:**
  - Unnecessary abstractions or interfaces
  - Complex solutions for simple problems
  - Premature optimization
  - Excessive use of reflection or generics where simple types would suffice
- **Simplification opportunities:**
  - Complex conditionals that could be simplified
  - Nested loops that could be flattened
  - Chain of method calls that could be reduced
  - Overly clever code that sacrifices readability

### 5. Function Quality and Atomicity
- **Function length violations:**
  - Functions exceeding 40 lines
  - Functions with multiple responsibilities
  - Functions with high cyclomatic complexity (>10)
- **Atomicity issues:**
  - Functions doing more than one thing
  - Side effects not clearly indicated
  - Mixed levels of abstraction within functions
- **Function signature problems:**
  - Too many parameters (>5)
  - Boolean parameters that reduce readability
  - Return values that could be simplified

### 6. Common Go Code Smells
- **Error handling issues:**
  - Ignored errors (using `_`)
  - Errors not being wrapped with context
  - Panic/recover used inappropriately
  - In-band error signaling instead of explicit returns
- **Concurrency problems:**
  - Race conditions
  - Goroutine leaks
  - Missing context propagation
  - Improper channel usage
- **Resource management:**
  - Missing `defer` for cleanup
  - File/connection leaks
  - Not closing response bodies
- **Performance anti-patterns:**
  - Unnecessary allocations in hot paths
  - String concatenation in loops
  - Inefficient slice operations
  - Map misuse

### 7. Testing and Documentation Gaps
- **Test coverage analysis:**
  - Functions without tests
  - Untested error paths
  - Missing edge cases
  - Lack of table-driven tests where appropriate
- **Documentation issues:**
  - Exported functions/types without comments
  - Comments not following Go conventions
  - Outdated or incorrect documentation
  - Missing package-level documentation

### 8. Specific Refactoring Recommendations

For each issue found, provide:
1. **Location**: File path and line number
2. **Issue**: Clear description of the problem
3. **Impact**: Why this matters (performance, maintainability, bugs)
4. **Solution**: Specific refactoring approach
5. **Example**: Before/after code snippet if helpful

### 9. Priority Action Items

Organize findings into:
- **Critical** (bugs, security issues, race conditions)
- **High** (significant performance issues, major architectural problems)
- **Medium** (code duplication, long functions, missing tests)
- **Low** (style issues, minor optimizations)

### 10. Automation Opportunities

Identify which improvements can be:
- **Auto-fixed** by tools (gofmt, goimports, golint --fix)
- **Semi-automated** with careful review
- **Manual only** requiring human judgment

## Output Format

Please provide:
1. **Executive Summary** - Overall code quality assessment (1-2 paragraphs)
2. **Metrics Overview** - Key statistics (lines of code, test coverage, complexity scores)
3. **Critical Issues** - Must-fix problems with specific locations
4. **Improvement Opportunities** - Organized by category with examples
5. **Refactoring Roadmap** - Prioritized list of changes with effort estimates
6. **Quick Wins** - Changes that can be made immediately with high impact

## Additional Context Questions

Before analysis, consider:
- What is the intended use case of this codebase?
- Are there performance requirements?
- What Go version is being targeted?
- Are there team-specific conventions to follow?
- What is the deployment environment?

---

**Note**: Focus on actionable, specific feedback rather than generic advice. Each finding should include the file path and line number where applicable. Prioritize issues that have the highest impact on code quality, maintainability, and correctness.
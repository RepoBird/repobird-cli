# Development TODO

## High Priority Tasks

- Usage progress bar 0.00% always on status page.
- It doesnt know free/pro tier usage limits lets hard code it and use that
  (allow more than total allowed to exist because admin can credit extra runs.)

### 0. Enhanced Pagination for 100+ Runs
**Task File:** `tasks/runs-pagination-autoload.md`
- [ ] Implement cache infrastructure for run pagination
- [ ] Add API pagination enhancement with retry logic  
- [ ] Modify TUI navigation to prevent wrap-around behavior
- [ ] Add Load More button for manual pagination
- [ ] Show "X of Y total runs loaded" in status line
- [ ] Test with users having 1000+ runs
- [ ] Update documentation

**Current Status:** ✅ Simple fix implemented - increased limit to 1000 runs

### 1. Add Global Configuration Support
**Task File:** `tasks/add-global-configs.md`
- [ ] Implement global config file support (~/.repobird/global.yaml)
- [ ] Add environment-specific configurations (dev, staging, prod)
- [ ] Support config inheritance and overrides
- [ ] Add config validation and schema

### 2. Improve FZF Integration
**Task File:** `tasks/improve-fzf-handling.md`
- [ ] Add inline FZF selection for run IDs
- [ ] Implement FZF for task file selection
- [ ] Add FZF preview for task JSON files
- [ ] Support FZF for branch selection
- [ ] Handle FZF binary detection gracefully

### 3. Enhanced TUI Features
**Task File:** `tasks/enhance-tui-features.md`
- [ ] Add real-time log streaming in TUI
- [ ] Implement split pane view for multiple runs
- [ ] Add keyboard shortcuts for common actions
- [ ] Support theme customization
- [ ] Add TUI configuration persistence

### 4. Batch Operations Support
**Task File:** `tasks/implement-batch-operations.md`
- [ ] Support running multiple tasks from directory
- [ ] Add batch status checking
- [ ] Implement parallel task execution
- [ ] Add batch cancellation support
- [ ] Create batch results summary view

### 5. Offline Mode Implementation
**Task File:** `tasks/add-offline-mode.md`
- [ ] Cache run data locally
- [ ] Queue tasks for later submission
- [ ] Sync when connection restored
- [ ] Add offline status indicators
- [ ] Implement conflict resolution

## Medium Priority Tasks

### 6. Enhanced Error Recovery
**Task File:** `tasks/enhance-error-recovery.md`
- [ ] Add automatic retry with exponential backoff
- [ ] Implement circuit breaker pattern
- [ ] Add detailed error diagnostics
- [ ] Create error recovery suggestions
- [ ] Log errors to file for debugging

### 7. Performance Optimizations
**Task File:** `tasks/optimize-performance.md`
- [ ] Implement request caching
- [ ] Add connection pooling
- [ ] Optimize large file handling
- [ ] Add progress bars for long operations
- [ ] Implement lazy loading for lists

### 8. Security Enhancements
**Task File:** `tasks/enhance-security.md`
- [ ] Add OS keychain integration for API keys
- [ ] Implement secure credential storage
- [ ] Add API key rotation support
- [ ] Implement audit logging
- [ ] Add security scan integration

### 9. Testing Infrastructure
**Task File:** `tasks/improve-testing.md`
- [ ] Add integration tests
- [ ] Implement E2E test suite
- [ ] Add performance benchmarks
- [ ] Create test data generators
- [ ] Add mutation testing

### 10. Documentation Generation
**Task File:** `tasks/add-docs-generation.md`
- [ ] Auto-generate CLI command docs
- [ ] Create API client documentation
- [ ] Add inline help system
- [ ] Generate configuration examples
- [ ] Create troubleshooting database

## Low Priority Tasks

### 11. Plugin System
**Task File:** `tasks/implement-plugin-system.md`
- [ ] Design plugin architecture
- [ ] Add plugin discovery mechanism
- [ ] Implement plugin API
- [ ] Create example plugins
- [ ] Add plugin marketplace support

### 12. Analytics and Telemetry
**Task File:** `tasks/add-analytics.md`
- [ ] Add usage analytics (opt-in)
- [ ] Implement performance metrics
- [ ] Create usage reports
- [ ] Add crash reporting
- [ ] Implement feature usage tracking

### 13. Multi-Repository Support
**Task File:** `tasks/add-multi-repo-support.md`
- [ ] Support GitLab repositories
- [ ] Add Bitbucket integration
- [ ] Implement generic Git support
- [ ] Add repository switching
- [ ] Create repository profiles

### 14. Advanced Filtering
**Task File:** `tasks/add-advanced-filtering.md`
- [ ] Add run filtering by status
- [ ] Implement date range filters
- [ ] Add regex pattern matching
- [ ] Create saved filter presets
- [ ] Add filter combination logic

### 15. Export and Import Features
**Task File:** `tasks/add-export-import.md`
- [ ] Export run history to JSON/CSV
- [ ] Import task definitions
- [ ] Add backup/restore functionality
- [ ] Create migration tools
- [ ] Support data archiving

## Bug Fixes and Improvements

### 16. Known Issues
**Task File:** `tasks/fix-known-issues.md`
- [ ] Fix timeout handling for long-running tasks
- [ ] Resolve Windows path separator issues
- [ ] Fix TUI rendering on small terminals
- [ ] Handle network interruptions gracefully
- [ ] Fix config file permission issues

### 17. UI/UX Improvements
**Task File:** `tasks/improve-ux.md`
- [ ] Add command aliases
- [ ] Improve error message clarity
- [ ] Add interactive prompts for missing params
- [ ] Create guided setup wizard
- [ ] Add command suggestions

## Future Features

### 18. AI Integration Enhancements
**Task File:** `tasks/enhance-ai-integration.md`
- [ ] Add context file auto-detection
- [ ] Implement smart task suggestions
- [ ] Add natural language task creation
- [ ] Create task templates library
- [ ] Add AI-powered error diagnosis

### 19. Collaboration Features
**Task File:** `tasks/add-collaboration.md`
- [ ] Add team workspace support
- [ ] Implement run sharing
- [ ] Add commenting system
- [ ] Create approval workflows
- [ ] Add notification system

### 20. Monitoring Dashboard
**Task File:** `tasks/add-monitoring-dashboard.md`
- [ ] Create web-based dashboard
- [ ] Add real-time status updates
- [ ] Implement metrics visualization
- [ ] Add alerting system
- [ ] Create custom dashboards

## Notes

- Tasks should be tackled based on user feedback and usage patterns
- Each task file should include detailed requirements, implementation notes, and testing criteria
- Consider backward compatibility for all changes
- Update documentation alongside feature implementation
- Ensure cross-platform compatibility for all new features

## Contributing

When picking up a task:
1. Create the corresponding task file in `tasks/` directory
2. Update this TODO with progress
3. Create a feature branch following naming conventions
4. Implement with tests and documentation
5. Submit PR for review

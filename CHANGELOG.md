# Changelog

All notable changes to RepoBird CLI will be documented in this file.

## [Unreleased]

### Improved
- Git operations now respond faster with automatic timeouts
- Better error handling when cache directories can't be created
- Cleaner interface with reduced visual clutter

### Fixed
- Repository statistics now display correctly in the dashboard
- Branch detection errors are handled more gracefully

## [0.1.2]

### Added
- Bug report template for easier issue reporting
- Feature request and pull request templates
- Command aliases in documentation for easier command discovery
- GitHub Actions CI for automated quality checks

### Improved
- Dashboard now displays repository statistics more accurately
- Better sorting of repositories in the interface
- Clearer documentation for configuration and setup
- Enhanced error messages when branches aren't found

### Fixed
- Git backup directories are now properly ignored
- Environment variable handling in tests
- Resource cleanup in various operations

### Changed
- Auto-detection of git information temporarily disabled while improvements are made
- Default branch handling now respects repository settings
- Development environment now uses localhost:3000 by default

## [0.1.1] and earlier

Initial releases establishing core functionality:
- Terminal UI for managing AI-powered code generation
- Command-line interface for submitting tasks
- Bulk operations support
- Configuration management
- API integration with RepoBird platform
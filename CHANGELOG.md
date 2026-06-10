# Changelog

All notable changes to RepoBird CLI will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- Single-run creation now supports retry-safe idempotency keys with a 30-second local duplicate-submission guard and --force override.

### Fixed

- Isolate command tests from real XDG RepoBird configuration so config command tests cannot overwrite user settings.

### Changed

- Show the selected repository in Basic/Pro run summaries before submission.

## [0.7.0] - 2026-06-06

### Added

- Added an installable default RepoBird CLI agent skill and README quickstart guidance for npx skills add.

### Changed

- Run creation success output now preserves public run IDs and canonical branch-output fields from the API response.

## [0.6.1] - 2026-06-06

### Fixed

- Make login use the production API URL by default instead of persisted custom API endpoint overrides.

## [0.6.0] - 2026-06-05

### Added

- Colored human-readable CLI output is enabled by default for terminal users, with config/env opt-outs via color=never, REPOBIRD_COLOR, and NO_COLOR.
- Run creation now supports --acknowledge-prompt-risk and acknowledgePromptRisk config files for explicit prompt-risk acknowledgement
- Add `repobird completion install` to configure shell completions for bash, zsh, fish, and PowerShell.

### Fixed

- Prevent login and progress output from emitting duplicate redraw lines in non-interactive terminals.
- Account info, status, and TUI usage displays now show credit balances instead of all-zero legacy run quotas.
- Cobra-generated help and version output now use the same colored output policy as other human-readable CLI output.
- Store API keys under the XDG config directory by default while reading and migrating legacy ~/.repobird keys.
- Prevent local tests from overwriting the desktop keyring API key.
- Secondary run-list repository pagination now sends page/limit parameters to match the current /api/v1/runs API
- Remove legacy API key files and plain-text config entries after migrating them into XDG secure storage

## [0.5.0] - 2026-06-01

### Added

- Branch-only run submission is now available from the CLI with --branch-only/--no-pr and branchOnly config files.
- Added feature-gated repo commands to inspect and update repository branch defaults.

### Changed

- Run creation now supports baseBranch/outputMode/outputBranch/prTargetBranch/outputBranchPolicy while preserving source/target/branchOnly compatibility.

## [0.4.0] - 2026-05-27

### Added

- Added Basic/Pro CLI run presets with model selection output and prompt shortcut commands

## [0.3.0] - 2026-05-25

### Changed

- Synced run pagination and account usage display with current RepoBird API behavior
- Legacy bulk run workflows are now gated behind development-only environment flags

## [0.2.0] - 2026-05-23

### Changed

- Single-run creation now sends the current OpenCode agent contract to the API
- Git operations now respond faster with automatic timeouts
- Better error handling when cache directories can't be created
- Cleaner interface with reduced visual clutter
- Bulk run workflows are hidden and disabled while the API keeps them behind a legacy gate
- README now points users to the live RepoBird docs and CLI quick start, with OpenCode and credit-based positioning
- Public docs and examples no longer advertise bulk runs while they remain legacy-gated

### Fixed

- Repository statistics now display correctly in the dashboard
- Branch detection errors are handled more gracefully

## [0.1.2] - 2024-01-01

### Added

- Bug report template for easier issue reporting
- Feature request and pull request templates
- Command aliases in documentation for easier command discovery
- GitHub Actions CI for automated quality checks

### Changed

- Dashboard now displays repository statistics more accurately
- Better sorting of repositories in the interface
- Clearer documentation for configuration and setup
- Enhanced error messages when branches aren't found
- Auto-detection of git information temporarily disabled while improvements are made
- Default branch handling now respects repository settings
- Development environment now uses localhost:3000 by default

### Fixed

- Git backup directories are now properly ignored
- Environment variable handling in tests
- Resource cleanup in various operations

## [0.1.1] - 2024-01-01

### Added

- Initial terminal UI for managing AI-powered code generation
- Command-line interface for submitting tasks
- Bulk operations support
- Configuration management
- API integration with RepoBird platform

[Unreleased]: https://github.com/RepoBird/repobird-cli/compare/v0.7.0...HEAD
[0.7.0]: https://github.com/RepoBird/repobird-cli/compare/v0.6.1...v0.7.0
[0.6.1]: https://github.com/RepoBird/repobird-cli/compare/v0.6.0...v0.6.1
[0.6.0]: https://github.com/RepoBird/repobird-cli/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/RepoBird/repobird-cli/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/RepoBird/repobird-cli/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/RepoBird/repobird-cli/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/RepoBird/repobird-cli/compare/v0.1.2...v0.2.0
[0.1.2]: https://github.com/RepoBird/repobird-cli/compare/v0.1.1...v0.1.2

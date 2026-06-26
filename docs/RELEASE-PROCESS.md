# RepoBird CLI Release Process

## Overview

The RepoBird CLI uses GoReleaser for creating cross-platform builds and GitHub releases. This document explains the release workflow and common operations.

## Release Workflow

### 1. Standard Release

```bash
# Ensure VERSION file contains the desired version (e.g., 1.2.3)
echo "1.2.3" > VERSION

# Create a signed release (will prompt for GPG key)
make release-github
```

This will:
1. Run validation checks (tests, formatting)
2. Build the binary and generate completions/man pages
3. Test build with GoReleaser in snapshot mode
4. **Generate changelog preview and prompt for approval**
5. Create and push git tag after successful test
6. Run GoReleaser to create GitHub release with all artifacts
7. Sign release artifacts with GPG

### Changelog Preview

The release script now generates an **interactive changelog preview** before publishing:

- Automatically filters out internal commits (chore, docs, test, style, refactor, perf, build, ci)
- Shows only user-facing changes (feat, fix, BREAKING CHANGE)
- Groups commits by type (Features, Bug Fixes, Breaking Changes)
- Allows you to:
  - **[Y]** Accept and continue
  - **[e]** Edit changelog in your `$EDITOR` (defaults to nano)
  - **[N]** Cancel release

Example workflow:
```bash
make release-github

# ... validation and build steps ...

# Generated changelog preview:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Release Notes

## Features
- feat(run.go): add printing of RepoBird URL for each run
- feat(tui): add debug loading mode for debugging

## Bug Fixes
- fix(install.sh): update binary filename format

## Breaking Changes
(none)

---
**Note:** Internal changes are excluded from this changelog.
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Options:
  [Y] Accept and continue with release
  [e] Edit changelog in $EDITOR (nano)
  [N] Cancel release

Proceed with this changelog? (Y/e/N)
```

You can:
- Press **Y** to accept and continue
- Press **e** to edit in your editor (set with `export EDITOR=vim`)
- Press **N** to cancel and make changes

### Changelog Filtering

The changelog automatically excludes commits based on **Conventional Commits / OpenCommit standards**:

**Excluded (internal-only):**
- `chore:` - Routine maintenance
- `docs:` - Documentation changes
- `test:` - Test-only changes
- `style:` - Code formatting
- `refactor:` - Code restructuring
- `perf:` - Performance improvements (internal)
- `build:` - Build system changes
- `ci:` - CI/CD changes

**Included (user-facing):**
- `feat:` - New features
- `fix:` - Bug fixes
- `BREAKING CHANGE:` - Breaking changes (any type)

### 2. Draft Release

```bash
# Create a draft release for review
make release-github-draft
```

### 3. Local Build Only

```bash
# Build release artifacts without publishing
./scripts/local-github-release.sh --local-only
```

## How It Works

### Tag Management
- The script first tests the build in snapshot mode
- Only after successful test, it creates and pushes the git tag
- This prevents broken tags if the build fails
- GoReleaser expects the tag to exist before creating the release

### GoReleaser vs gh CLI
- **GoReleaser** handles the entire release process:
  - Creates GitHub release from existing tag
  - Uploads all binary artifacts
  - Generates release notes
  - Creates checksums
  - Signs artifacts (if configured)
- **We do NOT use `gh release create`** because GoReleaser already creates the release
- Using both would cause duplicate release errors

### Generated Artifacts
During the release process, these artifacts are generated:
- `completions/` - Shell completion files
- `man/` - Manual pages
- `dist/` - GoReleaser build output

These are automatically cleaned up and not committed to git.

### Installer Distribution Contract

GitHub Releases are the canonical host for CLI installers and binary archives. The web app should expose `https://repobird.ai/install`, `https://repobird.ai/install.sh`, and `https://repobird.ai/install.ps1` as thin install pages, script responses, or redirects to this repository's installer scripts.

GoReleaser archive names must remain stable because the installers and web entrypoints depend on `releases/latest/download/<asset>` URLs:

- `repobird-cli_darwin_amd64.tar.gz`
- `repobird-cli_darwin_arm64.tar.gz`
- `repobird-cli_linux_amd64.tar.gz`
- `repobird-cli_linux_arm64.tar.gz`
- `repobird-cli_windows_amd64.zip`
- `repobird-cli_windows_386.zip`
- `checksums.txt`

The shell and PowerShell installers verify downloaded archives against `checksums.txt` before installing. Do not change `.goreleaser.yml` `archives.name_template` or `checksum.name_template` without updating both installers and `docs/INSTALLATION.md`.

## Common Operations

### Reset a Failed Release

If a release fails or needs to be redone:

```bash
# Clean up the last release attempt
make release-reset

# This will:
# - Delete local git tag
# - Delete remote git tag from GitHub
# - Delete GitHub release (if exists)
# - Clean up dist/ and generated artifacts
```

### Manual Tag Management

```bash
# Create tag manually (not recommended - use release script)
git tag -a v1.2.3 -m "Release v1.2.3"
git push gh v1.2.3

# Delete tag
git tag -d v1.2.3
git push gh --delete v1.2.3
```

### Check Release Status

```bash
# View current version
cat VERSION

# Check if tag exists locally
git tag -l v1.2.3

# Check if release exists on GitHub
gh release view v1.2.3
```

## Package Manager Support

Currently disabled package managers (require additional setup):
- **Homebrew** - Requires HOMEBREW_TAP_GITHUB_TOKEN
- **Scoop** - Requires SCOOP_GITHUB_TOKEN  
- **Chocolatey** - Requires Windows build environment
- **Snapcraft** - Requires snapcraft tools
- **AUR** - Requires AUR SSH key

Supported formats:
- **.deb** packages (Debian/Ubuntu)
- **.rpm** packages (RedHat/Fedora)
- **tar.gz** archives (Linux/macOS)
- **.zip** archives (Windows)

## Troubleshooting

### "Release already exists" Error

This happens when GoReleaser successfully creates the release. The script used to try creating it again with `gh release create`, which is now fixed.

If you see this error:
1. The release was likely successful - check GitHub
2. Run `make release-reset` to clean up if needed

### "Uncommitted changes" Error

The release script checks for clean git state. Either:
- Commit your changes: `git add . && git commit -m "..."`
- Stash temporarily: `git stash`

### GPG Signing Issues

If GPG signing fails:
1. Check you have a GPG key: `gpg --list-secret-keys`
2. Generate one if needed: `gpg --full-generate-key`
3. Or release without signing: remove `--sign` flag

### Wrong Git Remote

The script defaults to using 'gh' remote for GitHub. If your remote is named differently:

```bash
# Check your remotes
git remote -v

# Use --remote flag
./scripts/local-github-release.sh --remote origin
```

## Environment Variables

- `GITHUB_TOKEN` - GitHub API token (auto-detected from `gh auth token`)
- `GPG_FINGERPRINT` - GPG key for signing (auto-detected)
- `GITLAB_TOKEN` - Set to empty string to prevent conflicts

## Best Practices

1. **Always test locally first**: Use `--local-only` flag
2. **Keep VERSION file updated**: This is the source of truth
3. **Use semantic versioning**: MAJOR.MINOR.PATCH
4. **Create releases from main branch**: Unless hotfix
5. **Sign releases when possible**: Improves security
6. **Clean up failed attempts**: Use `make release-reset`

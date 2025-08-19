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
4. Create and push git tag after successful test
5. Run GoReleaser to create GitHub release with all artifacts
6. Sign release artifacts with GPG

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
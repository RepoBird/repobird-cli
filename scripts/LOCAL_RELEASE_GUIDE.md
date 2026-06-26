# Complete Local Release Guide

This guide explains how to create and publish RepoBird CLI releases entirely from your local machine without GitHub Actions.

## Prerequisites

### Git Remote Setup

**IMPORTANT**: If your GitHub remote is not named `origin`, you need to specify it:
```bash
# Check your remotes
git remote -v

# If GitHub remote is named 'gh' instead of 'origin', use:
./scripts/local-github-release.sh --remote gh
```

The script defaults to using `gh` as the remote name for GitHub.

### Required Tools

1. **GitHub CLI (`gh`)**
   ```bash
   # macOS/Linux
   brew install gh
   
   # Or download from https://cli.github.com
   ```

2. **GoReleaser**
   ```bash
   # macOS/Linux
   brew install goreleaser
   
   # Or see https://goreleaser.com/install
   ```

3. **Git** and **Go** (already required for development)

### Optional Tools

For signing releases:
```bash
# GPG for signing
brew install gnupg
```

For package managers:
```bash
# Chocolatey (Windows)
choco install chocolatey

# For building packages
apt-get install dpkg-dev rpm  # Linux
```

### Initial Setup

1. **Authenticate GitHub CLI**
   ```bash
   gh auth login
   ```

2. **Set up GPG signing (optional)**
   ```bash
   # Generate GPG key
   gpg --full-generate-key
   
   # Export public key
   gpg --armor --export your-email@example.com
   # Add this to GitHub: Settings → SSH and GPG keys
   ```

## Release Workflow

### Step 1: Prepare the Release

1. **Ensure you're on main branch**
   ```bash
   git checkout main
   git pull origin main
   ```

2. **Update VERSION file**
   ```bash
   echo "v1.2.3" > VERSION
   ```

3. **Update CHANGELOG.md with release notes**
   ```bash
   # Edit CHANGELOG.md with your release notes
   ```

4. **Run tests and checks**
   ```bash
   ./scripts/local-ci.sh
   ```

5. **Commit the changes**
   ```bash
   git add VERSION CHANGELOG.md
   git commit -m "chore: prepare release v1.2.3"
   git push origin main
   ```

### Step 2: Create GitHub Release

#### Option A: Full Automated Release (Recommended)

```bash
# Create and publish release with all artifacts (reads VERSION file)
./scripts/local-github-release.sh

# For a pre-release
./scripts/local-github-release.sh --prerelease

# For a signed release
./scripts/local-github-release.sh --sign

# With custom release notes
./scripts/local-github-release.sh --notes RELEASE_NOTES.md

# Override version (only if needed)
./scripts/local-github-release.sh --version v1.2.3-hotfix
```

#### Option B: Manual Steps

1. **Build release artifacts**
   ```bash
   # Using GoReleaser (recommended)
   ./scripts/local-release.sh --version v1.2.3 --goreleaser
   
   # Or manual cross-compilation
   ./scripts/local-release.sh --version v1.2.3 --cross-compile
   ```

2. **Create GitHub release**
   ```bash
   # Create tag
   git tag -a v1.2.3 -m "Release v1.2.3"
   git push origin v1.2.3
   
   # Create release with artifacts
   gh release create v1.2.3 \
     ./dist/*.tar.gz \
     ./dist/*.zip \
     ./dist/checksums.txt \
     --title "RepoBird CLI v1.2.3" \
     --generate-notes
   ```

### Step 3: Update Package Managers

After the GitHub release is created, update package managers:

```bash
# Update all package managers (reads VERSION file)
./scripts/local-package-publish.sh --all

# Or update specific ones
./scripts/local-package-publish.sh --homebrew --scoop

# Dry run to see what would be done
./scripts/local-package-publish.sh --all --dry-run

# Override version (only if needed)
./scripts/local-package-publish.sh --version v1.2.3-hotfix --all
```

#### Individual Package Manager Updates

##### Homebrew
```bash
./scripts/local-package-publish.sh --homebrew
```
This will:
- Clone/update the homebrew-tap repository
- Update the formula with new version and checksums
- Commit and push changes

##### Scoop
```bash
./scripts/local-package-publish.sh --scoop
```
This will:
- Create/update the Scoop manifest
- Update checksums automatically
- Push to scoop-bucket repository if it exists

##### Chocolatey
```bash
./scripts/local-package-publish.sh --chocolatey

# If Chocolatey is installed, push the package
cd /tmp/repobird-chocolatey
choco push repobird.*.nupkg --source https://push.chocolatey.org/ --api-key YOUR_API_KEY
```

##### AUR (Arch Linux)
```bash
./scripts/local-package-publish.sh --aur

# Then manually push to AUR
cd /tmp/repobird-aur
makepkg --printsrcinfo > .SRCINFO
git clone ssh://aur@aur.archlinux.org/repobird.git aur-repo
cp PKGBUILD .SRCINFO aur-repo/
cd aur-repo
git add .
git commit -m "Update to v1.2.3"
git push
```

## Complete Release Checklist

```bash
# 1. Preparation
□ On main branch and up to date
□ Version updated in VERSION file
□ CHANGELOG.md updated
□ All tests passing (./scripts/local-ci.sh)
□ Changes committed and pushed

# 2. Build and Release
□ Run: ./scripts/local-github-release.sh
□ Verify release on GitHub

# 3. Package Managers
□ Run: ./scripts/local-package-publish.sh --all
□ Verify Homebrew formula updated
□ Verify Scoop manifest updated
□ Push Chocolatey package if applicable
□ Update AUR package if applicable

# 4. Announcements
□ Tweet/post about release
□ Update documentation if needed
□ Notify users via appropriate channels
```

## Quick Release Commands

### Standard Release
```bash
# Update VERSION file first
echo "v1.2.3" > VERSION

# Complete release in one command
./scripts/local-github-release.sh && \
./scripts/local-package-publish.sh --all
```

### Pre-release/Beta
```bash
# Update VERSION file with pre-release version
echo "v1.2.3-beta.1" > VERSION

# Create pre-release
./scripts/local-github-release.sh --prerelease

# Don't update stable package managers for pre-releases
```

### Hotfix Release
```bash
# Quick hotfix release
git checkout -b hotfix/v1.2.4
# Make fixes
git commit -am "fix: critical bug"
git checkout main
git merge hotfix/v1.2.4

# Update VERSION file
echo "v1.2.4" > VERSION
git add VERSION
git commit -m "chore: bump version to v1.2.4"

./scripts/local-github-release.sh --skip-validation
./scripts/local-package-publish.sh --homebrew --scoop
```

## Troubleshooting

### Issue: "gh: command not found"
Install GitHub CLI:
```bash
brew install gh
# Or download from https://cli.github.com
```

### Issue: "goreleaser: command not found"
Install GoReleaser:
```bash
brew install goreleaser
# Or see https://goreleaser.com/install
```

### Issue: "Release already exists"
Delete and recreate:
```bash
gh release delete v1.2.3 --yes
git tag -d v1.2.3
git push origin :v1.2.3
# Then run release script again
```

### Issue: "GPG signing failed"
Check GPG setup:
```bash
gpg --list-secret-keys
# If no keys, generate one:
gpg --full-generate-key
```

### Issue: "Checksum mismatch in package manager"
Re-download and verify:
```bash
# Get correct checksum
curl -L https://github.com/repobird/repobird-cli/releases/download/v1.2.3/checksums.txt
# Update package manager config with correct checksum
```

## Environment Variables

- `GITHUB_TOKEN` - GitHub authentication token (auto-detected from `gh`)
- `GPG_FINGERPRINT` - GPG key fingerprint for signing
- `HOMEBREW_TAP_GITHUB_TOKEN` - Token for updating Homebrew tap
- `CHOCOLATEY_API_KEY` - API key for pushing to Chocolatey

## Advanced Usage

### Building Without Publishing
```bash
# Build locally without any GitHub interaction
./scripts/local-github-release.sh --version v1.2.3 --local-only
```

### Custom GoReleaser Config
```bash
# Use custom .goreleaser.yml
goreleaser release --config .goreleaser.custom.yml --skip=publish
```

### Parallel Package Updates
```bash
# Update multiple package managers in parallel
parallel -j 4 ./scripts/local-package-publish.sh --version v1.2.3 ::: \
  --homebrew --scoop --chocolatey --aur
```

## Script Reference

| Script | Purpose | Key Options |
|--------|---------|-------------|
| `local-github-release.sh` | Create GitHub releases | `--version`, `--draft`, `--prerelease`, `--sign` |
| `local-package-publish.sh` | Update package managers | `--version`, `--homebrew`, `--scoop`, `--all` |
| `local-release.sh` | Build artifacts | `--version`, `--cross-compile`, `--goreleaser` |
| `local-ci.sh` | Run CI checks | None |
| `local-test.sh` | Run tests | `--coverage`, `--go-version` |

## Security Notes

1. **Never commit API keys or tokens**
2. **Use environment variables for sensitive data**
3. **Sign releases with GPG when possible**
4. **Verify checksums after building**
5. **Test releases before marking as latest**

## Automation Tips

Create a release alias in your shell:
```bash
# Add to ~/.bashrc or ~/.zshrc
alias release='function _release() {
  version=$1
  ./scripts/local-github-release.sh --version $version && \
  ./scripts/local-package-publish.sh --version $version --all
}; _release'

# Usage
release v1.2.3
```

## Support

For issues with the release process:
1. Check the script output for detailed error messages
2. Run with `bash -x` for debug output
3. Check GitHub/package manager documentation
4. File an issue in the repository
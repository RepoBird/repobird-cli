# Package Manager Testing Guide

This document explains how to test RepoBird CLI across different package managers and platforms using GitHub Actions CI and local testing methods.

## Overview

We have two comprehensive testing workflows:

1. **`test-local-packages.yml`** - Tests you can run immediately on your current codebase
2. **`test-packages.yml`** - Full integration tests requiring external package repositories

## 🚀 Quick Start - Local Testing

### Run Immediate Tests

These tests work with your current code without external dependencies:

```bash
# Test on your current machine
make build
make test
make install
repobird version
make uninstall

# Test cross-compilation
make build-all

# Test documentation and completions
./scripts/generate-completions.sh
./scripts/generate-docs.sh

# Test package signing (simulation)
./scripts/setup-signing.sh
```

### GitHub Actions - Local Package Tests

The `test-local-packages.yml` workflow runs automatically on push/PR and tests:

- ✅ Cross-platform builds (Linux, macOS, Windows)
- ✅ Local installations
- ✅ Shell completion generation
- ✅ Documentation generation
- ✅ Package format creation (DEB, RPM, archives)
- ✅ Container compatibility
- ✅ GoReleaser configuration
- ✅ Package signing simulation

**Trigger the workflow:**
```bash
git push origin main
# or create a pull request
```

## 📦 Full Package Manager Testing

### Prerequisites

Before running full package manager tests, you need:

1. **Package Repositories Set Up:**
   ```bash
   # Homebrew tap
   github.com/repobird/homebrew-tap
   
   # APT repository
   https://apt.repobird.ai
   
   # YUM repository  
   https://yum.repobird.ai
   
   # Scoop bucket
   github.com/repobird/scoop-bucket
   ```

2. **GitHub Secrets Configured:**
   ```
   GPG_PRIVATE_KEY
   GPG_PASSPHRASE
   HOMEBREW_TAP_GITHUB_TOKEN
   SCOOP_GITHUB_TOKEN
   CHOCOLATEY_API_KEY
   AUR_KEY
   ```

### Package Manager Test Matrix

| Package Manager | OS/Container | Test Coverage |
|----------------|-------------|---------------|
| **Homebrew** | macOS-latest | Install, update, uninstall, functionality |
| **APT** | Ubuntu-latest | Full package lifecycle |
| **Snap** | Ubuntu-latest | Install, refresh, remove |
| **Chocolatey** | Windows-latest | Windows package management |
| **Scoop** | Windows-latest | Windows bucket management |
| **DNF/YUM** | Fedora container | RPM package testing |
| **YUM** | CentOS container | Enterprise Linux testing |
| **AUR** | Arch container | AUR helper integration |
| **APT** | Debian container | Pure Debian testing |

### Running Package Manager Tests

1. **Set up your package repositories first:**
   ```bash
   # Create Homebrew tap repository
   gh repo create repobird/homebrew-tap --public
   
   # Set up APT repository hosting (S3 + CloudFront)
   # Configure domain: apt.repobird.ai
   
   # Set up YUM repository hosting
   # Configure domain: yum.repobird.ai
   ```

2. **Configure GitHub secrets in your repository settings**

3. **Trigger the workflow:**
   ```bash
   # Push to main or create PR
   git push origin main
   
   # Or trigger manually
   gh workflow run test-packages.yml
   
   # Or run on schedule (weekly)
   # The workflow runs automatically every Sunday at 2 AM UTC
   ```

## 🧪 Test Categories Explained

### Local Build Tests
- **Purpose:** Verify code compiles and runs on all platforms
- **Coverage:** Linux, macOS, Windows
- **Tests:** Build, install, version check, uninstall
- **Runtime:** ~5-10 minutes

### Package Format Tests  
- **Purpose:** Verify package creation for all formats
- **Coverage:** DEB, RPM, TAR.GZ, ZIP
- **Tests:** Package creation, content verification
- **Runtime:** ~15-20 minutes

### Container Installation Tests
- **Purpose:** Test compatibility across Linux distributions
- **Coverage:** Ubuntu, Debian, Fedora, Arch Linux
- **Tests:** Build and install in clean containers
- **Runtime:** ~20-30 minutes

### Package Manager Integration Tests
- **Purpose:** Test real-world installation scenarios
- **Coverage:** All major package managers
- **Tests:** Install → Update → Functionality → Uninstall
- **Runtime:** ~30-45 minutes

### Performance Tests
- **Purpose:** Ensure installation speed and reliability
- **Coverage:** Installation benchmarking, concurrent installs
- **Tests:** Timing, load testing, race condition detection
- **Runtime:** ~10-15 minutes

## 🔍 Interpreting Test Results

### Success Indicators
- ✅ All builds complete without errors
- ✅ Binaries execute and show correct version
- ✅ Package installations work on target platforms
- ✅ Uninstalls clean up completely
- ✅ Shell completions generate correctly
- ✅ Documentation builds successfully

### Common Failure Patterns

#### Build Failures
```bash
# Check Go version compatibility
go version

# Check missing dependencies
make deps

# Check build flags
make build
```

#### Package Manager Issues
```bash
# Repository not found
curl -I https://apt.repobird.ai  # Should return 200

# GPG key issues
curl -fsSL https://apt.repobird.ai/gpg | gpg --import

# Permission issues
sudo make install-global
```

#### Container Issues
```bash
# Network connectivity
docker run --rm ubuntu:latest ping google.com

# Package manager updates
apt update && apt install -y curl
```

## 🛠️ Local Testing Commands

### Test Individual Package Managers

```bash
# Test Homebrew (macOS)
brew tap repobird/tap
brew install repobird
repobird version
brew uninstall repobird

# Test APT (Ubuntu/Debian)
sudo apt install repobird
repobird version  
sudo apt remove repobird

# Test Snap (Ubuntu)
sudo snap install repobird
repobird version
sudo snap remove repobird

# Test Chocolatey (Windows)
choco install repobird
repobird version
choco uninstall repobird

# Test manual installation
curl -fsSL https://get.repobird.ai | sh
repobird version
rm ~/.local/bin/repobird
```

### Test Package Creation

```bash
# Test DEB package creation
make build
./scripts/create-deb-package.sh

# Test RPM package creation  
./scripts/create-rpm-package.sh

# Test archive creation
make build-all
```

### Test Signing

```bash
# Set up signing
./scripts/setup-signing.sh

# Sign packages
./scripts/sign-packages.sh

# Verify signatures
./scripts/verify-packages.sh
```

## 📊 Test Reports and Artifacts

### GitHub Actions Artifacts

Each test run produces downloadable artifacts:

- **`test-packages-*`** - Generated package files
- **`goreleaser-dist`** - GoReleaser build outputs  
- **`installation-logs`** - Detailed installation logs
- **`performance-reports`** - Timing and performance data

### Accessing Test Results

```bash
# Download artifacts using GitHub CLI
gh run download <run-id>

# View workflow logs
gh run view <run-id> --log

# List recent runs
gh run list --workflow=test-packages.yml
```

## 🚨 Troubleshooting

### Common Issues and Solutions

#### Repository Setup Issues
```bash
# Verify repository accessibility
curl -I https://apt.repobird.ai
curl -I https://yum.repobird.ai

# Check DNS resolution
nslookup apt.repobird.ai
```

#### GPG Signing Issues
```bash
# Check GPG key
gpg --list-secret-keys

# Test signing
echo "test" | gpg --clearsign
```

#### Container Permission Issues
```bash
# Run with privileged mode
docker run --privileged ubuntu:latest

# Check user permissions
docker run --user root ubuntu:latest
```

### Getting Help

1. **Check workflow logs** - View detailed error messages
2. **Review artifacts** - Download and inspect generated files
3. **Test locally** - Reproduce issues on your machine
4. **Check package repositories** - Verify external services are up

## 📈 Continuous Improvement

### Adding New Package Managers

To add a new package manager:

1. **Add to test matrix** in `test-packages.yml`
2. **Create container config** if needed
3. **Add installation commands**
4. **Update documentation**

### Performance Optimization

Monitor and optimize:
- Installation time benchmarks
- Package size optimization
- CI runtime efficiency
- Artifact retention policies

## 🔄 Maintenance Schedule

- **Daily:** Automated tests on push/PR
- **Weekly:** Full package manager integration tests
- **Monthly:** Performance benchmarking
- **Quarterly:** Test infrastructure review

This comprehensive testing ensures RepoBird CLI works reliably across all supported platforms and package managers.
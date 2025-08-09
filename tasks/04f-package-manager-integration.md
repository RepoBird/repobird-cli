# Task 04f: Package Manager Integration

## Overview
Integrate RepoBird CLI with major package managers across all platforms, enabling easy installation, updates, and dependency management for users.

## Background Research

### Package Manager Best Practices
Based on industry standards:
- **Support multiple distribution channels** - Match user OS and preferences
- **Automate package builds** - Use CI/CD to reduce errors and maintain versions
- **Synchronize versions** - Keep all channels up-to-date simultaneously
- **Static binaries preferred** - Minimize dependencies for compatibility
- **Sign packages** - Ensure authenticity and security
- **Auto-update support** - Enable seamless upgrades for users

### Target Package Managers
- **Homebrew** - macOS/Linux
- **APT/DEB** - Debian/Ubuntu
- **YUM/RPM** - RHEL/Fedora/CentOS
- **Chocolatey** - Windows
- **Scoop** - Windows
- **Snap** - Universal Linux
- **AUR** - Arch Linux

## Implementation Tasks

### 1. Homebrew Formula
- [ ] Create Homebrew tap repository:
  ```bash
  github.com/repobird/homebrew-repobird
  ```
- [ ] Write formula `repobird.rb`:
  ```ruby
  class Repobird < Formula
    desc "Fast CLI for RepoBird AI agent platform"
    homepage "https://github.com/repobird/repobird-cli"
    version "1.0.0"
    
    on_macos do
      if Hardware::CPU.arm?
        url "https://github.com/repobird/repobird-cli/releases/download/v1.0.0/repobird_darwin_arm64.tar.gz"
        sha256 "..."
      else
        url "https://github.com/repobird/repobird-cli/releases/download/v1.0.0/repobird_darwin_amd64.tar.gz"
        sha256 "..."
      end
    end
    
    on_linux do
      if Hardware::CPU.arm?
        url "https://github.com/repobird/repobird-cli/releases/download/v1.0.0/repobird_linux_arm64.tar.gz"
        sha256 "..."
      else
        url "https://github.com/repobird/repobird-cli/releases/download/v1.0.0/repobird_linux_amd64.tar.gz"
        sha256 "..."
      end
    end
    
    def install
      bin.install "repobird"
      
      # Install shell completions
      bash_completion.install "completions/repobird.bash"
      zsh_completion.install "completions/_repobird"
      fish_completion.install "completions/repobird.fish"
      
      # Install man pages
      man1.install "man/repobird.1"
    end
    
    test do
      assert_match "repobird version", shell_output("#{bin}/repobird version")
    end
  end
  ```
- [ ] Set up auto-update workflow
- [ ] Test installation: `brew install repobird/repobird/repobird`
- [ ] Submit to Homebrew Core (optional)

### 2. Debian/Ubuntu Packages (.deb)
- [ ] Create debian package structure:
  ```
  debian/
  ├── control
  ├── rules
  ├── changelog
  ├── copyright
  └── repobird.install
  ```
- [ ] Write `debian/control`:
  ```
  Package: repobird
  Version: 1.0.0
  Section: utils
  Priority: optional
  Architecture: amd64
  Maintainer: RepoBird Team <team@repobird.ai>
  Homepage: https://github.com/repobird/repobird-cli
  Description: Fast CLI for RepoBird AI agent platform
   RepoBird CLI enables users to submit AI-powered code
   generation tasks and track their progress.
  ```
- [ ] Set up APT repository hosting
- [ ] Create repository signing key
- [ ] Automate .deb building in CI
- [ ] Document repository addition:
  ```bash
  curl -fsSL https://apt.repobird.ai/gpg | sudo apt-key add -
  echo "deb https://apt.repobird.ai stable main" | sudo tee /etc/apt/sources.list.d/repobird.list
  sudo apt update && sudo apt install repobird
  ```

### 3. RPM Packages (Red Hat/Fedora)
- [ ] Create RPM spec file:
  ```spec
  Name:           repobird
  Version:        1.0.0
  Release:        1%{?dist}
  Summary:        Fast CLI for RepoBird AI agent platform
  License:        MIT
  URL:            https://github.com/repobird/repobird-cli
  Source0:        %{url}/releases/download/v%{version}/repobird_%{version}_linux_amd64.tar.gz
  
  %description
  RepoBird CLI enables users to submit AI-powered code generation tasks
  
  %prep
  %autosetup
  
  %install
  install -D -m 755 repobird %{buildroot}%{_bindir}/repobird
  
  %files
  %{_bindir}/repobird
  ```
- [ ] Set up Copr repository for Fedora
- [ ] Configure OBS for openSUSE Build Service
- [ ] Automate RPM building
- [ ] Sign RPM packages

### 4. Chocolatey Package (Windows)
- [ ] Create Chocolatey package:
  ```powershell
  # repobird.nuspec
  <?xml version="1.0"?>
  <package>
    <metadata>
      <id>repobird</id>
      <version>1.0.0</version>
      <title>RepoBird CLI</title>
      <authors>RepoBird Team</authors>
      <projectUrl>https://github.com/repobird/repobird-cli</projectUrl>
      <licenseUrl>https://github.com/repobird/repobird-cli/blob/main/LICENSE</licenseUrl>
      <description>Fast CLI for RepoBird AI agent platform</description>
      <tags>cli ai code-generation</tags>
    </metadata>
    <files>
      <file src="tools\**" target="tools" />
    </files>
  </package>
  ```
- [ ] Create install script `tools/chocolateyinstall.ps1`:
  ```powershell
  $ErrorActionPreference = 'Stop'
  $toolsDir = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
  $url = 'https://github.com/repobird/repobird-cli/releases/download/v1.0.0/repobird_windows_amd64.zip'
  
  Install-ChocolateyZipPackage $packageName $url $toolsDir
  ```
- [ ] Set up automatic updates (AU)
- [ ] Submit to Chocolatey Community Repository
- [ ] Test: `choco install repobird`

### 5. Scoop Manifest (Windows)
- [ ] Create Scoop bucket repository
- [ ] Write manifest `repobird.json`:
  ```json
  {
    "version": "1.0.0",
    "description": "Fast CLI for RepoBird AI agent platform",
    "homepage": "https://github.com/repobird/repobird-cli",
    "license": "MIT",
    "architecture": {
      "64bit": {
        "url": "https://github.com/repobird/repobird-cli/releases/download/v1.0.0/repobird_windows_amd64.zip",
        "hash": "...",
        "extract_dir": "repobird_windows_amd64"
      },
      "32bit": {
        "url": "https://github.com/repobird/repobird-cli/releases/download/v1.0.0/repobird_windows_386.zip",
        "hash": "...",
        "extract_dir": "repobird_windows_386"
      }
    },
    "bin": "repobird.exe",
    "checkver": "github",
    "autoupdate": {
      "architecture": {
        "64bit": {
          "url": "https://github.com/repobird/repobird-cli/releases/download/v$version/repobird_windows_amd64.zip"
        }
      }
    }
  }
  ```
- [ ] Submit to Scoop Main bucket
- [ ] Test: `scoop install repobird`

### 6. Snap Package (Universal Linux)
- [ ] Create `snapcraft.yaml`:
  ```yaml
  name: repobird
  version: '1.0.0'
  summary: Fast CLI for RepoBird AI agent platform
  description: |
    RepoBird CLI enables users to submit AI-powered code
    generation tasks and track their progress.
  
  grade: stable
  confinement: strict
  base: core22
  
  apps:
    repobird:
      command: bin/repobird
      plugs:
        - network
        - home
        - removable-media
  
  parts:
    repobird:
      plugin: dump
      source: https://github.com/repobird/repobird-cli/releases/download/v1.0.0/repobird_linux_amd64.tar.gz
      source-type: tar
      stage-packages:
        - ca-certificates
  ```
- [ ] Build and test snap locally
- [ ] Publish to Snap Store
- [ ] Set up automatic builds
- [ ] Test: `snap install repobird`

### 7. AUR Package (Arch Linux)
- [ ] Create PKGBUILD:
  ```bash
  # Maintainer: RepoBird Team <team@repobird.ai>
  pkgname=repobird
  pkgver=1.0.0
  pkgrel=1
  pkgdesc="Fast CLI for RepoBird AI agent platform"
  arch=('x86_64' 'aarch64')
  url="https://github.com/repobird/repobird-cli"
  license=('MIT')
  depends=()
  source_x86_64=("$url/releases/download/v$pkgver/repobird_linux_amd64.tar.gz")
  source_aarch64=("$url/releases/download/v$pkgver/repobird_linux_arm64.tar.gz")
  sha256sums_x86_64=('...')
  sha256sums_aarch64=('...')
  
  package() {
    install -Dm755 repobird "$pkgdir/usr/bin/repobird"
    install -Dm644 LICENSE "$pkgdir/usr/share/licenses/$pkgname/LICENSE"
    install -Dm644 README.md "$pkgdir/usr/share/doc/$pkgname/README.md"
  }
  ```
- [ ] Submit to AUR
- [ ] Set up AUR auto-update bot
- [ ] Document installation: `yay -S repobird`

### 8. Package Repository Automation
- [ ] Create unified release workflow:
  ```yaml
  # .github/workflows/package-release.yml
  name: Package Release
  on:
    release:
      types: [published]
  
  jobs:
    homebrew:
      runs-on: ubuntu-latest
      steps:
        - name: Update Homebrew Formula
          run: |
            # Update formula with new version and checksums
            # Push to homebrew-repobird tap
    
    debian:
      runs-on: ubuntu-latest
      steps:
        - name: Build .deb packages
        - name: Upload to APT repository
    
    rpm:
      runs-on: ubuntu-latest
      steps:
        - name: Build RPM packages
        - name: Upload to YUM repository
    
    chocolatey:
      runs-on: windows-latest
      steps:
        - name: Pack Chocolatey package
        - name: Push to Chocolatey
    
    snap:
      runs-on: ubuntu-latest
      steps:
        - name: Build Snap
        - name: Push to Snap Store
    
    aur:
      runs-on: ubuntu-latest
      steps:
        - name: Update PKGBUILD
        - name: Push to AUR
  ```

### 9. Version Management
- [ ] Implement version sync script:
  ```bash
  #!/bin/bash
  VERSION=$1
  
  # Update all package definitions
  sed -i "s/version: .*/version: $VERSION/" snapcraft.yaml
  sed -i "s/pkgver=.*/pkgver=$VERSION/" PKGBUILD
  sed -i "s/Version: .*/Version: $VERSION/" debian/control
  # ... etc for all package formats
  ```
- [ ] Create version validation
- [ ] Set up version bump automation
- [ ] Document release process

### 10. Installation Documentation
- [ ] Create installation guide:
  ```markdown
  # Installation
  
  ## macOS/Linux (Homebrew)
  ```bash
  brew install repobird/repobird/repobird
  ```
  
  ## Ubuntu/Debian
  ```bash
  sudo apt install repobird
  ```
  
  ## Fedora/RHEL
  ```bash
  sudo dnf install repobird
  ```
  
  ## Windows (Chocolatey)
  ```powershell
  choco install repobird
  ```
  
  ## Windows (Scoop)
  ```powershell
  scoop install repobird
  ```
  
  ## Linux (Snap)
  ```bash
  snap install repobird
  ```
  
  ## Arch Linux
  ```bash
  yay -S repobird
  ```
  
  ## Manual Installation
  Download from [releases](https://github.com/repobird/repobird-cli/releases)
  ```
- [ ] Add OS detection script
- [ ] Create one-liner installers
- [ ] Document uninstallation

## Testing Matrix

| Package Manager | Test Command | Update Test | Uninstall Test |
|----------------|--------------|-------------|----------------|
| Homebrew | `brew install repobird` | `brew upgrade repobird` | `brew uninstall repobird` |
| APT | `apt install repobird` | `apt upgrade repobird` | `apt remove repobird` |
| YUM | `yum install repobird` | `yum update repobird` | `yum remove repobird` |
| Chocolatey | `choco install repobird` | `choco upgrade repobird` | `choco uninstall repobird` |
| Scoop | `scoop install repobird` | `scoop update repobird` | `scoop uninstall repobird` |
| Snap | `snap install repobird` | `snap refresh repobird` | `snap remove repobird` |
| AUR | `yay -S repobird` | `yay -Syu repobird` | `yay -R repobird` |

## Auto-Update Configuration

| Manager | Update Method | Frequency | User Control |
|---------|--------------|-----------|--------------|
| Homebrew | `brew upgrade` | Manual/Daily | Full |
| APT | `unattended-upgrades` | Daily | Configurable |
| YUM | `yum-cron` | Daily | Configurable |
| Chocolatey | `choco upgrade all` | Manual/Scheduled | Full |
| Scoop | `scoop update *` | Manual | Full |
| Snap | Automatic | 4x daily | Limited |
| AUR | Manual via helper | Manual | Full |

## Success Metrics
- All 7 package managers have working packages
- Installation success rate >99%
- Update propagation <24 hours
- Package signatures verified
- Zero dependency conflicts
- Documentation rated helpful >90%

## Repository Hosting

| Package Type | Hosting Solution | Cost | Maintenance |
|-------------|-----------------|------|-------------|
| Homebrew | GitHub (tap) | Free | Low |
| APT | S3 + CloudFront | ~$10/mo | Medium |
| YUM | Copr/OBS | Free | Low |
| Chocolatey | Community repo | Free | Low |
| Scoop | GitHub | Free | Low |
| Snap | Snap Store | Free | Low |
| AUR | AUR | Free | Low |

## References
- [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)
- [Debian New Maintainers' Guide](https://www.debian.org/doc/manuals/maint-guide/)
- [RPM Packaging Guide](https://rpm-packaging-guide.github.io/)
- [Chocolatey Package Creation](https://docs.chocolatey.org/en-us/create/create-packages)
- [Snapcraft Documentation](https://snapcraft.io/docs)
- [AUR Submission Guidelines](https://wiki.archlinux.org/title/AUR_submission_guidelines)
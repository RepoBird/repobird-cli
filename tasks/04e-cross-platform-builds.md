# Task 04e: Cross-Platform Builds

## Overview
Set up cross-platform build system for RepoBird CLI supporting Windows, macOS, and Linux across multiple architectures (x64, ARM, Apple Silicon).

## Background Research

### Cross-Platform Build Best Practices
Based on industry standards:
- **Static Binaries** - Use `CGO_ENABLED=0` for portable, dependency-free binaries
- **GoReleaser** - Automate multi-platform builds and releases
- **Version Embedding** - Inject version info at build time via `-ldflags`
- **Binary Optimization** - Strip debug info with `-s -w` flags
- **Reproducible Builds** - Use `-trimpath` for consistent outputs
- **Platform Testing** - Test on actual target platforms via CI/VMs

### Target Platforms
- **Operating Systems:** Windows, macOS, Linux
- **Architectures:** amd64 (x64), arm64 (Apple Silicon, ARM servers), arm (32-bit ARM)
- **Total Matrix:** 9+ platform combinations

## Implementation Tasks

### 1. GoReleaser Configuration
- [ ] Create `.goreleaser.yml`:
  ```yaml
  project_name: repobird-cli
  
  before:
    hooks:
      - go mod tidy
      - go generate ./...
      - go test ./...
  
  builds:
    - id: repobird
      main: ./cmd/repobird
      binary: repobird
      env:
        - CGO_ENABLED=0
      goos:
        - linux
        - windows
        - darwin
        - freebsd
      goarch:
        - amd64
        - arm64
        - arm
        - 386
      goarm:
        - 6
        - 7
      ignore:
        - goos: darwin
          goarch: 386
        - goos: darwin
          goarch: arm
        - goos: windows
          goarch: arm
      flags:
        - -trimpath
      ldflags:
        - -s -w
        - -X github.com/repobird/repobird-cli/pkg/version.Version={{.Version}}
        - -X github.com/repobird/repobird-cli/pkg/version.Commit={{.Commit}}
        - -X github.com/repobird/repobird-cli/pkg/version.Date={{.Date}}
        - -X github.com/repobird/repobird-cli/pkg/version.BuiltBy=goreleaser
  ```
- [ ] Configure archive formats
- [ ] Set up checksum generation
- [ ] Configure changelog generation
- [ ] Add snapshot builds

### 2. Version Information System
- [ ] Create `pkg/version/version.go`:
  ```go
  package version
  
  var (
      Version = "dev"
      Commit  = "none"
      Date    = "unknown"
      BuiltBy = "unknown"
  )
  
  func GetVersion() string {
      return fmt.Sprintf("repobird version %s (%s) built on %s by %s",
          Version, Commit, Date, BuiltBy)
  }
  ```
- [ ] Implement `version` command
- [ ] Add version to user agent for API calls
- [ ] Include version in bug reports
- [ ] Add build info to crash reports

### 3. Build Matrix Configuration
- [ ] Define supported platforms:
  ```yaml
  platforms:
    - os: linux
      arch: [amd64, arm64, arm/v6, arm/v7, 386]
    - os: darwin
      arch: [amd64, arm64]  # Intel and Apple Silicon
    - os: windows
      arch: [amd64, arm64, 386]
    - os: freebsd
      arch: [amd64, arm64]
  ```
- [ ] Test build combinations
- [ ] Document minimum OS versions
- [ ] Identify platform-specific features
- [ ] Create compatibility matrix

### 4. Binary Optimization
- [ ] Implement build flags:
  ```bash
  # Development build
  go build -o repobird ./cmd/repobird
  
  # Production build
  CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags="-s -w -X main.version=$(git describe --tags)" \
    -o repobird \
    ./cmd/repobird
  ```
- [ ] Measure binary sizes
- [ ] Implement UPX compression (optional)
- [ ] Test startup performance
- [ ] Verify stripped binaries work

### 5. Platform-Specific Code
- [ ] Use build tags for OS-specific code:
  ```go
  // +build windows
  
  package config
  
  func GetConfigPath() string {
      return filepath.Join(os.Getenv("APPDATA"), "repobird")
  }
  ```
- [ ] Abstract file paths
- [ ] Handle line endings (CRLF vs LF)
- [ ] Manage executable permissions
- [ ] Test terminal colors per OS

### 6. Cross-Compilation Scripts
- [ ] Create `scripts/build-all.sh`:
  ```bash
  #!/bin/bash
  
  VERSION=${1:-dev}
  PLATFORMS=(
      "linux/amd64"
      "linux/arm64"
      "darwin/amd64"
      "darwin/arm64"
      "windows/amd64"
      "windows/arm64"
  )
  
  for PLATFORM in "${PLATFORMS[@]}"; do
      OS="${PLATFORM%/*}"
      ARCH="${PLATFORM#*/}"
      OUTPUT="dist/repobird_${OS}_${ARCH}"
      
      if [ "$OS" = "windows" ]; then
          OUTPUT="${OUTPUT}.exe"
      fi
      
      echo "Building $OS/$ARCH..."
      GOOS=$OS GOARCH=$ARCH CGO_ENABLED=0 go build \
          -trimpath \
          -ldflags="-s -w -X main.version=$VERSION" \
          -o "$OUTPUT" \
          ./cmd/repobird
  done
  ```
- [ ] Add build verification
- [ ] Generate checksums
- [ ] Create archives per platform
- [ ] Sign binaries (macOS/Windows)

### 7. Testing Infrastructure
- [ ] Set up CI matrix testing:
  ```yaml
  # .github/workflows/test.yml
  strategy:
    matrix:
      os: [ubuntu-latest, macos-latest, windows-latest]
      arch: [amd64, arm64]
      go: ['1.20', '1.21']
      exclude:
        - os: windows-latest
          arch: arm64  # No GitHub runner yet
  ```
- [ ] Test on real hardware (not just cross-compile)
- [ ] Verify keyring access per platform
- [ ] Test file operations
- [ ] Check terminal compatibility

### 8. Archive Formats
- [ ] Configure platform-appropriate archives:
  ```yaml
  archives:
    - id: default
      format: tar.gz
      format_overrides:
        - goos: windows
          format: zip
      name_template: >-
        {{ .ProjectName }}_
        {{ .Version }}_
        {{ .Os }}_
        {{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}
      files:
        - README.md
        - LICENSE
        - CHANGELOG.md
  ```
- [ ] Include documentation
- [ ] Add completion scripts
- [ ] Bundle man pages
- [ ] Create installer scripts

### 9. Platform Testing Matrix
- [ ] Create test plan:
  | OS | Version | Architecture | Terminal | Tests |
  |----|---------|--------------|----------|-------|
  | Windows | 10, 11 | x64, ARM | CMD, PowerShell, Terminal | Full |
  | macOS | 12, 13, 14 | Intel, M1/M2 | Terminal, iTerm2 | Full |
  | Ubuntu | 20.04, 22.04 | x64, ARM64 | GNOME, WSL | Full |
  | Alpine | 3.18 | x64, ARM64 | sh, bash | Smoke |
  | FreeBSD | 13 | x64 | sh | Smoke |

- [ ] Document known issues per platform
- [ ] Test Unicode support
- [ ] Verify color output
- [ ] Check path handling

### 10. Binary Signing
- [ ] macOS code signing:
  ```bash
  codesign --sign "Developer ID" \
           --options runtime \
           --timestamp \
           dist/repobird_darwin_*
  ```
- [ ] Windows code signing:
  ```bash
  signtool sign /fd SHA256 \
                /tr http://timestamp.digicert.com \
                /td SHA256 \
                dist/repobird_windows_*.exe
  ```
- [ ] Create signing certificates
- [ ] Automate in CI with secrets
- [ ] Verify signatures post-build
- [ ] Document trust requirements

## Build Commands

```bash
# Local development build
make build

# Build for current platform (optimized)
make build-prod

# Build for all platforms
make build-all

# Build with GoReleaser (dry run)
goreleaser build --snapshot --clean

# Create release
goreleaser release --clean

# Build specific platform
GOOS=linux GOARCH=arm64 make build

# Build and test locally
make build test

# Verify binary
file ./build/repobird
ldd ./build/repobird  # Should show "not a dynamic executable"
```

## Size Optimization Targets

| Platform | Target Size | Maximum |
|----------|------------|---------|
| Linux x64 | < 15MB | 20MB |
| macOS Universal | < 30MB | 35MB |
| Windows x64 | < 15MB | 20MB |
| ARM64 | < 14MB | 18MB |

## Performance Targets

| Metric | Target | Maximum |
|--------|--------|---------|
| Startup time | < 50ms | 100ms |
| Memory usage | < 20MB | 50MB |
| Binary load time | < 100ms | 200ms |

## Makefile Targets

```makefile
# Makefile additions
.PHONY: build-all
build-all:
	@echo "Building for all platforms..."
	@./scripts/build-all.sh $(VERSION)

.PHONY: release-dry
release-dry:
	goreleaser release --snapshot --skip-publish --clean

.PHONY: release
release:
	goreleaser release --clean

.PHONY: checksums
checksums:
	cd dist && sha256sum repobird_* > checksums.txt

.PHONY: sign-macos
sign-macos:
	@./scripts/sign-macos.sh

.PHONY: sign-windows
sign-windows:
	@./scripts/sign-windows.sh
```

## Success Metrics
- All 9 platform combinations build successfully
- Binary sizes meet targets
- Zero runtime dependencies (fully static)
- Signed binaries on macOS/Windows
- CI builds complete < 10 minutes
- Version info correctly embedded

## Dependencies
- GoReleaser for automation
- UPX for compression (optional)
- Code signing certificates
- GitHub Actions for CI/CD

## References
- [GoReleaser Documentation](https://goreleaser.com/documentation/)
- [Go Cross-Compilation](https://opensource.com/article/21/1/go-cross-compiling)
- [Static Go Binaries](https://freshman.tech/snippets/go/cross-compile-go-programs/)
- [Code Signing Best Practices](https://developer.apple.com/documentation/security/notarizing_macos_software_before_distribution)
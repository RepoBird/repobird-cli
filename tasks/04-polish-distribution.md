# Phase 4: Polish & Distribution (Week 6)

## Overview
Finalize the CLI tool with comprehensive error handling, testing, documentation, and cross-platform distribution.

## Tasks

### Error Handling & Recovery
- [ ] Implement comprehensive error types
- [ ] Map API status enums to user-friendly messages
- [ ] Handle common errors:
  - "No runs remaining" with tier info
  - "Repository not found or not connected"
  - "Invalid API key"
  - Rate limit errors (future)
- [ ] Implement retry logic with exponential backoff
- [ ] Add 5-second polling for status updates
- [ ] Stop polling when status is DONE or FAILED
- [ ] Handle network interruptions gracefully
- [ ] Display remaining runs from user tier

### Authentication & Security
- [ ] Integrate OS keyring (keyring-go library)
- [ ] Implement secure API key storage (never plain text)
- [ ] Support REPOBIRD_API_KEY environment variable
- [ ] Support REPOBIRD_API_URL for dev override
- [ ] Add API key verification via GET /api/v1/auth/verify
- [ ] Display clear instructions to get API key from dashboard
- [ ] Cache user tier info for offline usage checks
- [ ] Never log or display full API keys
- [ ] Handle "No runs remaining" errors gracefully

### Comprehensive Testing
- [ ] Achieve 80%+ code coverage
- [ ] Add property-based testing
- [ ] Implement integration test suite
- [ ] Add performance benchmarks
- [ ] Create end-to-end test scenarios
- [ ] Add fuzz testing for parsers
- [ ] Implement regression test suite
- [ ] Set up continuous testing in CI

### Documentation
- [ ] Write comprehensive README.md
- [ ] Create detailed API documentation
- [ ] Generate man pages
- [ ] Write user guide with examples
- [ ] Create troubleshooting guide
- [ ] Add configuration reference
- [ ] Document all command flags
- [ ] Create video tutorials/demos
- [ ] Write contributing guidelines
- [ ] Add architecture documentation

### Cross-Platform Builds
- [ ] Set up GoReleaser configuration
- [ ] Configure multi-arch builds
- [ ] Test on Windows (x64, ARM)
- [ ] Test on macOS (Intel, Apple Silicon)
- [ ] Test on Linux (x64, ARM, ARM64)
- [ ] Create build matrix for CI
- [ ] Optimize binary size
- [ ] Add version information to binaries

### Package Manager Integration
- [ ] Create Homebrew formula
- [ ] Build .deb packages for Debian/Ubuntu
- [ ] Build .rpm packages for RHEL/Fedora
- [ ] Create Chocolatey package for Windows
- [ ] Add Scoop manifest
- [ ] Create Snap package
- [ ] Set up AUR package for Arch Linux
- [ ] Configure auto-update mechanism

### GitHub Release Automation
- [ ] Set up GitHub Actions workflow
- [ ] Configure automatic changelog generation
- [ ] Implement semantic versioning
- [ ] Create release notes template
- [ ] Add asset uploading
- [ ] Configure release drafts
- [ ] Set up release notifications
- [ ] Implement rollback mechanism

### Performance Optimization
- [ ] Profile CPU usage
- [ ] Optimize memory allocations
- [ ] Reduce binary size (< 20MB)
- [ ] Optimize startup time (< 100ms)
- [ ] Implement caching strategies
- [ ] Add connection pooling
- [ ] Optimize JSON/YAML parsing
- [ ] Implement lazy loading

### Quality Assurance
- [ ] Set up linting (golangci-lint)
- [ ] Add security scanning (gosec)
- [ ] Implement code formatting (gofmt)
- [ ] Add dependency scanning
- [ ] Set up license compliance checking
- [ ] Create pre-commit hooks
- [ ] Add code review checklist
- [ ] Implement automated code quality gates

## Build & Release Configuration

### GoReleaser Configuration (.goreleaser.yml)
```yaml
project_name: repobird-cli

before:
  hooks:
    - go mod tidy
    - go generate ./...

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
    goarch:
      - amd64
      - arm64
      - arm
    goarm:
      - 6
      - 7
    ldflags:
      - -s -w
      - -X github.com/repobird/repobird-cli/pkg/version.Version={{.Version}}
      - -X github.com/repobird/repobird-cli/pkg/version.Commit={{.Commit}}
      - -X github.com/repobird/repobird-cli/pkg/version.Date={{.Date}}

archives:
  - format: tar.gz
    format_overrides:
      - goos: windows
        format: zip
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

checksum:
  name_template: 'checksums.txt'

snapshot:
  name_template: "{{ incpatch .Version }}-next"

changelog:
  sort: asc
  filters:
    exclude:
      - '^docs:'
      - '^test:'
      - '^chore:'

brews:
  - repository:
      owner: repobird
      name: homebrew-tap
    folder: Formula
    homepage: https://github.com/repobird/repobird-cli
    description: Fast CLI for RepoBird AI agent platform
    license: MIT

nfpms:
  - maintainer: RepoBird Team <team@repobird.ai>
    description: Fast CLI for RepoBird AI agent platform
    homepage: https://github.com/repobird/repobird-cli
    license: MIT
    formats:
      - deb
      - rpm
      - apk
```

### GitHub Actions Workflow (.github/workflows/release.yml)
```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      
      - name: Run tests
        run: go test -v ./...
      
      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v5
        with:
          distribution: goreleaser
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## Testing Strategy

### Unit Test Coverage Goals
- API client: 90%
- Parsers: 95%
- Core logic: 85%
- TUI components: 70%
- Overall: 80%

### Platform Testing Matrix
| OS | Architecture | Terminal | Package Manager |
|----|-------------|----------|-----------------|
| Windows 11 | x64, ARM | Windows Terminal, CMD | Chocolatey, Scoop |
| macOS 14 | Intel, M1/M2 | Terminal, iTerm2 | Homebrew |
| Ubuntu 22.04 | x64, ARM64 | GNOME Terminal | apt |
| Fedora 39 | x64 | Konsole | dnf |
| Arch Linux | x64 | Alacritty | pacman/AUR |

## Documentation Structure

```
docs/
├── README.md                 # Quick start guide
├── installation.md           # Detailed installation
├── configuration.md          # Configuration reference
├── commands/
│   ├── run.md               # Run command docs
│   ├── status.md            # Status command docs
│   ├── config.md            # Config command docs
│   └── auth.md              # Auth command docs
├── formats/
│   ├── json.md              # JSON format guide
│   ├── yaml.md              # YAML format guide
│   ├── toml.md              # TOML format guide
│   └── markdown.md          # Markdown format guide
├── tui/
│   ├── navigation.md        # TUI navigation guide
│   └── keybindings.md       # Keybinding reference
├── api/
│   └── integration.md       # API integration docs
├── troubleshooting.md       # Common issues
└── contributing.md          # Contribution guide
```

## Deliverables

1. Production-ready CLI tool
2. Cross-platform binary distributions
3. Package manager integrations
4. Comprehensive documentation
5. Automated release pipeline
6. Performance benchmarks
7. Security audit report

## Success Criteria

- [ ] Zero critical bugs in production
- [ ] < 0.1% crash rate
- [ ] All performance targets met
- [ ] 80%+ test coverage achieved
- [ ] Successfully published to package managers
- [ ] Documentation rated helpful by users
- [ ] Automated releases working smoothly
- [ ] Cross-platform compatibility verified
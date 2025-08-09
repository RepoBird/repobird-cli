# Task 04g: GitHub Release Automation

## Overview
Implement comprehensive GitHub release automation for RepoBird CLI, including semantic versioning, changelog generation, multi-platform asset building, and automated publishing workflows.

## Background Research

### Release Automation Best Practices
Based on industry standards:
- **Event-driven workflows** - Trigger on tags, merges, or manual dispatch
- **Semantic versioning** - Follow major.minor.patch conventions
- **Automated changelogs** - Generate from commit history and PR metadata
- **Consistent templates** - Standardized release notes format
- **Multi-platform assets** - Build and upload all platform binaries
- **Draft releases** - Allow review before publishing
- **Immutable tags** - Never delete tags for traceability
- **GoReleaser integration** - Consolidated build and release tool

## Implementation Tasks

### 1. Semantic Versioning Setup
- [ ] Implement version management:
  ```go
  // pkg/version/version.go
  package version
  
  var (
      Version   = "dev"
      GitCommit = "unknown"
      BuildDate = "unknown"
      GoVersion = runtime.Version()
  )
  
  type Info struct {
      Version   string `json:"version"`
      GitCommit string `json:"commit"`
      BuildDate string `json:"buildDate"`
      GoVersion string `json:"goVersion"`
  }
  ```
- [ ] Create version bumping script:
  ```bash
  #!/bin/bash
  # scripts/bump-version.sh
  
  TYPE=${1:-patch} # major, minor, patch
  CURRENT=$(git describe --tags --abbrev=0)
  
  # Parse current version
  IFS='.' read -r -a VERSION_PARTS <<< "${CURRENT#v}"
  MAJOR="${VERSION_PARTS[0]}"
  MINOR="${VERSION_PARTS[1]}"
  PATCH="${VERSION_PARTS[2]}"
  
  # Bump version
  case $TYPE in
    major)
      MAJOR=$((MAJOR + 1))
      MINOR=0
      PATCH=0
      ;;
    minor)
      MINOR=$((MINOR + 1))
      PATCH=0
      ;;
    patch)
      PATCH=$((PATCH + 1))
      ;;
  esac
  
  NEW_VERSION="v${MAJOR}.${MINOR}.${PATCH}"
  echo $NEW_VERSION
  ```
- [ ] Add pre-release version support (alpha, beta, rc)
- [ ] Document versioning strategy
- [ ] Create version validation hooks

### 2. GitHub Actions Workflow
- [ ] Create main release workflow `.github/workflows/release.yml`:
  ```yaml
  name: Release
  
  on:
    push:
      tags:
        - 'v*.*.*'
    workflow_dispatch:
      inputs:
        version:
          description: 'Version to release (e.g., v1.2.3)'
          required: true
  
  permissions:
    contents: write
    packages: write
    id-token: write
  
  jobs:
    test:
      runs-on: ubuntu-latest
      steps:
        - uses: actions/checkout@v4
        - uses: actions/setup-go@v5
          with:
            go-version: '1.21'
        - name: Run tests
          run: |
            go test -v -race ./...
            go test -v -tags=integration ./tests/integration
    
    release:
      needs: test
      runs-on: ubuntu-latest
      steps:
        - uses: actions/checkout@v4
          with:
            fetch-depth: 0  # For changelog generation
        
        - uses: actions/setup-go@v5
          with:
            go-version: '1.21'
        
        - name: Install GoReleaser
          uses: goreleaser/goreleaser-action@v5
          with:
            distribution: goreleaser
            version: latest
        
        - name: Run GoReleaser
          env:
            GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
            HOMEBREW_TAP_TOKEN: ${{ secrets.HOMEBREW_TAP_TOKEN }}
            CHOCOLATEY_API_KEY: ${{ secrets.CHOCOLATEY_API_KEY }}
            SNAP_TOKEN: ${{ secrets.SNAP_TOKEN }}
          run: |
            goreleaser release --clean
    
    notify:
      needs: release
      runs-on: ubuntu-latest
      steps:
        - name: Notify Slack
          uses: slackapi/slack-github-action@v1
          with:
            payload: |
              {
                "text": "RepoBird CLI ${{ github.ref_name }} released!"
              }
          env:
            SLACK_WEBHOOK_URL: ${{ secrets.SLACK_WEBHOOK }}
  ```

### 3. GoReleaser Configuration
- [ ] Enhance `.goreleaser.yml`:
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
      ldflags:
        - -s -w
        - -X github.com/repobird/repobird-cli/pkg/version.Version={{.Version}}
        - -X github.com/repobird/repobird-cli/pkg/version.GitCommit={{.Commit}}
        - -X github.com/repobird/repobird-cli/pkg/version.BuildDate={{.Date}}
  
  archives:
    - id: default
      name_template: >-
        {{ .ProjectName }}_
        {{- .Version }}_
        {{- .Os }}_
        {{- .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}
      format_overrides:
        - goos: windows
          format: zip
      files:
        - README.md
        - LICENSE
        - CHANGELOG.md
        - completions/*
  
  checksum:
    name_template: 'checksums.txt'
    algorithm: sha256
  
  signs:
    - artifacts: checksum
      cmd: gpg
      args:
        - "--batch"
        - "--local-user"
        - "{{ .Env.GPG_FINGERPRINT }}"
        - "--output"
        - "${signature}"
        - "--detach-sign"
        - "${artifact}"
  
  snapshot:
    name_template: "{{ incpatch .Version }}-next"
  
  changelog:
    sort: asc
    use: github
    filters:
      exclude:
        - '^docs:'
        - '^test:'
        - '^chore:'
        - Merge pull request
        - Merge branch
    groups:
      - title: '🚀 Features'
        regexp: "^.*feat[(\\w)]*:+.*$"
        order: 0
      - title: '🐛 Bug Fixes'
        regexp: "^.*fix[(\\w)]*:+.*$"
        order: 1
      - title: '⚡ Performance'
        regexp: "^.*perf[(\\w)]*:+.*$"
        order: 2
      - title: Others
        order: 999
  
  release:
    github:
      owner: repobird
      name: repobird-cli
    draft: false
    prerelease: auto
    mode: append
    header: |
      ## RepoBird CLI {{ .Tag }}
      
      Thanks to all contributors!
    footer: |
      ## Installation
      
      ### Homebrew
      ```bash
      brew install repobird/repobird/repobird
      ```
      
      ### Direct Download
      Download the appropriate archive for your platform from the assets below.
      
      **Full Changelog**: https://github.com/repobird/repobird-cli/compare/{{ .PreviousTag }}...{{ .Tag }}
  
  announce:
    twitter:
      enabled: true
      message_template: '🎉 RepoBird CLI {{ .Tag }} is out! {{ .ReleaseURL }}'
    slack:
      enabled: true
      channel: '#releases'
      message_template: 'RepoBird CLI {{ .Tag }} released: {{ .ReleaseURL }}'
  ```

### 4. Changelog Generation
- [ ] Create `.github/release.yml`:
  ```yaml
  changelog:
    exclude:
      labels:
        - ignore-for-release
        - skip-changelog
      authors:
        - dependabot
        - github-actions
    categories:
      - title: 🚀 Features
        labels:
          - feature
          - enhancement
      - title: 🐛 Bug Fixes
        labels:
          - bug
          - fix
      - title: ⚡ Performance
        labels:
          - performance
      - title: 📚 Documentation
        labels:
          - documentation
      - title: 🔒 Security
        labels:
          - security
      - title: 🧰 Maintenance
        labels:
          - chore
          - dependencies
      - title: Other Changes
        labels:
          - "*"
  ```
- [ ] Implement conventional commits
- [ ] Add commit lint enforcement
- [ ] Create changelog update script
- [ ] Generate migration guides for breaking changes

### 5. Draft Release Workflow
- [ ] Create draft release workflow:
  ```yaml
  name: Draft Release
  
  on:
    workflow_dispatch:
      inputs:
        version:
          description: 'Version to draft'
          required: true
  
  jobs:
    draft:
      runs-on: ubuntu-latest
      steps:
        - uses: actions/checkout@v4
        
        - name: Create Draft Release
          uses: goreleaser/goreleaser-action@v5
          with:
            args: release --draft --skip-validate
          env:
            GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
  ```
- [ ] Add review checklist
- [ ] Implement approval workflow
- [ ] Create release candidate process

### 6. Asset Management
- [ ] Configure asset uploads:
  ```yaml
  # In .goreleaser.yml
  release:
    extra_files:
      - glob: ./dist/*.deb
      - glob: ./dist/*.rpm
      - glob: ./dist/*.msi
      - glob: ./completions/*
      - glob: ./man/*
  ```
- [ ] Generate installation scripts
- [ ] Create universal installer
- [ ] Add signature files
- [ ] Generate SBOM (Software Bill of Materials)

### 7. Rollback Mechanism
- [ ] Create rollback workflow:
  ```yaml
  name: Rollback Release
  
  on:
    workflow_dispatch:
      inputs:
        version:
          description: 'Version to rollback to'
          required: true
  
  jobs:
    rollback:
      runs-on: ubuntu-latest
      steps:
        - name: Create Rollback Release
          run: |
            # Download previous release assets
            gh release download ${{ inputs.version }} --dir ./assets
            
            # Create new release with rollback notice
            ROLLBACK_VERSION="${{ inputs.version }}-rollback-$(date +%Y%m%d)"
            gh release create $ROLLBACK_VERSION \
              --title "Rollback to ${{ inputs.version }}" \
              --notes "This is a rollback release to version ${{ inputs.version }}" \
              ./assets/*
  ```
- [ ] Document rollback procedure
- [ ] Test rollback scenarios
- [ ] Create hotfix workflow

### 8. Release Notes Template
- [ ] Create release notes template:
  ```markdown
  # RepoBird CLI v{{ .Version }}
  
  Released: {{ .Date }}
  
  ## 🎯 Highlights
  - Major feature or improvement
  - Important fix or change
  
  ## 📦 What's Changed
  {{ .Changelog }}
  
  ## 📊 Statistics
  - {{ .Contributors }} contributors
  - {{ .Commits }} commits
  - {{ .FilesChanged }} files changed
  
  ## 🔧 Installation & Upgrade
  
  ### New Users
  See our [installation guide](https://github.com/repobird/repobird-cli#installation)
  
  ### Existing Users
  - Homebrew: `brew upgrade repobird`
  - Apt: `sudo apt update && sudo apt upgrade repobird`
  - Manual: Download from assets below
  
  ## ⚠️ Breaking Changes
  - List any breaking changes
  
  ## 🙏 Acknowledgments
  Thanks to all our contributors!
  
  ## 📝 Full Changelog
  https://github.com/repobird/repobird-cli/compare/{{ .PreviousTag }}...{{ .Tag }}
  ```

### 9. Notification System
- [ ] Configure notifications:
  ```yaml
  # .github/workflows/notify-release.yml
  name: Release Notifications
  
  on:
    release:
      types: [published]
  
  jobs:
    notify:
      runs-on: ubuntu-latest
      steps:
        - name: Send Discord Notification
        - name: Send Slack Notification
        - name: Send Email to Subscribers
        - name: Update Status Page
        - name: Post to Twitter/X
        - name: Update Documentation Site
  ```
- [ ] Set up webhook integrations
- [ ] Create release announcement template
- [ ] Configure RSS feed

### 10. Release Validation
- [ ] Create validation workflow:
  ```yaml
  name: Validate Release
  
  on:
    release:
      types: [published]
  
  jobs:
    validate:
      strategy:
        matrix:
          os: [ubuntu-latest, macos-latest, windows-latest]
      runs-on: ${{ matrix.os }}
      steps:
        - name: Download Release
          run: |
            gh release download ${{ github.event.release.tag_name }}
        
        - name: Verify Checksums
        - name: Test Installation
        - name: Run Smoke Tests
        - name: Report Status
  ```

## Release Process

### Standard Release
1. Create and push version tag: `git tag v1.2.3 && git push origin v1.2.3`
2. GitHub Actions triggers automatically
3. Tests run, build artifacts created
4. GoReleaser generates changelog and creates release
5. Assets uploaded, notifications sent
6. Package managers updated automatically

### Hotfix Release
1. Create hotfix branch from tag
2. Apply fix and test
3. Tag with patch version
4. Follow standard release process

### Major Release
1. Create release branch
2. Update documentation for breaking changes
3. Create migration guide
4. Test extensively
5. Create release candidate first
6. After validation, create final release

## Success Metrics
- Release creation time < 15 minutes
- Zero manual steps required
- All platform assets generated
- Changelog accuracy > 95%
- Package manager updates < 1 hour
- Rollback capability tested monthly

## Monitoring & Alerts

| Check | Frequency | Alert Channel |
|-------|-----------|---------------|
| Release build status | On trigger | Slack, Email |
| Asset upload verification | Post-release | GitHub Issues |
| Package manager sync | Hourly | Dashboard |
| Download statistics | Daily | Analytics |
| User feedback | Continuous | Support channel |

## Dependencies
- GoReleaser for build orchestration
- GitHub Actions for automation
- GPG for signing
- Various package manager CLIs

## References
- [GoReleaser Documentation](https://goreleaser.com/documentation/)
- [GitHub Actions for Releases](https://docs.github.com/en/actions/creating-actions/releasing-and-maintaining-actions)
- [Semantic Versioning](https://semver.org/)
- [Conventional Commits](https://www.conventionalcommits.org/)
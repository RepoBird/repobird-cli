# RepoBird CLI Makefile
# Common commands for development, testing, and building

# Load .env file if it exists
-include .env
export

# Variables
BINARY_NAME=repobird
MAIN_PATH=./cmd/repobird
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-s -w -X github.com/repobird/repobird-cli/pkg/version.Version=$(VERSION) -X github.com/repobird/repobird-cli/pkg/version.GitCommit=$(COMMIT) -X github.com/repobird/repobird-cli/pkg/version.BuildDate=$(DATE)"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint
GOVET=$(GOCMD) vet

# Build directories
BUILD_DIR=build
DIST_DIR=dist

# Platforms for cross-compilation
PLATFORMS=darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64

.PHONY: all build clean test coverage fmt lint vet deps run install uninstall help

# Default target
all: test build

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^##' Makefile | sed 's/## /  /'

## init: Initialize project and download dependencies
init:
	$(GOMOD) init github.com/repobird/repobird-cli || true
	$(GOMOD) tidy
	$(GOMOD) download
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/goreleaser/goreleaser@latest

## deps: Download and verify dependencies
deps:
	$(GOMOD) download
	$(GOMOD) verify

## build: Build the binary for current platform (without CGO for portability)
build:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## build-cgo: Build the binary with CGO enabled (better clipboard support)
build-cgo:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete with CGO: $(BUILD_DIR)/$(BINARY_NAME)"

## build-all: Build binaries for all platforms
build-all:
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} \
		CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) \
		-o $(BUILD_DIR)/$(BINARY_NAME)-$${platform%/*}-$${platform#*/}$${ext} \
		$(MAIN_PATH); \
		echo "Built: $(BUILD_DIR)/$(BINARY_NAME)-$${platform%/*}-$${platform#*/}"; \
	done

## run: Run the application
run:
	$(GOCMD) run $(LDFLAGS) $(MAIN_PATH)

## tui: Build and run the TUI interface (portable build)
tui: build
	./$(BUILD_DIR)/$(BINARY_NAME) tui

## tui-cgo: Build and run the TUI interface with CGO (better clipboard support)
tui-cgo: build-cgo
	./$(BUILD_DIR)/$(BINARY_NAME) tui

## status: Build and run status command
status: build
	./$(BUILD_DIR)/$(BINARY_NAME) status

## status-debug: Build and run status command with debug output
status-debug: build
	./$(BUILD_DIR)/$(BINARY_NAME) status --debug

## install: Install repobird to ~/.local/bin with rb alias
install: build
	@echo "Installing repobird to ~/.local/bin..."
	@mkdir -p ~/.local/bin
	@cp $(BUILD_DIR)/$(BINARY_NAME) ~/.local/bin/$(BINARY_NAME)
	@ln -sf ~/.local/bin/$(BINARY_NAME) ~/.local/bin/rb
	@echo "✓ Installation complete!"
	@echo ""
	@echo "Make sure ~/.local/bin is in your PATH. Add this to your ~/.zshrc or ~/.bashrc:"
	@echo '  export PATH="$$HOME/.local/bin:$$PATH"'
	@echo ""
	@echo "You can now use 'repobird' or 'rb' commands"

## install-global: Install repobird globally to /usr/local/bin (requires sudo)
install-global: build
	@echo "Installing repobird globally to /usr/local/bin (requires sudo)..."
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	sudo ln -sf /usr/local/bin/$(BINARY_NAME) /usr/local/bin/rb
	@echo "✓ Global installation complete!"
	@echo "You can now use 'repobird' or 'rb' commands from anywhere"

## uninstall: Remove repobird from ~/.local/bin
uninstall:
	@echo "Uninstalling repobird from ~/.local/bin..."
	@rm -f ~/.local/bin/$(BINARY_NAME) ~/.local/bin/rb
	@echo "✓ Uninstall complete"

## uninstall-global: Uninstall repobird globally from /usr/local/bin (requires sudo)
uninstall-global:
	@echo "Uninstalling repobird globally from /usr/local/bin (requires sudo)..."
	sudo rm -f /usr/local/bin/$(BINARY_NAME) /usr/local/bin/rb
	@echo "✓ Global uninstall complete"

## clean: Remove build artifacts
clean:
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR) $(DIST_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

## test: Run all tests
test:
	$(GOTEST) -v -race -timeout 30s ./...

## test-unit: Run unit tests only
test-unit:
	$(GOTEST) -v -race -timeout 30s -short ./internal/... ./pkg/...

## test-integration: Run integration tests only
test-integration:
	$(GOTEST) -v -race -timeout 2m -tags=integration ./tests/integration/...

## test-commands: Run command tests only
test-commands:
	$(GOTEST) -v -race -timeout 30s ./internal/commands/...

## test-api: Run API tests only
test-api:
	$(GOTEST) -v -race -timeout 30s ./internal/api/...

## test-config: Run config tests only
test-config:
	$(GOTEST) -v -race -timeout 30s ./internal/config/...

## test-short: Run short tests only
test-short:
	$(GOTEST) -v -short ./...

## test-verbose: Run tests with verbose output
test-verbose:
	$(GOTEST) -v -race -timeout 30s -count=1 ./...

## coverage: Run tests with coverage
coverage:
	$(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"
	@$(GOCMD) tool cover -func=coverage.out | grep total | awk '{print "Total coverage: " $$3}'

## coverage-unit: Generate coverage for unit tests only
coverage-unit:
	$(GOTEST) -v -race -coverprofile=coverage-unit.out -covermode=atomic -short ./internal/... ./pkg/...
	$(GOCMD) tool cover -html=coverage-unit.out -o coverage-unit.html
	@echo "Unit test coverage report: coverage-unit.html"
	@$(GOCMD) tool cover -func=coverage-unit.out | grep total | awk '{print "Unit test coverage: " $$3}'

## benchmark: Run benchmarks
benchmark:
	$(GOTEST) -bench=. -benchmem ./...

## benchmark-api: Run API benchmarks only
benchmark-api:
	$(GOTEST) -bench=. -benchmem ./internal/api/...

## fuzz: Run fuzz tests
fuzz:
	$(GOTEST) -fuzz=. -fuzztime=30s ./internal/models/...
	$(GOTEST) -fuzz=. -fuzztime=30s ./pkg/utils/...

## fmt: Format code
fmt:
	@$(GOFMT) -s -w .
	@echo "Code formatted"

## fmt-check: Check if code is formatted
fmt-check:
	@test -z "$$($(GOFMT) -l .)" || (echo "Please run 'make fmt' to format code"; exit 1)

## lint: Run linter
lint:
	@which $(GOLINT) > /dev/null || (echo "golangci-lint not installed. Run 'make init'"; exit 1)
	$(GOLINT) run --timeout 5m ./...

## lint-fix: Run linter and fix issues where possible
lint-fix:
	@which $(GOLINT) > /dev/null || (echo "golangci-lint not installed. Run 'make init'"; exit 1)
	$(GOLINT) run --fix --timeout 5m ./...

## vet: Run go vet
vet:
	$(GOVET) ./...

## security: Run security checks
security:
	@which gosec > /dev/null || go install github.com/securego/gosec/v2/cmd/gosec@latest
	gosec -fmt json -out security-report.json ./...
	@echo "Security report: security-report.json"

## mod-tidy: Tidy and verify go modules
mod-tidy:
	$(GOMOD) tidy
	$(GOMOD) verify

## check: Run all checks (fmt, vet, lint, test)
check: fmt-check vet lint test

## ci: Run CI pipeline locally
ci: deps check coverage security

## release-dry: Dry run of release process
release-dry:
	@which goreleaser > /dev/null || (echo "goreleaser not installed. Run 'make init'"; exit 1)
	goreleaser release --snapshot --skip-publish --clean

## release: Create a new release (requires tag)
release:
	@which goreleaser > /dev/null || (echo "goreleaser not installed. Run 'make init'"; exit 1)
	goreleaser release --clean

## docker-build: Build Docker image
docker-build:
	docker build -t repobird-cli:$(VERSION) .

## docker-run: Run Docker container
docker-run:
	docker run --rm -it repobird-cli:$(VERSION)

## dev: Start development with file watching
dev:
	@which air > /dev/null || go install github.com/cosmtrek/air@latest
	air -c .air.toml

## docs: Generate documentation
docs:
	@which godoc > /dev/null || go install golang.org/x/tools/cmd/godoc@latest
	@echo "Starting godoc server on http://localhost:6060"
	godoc -http=:6060

## version: Show version information
version:
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"
	@echo "Date:    $(DATE)"

# Shortcuts
b: build
t: test
c: clean
r: run
i: install
# RepoBird CLI Makefile
# Common commands for development, testing, and building

# Variables
BINARY_NAME=repobird
MAIN_PATH=./cmd/repobird
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-s -w -X github.com/repobird/repobird-cli/pkg/version.Version=$(VERSION) -X github.com/repobird/repobird-cli/pkg/version.Commit=$(COMMIT) -X github.com/repobird/repobird-cli/pkg/version.Date=$(DATE)"

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

## build: Build the binary for current platform
build:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

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

## install: Install the binary to $GOPATH/bin
install: build
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/
	@echo "Installed to $(GOPATH)/bin/$(BINARY_NAME)"

## uninstall: Remove the binary from $GOPATH/bin
uninstall:
	@rm -f $(GOPATH)/bin/$(BINARY_NAME)
	@echo "Uninstalled from $(GOPATH)/bin/$(BINARY_NAME)"

## clean: Remove build artifacts
clean:
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR) $(DIST_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

## test: Run all tests
test:
	$(GOTEST) -v -race -timeout 30s ./...

## test-short: Run short tests only
test-short:
	$(GOTEST) -v -short ./...

## coverage: Run tests with coverage
coverage:
	$(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"
	@$(GOCMD) tool cover -func=coverage.out | grep total | awk '{print "Total coverage: " $$3}'

## benchmark: Run benchmarks
benchmark:
	$(GOTEST) -bench=. -benchmem ./...

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
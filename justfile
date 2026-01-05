# Crook Justfile - Task automation for development

# Variables
VERSION := `git describe --tags --always --dirty 2>/dev/null || echo "dev"`
COMMIT := `git rev-parse --short HEAD 2>/dev/null || echo "unknown"`
BUILD_DATE := `date -u +"%Y-%m-%dT%H:%M:%SZ"`
LDFLAGS := "-X main.version=" + VERSION + " -X main.commit=" + COMMIT + " -X main.buildDate=" + BUILD_DATE

# Default recipe
default:
    @just --list

# Build the crook binary with version information
build:
    go build -ldflags '{{LDFLAGS}}' -o bin/crook ./cmd/crook

# Build for release (stripped binary)
build-release:
    go build -ldflags '{{LDFLAGS}} -s -w' -o bin/crook ./cmd/crook

# Run all tests
test:
    go test ./...

# Run tests with verbose output
test-verbose:
    go test -v ./...

# Run tests with race detection
test-race:
    go test -race ./...

# Run linter
lint:
    golangci-lint run

# Run linter and fix auto-fixable issues
lint-fix:
    golangci-lint run --fix

# Format code
fmt:
    go fmt ./...
    goimports -w .

# Verify (lint + test + build)
verify: lint test build

# Clean build artifacts
clean:
    rm -rf bin/
    go clean -cache

# Install locally
install:
    go install -ldflags '{{LDFLAGS}}' ./cmd/crook

# Generate shell completions
completions:
    mkdir -p completions
    go run ./cmd/crook completion bash > completions/crook.bash
    go run ./cmd/crook completion zsh > completions/_crook
    go run ./cmd/crook completion fish > completions/crook.fish

# Show version info
version:
    @echo "Version: {{VERSION}}"
    @echo "Commit: {{COMMIT}}"
    @echo "Build Date: {{BUILD_DATE}}"

# Run the application (for testing)
run *ARGS:
    go run -ldflags '{{LDFLAGS}}' ./cmd/crook {{ARGS}}

# Cross-compilation targets
# Build for Linux AMD64
build-linux-amd64:
    GOOS=linux GOARCH=amd64 go build -ldflags '{{LDFLAGS}} -s -w' -o bin/crook-linux-amd64 ./cmd/crook

# Build for Linux ARM64
build-linux-arm64:
    GOOS=linux GOARCH=arm64 go build -ldflags '{{LDFLAGS}} -s -w' -o bin/crook-linux-arm64 ./cmd/crook

# Build for macOS AMD64
build-darwin-amd64:
    GOOS=darwin GOARCH=amd64 go build -ldflags '{{LDFLAGS}} -s -w' -o bin/crook-darwin-amd64 ./cmd/crook

# Build for macOS ARM64 (Apple Silicon)
build-darwin-arm64:
    GOOS=darwin GOARCH=arm64 go build -ldflags '{{LDFLAGS}} -s -w' -o bin/crook-darwin-arm64 ./cmd/crook

# Build all release binaries
build-all: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64
    @echo "Built all release binaries in bin/"
    @ls -la bin/

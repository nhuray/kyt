.PHONY: build build-ky build-k8s-diff test test-verbose test-coverage lint install clean help

# Binary names
KY_BINARY=ky
KY_PATH=bin/$(KY_BINARY)
LEGACY_BINARY=k8s-diff
LEGACY_PATH=bin/$(LEGACY_BINARY)

# Default binary (new ky tool)
BINARY_NAME=$(KY_BINARY)
BINARY_PATH=$(KY_PATH)

# Version information
VERSION ?= dev
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

build: build-ky ## Build the ky binary (default)

build-ky: ## Build the ky binary
	@echo "Building $(KY_BINARY)..."
	@mkdir -p bin
	@go build $(LDFLAGS) -o $(KY_PATH) ./cmd/$(KY_BINARY)
	@echo "✓ Binary built: $(KY_PATH)"

build-k8s-diff: ## Build the legacy k8s-diff binary
	@echo "Building $(LEGACY_BINARY)..."
	@mkdir -p bin
	@go build $(LDFLAGS) -o $(LEGACY_PATH) ./cmd/$(LEGACY_BINARY)
	@echo "✓ Binary built: $(LEGACY_PATH)"

test: ## Run tests
	@echo "Running tests..."
	@go test ./...

test-verbose: ## Run tests with verbose output
	@echo "Running tests with verbose output..."
	@go test -v ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"

lint: ## Run linter (requires golangci-lint)
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "⚠ golangci-lint not installed. Install it from: https://golangci-lint.run/usage/install/"; \
		exit 1; \
	fi

install: build-ky ## Install ky binary to GOPATH/bin
	@echo "Installing $(KY_BINARY)..."
	@go install $(LDFLAGS) ./cmd/$(KY_BINARY)
	@echo "✓ Installed: $(shell which $(KY_BINARY) 2>/dev/null || echo 'ky')"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/ dist/ coverage.out coverage.html
	@echo "✓ Cleaned"

run: build-ky ## Build and run ky diff with example arguments
	@echo "Running $(KY_BINARY) diff..."
	@$(KY_PATH) diff examples/manifests/basic examples/manifests/multi-doc

run-json: build-ky ## Build and run ky diff with JSON output
	@echo "Running $(KY_BINARY) diff with JSON output..."
	@$(KY_PATH) diff -o json examples/manifests/basic examples/manifests/multi-doc

run-lint: build-ky ## Build and run ky lint
	@echo "Running $(KY_BINARY) lint..."
	@$(KY_PATH) lint examples/manifests/basic

.DEFAULT_GOAL := help

.PHONY: build test test-verbose test-coverage lint install clean help

# Binary name
BINARY=kyt
BINARY_PATH=bin/$(BINARY)

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

build: ## Build the kyt binary
	@echo "Building $(BINARY)..."
	@mkdir -p bin
	@go build $(LDFLAGS) -o $(BINARY_PATH) ./cmd/$(BINARY)
	@echo "✓ Binary built: $(BINARY_PATH)"

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

install: build ## Install kyt binary to GOPATH/bin
	@echo "Installing $(BINARY)..."
	@go install $(LDFLAGS) ./cmd/$(BINARY)
	@echo "✓ Installed: $(shell which $(BINARY) 2>/dev/null || echo 'kyt')"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/ dist/ coverage.out coverage.html
	@echo "✓ Cleaned"

run: build ## Build and run kyt diff with example arguments
	@echo "Running $(BINARY) diff..."
	@$(BINARY_PATH) diff examples/manifests/basic examples/manifests/multi-doc

run-json: build ## Build and run kyt diff with JSON output
	@echo "Running $(BINARY) diff with JSON output..."
	@$(BINARY_PATH) diff -o json examples/manifests/basic examples/manifests/multi-doc

run-fmt: build ## Build and run kyt fmt
	@echo "Running $(BINARY) fmt..."
	@$(BINARY_PATH) fmt examples/manifests/basic

.DEFAULT_GOAL := help

.PHONY: build test test-verbose test-coverage lint install clean help

# Binary name
BINARY_NAME=k8s-diff
BINARY_PATH=bin/$(BINARY_NAME)

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

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@go build $(LDFLAGS) -o $(BINARY_PATH) ./cmd/$(BINARY_NAME)
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

install: build ## Install binary to GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	@go install $(LDFLAGS) ./cmd/$(BINARY_NAME)
	@echo "✓ Installed: $(shell which $(BINARY_NAME))"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/ dist/ coverage.out coverage.html
	@echo "✓ Cleaned"

run: build ## Build and run with example arguments
	@echo "Running $(BINARY_NAME)..."
	@$(BINARY_PATH) examples/manifests/basic examples/manifests/multi-doc

run-json: build ## Build and run with JSON output
	@echo "Running $(BINARY_NAME) with JSON output..."
	@$(BINARY_PATH) -o json examples/manifests/basic examples/manifests/multi-doc

.DEFAULT_GOAL := help

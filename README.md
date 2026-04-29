# k8s-diff

A CLI tool for comparing Kubernetes manifests with smart ignore rules.

## Overview

`k8s-diff` is a Go-based tool that intelligently compares Kubernetes manifests by applying configurable ignore rules before computing differences. It uses [ArgoCD](https://argo-cd.readthedocs.io/)-compatible ignore rules (JSON Pointers and JQ path expressions) and leverages [difftastic](https://difftastic.wilfred.me.uk/) for beautiful structural diffs.

**Key Features:**

- 🎯 **ArgoCD-Compatible Rules**: Uses the same ignore syntax as ArgoCD's `ignoreDifferences`
- 🔍 **JSON Pointer Support**: RFC 6901 compliant JSON Pointers for precise field targeting
- 🎨 **JQ Path Expressions**: Powerful filtering with wildcards and conditionals
- 📊 **Multiple Output Formats**: CLI (with colors), JSON
- 🎯 **Smart Normalization**: Sorts keys, removes managed fields, applies ignore rules
- ⚡ **Fast & Reliable**: Written in Go with 52 passing tests

## Use Cases

- **Validate Helm vs Kustomize migrations**: Compare rendered manifests while ignoring expected differences (labels, annotations)
- **Detect configuration drift**: Compare desired state with actual cluster state
- **CI/CD validation**: Ensure manifest changes are intentional
- **Release validation**: Compare what's deployed vs what should be deployed

## Status

✅ **Core functionality complete!** - See [docs/PLAN.md](docs/PLAN.md) for implementation details.

- ✅ Phases 1-8 complete (Setup, Parsing, Config, Normalization, Diff, Output, CLI, Testing)
- 🔨 Phase 9 in progress (Documentation)
- 📦 Phase 10 planned (Build & Release)

## Quick Start

```bash
# Build from source
git clone https://github.com/nhuray/k8s-diff.git
cd k8s-diff
go build -o bin/k8s-diff ./cmd/k8s-diff

# Compare two manifest files
./bin/k8s-diff source.yaml target.yaml

# Compare directories
./bin/k8s-diff ./kustomize-output ./helm-output

# Use custom config
./bin/k8s-diff source.yaml target.yaml --config my-rules.yaml

# Output JSON for CI/CD
./bin/k8s-diff source.yaml target.yaml --output json

# Show identical resources
./bin/k8s-diff source.yaml target.yaml --show-identical

# Verbose mode for debugging
./bin/k8s-diff -v source.yaml target.yaml

# Use unified diff instead of difftastic
./bin/k8s-diff --diff-tool diff source.yaml target.yaml

# Change difftastic display mode
./bin/k8s-diff --display inline source.yaml target.yaml
```

### Difftastic Integration

The tool automatically detects and uses [difftastic](https://difftastic.wilfred.me.uk/) if available, providing beautiful syntax-aware structural diffs. If difftastic is not found, it gracefully falls back to standard unified diff.

**Install difftastic:**

```bash
# macOS
brew install difftastic

# Other platforms: see https://difftastic.wilfred.me.uk/installation.html
```

### Exit Codes

- `0` - No differences found
- `1` - Differences detected
- `2` - Error (invalid YAML, missing files, etc.)

## Configuration Example

```yaml
# .k8s-diff.yaml
ignoreDifferences:
  # Ignore all labels and annotations
  - group: ""
    kind: "*"
    jsonPointers:
      - /metadata/labels
      - /metadata/annotations

  # Ignore Istio sidecar containers
  - group: "apps"
    kind: "Deployment"
    jqPathExpressions:
      - .spec.template.spec.containers[] | select(.name == "istio-proxy")
```

## Installation

### From Source

```bash
git clone https://github.com/nhuray/k8s-diff.git
cd k8s-diff

# Using Make (recommended)
make build

# Or directly with Go
go build -o bin/k8s-diff ./cmd/k8s-diff

# Optional: Install to your PATH
make install
# Or manually: cp bin/k8s-diff /usr/local/bin/
```

### Using Go Install

```bash
go install github.com/nhuray/k8s-diff/cmd/k8s-diff@latest
```

### Binary Releases

_(Coming soon)_ Download pre-built binaries from [GitHub Releases](https://github.com/nhuray/k8s-diff/releases)

## Documentation

- [Implementation Plan](docs/PLAN.md) - Detailed development roadmap with progress
- [Example Configs](examples/) - Sample configurations and manifests
- Configuration Guide - _(Coming soon in docs/configuration.md)_
- Usage Examples - _(Coming soon in docs/usage.md)_

## Testing

The project has comprehensive test coverage:

- **52 tests total** (45 unit + 7 integration)
- All tests passing
- Covers: config loading, manifest parsing, normalization, diffing, output formatting, CLI

Run tests:

```bash
# Run all tests
make test

# Run with verbose output
make test-verbose

# Run with coverage report
make test-coverage

# Or use go directly
go test ./...
go test -v ./...
go test -coverprofile=coverage.out ./...
```

## Development

```bash
# Show all available make targets
make help

# Build the binary
make build

# Run tests
make test

# Clean build artifacts
make clean

# Run with example manifests
make run
make run-json
```

## Dependencies

**Runtime:**

- [difftastic](https://difftastic.wilfred.me.uk/) - Structural diff tool (optional but recommended)
- [diff2html-cli](https://diff2html.xyz/) - For HTML report generation (optional)

**Go Libraries:**

- [ArgoCD](https://github.com/argoproj/argo-cd) - Ignore rules engine
- [gojq](https://github.com/itchyny/gojq) - JQ implementation in Go
- [cobra](https://github.com/spf13/cobra) - CLI framework

## Inspiration

This project is inspired by:

- [ArgoCD's diff customization](https://argo-cd.readthedocs.io/en/stable/user-guide/diff-customization/)
- [kubectl diff](https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands#diff)
- [dyff](https://github.com/homeport/dyff) - YAML diff tool
- [helm-drift](https://github.com/nikhilsbhat/helm-drift) - Helm drift detection

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) _(coming soon)_ for guidelines.

## License

MIT License - See [LICENSE](LICENSE) for details.

## Author

Created by Nicolas Huray ([@nhuray](https://github.com/nhuray))

---

**Status:** ✅ Core functionality complete! Phases 1-8 done. Documentation and release automation in progress.

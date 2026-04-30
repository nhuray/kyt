# kyt (Kubernetes YAML Toolkit)

A powerful CLI tool for formatting and comparing Kubernetes manifests with intelligent ignore rules.

## Overview

`kyt` is a Go-based toolkit designed to solve a common problem in Kubernetes workflows: **comparing and managing YAML manifests while ignoring irrelevant differences**.

When working with tools like Helm, Kustomize, or ArgoCD, you often need to compare manifests to detect drift, validate changes, or ensure consistency. However, raw YAML diffs are noisy - filled with differences in field ordering, timestamps, managed fields, and other metadata that doesn't matter for your use case.

**kyt solves this by:**

1. **Formatting manifests** - Normalizes YAML by sorting keys, removing default/managed fields, and applying custom ignore rules
2. **Smart comparison** - Compares only what matters by using ArgoCD-compatible ignore rules (JSON Pointers and JQ expressions)
3. **Beautiful diffs** - Leverages [difftastic](https://difftastic.wilfred.me.uk/) for structural, syntax-aware diffs that are easy to read

**Key Features:**

- 🎯 **ArgoCD-Compatible Rules**: Uses the same ignore syntax as ArgoCD's `ignoreDifferences`
- 🔍 **JSON Pointer Support**: RFC 6901 compliant JSON Pointers for precise field targeting
- 🎨 **JQ Path Expressions**: Powerful filtering with wildcards and conditionals
- 📊 **Multiple Output Formats**: CLI (with colors), JSON
- 🎯 **Smart Normalization**: Sorts keys, removes managed fields, applies ignore rules
- 🔧 **Lint & Format**: Format manifests with `ky fmt`
- 🔀 **Pipe-friendly**: Works seamlessly with kubectl, kustomize, helm
- 🤖 **Smart Similarity Matching**: Automatically detects renamed resources
- ⚡ **Fast & Reliable**: Written in Go with 60+ passing tests

## Use Cases

- **Compare Helm vs Kustomize outputs**: Validate migrations by comparing rendered manifests while ignoring expected differences (field order, formatting, etc.)
- **Detect configuration drift**: Compare desired state (Git) with actual cluster state (`kubectl get`), ignoring dynamic fields like timestamps and resource versions
- **CI/CD validation**: Ensure manifest changes are intentional by comparing PR changes against production, with rules to ignore acceptable differences
- **Pre-deployment validation**: Compare what's currently deployed vs what will be deployed, filtering out noise
- **Format and standardize**: Clean up YAML files by sorting keys, removing managed fields, and applying consistent formatting

## Status

✅ **Core functionality complete!** - See [docs/PLAN.md](docs/PLAN.md) for implementation details.

- ✅ Phases 1-8.6 complete (Setup, Parsing, Config, Normalization, Diff, Output, CLI, Testing, Tool Refactoring)
- 🔨 Phase 9 in progress (Documentation)
- 📦 Phase 10 planned (Build & Release)

## Quick Start

```bash
# Build from source
git clone https://github.com/nhuray/k8s-diff.git
cd k8s-diff
make build

# Compare two manifest files
./bin/kyt diff source.yaml target.yaml

# Compare directories
./bin/kyt diff ./kustomize-output ./helm-output

# Normalize a manifest file
./bin/kyt fmt deployment.yaml

# Pipe manifests through ky
kustomize build . | kyt fmt | kubectl apply -f -

# Use custom config
./bin/kyt diff -c .kyt.yaml source.yaml target.yaml

# Output JSON for CI/CD
./bin/kyt diff -o json source.yaml target.yaml
```

## Commands

### `ky diff` - Compare manifests

Compare two Kubernetes manifest files or directories with smart ignore rules.

```bash
# Basic comparison
kyt diff source.yaml target.yaml

# Compare directories
kyt diff ./helm-output ./kustomize-output

# Show identical resources
kyt diff --show-identical source.yaml target.yaml

# JSON output for CI/CD
kyt diff -o json source.yaml target.yaml

# Verbose mode for debugging
kyt diff -v source.yaml target.yaml

# Disable similarity matching (exact name match only)
kyt diff --exact-match source.yaml target.yaml

# Use unified diff instead of difftastic
kyt diff --diff-tool diff source.yaml target.yaml

# Change difftastic display mode
kyt diff --display inline source.yaml target.yaml
```

**Exit Codes:**
- `0` - No differences found
- `1` - Differences detected
- `2` - Error (invalid YAML, missing files, etc.)

### `ky fmt` - Format manifests

Format Kubernetes manifests by applying transformations like sorting keys and arrays.

```bash
# Format a file to stdout
kyt fmt deployment.yaml

# Format a directory to stdout
kyt fmt ./manifests

# Format and write back to source files
kyt fmt -w ./manifests

# Format from stdin
cat deployment.yaml | kyt fmt

# Chain with other tools
kustomize build . | kyt fmt | kubectl apply -f -
helm template . | kyt fmt > formatted.yaml
```

### `ky version` - Version information

```bash
kyt version
```

## Configuration

The tool searches for `.kyt.yaml` (or legacy `.k8s-diff.yaml`) in the current directory and parent directories.

```yaml
# .kyt.yaml
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

  # Ignore specific fields in Services
  - group: ""
    kind: "Service"
    jsonPointers:
      - /spec/clusterIP
      - /spec/clusterIPs
```

See [examples/.kyt.yaml](examples/.kyt.yaml) for a complete configuration example.

## Installation

### From Source

```bash
git clone https://github.com/nhuray/k8s-diff.git
cd k8s-diff

# Using Make (recommended)
make build

# Or directly with Go
go build -o bin/kyt ./cmd/ky

# Optional: Install to your PATH
make install
# Or manually: cp bin/kyt /usr/local/bin/
```

### Using Go Install

```bash
go install github.com/nhuray/k8s-diff/cmd/ky@latest
```

### Binary Releases

_(Coming soon)_ Download pre-built binaries from [GitHub Releases](https://github.com/nhuray/k8s-diff/releases)

### Difftastic Integration

The `ky diff` command automatically detects and uses [difftastic](https://difftastic.wilfred.me.uk/) if available, providing beautiful syntax-aware structural diffs. If difftastic is not found, it gracefully falls back to standard unified diff.

**Install difftastic:**

```bash
# macOS
brew install difftastic

# Other platforms: see https://difftastic.wilfred.me.uk/installation.html
```

## Documentation

- **[fmt Command Guide](docs/fmt.md)** - Complete guide to formatting manifests with configuration options
- **[diff Command Guide](docs/diff.md)** - Advanced comparison techniques with JQ expressions and examples
- [Implementation Plan](docs/PLAN.md) - Detailed development roadmap with progress
- [Example Configs](examples/) - Sample configurations and manifests

## Testing

The project has comprehensive test coverage:

- **60+ tests total** (52 unit + 9 integration)
- All tests passing
- Covers: config loading, manifest parsing, normalization, diffing, output formatting, CLI commands

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

**Status:** ✅ Phase 8.6 complete! Tool refactored to `kyt` with subcommands. Documentation and release automation in progress.

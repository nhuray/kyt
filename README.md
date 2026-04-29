# k8s-diff

A CLI tool for comparing Kubernetes manifests with smart ignore rules.

## Overview

`k8s-diff` is a Go-based tool that intelligently compares Kubernetes manifests by applying configurable ignore rules before computing differences. It uses [ArgoCD](https://argo-cd.readthedocs.io/)-compatible ignore rules (JSON Pointers and JQ path expressions) and leverages [difftastic](https://difftastic.wilfred.me.uk/) for beautiful structural diffs.

**Key Features:**

- 🎯 **ArgoCD-Compatible Rules**: Uses the same ignore syntax as ArgoCD's `ignoreDifferences`
- 🔍 **JSON Pointer Support**: RFC 6901 compliant JSON Pointers for precise field targeting
- 🎨 **JQ Path Expressions**: Powerful filtering with wildcards and conditionals
- 📊 **Multiple Output Formats**: CLI (difftastic), JSON, unified diff, HTML
- ⚡ **Fast & Reliable**: Written in Go, leveraging ArgoCD's battle-tested code

## Use Cases

- **Validate Helm vs Kustomize migrations**: Compare rendered manifests while ignoring expected differences (labels, annotations)
- **Detect configuration drift**: Compare desired state with actual cluster state
- **CI/CD validation**: Ensure manifest changes are intentional
- **Release validation**: Compare what's deployed vs what should be deployed

## Status

🚧 **Work in Progress** - See [docs/PLAN.md](docs/PLAN.md) for detailed implementation plan.

## Quick Start (Planned)

```bash
# Compare two manifest files
k8s-diff source.yaml target.yaml

# Use custom config
k8s-diff source.yaml target.yaml --config my-rules.yaml

# Output JSON for CI/CD
k8s-diff source.yaml target.yaml --output json

# Generate HTML report
k8s-diff source.yaml target.yaml --output html > report.html
```

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

## Installation (Planned)

```bash
# Homebrew (coming soon)
brew install nicolasleigh/tap/k8s-diff

# Go install
go install github.com/nicolasleigh/k8s-diff/cmd/k8s-diff@latest

# Download binary from releases
# See: https://github.com/nicolasleigh/k8s-diff/releases
```

## Documentation

- [Implementation Plan](docs/PLAN.md) - Detailed development roadmap
- [Configuration Guide](docs/configuration.md) - _(Coming soon)_
- [Usage Examples](docs/usage.md) - _(Coming soon)_
- [Architecture](docs/architecture.md) - _(Coming soon)_

## Development

```bash
# Build
make build

# Run tests
make test

# Run linter
make lint

# Install locally
make install
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

Created by Nicolas Leigh ([@nicolasleigh](https://github.com/nicolasleigh))

---

**Status:** 🚧 Under active development - See [PLAN.md](docs/PLAN.md) for progress

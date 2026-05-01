# fmt Command Guide

The `kyt fmt` command formats Kubernetes manifests by sorting keys alphabetically to ensure consistent YAML structure.

## Table of Contents

- [Overview](#overview)
- [Usage](#usage)
- [Configuration](#configuration)
- [Examples](#examples)
- [Use Cases](#use-cases)

## Overview

The `fmt` command formats Kubernetes YAML manifests by sorting keys alphabetically. This ensures consistent field ordering and makes manifests easier to read and compare.

### What Does Formatting Do?

`kyt fmt` applies one transformation:

1. **Sorts keys alphabetically** - Ensures consistent field ordering throughout the YAML structure

### Formatting vs. Normalization

It's important to understand the difference between `fmt` and `diff`:

- **`kyt fmt`** - Only sorts keys alphabetically. Does not remove fields or apply ignore rules. Use this when you want to standardize YAML formatting without changing content.
- **`kyt diff`** - Performs full normalization (removes default fields, applies ignore rules, sorts keys) before comparison. Use this when you want to compare manifests and ignore irrelevant differences.

### Use Cases

The `fmt` command is useful for:

- **Standardizing YAML files** - Ensure consistent key ordering across your repository
- **Pre-commit formatting** - Automatically sort keys before committing
- **Improving readability** - Make YAML files easier to read with consistent ordering
- **Git diffs** - Reduce noise in version control by maintaining consistent ordering

## Usage

### Basic Usage

```bash
# Format a single file to stdout
kyt fmt deployment.yaml

# Format a directory to stdout
kyt fmt ./manifests

# Format and write back to source files
kyt fmt -w deployment.yaml

# Format from stdin
cat deployment.yaml | kyt fmt
kubectl get deployment nginx -o yaml | kyt fmt
```

### Command Options

```bash
kyt fmt [path] [flags]

Flags:
  -h, --help    help for fmt
  -w, --write   write formatted output back to source files

Note: The fmt command does not use configuration files. It only sorts keys alphabetically.
```

### Integration with Other Tools

```bash
# Format Kustomize output
kustomize build . | kyt fmt > formatted.yaml

# Format Helm output
helm template my-release ./chart | kyt fmt > formatted.yaml

# Format and apply
kustomize build . | kyt fmt | kubectl apply -f -

# Format all YAML files in a directory
find ./manifests -name "*.yaml" -exec kyt fmt -w {} \;

# Format in a pre-commit hook
#!/bin/bash
for file in $(git diff --cached --name-only | grep '\.ya*ml$'); do
  kyt fmt -w "$file"
  git add "$file"
done
```

## Configuration

**The `fmt` command does not use configuration files.** It only sorts keys alphabetically and does not remove fields or apply ignore rules.

If you need to remove fields, apply ignore rules, or perform other transformations, use the `kyt diff` command which applies full normalization before comparison.

### Want to Remove Fields or Apply Rules?

If you need advanced normalization (removing fields, applying ignore rules), you have two options:

1. **Use `kyt diff`** - The diff command normalizes both sides before comparison
2. **Process with other tools** - Pipe through tools like `yq` or `jq` for custom transformations

For example, to remove status fields:
```bash
# Using yq to remove status before formatting
yq 'del(.status)' manifest.yaml | kyt fmt
```

## Examples

### Example 1: Standardize Key Ordering

Sort keys alphabetically for consistent formatting:

```bash
# Format a single file
kyt fmt deployment.yaml > deployment-formatted.yaml

# Format and write back
kyt fmt -w deployment.yaml
```

### Example 2: Format Generated Manifests

Sort keys in Helm or Kustomize output:

```bash
# Format Helm output
helm template my-release ./chart | kyt fmt > formatted-output.yaml

# Format Kustomize output
kustomize build . | kyt fmt > formatted-output.yaml
```

### Example 3: Pre-Commit Hook

Sort keys in all YAML files before committing:

```bash
#!/bin/bash
# .git/hooks/pre-commit

# Get list of staged YAML files
YAML_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep '\.ya*ml$')

if [ -n "$YAML_FILES" ]; then
  echo "Formatting YAML files..."
  for file in $YAML_FILES; do
    if kyt fmt -w "$file" 2>/dev/null; then
      git add "$file"
      echo "  ✓ Formatted $file"
    fi
  done
fi
```



### Example 4: Batch Format Directory

Format all manifests in a directory:

```bash
# Format all YAML files
find ./manifests -name "*.yaml" -type f -exec kyt fmt -w {} \;

# Or using a loop for better error handling
for file in ./manifests/**/*.yaml; do
  echo "Formatting $file..."
  kyt fmt -w "$file"
done
```

## Use Cases

### 1. Standardizing Generated Manifests

When using tools like Helm or Kustomize, the generated YAML often has inconsistent key ordering.

```bash
# Standardize key ordering
helm template . | kyt fmt > standardized.yaml
kustomize build . | kyt fmt > standardized.yaml
```

### 2. Repository Standardization

Ensure all YAML files in your repository have consistent key ordering.

```bash
# Format all manifests in repository
find . -name "*.yaml" -path "*/k8s/*" -exec kyt fmt -w {} \;

# Add to CI to enforce formatting
# .github/workflows/lint.yml
- name: Check YAML formatting
  run: |
    # Check if any files would change
    if ! git diff --exit-code $(find . -name "*.yaml" -exec kyt fmt {} \;); then
      echo "YAML files need formatting. Run: kyt fmt -w"
      exit 1
    fi
```

### 3. Improving Git Diffs

Consistent key ordering reduces noise in git diffs and makes code reviews easier.

```bash
# Format files before committing
kyt fmt -w deployment.yaml
git add deployment.yaml
git commit -m "Update deployment"
```

### 4. Cleaning Up Cluster Exports

When exporting resources from a cluster, sort keys for readability.

```bash
# Export with sorted keys
kubectl get deployment nginx -o yaml | kyt fmt > nginx-export.yaml
```

## Best Practices

1. **Use fmt for formatting only**: The `fmt` command only sorts keys. For removing fields or applying rules, use `kyt diff` or external tools like `yq`

2. **Commit formatted files**: Always format files before committing to maintain consistency across the team

3. **Use in CI**: Add formatting checks to your CI pipeline to enforce consistent key ordering

4. **Pre-commit hooks**: Automate formatting with git pre-commit hooks

5. **Combine with other tools**: Pipe through `yq` or `jq` for additional transformations before formatting

## See Also

- [diff Command Guide](diff.md) - Compare formatted manifests
- [Configuration Examples](../examples/.kyt.yaml) - Full configuration examples
- [Implementation Plan](PLAN.md) - Technical implementation details

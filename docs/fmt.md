# fmt Command Guide

The `kyt fmt` command formats Kubernetes manifests by applying transformations like sorting keys, sorting arrays, removing default fields, and applying custom ignore rules.

## Table of Contents

- [Overview](#overview)
- [Usage](#usage)
- [Configuration](#configuration)
- [Examples](#examples)
- [Use Cases](#use-cases)

## Overview

The `fmt` command normalizes Kubernetes YAML manifests to ensure consistent formatting. This is useful for:

- **Cleaning up generated manifests** - Remove noise from Helm/Kustomize output
- **Standardizing YAML files** - Ensure consistent formatting across your repository
- **Pre-commit formatting** - Automatically format manifests before committing
- **Preparing for comparison** - Normalize manifests before diffing to reduce noise

### What Does Formatting Do?

By default, `kyt fmt` applies the following transformations:

1. **Sorts keys alphabetically** - Ensures consistent field ordering
2. **Removes default fields** - Strips out fields like `status`, `managedFields`, timestamps
3. **Applies ignore rules** - Uses your `.kyt.yaml` configuration to remove or transform specific fields
4. **Normalizes YAML structure** - Consistent indentation and spacing

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

Global Flags:
  -c, --config string   config file (default: .kyt.yaml)
  -v, --verbose         verbose output to stderr
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

The `fmt` command uses the `.kyt.yaml` configuration file to determine how to format manifests. The configuration is automatically detected in the current directory or parent directories.

### Configuration File Location

```bash
# Explicit config file
kyt fmt -c /path/to/.kyt.yaml deployment.yaml

# Auto-detected (searches current and parent directories)
kyt fmt deployment.yaml  # looks for .kyt.yaml
```

### Normalization Configuration

The `normalization` section controls how manifests are formatted:

```yaml
# .kyt.yaml
normalization:
  # Sort object keys alphabetically (default: true)
  sortKeys: true

  # Sort arrays where order doesn't matter
  sortArrays:
    - path: ".spec.template.spec.containers[].ports"
      sortBy: "containerPort"
    - path: ".spec.template.spec.containers[].env"
      sortBy: "name"
    - path: ".spec.template.spec.volumes"
      sortBy: "name"
    - path: ".spec.template.spec.containers[].volumeMounts"
      sortBy: "name"

  # Remove fields that shouldn't be included
  removeDefaultFields:
    - "/status"
    - "/metadata/managedFields"
    - "/metadata/creationTimestamp"
    - "/metadata/generation"
    - "/metadata/resourceVersion"
    - "/metadata/uid"
```

### Ignore Differences Configuration

The `ignoreDifferences` section allows you to remove or transform specific fields during formatting:

```yaml
# .kyt.yaml
ignoreDifferences:
  # Remove specific labels from all resources
  - group: ""
    kind: "*"
    jsonPointers:
      - /metadata/labels/app.kubernetes.io~1version
      - /metadata/annotations/kubectl.kubernetes.io~1last-applied-configuration

  # Remove Istio sidecar containers from Deployments
  - group: "apps"
    kind: "Deployment"
    jqPathExpressions:
      - .spec.template.spec.containers[] | select(.name == "istio-proxy")
      - .spec.template.spec.initContainers[] | select(.name == "istio-init")

  # Remove replica count from Deployments
  - group: "apps"
    kind: "Deployment"
    jsonPointers:
      - /spec/replicas

  # Remove fields managed by specific controllers
  - group: "apps"
    kind: "Deployment"
    managedFieldsManagers:
      - "kube-controller-manager"
```

### Configuration Options Reference

#### `normalization.sortKeys`

**Type:** `boolean`  
**Default:** `true`

Sorts object keys alphabetically for consistent ordering.

```yaml
normalization:
  sortKeys: true
```

#### `normalization.sortArrays`

**Type:** `array of objects`  
**Default:** `[]`

Defines which arrays should be sorted before output. Useful for arrays where order doesn't matter semantically.

```yaml
normalization:
  sortArrays:
    - path: ".spec.template.spec.containers[].env"
      sortBy: "name"
```

- `path`: JQ-style path to the array (e.g., `.spec.template.spec.containers[].env`)
- `sortBy`: Field name to sort by (e.g., `"name"`, `"containerPort"`)

#### `normalization.removeDefaultFields`

**Type:** `array of strings`  
**Default:** See below

JSON Pointer paths to remove from all resources.

**Default fields removed:**
- `/status` - Runtime status (ephemeral)
- `/metadata/managedFields` - Server-side apply metadata
- `/metadata/creationTimestamp` - Creation time
- `/metadata/generation` - Resource version counter
- `/metadata/resourceVersion` - Cluster-specific version
- `/metadata/uid` - Cluster-specific unique ID

```yaml
normalization:
  removeDefaultFields:
    - "/status"
    - "/metadata/managedFields"
    - "/metadata/annotations/kubectl.kubernetes.io~1last-applied-configuration"
```

#### `ignoreDifferences`

**Type:** `array of objects`

Defines rules for ignoring specific fields in specific resource types. See the [Ignore Rules](#ignore-rules) section below.

### Ignore Rules

Ignore rules allow you to selectively remove or transform fields during formatting. This is useful when you want to clean up manifests by removing generated or environment-specific values.

#### Rule Matching

Rules match resources based on:

- `group` - API group (empty string for core resources)
- `kind` - Resource kind (use `"*"` to match all kinds)
- `name` - Resource name (optional, supports glob patterns)
- `namespace` - Resource namespace (optional, supports glob patterns)

#### Ignore Methods

Each rule can use one or more of these methods:

##### 1. JSON Pointers

**Best for:** Removing specific fields by path

```yaml
ignoreDifferences:
  - group: ""
    kind: "Service"
    jsonPointers:
      - /spec/clusterIP
      - /spec/clusterIPs
      - /metadata/labels/app.kubernetes.io~1version  # ~ escape for /
```

##### 2. JQ Path Expressions

**Best for:** Complex filtering and conditional removal

```yaml
ignoreDifferences:
  - group: "apps"
    kind: "Deployment"
    jqPathExpressions:
      # Remove containers by name
      - .spec.template.spec.containers[] | select(.name == "istio-proxy")
      
      # Remove env vars by name
      - .spec.template.spec.containers[].env[] | select(.name == "DEBUG")
      
      # Remove annotations matching a pattern
      - .metadata.annotations | to_entries[] | select(.key | startswith("kubectl.kubernetes.io/"))
```

##### 3. Managed Fields Managers

**Best for:** Removing fields managed by specific controllers

```yaml
ignoreDifferences:
  - group: "apps"
    kind: "Deployment"
    managedFieldsManagers:
      - "kube-controller-manager"
      - "kubectl-client-side-apply"
```

## Examples

### Example 1: Clean Up Helm Output

Remove Helm-specific annotations and standardize formatting:

```yaml
# .kyt.yaml
normalization:
  sortKeys: true
  removeDefaultFields:
    - "/status"
    - "/metadata/managedFields"

ignoreDifferences:
  - group: ""
    kind: "*"
    jsonPointers:
      - /metadata/annotations/meta.helm.sh~1release-name
      - /metadata/annotations/meta.helm.sh~1release-namespace
      - /metadata/labels/app.kubernetes.io~1managed-by
```

```bash
helm template my-release ./chart | kyt fmt > clean-output.yaml
```

### Example 2: Remove Istio Sidecars

Format manifests without Istio sidecar noise:

```yaml
# .kyt.yaml
ignoreDifferences:
  - group: "apps"
    kind: "Deployment"
    jqPathExpressions:
      - .spec.template.spec.containers[] | select(.name == "istio-proxy")
      - .spec.template.spec.initContainers[] | select(.name == "istio-init")
      - .spec.template.metadata.annotations["sidecar.istio.io/status"]
```

```bash
kubectl get deployment -o yaml | kyt fmt > without-istio.yaml
```

### Example 3: Standardize Environment Variables

Sort environment variables and volumes for consistency:

```yaml
# .kyt.yaml
normalization:
  sortKeys: true
  sortArrays:
    - path: ".spec.template.spec.containers[].env"
      sortBy: "name"
    - path: ".spec.template.spec.volumes"
      sortBy: "name"
    - path: ".spec.template.spec.containers[].volumeMounts"
      sortBy: "name"
```

```bash
kyt fmt -w deployment.yaml
```

### Example 4: Pre-Commit Hook

Format all YAML files before committing:

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

### Example 5: Batch Format Directory

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

When using tools like Helm or Kustomize, the generated YAML often has inconsistent formatting, field ordering, or extra metadata.

```bash
# Before: Inconsistent field order, extra annotations
helm template . | kyt fmt > standardized.yaml
```

### 2. Cleaning Cluster Resources

When exporting resources from a cluster, they contain runtime fields that aren't needed.

```bash
# Export clean deployment
kubectl get deployment nginx -o yaml | kyt fmt > nginx-clean.yaml

# Export all deployments without noise
kubectl get deployments -o yaml | kyt fmt > deployments-clean.yaml
```

### 3. Pre-Diff Formatting

Format manifests before comparing to reduce noise from irrelevant differences.

```bash
# Format both sides before comparing
kyt fmt old-manifest.yaml > old-formatted.yaml
kyt fmt new-manifest.yaml > new-formatted.yaml
diff old-formatted.yaml new-formatted.yaml
```

Or better yet, use `kyt diff` which does this automatically:

```bash
kyt diff old-manifest.yaml new-manifest.yaml
```

### 4. Repository Standardization

Ensure all YAML files in your repository follow the same formatting conventions.

```bash
# Format all manifests in repository
find . -name "*.yaml" -path "*/k8s/*" -exec kyt fmt -w {} \;

# Add to CI to enforce formatting
# .github/workflows/lint.yml
- name: Check YAML formatting
  run: |
    make fmt-check
```

### 5. Migration Prep

When migrating from one deployment tool to another, format outputs to make comparison easier.

```bash
# Compare Helm vs Kustomize
helm template release1 chart1 | kyt fmt > helm-output.yaml
kustomize build overlay | kyt fmt > kustomize-output.yaml
kyt diff helm-output.yaml kustomize-output.yaml
```

## Best Practices

1. **Commit the config**: Always commit `.kyt.yaml` to version control so formatting is consistent across the team

2. **Start minimal**: Begin with just `sortKeys: true` and add more rules as needed

3. **Use in CI**: Add formatting checks to your CI pipeline to enforce consistency

4. **Document rules**: Add comments to your `.kyt.yaml` explaining why each ignore rule exists

5. **Test formatting**: Always review formatted output before using `-w` to ensure rules work as expected

## See Also

- [diff Command Guide](diff.md) - Compare formatted manifests
- [Configuration Examples](../examples/.kyt.yaml) - Full configuration examples
- [Implementation Plan](PLAN.md) - Technical implementation details

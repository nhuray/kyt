# diff Command Guide

The `kyt diff` command compares Kubernetes manifests with smart ignore rules, providing clear visibility into what actually changed while filtering out noise.

## Table of Contents

- [Overview](#overview)
- [Usage](#usage)
- [Configuration](#configuration)
- [Ignore Rules](#ignore-rules)
- [Advanced Examples](#advanced-examples)
- [JQ Expression Cookbook](#jq-expression-cookbook)
- [Use Cases](#use-cases)

## Overview

The `diff` command solves a common problem: comparing Kubernetes manifests produces too much noise. Fields like timestamps, resource versions, managed fields, and even container ordering can create false positives that hide real changes.

`kyt diff` fixes this by:

1. **Normalizing both inputs** - Sorts keys, removes default fields
2. **Applying ignore rules** - Uses JSON Pointers and JQ expressions to ignore specific differences
3. **Smart resource matching** - Automatically pairs resources even if they've been renamed
4. **Beautiful output** - Uses difftastic for structural, syntax-aware diffs

## Usage

### Basic Usage

```bash
# Compare two files
kyt diff left.yaml right.yaml

# Compare two directories
kyt diff ./helm-output ./kustomize-output

# Compare file to directory
kyt diff deployment.yaml ./manifests

# Compare with custom config
kyt diff -c .kyt.yaml left.yaml right.yaml
```

### Command Options

```bash
kyt diff <left> <right> [flags]

Flags:
  -c, --config string      config file (default: .kyt.yaml)
  -o, --output string      output format: cli, json (default "cli")
      --show-identical     show resources with no differences
      --exact-match        disable similarity matching (exact name match only)
      --diff-tool string   diff tool: auto, difft, treesitter, diff (default "auto")
      --display string     difftastic display mode: side-by-side, inline (default "side-by-side")
      --include string     comma-separated list of resource kinds to include (e.g., 'cm,svc,deploy')
      --exclude string     comma-separated list of resource kinds to exclude (e.g., 'secrets,configmaps')
  -v, --verbose            verbose output to stderr
  -h, --help               help for diff
```

### Resource Filtering

The `--include` and `--exclude` flags allow you to filter which resources are compared. This is useful when you only care about specific resource types or want to skip certain resources.

**Supported name forms:**
- **Short names**: `cm`, `svc`, `deploy`, `sts`, `ds`, `po`, etc.
- **Singular**: `configmap`, `service`, `deployment`, etc.
- **Plural**: `configmaps`, `services`, `deployments`, etc.
- **Kind names**: `ConfigMap`, `Service`, `Deployment`, etc.

All forms are case-insensitive and can be mixed in the same command.

**Examples:**

```bash
# Include only ConfigMaps and Secrets
kyt diff --include cm,secrets ./source ./target

# Exclude Secrets from comparison
kyt diff --exclude secrets ./source ./target

# Include multiple resource types (using different name forms)
kyt diff --include deploy,svc,cm ./source ./target
kyt diff --include deployments,services,configmaps ./source ./target
kyt diff --include Deployment,Service,ConfigMap ./source ./target

# Compare only StatefulSets and DaemonSets
kyt diff --include sts,ds ./source ./target

# Exclude multiple types
kyt diff --exclude secrets,cm,svc ./source ./target
```

**Note:** 
- `--include` and `--exclude` can be used together
- When `--include` is specified, only those resource kinds are compared
- When `--exclude` is specified, those resource kinds are skipped
- Filters apply to both source and target manifests

### Exit Codes

- `0` - No differences found (success)
- `1` - Differences detected
- `2` - Error (invalid YAML, missing files, etc.)

### Output Formats

#### CLI (Default)

Human-readable output with colors and structure:

```bash
kyt diff left.yaml right.yaml

================================================================
  kyt diff Report
================================================================

Summary:
  Identical Resources: 5
  Modified Resources:  2
  Added Resources:     1
  Removed Resources:   0

Modified Resources (2):
  • Deployment.apps/nginx
  • Service/nginx

[... detailed diffs using difftastic ...]
```

#### JSON

Machine-readable output for CI/CD pipelines:

```bash
kyt diff -o json left.yaml right.yaml

{
  "summary": {
    "identical": 5,
    "modified": 2,
    "added": 1,
    "removed": 0,
    "hasDifferences": true
  },
  "modified": [
    {
      "key": {
        "apiVersion": "apps/v1",
        "kind": "Deployment",
        "name": "nginx"
      },
      "diff": "..."
    }
  ],
  ...
}
```

## Configuration

The `diff` command uses `.kyt.yaml` to configure ignore rules, normalization, and output options.

### Full Configuration Example

```yaml
# .kyt.yaml

# Ignore specific differences
ignoreDifferences:
  # Ignore replica count in production (HPA-managed)
  - group: "apps"
    kind: "Deployment"
    namespace: "production"
    jsonPointers:
      - /spec/replicas

  # Ignore Istio sidecars
  - group: "apps"
    kind: "Deployment"
    jqPathExpressions:
      - .spec.template.spec.containers[] | select(.name == "istio-proxy")

  # Ignore kubectl last-applied-configuration
  - group: ""
    kind: "*"
    jsonPointers:
      - /metadata/annotations/kubectl.kubernetes.io~1last-applied-configuration

# Normalization options
normalization:
  sortKeys: true
  sortArrays:
    - path: ".spec.template.spec.containers[].env"
      sortBy: "name"
  removeDefaultFields:
    - "/status"
    - "/metadata/managedFields"
    - "/metadata/creationTimestamp"

# Output options
output:
  format: cli            # cli or json
  diffTool: auto         # auto, difft, treesitter, diff
  colorize: true
  showUnchanged: false
  contextLines: 3
```

## Ignore Rules

Ignore rules are the heart of `kyt diff`. They let you filter out expected or irrelevant differences to focus on what actually matters.

### Rule Structure

```yaml
ignoreDifferences:
  - group: "apps"              # API group (empty for core resources)
    kind: "Deployment"         # Resource kind (use "*" for all)
    name: "nginx-*"            # Optional: resource name (supports globs)
    namespace: "prod-*"        # Optional: namespace (supports globs)
    
    # Choose one or more ignore methods:
    jsonPointers: [...]        # JSON Pointer paths
    jqPathExpressions: [...]   # JQ expressions
    managedFieldsManagers: [...] # Field manager names
```

### JSON Pointers

**Best for:** Simple, direct field paths

JSON Pointers (RFC 6901) provide a simple syntax for targeting specific fields:

```yaml
ignoreDifferences:
  # Single field
  - group: ""
    kind: "Service"
    jsonPointers:
      - /spec/clusterIP

  # Nested field
  - group: "apps"
    kind: "Deployment"
    jsonPointers:
      - /spec/template/metadata/annotations/prometheus.io~1scrape

  # Array element by index
  - group: "apps"
    kind: "Deployment"
    jsonPointers:
      - /spec/template/spec/containers/0/image
```

**JSON Pointer Escaping:**
- `/` in field names must be escaped as `~1`
- `~` in field names must be escaped as `~0`

Example: `kubectl.kubernetes.io/last-applied-configuration` becomes:
```
/metadata/annotations/kubectl.kubernetes.io~1last-applied-configuration
```

### JQ Path Expressions

**Best for:** Complex filtering, conditionals, and transformations

JQ expressions provide powerful filtering capabilities:

```yaml
ignoreDifferences:
  # Select by field value
  - group: "apps"
    kind: "Deployment"
    jqPathExpressions:
      - .spec.template.spec.containers[] | select(.name == "istio-proxy")

  # Select by field existence
  - group: ""
    kind: "ConfigMap"
    jqPathExpressions:
      - .data | keys[] | select(startswith("temp-"))

  # Complex condition
  - group: "apps"
    kind: "Deployment"
    jqPathExpressions:
      - .spec.template.spec.containers[] | select(.image | contains(":latest"))
```

### Managed Fields Managers

**Best for:** Ignoring fields updated by controllers

```yaml
ignoreDifferences:
  - group: "apps"
    kind: "Deployment"
    managedFieldsManagers:
      - "kube-controller-manager"  # Ignore HPA updates
      - "kubectl-client-side-apply"
```

## Advanced Examples

### Example 1: Helm vs Kustomize Migration

Compare Helm and Kustomize outputs while ignoring tool-specific differences:

```yaml
# .kyt.yaml
ignoreDifferences:
  # Ignore Helm metadata
  - group: ""
    kind: "*"
    jsonPointers:
      - /metadata/annotations/meta.helm.sh~1release-name
      - /metadata/annotations/meta.helm.sh~1release-namespace
      - /metadata/labels/app.kubernetes.io~1managed-by
      - /metadata/labels/helm.sh~1chart

  # Ignore Kustomize metadata
  - group: ""
    kind: "*"
    jsonPointers:
      - /metadata/annotations/config.kubernetes.io~1index
      - /metadata/annotations/config.kubernetes.io~1path

normalization:
  sortKeys: true
  sortArrays:
    - path: ".spec.template.spec.containers[].env"
      sortBy: "name"
```

```bash
# Generate outputs
helm template my-release ./chart > helm-output.yaml
kustomize build ./overlay > kustomize-output.yaml

# Compare
kyt diff helm-output.yaml kustomize-output.yaml
```

### Example 2: Cluster Drift Detection

Compare desired state (Git) with actual state (cluster):

```yaml
# .kyt.yaml
ignoreDifferences:
  # Ignore runtime/ephemeral fields
  - group: ""
    kind: "*"
    jsonPointers:
      - /metadata/resourceVersion
      - /metadata/uid
      - /metadata/generation
      - /metadata/creationTimestamp
      - /metadata/managedFields
      - /status

  # Ignore HPA-managed replicas
  - group: "apps"
    kind: "Deployment"
    managedFieldsManagers:
      - "kube-controller-manager"
    jsonPointers:
      - /spec/replicas

  # Ignore service IPs (assigned by cluster)
  - group: ""
    kind: "Service"
    jsonPointers:
      - /spec/clusterIP
      - /spec/clusterIPs
```

```bash
# Export current state
kubectl get deployment nginx -o yaml > cluster-state.yaml

# Compare with desired state
kyt diff git-repo/nginx-deployment.yaml cluster-state.yaml
```

### Example 3: Ignore Istio Injection

Compare manifests with and without Istio sidecar injection:

```yaml
# .kyt.yaml
ignoreDifferences:
  - group: "apps"
    kind: "Deployment"
    jqPathExpressions:
      # Remove istio-proxy container
      - .spec.template.spec.containers[] | select(.name == "istio-proxy")
      
      # Remove istio-init container
      - .spec.template.spec.initContainers[] | select(.name == "istio-init")
      
      # Remove Istio annotations
      - .spec.template.metadata.annotations | to_entries[] | select(.key | startswith("sidecar.istio.io/"))
      - .spec.template.metadata.annotations | to_entries[] | select(.key | startswith("prometheus.io/"))
      
      # Remove Istio volumes
      - .spec.template.spec.volumes[] | select(.name | startswith("istio-"))
```

```bash
kyt diff without-istio.yaml with-istio.yaml
# Should show no differences!
```

### Example 4: Environment-Specific Differences

Compare staging and production while ignoring expected differences:

```yaml
# .kyt.yaml
ignoreDifferences:
  # Different replica counts per environment
  - group: "apps"
    kind: "Deployment"
    jsonPointers:
      - /spec/replicas

  # Different resource limits per environment
  - group: "apps"
    kind: "Deployment"
    jsonPointers:
      - /spec/template/spec/containers/0/resources/limits
      - /spec/template/spec/containers/0/resources/requests

  # Environment-specific config
  - group: ""
    kind: "ConfigMap"
    name: "app-config"
    jsonPointers:
      - /data/ENVIRONMENT
      - /data/LOG_LEVEL

  # Environment-specific secrets
  - group: ""
    kind: "Secret"
    jsonPointers:
      - /data
```

```bash
kyt diff staging/ production/
```

### Example 5: Pre-Deployment Validation

Compare current deployment with what will be deployed:

```yaml
# .kyt.yaml for pre-deploy checks
ignoreDifferences:
  # Allow image tag updates (expected)
  - group: "apps"
    kind: "Deployment"
    jqPathExpressions:
      - .spec.template.spec.containers[].image

  # Allow replica count changes (HPA managed)
  - group: "apps"
    kind: "Deployment"
    jsonPointers:
      - /spec/replicas

  # But catch everything else!
```

```bash
# In CI/CD pipeline
kubectl get -f k8s/ -o yaml > current.yaml
kustomize build k8s/ > desired.yaml

if kyt diff current.yaml desired.yaml --exact-match; then
  echo "✓ No unexpected changes"
else
  echo "✗ Unexpected changes detected!"
  exit 1
fi
```

### Example 6: Comparing Only Specific Resource Types

Compare only certain resource types using the `--include` and `--exclude` flags:

```bash
# Compare only ConfigMaps and Secrets (useful for config drift detection)
kyt diff --include cm,secrets ./prod ./staging

# Compare all resources except Secrets (skip sensitive data)
kyt diff --exclude secrets ./source ./target

# Compare only workload resources
kyt diff --include deploy,sts,ds,job ./helm-output ./kustomize-output

# Compare infrastructure resources only
kyt diff --include svc,ing,cm ./old-infra ./new-infra

# Skip testing resources when comparing environments
kyt diff --exclude job,cronjob,po ./dev ./prod
```

**Use cases for resource filtering:**

- **Security audits**: Compare only RBAC resources (`--include role,rolebinding,clusterrole,clusterrolebinding`)
- **Config validation**: Compare only ConfigMaps and Secrets (`--include cm,secrets`)
- **Workload comparison**: Focus on Deployments and StatefulSets (`--include deploy,sts`)
- **Skip ephemeral resources**: Exclude Pods and Jobs (`--exclude po,job`)
- **Network comparison**: Compare only Services and Ingresses (`--include svc,ing`)

## JQ Expression Cookbook

### Common Patterns

#### 1. Select Container by Name

```yaml
jqPathExpressions:
  - .spec.template.spec.containers[] | select(.name == "nginx")
```

#### 2. Filter Environment Variables

```yaml
jqPathExpressions:
  # Remove specific env var
  - .spec.template.spec.containers[].env[] | select(.name == "DEBUG")
  
  # Remove env vars starting with "TMP_"
  - .spec.template.spec.containers[].env[] | select(.name | startswith("TMP_"))
```

#### 3. Ignore Image Tags

```yaml
jqPathExpressions:
  # Ignore all :latest tags
  - .spec.template.spec.containers[] | select(.image | endswith(":latest"))
  
  # Ignore images from specific registry
  - .spec.template.spec.containers[] | select(.image | startswith("docker.io/"))
```

#### 4. Filter Annotations by Pattern

```yaml
jqPathExpressions:
  # Remove all kubectl annotations
  - .metadata.annotations | to_entries[] | select(.key | startswith("kubectl."))
  
  # Remove timestamp annotations
  - .metadata.annotations | to_entries[] | select(.key | endswith(".timestamp"))
```

#### 5. Filter by Label

```yaml
jqPathExpressions:
  # Ignore resources with specific label
  - select(.metadata.labels["ignore-diff"] == "true")
  
  # Ignore version labels
  - .metadata.labels | to_entries[] | select(.key == "version")
```

#### 6. Conditional on Resource Properties

```yaml
jqPathExpressions:
  # Only in specific namespace
  - select(.metadata.namespace == "default") | .spec.replicas
  
  # Only for resources with HPA
  - select(.metadata.annotations["autoscaling.enabled"] == "true") | .spec.replicas
```

#### 7. Array Filtering

```yaml
jqPathExpressions:
  # Remove empty volumes
  - .spec.template.spec.volumes[] | select(.emptyDir != null)
  
  # Remove volumes by name pattern
  - .spec.template.spec.volumes[] | select(.name | startswith("cache-"))
```

#### 8. Nested Selections

```yaml
jqPathExpressions:
  # Remove probes from specific containers
  - .spec.template.spec.containers[] | select(.name == "app") | .livenessProbe
  - .spec.template.spec.containers[] | select(.name == "app") | .readinessProbe
```

### JQ Tips & Tricks

1. **Test expressions**: Use `jq` CLI to test your expressions
   ```bash
   kubectl get deployment nginx -o json | jq '.spec.template.spec.containers[] | select(.name == "istio-proxy")'
   ```

2. **Combine multiple selectors**: Use `or` for multiple conditions
   ```yaml
   - .spec.template.spec.containers[] | select(.name == "istio-proxy" or .name == "envoy")
   ```

3. **Use `has()` to check field existence**:
   ```yaml
   - .spec.template.spec.containers[] | select(has("securityContext"))
   ```

4. **Array filtering with `any`**:
   ```yaml
   - select(.spec.template.spec.containers | any(.name == "sidecar"))
   ```

## Use Cases

### 1. CI/CD Validation

Ensure PR changes are intentional:

```bash
#!/bin/bash
# .github/workflows/validate.yml

# Render both versions
kustomize build main > main.yaml
kustomize build PR_BRANCH > pr.yaml

# Compare with ignore rules
if kyt diff main.yaml pr.yaml; then
  echo "✓ No unexpected changes"
  exit 0
else
  echo "✗ Changes detected - review required"
  kyt diff main.yaml pr.yaml -o json > changes.json
  exit 1
fi
```

### 2. Release Validation

Before deploying, verify what will change:

```bash
#!/bin/bash
# compare current vs new deployment

# Get current state
kubectl get -f k8s/ -o yaml > current.yaml

# Get desired state
helm template release-v2 . > desired.yaml

# Show differences
echo "The following changes will be deployed:"
kyt diff current.yaml desired.yaml

read -p "Proceed with deployment? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
  kubectl apply -f desired.yaml
fi
```

### 3. Multi-Environment Consistency

Verify staging and production have same configuration (except expected differences):

```bash
# Compare with environment-specific ignore rules
kyt diff -c .kyt-env-diff.yaml \
  envs/staging/ \
  envs/production/
```

### 4. Tool Migration Validation

Verify Helm → Kustomize migration produces same results:

```bash
helm template old-release ./helm-chart > helm.yaml
kustomize build ./kustomize > kustomize.yaml
kyt diff helm.yaml kustomize.yaml
```

### 5. Drift Detection Automation

Run regularly to detect cluster drift:

```bash
#!/bin/bash
# cron job: check-drift.sh

for ns in production staging; do
  echo "Checking namespace: $ns"
  
  # Export current state
  kubectl get all -n $ns -o yaml > /tmp/current-$ns.yaml
  
  # Get desired state from git
  kustomize build git-repo/overlays/$ns > /tmp/desired-$ns.yaml
  
  # Compare
  if ! kyt diff /tmp/desired-$ns.yaml /tmp/current-$ns.yaml; then
    echo "⚠️  Drift detected in $ns!"
    # Send alert
  fi
done
```

## Diff Tool Options

### Difftastic (Default)

Beautiful structural diffs with syntax highlighting:

```bash
kyt diff --diff-tool difft left.yaml right.yaml

# Change display mode
kyt diff --display inline left.yaml right.yaml
kyt diff --display side-by-side left.yaml right.yaml
```

### Tree-sitter (Built-in Fallback)

Go-native structural diff (no external dependencies):

```bash
kyt diff --diff-tool treesitter left.yaml right.yaml
```

### Unified Diff

Traditional diff format:

```bash
kyt diff --diff-tool diff left.yaml right.yaml
```

### Auto (Recommended)

Tries difftastic → tree-sitter → unified diff:

```bash
kyt diff --diff-tool auto left.yaml right.yaml
```

## Best Practices

1. **Start small**: Begin with minimal ignore rules and add as needed
2. **Document rules**: Add comments explaining why each rule exists
3. **Test rules**: Use `kyt fmt` to preview what will be ignored
4. **Version control**: Commit `.kyt.yaml` with your manifests
5. **Use in CI**: Automate diff checks in your pipeline
6. **Review JSON output**: Use `-o json` for programmatic analysis
7. **Enable similarity matching**: Let kyt detect renamed resources (default)
8. **Use verbose mode**: Add `-v` when debugging ignore rules

## Troubleshooting

### Differences Not Being Ignored

1. Check resource matching (group, kind, name, namespace)
2. Test JQ expression with `jq` CLI
3. Use `-v` verbose mode to see which rules match
4. Verify JSON Pointer escaping (`/` → `~1`)

### Too Many Differences

1. Add more `removeDefaultFields` in normalization
2. Use wildcard (`*`) for kind to apply rules broadly
3. Enable `sortKeys` and `sortArrays` in normalization

### Resource Not Matching

1. Check API group (use `""` for core resources)
2. Verify similarity threshold with `--exact-match`
3. Use `--show-identical` to see all resources

## See Also

- [fmt Command Guide](fmt.md) - Format manifests before comparing
- [Configuration Examples](../examples/.kyt.yaml) - Full configuration examples
- [JQ Manual](https://stedolan.github.io/jq/manual/) - JQ expression reference
- [JSON Pointer RFC](https://tools.ietf.org/html/rfc6901) - JSON Pointer specification

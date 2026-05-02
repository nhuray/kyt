# Cluster Comparison Guide

This guide explains how to use kyt's live cluster comparison feature to compare Kubernetes resources directly from your clusters.

## Overview

kyt can fetch resources from live Kubernetes clusters and compare them with:
- Local manifest files or directories
- Resources from other namespaces
- Resources from different clusters

This is useful for:
- Detecting configuration drift between Git (desired state) and cluster (actual state)
- Comparing environments (prod vs staging)
- Validating deployments before applying changes
- Troubleshooting differences across namespaces

## Basic Usage

### Namespace Syntax

Use the `ns:namespace` syntax to reference a Kubernetes namespace:

```bash
# Compare local manifests against production cluster
kyt diff ./manifests ns:production

# Compare two namespaces
kyt diff ns:production ns:staging

# Compare cluster against local manifests (reversed)
kyt diff ns:production ./manifests
```

### Context Selection

kyt uses your current kubectl context by default, but you can specify a different context:

```bash
# Uses current context (whatever kubectl is using)
kyt diff ./manifests ns:production

# Explicitly specify context
kyt diff --context prod ./manifests ns:production

# Compare namespaces in specific cluster
kyt diff --context prod ns:production ns:staging
```

Check your current context:
```bash
kubectl config current-context
```

List available contexts:
```bash
kubectl config get-contexts
```

## Resource Types

kyt fetches approximately 15 common resource types from the cluster:

**Core (v1):**
- pods
- services
- replicationcontrollers
- configmaps
- secrets
- persistentvolumeclaims

**apps/v1:**
- deployments
- daemonsets
- replicasets
- statefulsets

**batch/v1:**
- jobs
- cronjobs

**networking.k8s.io/v1:**
- ingresses

**autoscaling/v1:**
- horizontalpodautoscalers

### Filtering Resources

Use `--include` or `--exclude` to filter specific resource types:

```bash
# Compare only Deployments and Services
kyt diff --include deploy,svc ns:production ns:staging

# Compare all except Secrets
kyt diff --exclude secrets ns:production ns:staging

# Multiple resource types
kyt diff --include cm,secrets,deploy ./manifests ns:production
```

Supports multiple formats:
- Short names: `cm`, `svc`, `deploy`
- Singular: `configmap`, `service`, `deployment`
- Plural: `configmaps`, `services`, `deployments`

## Verbose Mode

Use `-v` or `--verbose` to see detailed information about the cluster connection and resource fetching:

```bash
kyt diff -v ns:production ns:staging
```

Output includes:
- Current context being used
- Cluster connection status
- Resources found per type
- Total resources fetched
- Resource types skipped (due to permissions or not available)

Example verbose output:
```
Using current context: gke-prod

Comparing:
  Source: ns:production (context: gke-prod)
  Target: ns:staging (context: gke-prod)

Loading source manifests...
  Connected to cluster (context: gke-prod)
  Fetching resources from namespace "production"...
  Found 12 pods
  Found 8 services
  Found 15 configmaps
  Found 5 secrets
  Found 6 deployments
  Total: 46 resources from 5 resource types (0 types skipped)
```

## Common Workflows

### Drift Detection

Compare your Git repository against the live cluster to detect configuration drift:

```bash
# Compare local desired state against actual cluster state
kyt diff ./k8s/production ns:production

# Show only a summary
kyt diff --summary ./k8s/production ns:production

# Filter to specific types
kyt diff --include deploy,svc,cm ./k8s/production ns:production
```

### Environment Comparison

Compare production vs staging environments:

```bash
# Same cluster, different namespaces
kyt diff ns:production ns:staging

# Different clusters
kyt diff --context prod ns:default ./staging-export.yaml

# With filtering
kyt diff --include deploy,svc ns:production ns:staging
```

### Pre-Deployment Validation

Before applying changes, compare what will be deployed vs what's currently running:

```bash
# Generate manifests and compare
kustomize build ./overlays/production | kyt diff - ns:production

# With Helm
helm template my-release ./chart | kyt diff - ns:production

# From directory
kyt diff ./manifests-to-apply ns:production
```

### CI/CD Integration

Use in CI/CD pipelines to validate changes:

```bash
# Exit code 0 = no differences, 1 = differences found
kyt diff ./manifests ns:production
if [ $? -eq 1 ]; then
  echo "Differences detected between Git and cluster"
  exit 1
fi

# Write diff to file for artifact storage
kyt diff -o drift-report.txt ./manifests ns:production
```

## Troubleshooting

### Kubeconfig Not Found

```
Error: kubeconfig not found at /home/user/.kube/config

Troubleshooting:
- Check if KUBECONFIG environment variable is set correctly
- Ensure kubectl is configured: run 'kubectl config view'
- Default location is ~/.kube/config
```

**Solutions:**
- Set KUBECONFIG: `export KUBECONFIG=/path/to/kubeconfig`
- Initialize kubectl: `kubectl config view`
- Check file exists: `ls -la ~/.kube/config`

### Context Not Found

```
Error: context "prod" not found in kubeconfig

Troubleshooting:
- List available contexts: kubectl config get-contexts
- Check kubeconfig file: /home/user/.kube/config
- Ensure the context name is spelled correctly
```

**Solutions:**
- List contexts: `kubectl config get-contexts`
- Use correct context name from the list
- Check for typos in context name

### Namespace Not Found

```
Error: namespace "nonexistent" does not exist in cluster (context: gke-prod)
```

**Solutions:**
- List namespaces: `kubectl get namespaces --context gke-prod`
- Verify namespace name spelling
- Ensure you're using the correct context

### Permission Denied

Some resource types may be skipped if you lack RBAC permissions:

```bash
# Use verbose mode to see which types are skipped
kyt diff -v ns:production ns:staging
```

Output will show:
```
Skipped batch.k8s.io/jobs: forbidden: User cannot list jobs
```

**Solutions:**
- Check permissions: `kubectl auth can-i list jobs -n production`
- Contact cluster administrator for access
- Use `--include` to only fetch resource types you have access to

### Connection Errors

```
Error: failed to connect to cluster (context: prod): connection refused

Troubleshooting:
- Verify cluster is accessible: kubectl cluster-info --context prod
- Check network connectivity and VPN status
- Verify credentials are valid and not expired
```

**Solutions:**
- Test connection: `kubectl cluster-info --context prod`
- Check VPN connection if required
- Refresh credentials if expired (e.g., `gcloud auth login` for GKE)
- Verify kubeconfig is up to date

### Empty Results

If namespace comparison returns no resources:

```bash
# Check if resources exist in namespace
kubectl get all -n production

# Use verbose mode to see what's being fetched
kyt diff -v ns:production ns:staging

# Explicitly specify resource types
kyt diff --include deploy,svc,cm ns:production ns:staging
```

## Advanced Examples

### Compare Specific Application

```bash
# Use label selectors (requires pre-filtering with kubectl)
kubectl get deploy,svc,cm -n production -l app=myapp -o yaml > /tmp/prod-myapp.yaml
kyt diff /tmp/prod-myapp.yaml ns:staging

# Or compare full namespaces and use include filter
kyt diff --include deploy,svc,cm ns:production ns:staging
```

### Multi-Cluster Comparison

```bash
# Export from one cluster, compare with another
kubectl get all -n production --context cluster-a -o yaml > /tmp/cluster-a.yaml
kyt diff --context cluster-b /tmp/cluster-a.yaml ns:production
```

### Continuous Drift Detection

```bash
#!/bin/bash
# Monitor drift every hour
while true; do
  echo "Checking for drift at $(date)"
  if kyt diff --summary ./k8s/production ns:production; then
    echo "No drift detected"
  else
    echo "DRIFT DETECTED!"
    kyt diff ./k8s/production ns:production > drift-$(date +%Y%m%d-%H%M%S).txt
  fi
  sleep 3600
done
```

### Ignore Dynamic Fields

Use `.kyt.yaml` to ignore dynamic fields that change frequently:

```yaml
# .kyt.yaml
diff:
  ignoreDifferences:
    # Ignore Pod status and metadata changes
    - group: ""
      kind: "Pod"
      jsonPointers:
        - /status
        - /metadata/resourceVersion
        - /metadata/uid
        - /metadata/creationTimestamp
    
    # Ignore Service IPs that are auto-assigned
    - group: ""
      kind: "Service"
      jsonPointers:
        - /spec/clusterIP
        - /spec/clusterIPs
```

Then run:
```bash
kyt diff -c .kyt.yaml ns:production ns:staging
```

## Best Practices

1. **Use version control for configurations**: Store your `.kyt.yaml` in Git alongside manifests
2. **Filter to relevant resources**: Use `--include` to reduce noise and improve performance
3. **Automate drift detection**: Run comparisons in CI/CD or cron jobs
4. **Use verbose mode for debugging**: Add `-v` when troubleshooting connection issues
5. **Leverage ignore rules**: Configure `.kyt.yaml` to ignore expected differences
6. **Test context access**: Verify `kubectl` works before using kyt cluster features
7. **Document context names**: Keep track of which contexts map to which environments

## Integration with kubectl

kyt works seamlessly with kubectl:

```bash
# Export current state and compare
kubectl get all -n production -o yaml | kyt diff ./manifests -

# Compare rendered manifests before apply
kustomize build ./overlays/prod | tee /tmp/manifests.yaml | kubectl apply -f -
kyt diff /tmp/manifests.yaml ns:production  # Verify what was applied

# Pipe through kyt for formatting
kubectl get deploy -n production -o yaml | kyt fmt | kubectl apply -f -
```

## Security Considerations

- **Credentials**: kyt uses your kubectl credentials from `~/.kube/config`
- **RBAC**: Requires `list` permission on resource types in target namespaces
- **Secrets**: Consider using `--exclude secrets` to avoid displaying sensitive data
- **Audit logs**: Cluster access is logged by Kubernetes audit logs
- **Read-only**: kyt only performs read operations (list/get), never modifies resources

## Limitations

- **Resource types**: Fetches ~15 common types; custom resources (CRDs) not included
- **Single cluster per command**: Cannot compare across different clusters in one command (use two-step process)
- **No wildcard namespaces**: Must specify exact namespace names
- **Label selectors**: Not directly supported (use kubectl to pre-filter)
- **Field selectors**: Not directly supported (use kubectl to pre-filter)

## See Also

- [diff Command Guide](diff.md) - Complete diff command reference
- [Configuration Guide](../examples/.kyt.yaml) - Example configuration file
- [kubectl documentation](https://kubernetes.io/docs/reference/kubectl/) - kubectl reference

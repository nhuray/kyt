# Cluster Comparison Implementation Plan

## Overview

Add support for comparing live Kubernetes manifests from different namespaces using the syntax `kyt diff ns:namespace1 ns:namespace2`.

## Goals

- Enable direct comparison of resources running in Kubernetes clusters
- Support comparing namespace-to-namespace and namespace-to-files
- Reuse existing diff/normalize/report infrastructure
- Follow kubectl conventions for context and authentication
- Maintain fail-fast error handling for reliability

## Non-Goals

- Cross-cluster comparison in single command (use separate commands + file output)
- All-namespaces wildcard (`ns:*` or `ns:--all`)
- Custom server-side field cleaning (normalization handles this)

---

## Design Decisions

### 1. Resource Scope
**Decision: Option B** - Include all resources kubectl considers "all" (~15 resource types)

**Rationale:**
- Comprehensive comparison out of the box
- Use `--include`/`--exclude` flags to filter (already implemented in kyt)
- Matches user expectations from `kubectl get all`

**Resource types to include:**
```go
- Core/v1: pods, services, replicationcontrollers, configmaps, secrets, persistentvolumeclaims
- apps/v1: deployments, daemonsets, replicasets, statefulsets
- batch/v1: jobs, cronjobs
- networking.k8s.io/v1: ingresses
- autoscaling/v1: horizontalpodautoscalers
```

### 2. Context Handling
**Decision: Option A** - Single `--context` flag per command

**Rationale:**
- Simpler implementation and UX
- Follows kubectl conventions
- For cross-cluster comparison, users can run two commands and compare outputs

**Syntax:**
```bash
# Compare two namespaces in same cluster
kyt diff ns:prod ns:staging

# Use specific context
kyt diff ns:prod ns:staging --context my-cluster

# Compare namespace with files
kyt diff ns:prod ./manifests

# Cross-cluster (two-step process)
kyt diff ns:prod --context cluster1 -o /tmp/cluster1.yaml
kyt diff ns:prod --context cluster2 -o /tmp/cluster2.yaml
kyt diff /tmp/cluster1.yaml /tmp/cluster2.yaml
```

### 3. Namespace Validation
**Decision: Option A** - Fail immediately if namespace doesn't exist

**Rationale:**
- Clear error messages prevent confusion
- Fail-fast is more predictable than silent failures
- Typos are caught immediately

**Error example:**
```
Error: namespace "productin" does not exist in cluster (context: prod-cluster)
Did you mean: production?
```

### 4. RBAC & Permission Failures
**Decision: Option A** - Fail entire operation on permission errors

**Rationale:**
- Consistent with fail-fast philosophy
- Prevents incomplete/misleading diffs
- Users can address permissions explicitly

**Error example:**
```
Error: insufficient permissions to list deployments.apps in namespace "kube-system"
Required: get, list permissions on deployments.apps
```

### 5. Server-Side Field Cleaning
**Decision: Not needed** - Normalization already handles this

**Rationale:**
- kyt's normalizer already removes fields via `RemoveDefaultFields` config
- Server-side fields already in default removal list
- Consistent behavior: cluster resources normalized same as file resources

**No additional flags needed.**

### 6. Wildcard Namespaces
**Decision: Not supported** - Only exact namespace names

**Rationale:**
- Keeps implementation simple
- Large clusters could have hundreds of namespaces
- Users can script loops if needed: `for ns in $(kubectl get ns -o name); do ...; done`

---

## Architecture

### New Package: `pkg/cluster/`

```
pkg/cluster/
├── client.go          # ClusterClient implementation
├── resources.go       # Resource type definitions
├── namespace.go       # Namespace validation
└── client_test.go     # Unit tests
```

### Key Components

#### 1. ClusterClient

```go
type ClusterClient struct {
    client   dynamic.Interface
    context  string
}

// NewClusterClient creates a client for a specific context
func NewClusterClient(kubeconfigPath, contextName string) (*ClusterClient, error)

// GetResourcesInNamespace fetches all resources from a namespace
func (c *ClusterClient) GetResourcesInNamespace(
    namespace string,
    resourceTypes []schema.GroupVersionResource,
) ([]*unstructured.Unstructured, error)

// ValidateNamespace checks if namespace exists
func (c *ClusterClient) ValidateNamespace(namespace string) error
```

#### 2. Resource Type Registry

```go
// CommonResourceTypes returns GVRs for "all" resources
func CommonResourceTypes() []schema.GroupVersionResource

// FilterResourceTypes filters by include/exclude patterns
func FilterResourceTypes(
    all []schema.GroupVersionResource,
    include []string,
    exclude []string,
) ([]schema.GroupVersionResource, error)
```

#### 3. Integration with diff Command

```go
// Input represents either a file path or namespace
type Input struct {
    Type      string // "file" or "namespace"
    Value     string // path or namespace name
}

// ParseInput detects ns:* format
func ParseInput(arg string) Input

// LoadManifests loads from file or cluster
func LoadManifests(
    input Input,
    context string,
    includeKinds []string,
    excludeKinds []string,
) (*manifest.ManifestSet, error)
```

---

## Implementation Phases

### Phase 0: Dependencies
**Goal:** Add required dependencies

**Tasks:**
- [ ] Add `k8s.io/client-go@v0.36.0` to go.mod
- [ ] Run `go mod tidy`
- [ ] Verify no conflicts with existing dependencies

**Estimate:** 15 minutes

---

### Phase 1: Core Cluster Client
**Goal:** Create basic cluster connectivity

**Package:** `pkg/cluster/`

**Tasks:**
- [ ] Create `client.go` with `ClusterClient` struct
- [ ] Implement `NewClusterClient()` using clientcmd
  - Load kubeconfig from default location or KUBECONFIG env
  - Support explicit context selection
  - Return error if context not found
- [ ] Implement `GetResourcesInNamespace()`
  - Use dynamic.Interface to list resources
  - Handle each GVR separately
  - Collect all resources into slice
- [ ] Add error handling for:
  - Kubeconfig not found
  - Invalid context
  - Cluster connection failures
  - API errors

**Files:**
- `pkg/cluster/client.go` (new)
- `pkg/cluster/client_test.go` (new)

**Estimate:** 2-3 hours

---

### Phase 2: Resource Type Management
**Goal:** Define and filter resource types

**Package:** `pkg/cluster/`

**Tasks:**
- [ ] Create `resources.go`
- [ ] Implement `CommonResourceTypes()` returning:
  ```go
  []schema.GroupVersionResource{
      {Group: "", Version: "v1", Resource: "pods"},
      {Group: "", Version: "v1", Resource: "services"},
      {Group: "", Version: "v1", Resource: "configmaps"},
      {Group: "", Version: "v1", Resource: "secrets"},
      {Group: "", Version: "v1", Resource: "persistentvolumeclaims"},
      {Group: "apps", Version: "v1", Resource: "deployments"},
      {Group: "apps", Version: "v1", Resource: "daemonsets"},
      {Group: "apps", Version: "v1", Resource: "replicasets"},
      {Group: "apps", Version: "v1", Resource: "statefulsets"},
      {Group: "batch", Version: "v1", Resource: "jobs"},
      {Group: "batch", Version: "v1", Resource: "cronjobs"},
      {Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},
      {Group: "autoscaling", Version: "v1", Resource: "horizontalpodautoscalers"},
  }
  ```
- [ ] Implement `FilterResourceTypes()` using existing `pkg/resourcekind` logic
- [ ] Add tests for filtering

**Files:**
- `pkg/cluster/resources.go` (new)
- `pkg/cluster/resources_test.go` (new)

**Estimate:** 1-2 hours

---

### Phase 3: Namespace Validation
**Goal:** Validate namespace exists before fetching

**Package:** `pkg/cluster/`

**Tasks:**
- [ ] Create `namespace.go`
- [ ] Implement `ValidateNamespace()`:
  - Use client to check if namespace exists
  - Return clear error message if not found
  - Optionally suggest similar names (fuzzy match)
- [ ] Add error type `NamespaceNotFoundError`
- [ ] Add tests with fake client

**Files:**
- `pkg/cluster/namespace.go` (new)
- `pkg/cluster/namespace_test.go` (new)

**Estimate:** 1 hour

---

### Phase 4: diff Command Integration
**Goal:** Extend diff command to support ns: syntax

**Package:** `cmd/kyt/`

**Tasks:**
- [ ] Add `--context` flag to diff command:
  ```go
  var diffContext string
  diffCmd.Flags().StringVar(&diffContext, "context", "", 
      "Kubernetes context to use for cluster resources")
  ```
- [ ] Create input parser:
  - Detect `ns:namespace` format
  - Return input type (file/namespace) and value
- [ ] Create manifest loader:
  - Switch based on input type
  - For namespace: use ClusterClient
  - For file: use existing parser
- [ ] Update `runDiff()`:
  - Parse both source and target inputs
  - Load manifests based on type
  - Pass to existing differ
- [ ] Update command help text with examples
- [ ] Add error handling for:
  - Both inputs are namespaces but no cluster access
  - Context specified but no cluster inputs
  - Mixed cluster/file with wrong context

**Files:**
- `cmd/kyt/diff.go` (modify)
- `cmd/kyt/cluster_loader.go` (new - helper functions)

**Estimate:** 2-3 hours

---

### Phase 5: Error Messages & UX
**Goal:** Provide clear, actionable error messages

**Tasks:**
- [ ] Add helpful error messages for common issues:
  - `Error: namespace "prod" does not exist in context "my-cluster"`
  - `Error: no Kubernetes context specified (use --context flag)`
  - `Error: failed to connect to cluster: context "invalid" not found`
  - `Error: insufficient permissions to list pods in namespace "kube-system"`
- [ ] Add verbose logging for cluster operations:
  - `Connecting to cluster (context: my-cluster)...`
  - `Validating namespace "prod"...`
  - `Fetching resources from namespace "prod"...`
  - `Found 47 resources (12 deployments, 15 pods, ...)`
- [ ] Handle edge cases:
  - Empty namespace (no resources)
  - Timeout connecting to cluster
  - Network errors during resource fetch

**Files:**
- `cmd/kyt/diff.go` (modify)
- `pkg/cluster/errors.go` (new)

**Estimate:** 1-2 hours

---

### Phase 6: Testing
**Goal:** Comprehensive test coverage

**Tasks:**
- [ ] Unit tests for `pkg/cluster/`:
  - Client creation with valid/invalid context
  - Resource listing
  - Namespace validation
  - Resource filtering
- [ ] Integration tests for diff command:
  - `ns:foo ns:bar` syntax
  - `ns:foo ./manifests` mixed syntax
  - `--context` flag handling
  - Error cases (namespace not found, no context, etc.)
- [ ] Add test fixtures:
  - Fake kubeconfig with test contexts
  - Mock cluster data
- [ ] Manual testing with real cluster:
  - Test with minikube or kind
  - Verify resource listing
  - Test permission errors
  - Test network failures

**Files:**
- `pkg/cluster/*_test.go` (new)
- `cmd/kyt/diff_cluster_test.go` (new)
- `testdata/kubeconfig` (new)

**Estimate:** 3-4 hours

---

### Phase 7: Documentation
**Goal:** Document new feature

**Tasks:**
- [ ] Update `README.md`:
  - Add cluster comparison to features list
  - Add examples section
- [ ] Update `docs/diff.md`:
  - Add "Cluster Comparison" section
  - Document `ns:*` syntax
  - Document `--context` flag
  - Add examples
- [ ] Update command help text:
  - Add examples to `kyt diff --help`
- [ ] Add troubleshooting section:
  - Context not found
  - Namespace not found
  - Permission errors
  - Network timeouts

**Files:**
- `README.md` (modify)
- `docs/diff.md` (modify)
- `docs/CLUSTER_COMPARISON.md` (new)
- `cmd/kyt/diff.go` (modify help text)

**Estimate:** 1-2 hours

---

## Total Effort Estimate

- **Phase 0:** 15 minutes
- **Phase 1:** 2-3 hours
- **Phase 2:** 1-2 hours
- **Phase 3:** 1 hour
- **Phase 4:** 2-3 hours
- **Phase 5:** 1-2 hours
- **Phase 6:** 3-4 hours
- **Phase 7:** 1-2 hours

**Total: 11-17 hours**

**MVP (Phases 0-4):** ~6-9 hours

---

## Example Usage

### Compare Two Namespaces
```bash
# Compare production and staging in current context
kyt diff ns:production ns:staging

# Compare with specific context
kyt diff ns:production ns:staging --context my-cluster

# Show only summary
kyt diff ns:production ns:staging --summary

# Filter by resource types
kyt diff ns:production ns:staging --include deploy,svc,cm
kyt diff ns:production ns:staging --exclude secrets
```

### Compare Namespace with Files
```bash
# Compare cluster state with local manifests
kyt diff ns:production ./manifests

# Compare with built manifests
kustomize build ./overlay/prod | kyt diff ns:production -

# Compare with Helm output
helm template my-app | kyt diff ns:production -
```

### Cross-Cluster Comparison
```bash
# Two-step process for different clusters
kyt diff ns:app --context cluster1 -o /tmp/cluster1.yaml
kyt diff ns:app --context cluster2 -o /tmp/cluster2.yaml
kyt diff /tmp/cluster1.yaml /tmp/cluster2.yaml
```

---

## Error Handling Examples

### Namespace Not Found
```bash
$ kyt diff ns:productin ns:staging
Error: namespace "productin" does not exist in cluster (context: my-cluster)

Available namespaces:
  - production
  - staging
  - development

Did you mean: production?
```

### No Context Specified
```bash
$ kyt diff ns:prod ns:staging
Error: Kubernetes context not specified

Use --context flag to specify the cluster context:
  kyt diff ns:prod ns:staging --context my-cluster

Or set the default context:
  kubectl config use-context my-cluster
```

### Permission Denied
```bash
$ kyt diff ns:kube-system ns:default
Error: insufficient permissions to list resources in namespace "kube-system"

Required RBAC permissions:
  - get, list on pods
  - get, list on deployments.apps
  - get, list on services
  ...

Contact your cluster administrator to grant access.
```

### Connection Failed
```bash
$ kyt diff ns:prod ns:staging --context my-cluster
Error: failed to connect to cluster (context: my-cluster)

Details: dial tcp 10.0.0.1:6443: i/o timeout

Troubleshooting:
  - Verify cluster is accessible: kubectl cluster-info --context my-cluster
  - Check your network connection
  - Verify kubeconfig is up to date
```

---

## Testing Strategy

### Unit Tests
- Client creation with valid/invalid kubeconfig
- Resource listing with mocked responses
- Namespace validation
- Resource type filtering
- Error handling

### Integration Tests
- End-to-end with fake Kubernetes cluster
- Test all syntax variations
- Verify error messages
- Test with include/exclude filters

### Manual Tests
- Real cluster (minikube/kind)
- Various resource types
- Permission scenarios
- Network failure scenarios
- Large namespaces (performance)

---

## Security Considerations

1. **Credentials**
   - Use standard kubeconfig (~/.kube/config)
   - Respect KUBECONFIG environment variable
   - Never log credentials or tokens

2. **Permissions**
   - Fail gracefully on RBAC denials
   - Don't mask permission errors
   - Clear messages about required permissions

3. **Network**
   - Use TLS for all cluster connections
   - Respect certificate validation
   - Support proxy configurations

4. **Audit**
   - Respect cluster audit logs
   - Read-only operations only (GET/LIST)
   - No mutation of cluster resources

---

## Future Enhancements (Not in MVP)

### Discovery API Support
- Auto-discover available resource types
- Include CRDs automatically
- `--discover` flag to enable

### Cross-Cluster in Single Command
- Support `ns:foo:context1 ns:bar:context2` syntax
- More complex but convenient

### Watch Mode
- Continuous comparison with live cluster
- Detect drift in real-time
- `kyt diff ns:prod ./manifests --watch`

### Resource Caching
- Cache cluster resources for faster repeated diffs
- `--cache` flag with TTL
- Useful for CI/CD

### All Namespaces
- Support `ns:--all` or `ns:*`
- Compare all namespaces at once
- Group by namespace in output

---

## Dependencies

### New Dependencies
```
k8s.io/client-go@v0.36.0
```

### Existing Dependencies (No Change)
```
k8s.io/apimachinery@v0.36.0  (already present)
```

---

## Risks & Mitigations

### Risk 1: Large Namespaces
**Impact:** Slow performance with 1000+ resources

**Mitigation:**
- Use pagination in List calls
- Add progress indicator for verbose mode
- Consider adding `--limit` flag in future

### Risk 2: RBAC Complexity
**Impact:** Users may not have full access

**Mitigation:**
- Fail fast with clear error messages
- Document required permissions
- Provide troubleshooting guide

### Risk 3: Network Timeouts
**Impact:** Unreliable on slow/flaky networks

**Mitigation:**
- Use default client-go timeouts (30s)
- Allow configuration via environment variables
- Clear timeout error messages

### Risk 4: Version Skew
**Impact:** Different Kubernetes versions have different resources

**Mitigation:**
- Use discovery API (future enhancement)
- Document supported Kubernetes versions (1.19+)
- Gracefully skip unknown resource types

---

## Success Criteria

### MVP Success
- [ ] Can compare two namespaces in same cluster
- [ ] Can compare namespace with file/directory
- [ ] Respects `--context` flag
- [ ] Uses existing `--include`/`--exclude` flags
- [ ] Proper error handling (fail-fast)
- [ ] All tests passing
- [ ] Documentation complete

### Full Success
- [ ] All MVP criteria met
- [ ] Comprehensive test coverage (>80%)
- [ ] Real-world testing with various clusters
- [ ] Clear error messages for all failure modes
- [ ] Performance acceptable (<5s for 100 resources)
- [ ] Documentation with troubleshooting guide

---

## Open Questions

None - all decisions made based on requirements:
1. ✅ Resource scope: Option B (all kubectl resources)
2. ✅ Context handling: Option A (single context)
3. ✅ Namespace validation: Option A (fail-fast)
4. ✅ RBAC failures: Option A (fail)
5. ✅ Server-side fields: Handled by normalization
6. ✅ Wildcards: Not supported

---

## Next Steps

1. Review and approve this plan
2. Create feature branch: `feat/cluster-comparison`
3. Start with Phase 0 (dependencies)
4. Implement phases sequentially
5. Create PR after Phase 4 (MVP) for early feedback
6. Complete remaining phases
7. Final PR with full feature

---

**Plan Version:** 1.0  
**Created:** 2026-05-02  
**Author:** OpenCode  
**Status:** Ready for Implementation

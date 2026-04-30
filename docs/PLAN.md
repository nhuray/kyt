# k8s-diff Implementation Plan

## Project Overview

**Goal:** Build a Go CLI tool that compares Kubernetes manifests using ArgoCD-compatible ignore rules (JSON Pointers & JQ expressions) and outputs differences using difftastic.

**Repository:** `/Users/nicolas/Workspace/k8s-diff`

**GitHub:** `github.com/nhuray/k8s-diff`

---

## Project Structure

```
k8s-diff/
├── .gitignore
├── .github/
│   └── workflows/
│       ├── test.yml           # CI testing
│       └── release.yml        # Binary releases
├── README.md
├── LICENSE
├── Makefile
├── go.mod
├── go.sum
├── cmd/
│   └── k8s-diff/
│       └── main.go            # CLI entry point
├── pkg/
│   ├── config/
│   │   ├── config.go          # Config loading
│   │   ├── config_test.go
│   │   └── types.go           # Config types
│   ├── manifest/
│   │   ├── parser.go          # YAML parsing
│   │   ├── parser_test.go
│   │   └── types.go
│   ├── normalizer/
│   │   ├── normalizer.go      # Uses ArgoCD normalizers
│   │   ├── normalizer_test.go
│   │   ├── jq.go              # JQ expression support
│   │   └── types.go
│   ├── differ/
│   │   ├── differ.go          # Diff orchestration
│   │   ├── differ_test.go
│   │   ├── similarity.go      # Similarity scoring (Phase 8.5)
│   │   ├── similarity_test.go
│   │   ├── matcher.go         # Resource matching (Phase 8.5)
│   │   ├── matcher_test.go
│   │   └── types.go
│   └── reporter/
│       ├── cli.go             # CLI output formatting
│       ├── json.go            # JSON output
│       ├── reporter_test.go
│       └── types.go
│       ├── html.go            # HTML output (diff2html)
│       └── json.go            # JSON output
├── examples/
│   ├── .k8s-diff.yaml         # Example config
│   ├── source.yaml            # Example source manifest
│   └── target.yaml            # Example target manifest
└── docs/
    ├── PLAN.md                # This file
    ├── configuration.md       # Config documentation
    ├── usage.md              # Usage examples
    └── architecture.md       # Architecture overview
```

---

## Phase 1: Project Setup & Structure (Day 1) ✅

### 1.1 Initialize Repository ✅

- [x] Create directory structure
- [x] Initialize Git repository
- [x] Create `.gitignore` for Go projects
- [x] Initialize Go module (`go mod init github.com/nhuray/k8s-diff`)
- [x] Create initial `README.md` with project description
- [ ] Add `LICENSE` file (MIT)

### 1.2 Initial Dependencies ✅

```bash
# Core dependencies installed (v2.14.21)
go get github.com/argoproj/argo-cd/v2@latest
go get github.com/itchyny/gojq@latest           # v0.12.19
go get github.com/spf13/cobra@latest            # v1.10.2
go get gopkg.in/yaml.v3@latest
go get k8s.io/apimachinery@latest               # v0.36.0

# Testing dependencies
go get github.com/stretchr/testify@latest
```

---

## Phase 2: Core Manifest Handling (Day 2-3) ✅

### 2.1 Manifest Parser (`pkg/manifest/`) ✅

**Goals:**

- Parse single YAML files
- Parse multi-document YAML (separated by `---`)
- Parse directories of YAML files
- Split resources by GVK (Group/Version/Kind), namespace, and name
- Convert to `unstructured.Unstructured` (K8s native type)

**Tasks:**

- [x] Define types in `types.go`
- [x] Implement `ParseFile(path string) ([]*unstructured.Unstructured, error)`
- [x] Implement `ParseDirectory(dir string) ([]*unstructured.Unstructured, error)`
- [x] Implement `ParseMultiDoc(content []byte) ([]*unstructured.Unstructured, error)`
- [x] Add resource key generation (`kind/namespace/name`)
- [x] Write comprehensive tests with example manifests
- [x] Handle edge cases (empty files, invalid YAML, etc.)

**Implementation Notes:**

- Created `ResourceKey` struct with Group/Version/Kind/Namespace/Name
- Created `ManifestSet` for indexed resource collections
- Implemented `Parser` with methods: ParseFile, ParseBytes, ParseDirectory, ParseReader, ParseFiles
- Added `SkipInvalid` mode for resilient parsing
- All 10 tests passing with comprehensive coverage

**Key Types:**

```go
// pkg/manifest/types.go
type ResourceKey struct {
    Group     string
    Kind      string
    Namespace string
    Name      string
}

type ManifestSet struct {
    Resources map[ResourceKey]*unstructured.Unstructured
}

func (rk ResourceKey) String() string {
    if rk.Namespace != "" {
        return fmt.Sprintf("%s/%s/%s/%s", rk.Group, rk.Kind, rk.Namespace, rk.Name)
    }
    return fmt.Sprintf("%s/%s/%s", rk.Group, rk.Kind, rk.Name)
}
```

### 2.2 Key Sorting & Normalization ✅

**Goals:**

- Sort object keys alphabetically (for consistent diffs)
- Sort arrays where order doesn't matter (e.g., container ports)
- Remove fields that shouldn't be compared (status, managedFields)

**Tasks:**

- [x] Implement key sorting for JSON objects
- [x] Implement configurable array sorting (configuration structure in place)
- [x] Basic normalization (remove status, managedFields by default)
- [x] Write tests comparing before/after normalization

**Note:** Implemented in Phase 4 as part of the normalizer (pkg/normalizer/normalizer.go).

---

## Phase 3: Configuration System (Day 3-4) ✅

### 3.1 Config File Structure (`pkg/config/`) ✅

**Goals:**

- Load `.k8s-diff.yaml` configuration
- Support ArgoCD-compatible `ignoreDifferences` format
- Support multiple config files (merge rules)
- Validate configuration

**Config Format:**

```yaml
# .k8s-diff.yaml
ignoreDifferences:
  # Simple ignores using JSON Pointers
  - group: ""
    kind: "*"
    jsonPointers:
      - /metadata/labels
      - /metadata/annotations

  # Complex ignores using JQ expressions
  - group: "apps"
    kind: "Deployment"
    name: "" # Empty means all deployments
    namespace: "" # Empty means all namespaces
    jqPathExpressions:
      - .spec.template.spec.containers[] | select(.name == "istio-proxy")
      - .spec.template.spec.initContainers[] | select(.name == "istio-init")

  # Managed fields managers (for server-side apply scenarios)
  - group: "apps"
    kind: "Deployment"
    managedFieldsManagers:
      - "kube-controller-manager"

# Normalization options
normalization:
  sortKeys: true
  sortArrays:
    - path: ".spec.template.spec.containers[].ports"
      sortBy: "containerPort"
    - path: ".spec.template.spec.containers[].env"
      sortBy: "name"

# Output options
output:
  format: cli # cli, json, diff, html
  diffTool: difft # difft, diff, or none
  colorize: true
```

**Tasks:**

- [x] Define Go structs matching config schema in `types.go`
- [x] Implement config loading in `loader.go`
- [x] Implement config validation in `validator.go`
- [x] Support multiple config files (LoadMultiple)
- [x] Support config merging (Merge method)
- [x] Write comprehensive tests for config loading, merging, and validation
- [x] Create example config files (default, minimal, helm-migration)

**Implementation Notes:**

- Created Config, ResourceIgnoreDifferences, NormalizationConfig, OutputConfig types
- Implemented Loader with methods: Load, LoadBytes, LoadDefault, LoadMultiple, SearchConfig, Save
- Implemented Validator with validation for JSON Pointers (RFC 6901) and JQ expressions
- Added MatchesResource method for resource matching with glob pattern support
- Created 3 example config files showcasing different use cases
- All 14 test cases passing with comprehensive coverage

**Key Types:**

```go
// pkg/config/types.go
type Config struct {
    IgnoreDifferences []ResourceIgnoreDifferences `yaml:"ignoreDifferences"`
    Normalization     NormalizationConfig         `yaml:"normalization"`
    Output            OutputConfig                `yaml:"output"`
}

type ResourceIgnoreDifferences struct {
    Group                 string   `yaml:"group"`
    Kind                  string   `yaml:"kind"`
    Name                  string   `yaml:"name,omitempty"`
    Namespace             string   `yaml:"namespace,omitempty"`
    JSONPointers          []string `yaml:"jsonPointers,omitempty"`
    JQPathExpressions     []string `yaml:"jqPathExpressions,omitempty"`
    ManagedFieldsManagers []string `yaml:"managedFieldsManagers,omitempty"`
}

type NormalizationConfig struct {
    SortKeys   bool               `yaml:"sortKeys"`
    SortArrays []ArraySortConfig  `yaml:"sortArrays"`
}

type ArraySortConfig struct {
    Path   string `yaml:"path"`
    SortBy string `yaml:"sortBy"`
}

type OutputConfig struct {
    Format   string `yaml:"format"`   // cli, json, diff, html
    DiffTool string `yaml:"diffTool"` // difft, diff
    Colorize bool   `yaml:"colorize"`
}
```

---

## Phase 4: Ignore Rules Engine (Day 4-6) ✅

### 4.1 Normalizer Implementation ✅

**Goals:**

- Implement custom normalizer for JSON Pointers and JQ expressions
- Match resources based on group/kind/name/namespace (with glob support)
- Apply ignore rules to manifests
- Sort keys and remove default fields

**Tasks:**

- [x] Study ArgoCD's normalizer approach
- [x] Create types in `types.go` (Normalizer, NormalizeResult)
- [x] Implement core normalization in `normalizer.go`
- [x] Implement JQ expression support in `jq.go` using gojq
- [x] Implement JSON Pointer field removal (RFC 6901)
- [x] Implement managed fields filtering by manager
- [x] Support key sorting for consistent diffs
- [x] Write comprehensive tests with various ignore scenarios

**Implementation Notes:**

- Created custom normalizer (not using ArgoCD's internal types for better control)
- Implemented JSON Pointer parsing with proper escape sequence handling (~0, ~1)
- JQ expression support for common patterns:
  - Container removal: `.spec.template.spec.containers[] | select(.name == "istio-proxy")`
  - Init container removal: `.spec.template.spec.initContainers[] | select(.name == "istio-init")`
  - Volume removal: `.spec.template.spec.volumes[] | select(.name | startswith("istio-"))`
- Recursive key sorting for nested objects
- ManagedFields filtering by manager name
- All operations work on deep copies to preserve original objects

### 4.2 Testing Results ✅

**Test Cases Implemented:**

- [x] Basic normalization with field removal
- [x] JSON Pointer removes simple fields: `/metadata/labels`
- [x] JSON Pointer with escaping: `/metadata/annotations/kubectl.kubernetes.io~1last-applied-configuration`
- [x] JSON Pointer parsing with ~0 and ~1 escapes
- [x] JQ expression removes istio-proxy container
- [x] Managed fields filtering by manager
- [x] Key sorting (verified through JSON output)
- [x] Resource matching with ignore rules
- [x] Batch normalization (NormalizeAll)
- [x] Non-existent fields handled gracefully (no errors)

All 8 tests passing (0.476s)

---

## Phase 5: Diff Engine (Day 6-7) ✅

### 5.1 Differ Implementation (`pkg/differ/`) ✅

**Goals:**

- Match resources between source and target
- Identify added, removed, and modified resources
- Generate structured diff results
- Integrate with external diff tools (difftastic)

**Tasks:**

- [x] Define types in `types.go`
- [x] Implement resource matching algorithm in `differ.go`
- [x] Detect added resources (in target, not in source)
- [x] Detect removed resources (in source, not in target)
- [x] Detect modified resources (in both, but different)
- [x] Write normalized manifests to temp files
- [x] Shell out to difftastic for comparison
- [x] Fallback to unified diff if difftastic not available
- [x] Write tests with various diff scenarios

**Implementation Notes:**

- Created DiffResult type with Added, Removed, Modified, Identical slices
- Implemented resource matching by ResourceKey (group/version/kind/namespace/name)
- Normalizes both source and target before comparison
- Compares resources using JSON equality
- Generates diffs using difftastic (with fallback to unified diff)
- Supports difftastic display modes and color output
- Creates temporary files for diff generation
- Properly handles difftastic exit codes (exit 1 on differences is expected)

### 5.2 Difftastic Integration ✅

**Options Supported:**

- Display modes: `side-by-side`, `side-by-side-show-both`, `inline`
- Color output: configurable via DiffOptions
- Fallback to unified diff when difftastic unavailable

**Tasks:**

- [x] Check if `difft` binary is available
- [x] Generate temp files for comparison
- [x] Execute `difft` with appropriate flags
- [x] Capture and parse output
- [x] Handle errors (difft not found, etc.)
- [x] Clean up temp files
- [x] Fallback to unified diff command

**Test Results:**

- 7 comprehensive tests covering:
  - Added resources detection
  - Removed resources detection
  - Modified resources detection with diff generation
  - Identical resources detection
  - Mixed changes (added + removed + modified + identical)
  - HasDifferences() helper method
  - Resource equality comparison
- All tests passing (0.520s)

---

## Phase 6: Output Formatters (Day 7-8) ✅

### 6.1 CLI Output (Default) ✅

**Goals:**

- Human-readable output using difftastic
- Color support (when TTY)
- Summary statistics

**Tasks:**

- [x] Implement CLI reporter using difftastic output in `cli.go`
- [x] Add summary header/footer
- [x] Support color output with TTY detection
- [x] Write tests for output formatting

**Implementation Notes:**

- Created Reporter interface for pluggable output formats
- CLI reporter with colored output and TTY detection
- Summary header shows report title
- Sections for Added (green), Removed (red), Modified (yellow), Identical (gray)
- Modified resources show full diff text from difftastic/unified diff
- Summary footer with statistics and status message
- ANSI color codes with automatic TTY detection

### 6.2 JSON Output ✅

**Goals:**

- Machine-readable JSON output
- Structured diff information
- CI/CD friendly

**Tasks:**

- [x] Define JSON schema for diff results in `json.go`
- [x] Implement JSON serialization
- [x] Write tests

**JSON Format Implemented:**

```json
{
  "summary": {
    "totalResources": 10,
    "added": 1,
    "removed": 0,
    "modified": 2,
    "identical": 7
  },
  "added": [
    {
      "group": "apps",
      "version": "v1",
      "kind": "Service",
      "name": "new-service",
      "namespace": "default"
    }
  ],
  "removed": [],
  "modified": [
    {
      "key": {
        "group": "apps",
        "version": "v1",
        "kind": "Deployment",
        "name": "app",
        "namespace": "default"
      },
      "diff": "... diff text ...",
      "diffLines": 42
    }
  ]
}
```

**Implementation Notes:**

- JSONOutput structure with summary and resource arrays
- Compact and pretty-print modes
- Optional inclusion of identical resources
- Full resource key information (group/version/kind/namespace/name)
- Diff text and line count included for modified resources

### 6.3 Unified Diff Output ✅

**Note:** This was implemented as part of Phase 5 (differ.go) as a fallback when difftastic is not available. The differ automatically uses unified diff when difftastic is unavailable or disabled.

### 6.4 HTML Output

**Status:** Deferred - Not implemented in MVP

**Rationale:** HTML output using diff2html would require additional dependencies and complexity. Users can pipe JSON output to other tools if needed. May be added in a future release.

**Test Results:**

- 6 comprehensive tests covering:
  - CLI output with and without colors
  - CLI output with/without identical resources
  - No differences message
  - Differences detected message
  - JSON compact and pretty modes
  - JSON with/without identical resources
  - Resource key conversion
  - Empty result handling
- All tests passing (0.498s)

---

## Phase 7: CLI Implementation (Day 8-9) ✅

### 7.1 Command Structure ✅

**Main Command:**

```bash
k8s-diff <source> <target> [flags]
```

**Flags:**

```
--config, -c         Config file path (default: .k8s-diff.yaml)
--output, -o         Output format: cli, json (default: cli)
--display            Difftastic display mode: side-by-side, inline (default: side-by-side)
--diff-tool          Diff tool: difft, diff (default: difft)
--no-color           Disable colored output
--show-identical     Show identical resources in output
--verbose, -v        Verbose output
--version            Show version
--help, -h           Show help
```

### 7.2 Implementation Tasks ✅

**Using Cobra:**

- [x] Set up root command with Cobra in `main.go`
- [x] Add all flags with proper types and defaults
- [x] Add command-line help text
- [x] Handle errors gracefully with exit codes
- [x] Wire up all packages (config, manifest, normalizer, differ, reporter)
- [x] Add version command
- [x] Fix int64 conversion issue (YAML parser creates int, Kubernetes expects int64)

**Exit Codes:**

```
0 - Success (no differences or differences ignored)
1 - Differences found
2 - Error (invalid config, missing files, etc.)
```

---

## Phase 8: Testing & Examples (Day 9-10) ✅

### 8.1 Unit Tests ✅

**Coverage Goals: 80%+**

- [x] Config loading and validation (14 tests)
- [x] Manifest parsing (single, multi-doc, directory) (10 tests)
- [x] Normalization (key sorting, array sorting) (8 tests)
- [x] Ignore rules (JSON Pointers, JQ expressions) (8 tests)
- [x] Resource matching and diffing (7 tests)
- [x] Output formatters (6 tests)

**Total: 45 unit tests across 5 packages - all passing**

### 8.2 Integration Tests ✅

**Test Scenarios:**

- [x] Compare two identical manifests (no diff) - exit code 0
- [x] Compare different manifests - exit code 1, shows added/removed
- [x] Test JSON output format
- [x] Test --show-identical flag
- [x] Test error handling (invalid YAML, missing files) - exit code 2
- [x] Test version command

**Total: 7 integration tests - all passing**

**Grand Total: 52 tests passing**

### 8.3 Example Manifests ✅

**Existing Examples:**

- [x] Basic manifests (service.yaml, deployment.yaml)
- [x] Multi-doc manifests (all-in-one.yaml)
- [x] Example config files:
  - `.k8s-diff.yaml` - Full-featured with ignore rules
  - `.k8s-diff.minimal.yaml` - Minimal configuration
  - `.k8s-diff.helm-migration.yaml` - Helm-to-Kustomize migration

---

## Phase 8.5: Similarity Matching (Smart Resource Pairing)

### 8.5.1 Problem Statement

**Current limitation:** Resources are only compared when they have exact `(group, kind, namespace, name)` matches. If a resource is renamed (e.g., `app-config-v1` → `app-config-v2`), it's reported as "removed + added" instead of "modified".

**Real-world scenarios:**

- Helm release name changes: `myapp-prod` → `myapp-production`
- Version-suffixed resources: `redis-v1` → `redis-v2`
- Environment-specific naming: `app-dev-config` → `app-staging-config`
- Renamed deployments during refactoring

**Goal:** Intelligently match resources by structural similarity when exact name matching fails.

### 8.5.2 Design: Multi-Stage Matching

#### Stage 1: Exact Match (Current behavior - keep as-is)

- Match resources by full ResourceKey: `(group, kind, namespace, name)`
- This is the primary, fast path
- **Performance:** O(n) - hash map lookup

#### Stage 2: Similarity-Based Matching (New)

**For unmatched resources:**

1. **Group by GVK+Namespace:** `(group, version, kind, namespace)`
   - Only compare resources of the same type in the same namespace
   - Example: All unmatched ConfigMaps in `default` namespace

2. **Case A: 1-to-1 Mapping (Simple)**
   - If exactly 1 unmatched resource on each side → compare them
   - If similarity ≥ threshold → mark as "modified"
   - Example: 1 orphan ConfigMap source, 1 orphan ConfigMap target

3. **Case B: Many-to-Many Mapping (Complex)**
   - If multiple unmatched resources → compute pairwise similarity matrix
   - Use greedy matching: pair highest similarity first
   - Only pair if similarity ≥ threshold
   - Example: 3 ConfigMaps source, 2 ConfigMaps target

4. **Case C: Remaining Unpaired**
   - Resources that don't match → report as added/removed

**Performance:** O(n×m) per GVK group where n, m = unmatched resources

### 8.5.3 Similarity Algorithm: Hierarchical Structural Comparison

**Chosen approach:** Recursive comparison with weighted scoring

**Key design decisions:**

- ✅ **Compare `spec` field** - Labels/annotations likely differ or are ignored
- ✅ **Element-by-element array matching** - Normalization sorts arrays first
- ✅ **Treat all fields equally** - No field-specific weights (KISS principle)
- ✅ **Fall back to full object** - If `spec` missing, compare entire object minus metadata
- ✅ **Short-circuit optimization** - Skip deep comparison if top-level similarity < 0.3

**Scoring weights:**

```
ExactMatchScore    = 1.0   // Exact key+value match
StructuralMatch    = 0.8   // Recursive match for nested objects
KeyOnlyMatch       = 0.3   // Key exists but value differs
ArrayMatch         = 0.7   // Arrays compared element-by-element
MissingFieldScore  = 0.0   // Key missing in one side
```

**Example calculation:**

```yaml
# Source spec
replicas: 1 # Different value: +0.3
image: redis:7.0 # Different value: +0.3
ports: # Array with same elements: +0.7
  - containerPort: 6379
resources: # Nested object with exact match: +0.8
  limits:
    cpu: 500m

# Similarity = (0.3 + 0.3 + 0.7 + 0.8) / 4 = 0.525
```

### 8.5.4 Implementation Tasks

**New Files:**

- [ ] `pkg/differ/similarity.go` - Similarity scoring algorithm
  - `SimilarityScorer` struct with configurable weights
  - `CompareSpecs(a, b map[string]interface{}) float64`
  - `compareValues()` - recursive comparison dispatcher
  - `compareObjects()` - map comparison
  - `compareArrays()` - element-by-element array comparison
  - Short-circuit optimization for performance

- [ ] `pkg/differ/matcher.go` - Resource matching logic
  - `findExactMatches()` - Stage 1: current exact matching
  - `groupByGVKNamespace()` - Group unmatched resources
  - `findSimilarityMatches()` - Stage 2: similarity matching
  - `findBestMatches()` - Greedy pairing for many-to-many
  - `Match` struct with `MatchType` ("exact" or "similarity")

**Modified Files:**

- [ ] `pkg/differ/differ.go`
  - Update `Diff()` to use 2-stage matching
  - Pass similarity matches to diff generation

- [ ] `pkg/differ/types.go`
  - Add `MatchType` field to `ResourceDiff`: "exact" | "similarity"
  - Add `SimilarityScore` field to `ResourceDiff`
  - Update `DiffOptions` to include:
    - `EnableSimilarityMatching bool` (default: true)
    - `SimilarityThreshold float64` (default: 0.7)

- [ ] `pkg/reporter/cli.go`
  - Update output format to show similarity score
  - Format: `ConfigMap.core/app-v1 → ConfigMap.core/app-v2 (similarity: 0.85)`
  - Format: `Deployment.apps/redis (exact match)` or just current format

- [ ] `pkg/reporter/json.go`
  - Add `matchType` field to modified resources in JSON output
  - Add `similarityScore` field
  - Add `sourceKey` and `targetKey` to show name mapping

- [ ] `cmd/k8s-diff/main.go`
  - Add `--exact-match` flag to disable similarity matching
  - Add `--similarity-threshold` flag (default: 0.7)
  - Wire up flags to `DiffOptions`

- [ ] `pkg/config/types.go`
  - Add similarity matching configuration:
    ```yaml
    matching:
      enableSimilarity: true
      threshold: 0.7
    ```

**Tests:**

- [ ] `pkg/differ/similarity_test.go` - Algorithm tests
  - Test exact matches return 1.0
  - Test completely different specs return ~0.0
  - Test partial matches return intermediate scores
  - Test nested object comparison
  - Test array comparison (sorted)
  - Test primitive value differences
  - Test nil/missing field handling
  - Benchmark for performance

- [ ] `pkg/differ/matcher_test.go` - Matching logic tests
  - Test exact matching (preserve existing behavior)
  - Test 1-to-1 similarity matching
  - Test many-to-many greedy pairing
  - Test threshold filtering
  - Test grouping by GVK+namespace

- [ ] `cmd/k8s-diff/integration_test.go` - End-to-end tests
  - Test renamed resources matched by similarity
  - Test --exact-match flag disables similarity
  - Test --similarity-threshold configuration
  - Test output format includes similarity scores
  - Test JSON output includes match metadata

### 8.5.5 Configuration & CLI

**CLI Flags:**

```bash
# Disable similarity matching (use only exact name matching)
k8s-diff --exact-match source.yaml target.yaml

# Adjust similarity threshold (0.0 to 1.0)
k8s-diff --similarity-threshold 0.8 source.yaml target.yaml

# Verbose mode shows matching decisions
k8s-diff -v source.yaml target.yaml
# Output: "Matched ConfigMap/app-v1 with ConfigMap/app-v2 (similarity: 0.85)"
```

**Config File:**

```yaml
# .k8s-diff.yaml
matching:
  enableSimilarity: true # Can disable here too
  threshold: 0.7 # Require 70% similarity minimum
```

### 8.5.6 Expected Output Examples

#### CLI Output (with similarity):

```
Modified Resources (3):

───────────────────────────────────────────────────────────────
• ConfigMap.core/app-config-v1 → ConfigMap.core/app-config-v2 (similarity: 0.85)
───────────────────────────────────────────────────────────────
[diff output]

───────────────────────────────────────────────────────────────
• Deployment.apps/redis (exact match)
───────────────────────────────────────────────────────────────
[diff output]

───────────────────────────────────────────────────────────────
• Service.core/redis-svc → Service.core/redis-service (similarity: 0.92)
───────────────────────────────────────────────────────────────
[diff output]
```

#### JSON Output (with similarity metadata):

```json
{
  "modified": [
    {
      "sourceKey": {
        "group": "",
        "kind": "ConfigMap",
        "namespace": "default",
        "name": "app-config-v1"
      },
      "targetKey": {
        "group": "",
        "kind": "ConfigMap",
        "namespace": "default",
        "name": "app-config-v2"
      },
      "matchType": "similarity",
      "similarityScore": 0.85,
      "diffText": "..."
    },
    {
      "sourceKey": {
        "group": "apps",
        "kind": "Deployment",
        "namespace": "default",
        "name": "redis"
      },
      "targetKey": {
        "group": "apps",
        "kind": "Deployment",
        "namespace": "default",
        "name": "redis"
      },
      "matchType": "exact",
      "similarityScore": 1.0,
      "diffText": "..."
    }
  ]
}
```

### 8.5.7 Performance Considerations

**Optimization strategies:**

1. **Early exit:** If top-level keys similarity < 0.3, skip deep comparison
2. **Array sampling:** For arrays > 100 elements, sample representative elements
3. **Depth limit:** Limit recursion depth to prevent stack overflow
4. **Caching:** Cache similarity scores for repeated comparisons (if needed)

**Performance targets:**

- Exact matching: O(n) - no change from current
- Similarity matching: O(n×m) per GVK group
  - For typical case (1-5 unmatched per GVK): ~10ms overhead
  - For worst case (100 unmatched per GVK): ~500ms acceptable
  - Total overhead should be < 1s for 1000 resources

### 8.5.8 Edge Cases & Considerations

**Handled cases:**

- ✅ Resources with missing `spec` field → fall back to full object comparison
- ✅ Empty specs → return 1.0 similarity (both empty)
- ✅ Different namespaces → only match within same namespace
- ✅ Threshold filtering → pairs below threshold treated as unpaired
- ✅ Exact match priority → exact matches always chosen over similarity

**Known limitations:**

- ⚠️ Cross-namespace matching not supported (by design)
- ⚠️ Field importance not weighted (all fields equal)
- ⚠️ Array element order matters (mitigated by normalization sorting)

### 8.5.9 Testing Strategy

**Unit tests:** (~200 lines)

- Algorithm correctness (15 tests)
- Edge cases (nil, empty, deep nesting)
- Performance benchmarks

**Integration tests:** (~150 lines)

- End-to-end renamed resource scenarios
- Flag and config testing
- Output format validation

**Test coverage goal:** 85%+ for new code

### 8.5.10 Effort Estimate

- **Implementation:** ~6-8 hours
  - similarity.go: ~200 lines
  - matcher.go: ~250 lines
  - Updates to existing files: ~150 lines
  - CLI and config changes: ~50 lines

- **Testing:** ~4-5 hours
  - Unit tests: ~350 lines
  - Integration tests: ~150 lines

- **Documentation:** ~1 hour
  - Update README with examples
  - Add inline code documentation

**Total: ~11-14 hours**

---

## Phase 9: Documentation (Day 10-11) 🔄

### 9.1 README.md ✅

**Sections:**

- [x] Project overview and motivation
- [x] Installation instructions
- [x] Quick start guide
- [x] Basic usage examples
- [x] Configuration reference (example shown)
- [x] Exit codes documented
- [x] Testing section with Makefile commands
- [x] Development section
- [ ] Advanced usage (JQ expressions, etc.) - can be expanded
- [ ] Comparison with other tools - can be expanded
- [ ] Contributing guidelines - can be added later

### 9.2 docs/configuration.md

**Content:**

- [ ] Complete config file reference
- [ ] JSON Pointer syntax and examples
- [ ] JQ path expression syntax and examples
- [ ] Glob pattern matching
- [ ] Common configuration patterns
- [ ] Troubleshooting

### 9.3 docs/usage.md

**Content:**

- [ ] CLI command reference
- [ ] Common use cases
- [ ] Comparing Helm vs Kustomize
- [ ] CI/CD integration examples

---

## Phase 10: Build & Release (Day 11-12) 🔄

### 10.1 Makefile ✅

**Tasks:**

- [x] Create Makefile with all targets
- [x] Add build, test, test-verbose, test-coverage targets
- [x] Add lint, install, clean targets
- [x] Add help target with documentation
- [x] Add version information to build
- [x] Test all make targets

### 10.2 GitHub Actions CI

**Workflows:**

- [ ] `.github/workflows/test.yml` - Run tests on PR
- [ ] `.github/workflows/lint.yml` - Run linter
- [ ] `.github/workflows/release.yml` - Build and release binaries

### 10.3 Release Strategy

**Using GoReleaser:**

- [ ] Configure `.goreleaser.yml`
- [ ] Build for multiple platforms (linux, darwin, windows)
- [ ] Build for multiple architectures (amd64, arm64)
- [ ] Generate checksums
- [ ] Create GitHub releases with binaries
- [ ] Publish to Homebrew tap (optional)

---

## Milestone Checklist

### Milestone 1: Core Functionality (Day 1-6)

- [x] Project structure set up
- [ ] Manifest parsing works
- [ ] Config loading works
- [ ] Ignore rules engine functional
- [ ] Basic tests passing

### Milestone 2: Diff & Output (Day 7-9)

- [ ] Diff engine works
- [ ] All output formats implemented
- [ ] CLI fully functional
- [ ] Integration tests passing

### Milestone 3: Polish & Release (Day 10-12)

- [ ] Documentation complete
- [ ] CI/CD set up
- [ ] First release published
- [ ] Examples working

---

## Risk Mitigation

### Potential Risks

1. **ArgoCD API Changes**
   - **Risk:** ArgoCD libraries might have breaking changes
   - **Mitigation:** Pin to specific version, vendor if needed

2. **JQ Expression Complexity**
   - **Risk:** Complex JQ expressions might not work
   - **Mitigation:** Start with simple expressions, add comprehensive tests

3. **Performance with Large Manifests**
   - **Risk:** Slow with hundreds of resources
   - **Mitigation:** Profile and optimize, consider parallel processing

4. **Difftastic Dependency**
   - **Risk:** Users might not have difftastic installed
   - **Mitigation:** Provide clear error messages, fallback to unified diff

---

## Success Criteria

✅ **Must Have (MVP):**

- Parse YAML manifests
- Apply JSON Pointer ignore rules
- Apply JQ path ignore rules
- Smart resource matching with similarity scoring
- Compare using difftastic with color support
- Output CLI and JSON formats
- Comprehensive test coverage (52+ tests)
- Documentation for basic usage

🎯 **Should Have:**

- HTML output (future enhancement)
- CI/CD pipeline
- Binary releases for multiple platforms

🚀 **Nice to Have:**

- Performance optimizations for very large manifests (1000+ resources)
- Auto-detection of common ignore patterns
- Homebrew distribution
- Field-weighted similarity scoring

---

## Timeline Summary

| Phase     | Days           | Focus                          |
| --------- | -------------- | ------------------------------ |
| 1         | 1              | Setup & Structure ✅           |
| 2         | 2              | Manifest Parsing ✅            |
| 3         | 1-2            | Configuration ✅               |
| 4         | 2-3            | Ignore Rules (ArgoCD) ✅       |
| 5         | 1-2            | Diff Engine ✅                 |
| 6         | 1-2            | Output Formatters ✅           |
| 7         | 1-2            | CLI Implementation ✅          |
| 8         | 1-2            | Testing & Examples ✅          |
| 8.5       | 1-2            | Similarity Matching 🔨         |
| 9         | 1-2            | Documentation 🔄               |
| 10        | 1-2            | Build & Release 🔄             |
| **Total** | **13-16 days** | **MVP Ready + Smart Matching** |

---

## Out of Scope (Future Work)

The following items are explicitly out of scope for the MVP but may be added later:

- CLI subcommands (`validate-config`, `example-config`, etc.)
- Nx executor integration (will be done in the deployments repo later)
- kubectl plugin packaging
- Advanced performance optimizations
- Auto-detection of ignore patterns
- Interactive mode
- Configuration presets library

---

## Next Steps

1. ✅ Repository created and initialized
2. ✅ Plan documented
3. 📝 Review plan and get approval
4. 🚀 Start implementation with Phase 2 (Manifest Parsing)

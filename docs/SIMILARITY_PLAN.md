# Similarity Matching Improvements Plan

## Overview

This document outlines improvements to the similarity matching system in kyt. The changes address two separate concerns:

1. **Fuzzy String Matching**: Internal mechanism for comparing large string fields
2. **Data Field Boost**: User-facing control to prioritize data content over metadata

## Current State Analysis

### How Similarity Matching Works

#### Stage 1: Resource Grouping (Pre-filtering)
Resources are grouped by **GVK only** before any similarity scoring:
```go
gvk := GVK{
    Group:   key.Group,      // ← Must match
    Version: key.Version,    // ← Must match
    Kind:    key.Kind,       // ← Must match
}
```

**Implication**: Only resources with identical GVK are compared. Both **namespace and name** can differ in similarity matching.

**Important**: This allows comparing resources deployed in different namespaces (e.g., `prod/myapp` vs `staging/myapp`), which is common when comparing Helm charts or Kustomize overlays across environments.

#### Stage 2: Similarity Scoring

**For resources with `spec` (Deployments, Services, StatefulSets, etc.):**
- **Metadata is completely ignored**
- Only `spec` content is compared
- Score is 0.0-1.0 based on spec field structural similarity

**For ConfigMaps/Secrets:**
- Uses **weighted comparison** including both metadata and data
- Current weight calculation:
  ```
  baseWeight = 0.5 + (dataSize / 10000)
  max = 0.9
  
  Weights:
    data field: baseWeight (50-90%)
    metadata.name: (1.0 - baseWeight) * 0.4
    metadata.labels: (1.0 - baseWeight) * 0.6
  ```

**Example weights for different ConfigMap sizes:**

| Data Size | Data Weight | name Weight | labels Weight |
|-----------|-------------|-------------|---------------|
| 1KB       | 0.6 (60%)   | 0.16 (16%)  | 0.24 (24%)    |
| 5KB       | 0.9 (90%)   | 0.04 (4%)   | 0.06 (6%)     |
| 20KB+     | 0.9 (90%)   | 0.04 (4%)   | 0.06 (6%)     |

#### Stage 3: Fuzzy String Matching

Currently controlled by `StringSimilarityThreshold` (int, default 100):
- Strings **≥ 100 characters** use Levenshtein distance for comparison
- Strings **< 100 characters** use exact equality check
- This is critical for ConfigMaps with large scripts/configs

**Test case example:**
```bash
# With fuzzy matching (default):
redis-scripts → redis-ha-scripts: similarity 0.77 (MATCHED)

# Without fuzzy matching (--string-similarity-threshold=0):
redis-scripts → redis-ha-scripts: NOT MATCHED (treated as add+remove)
```

The script has minor string differences (`redis-headless` → `redis-ha-headless`) that fuzzy matching detects.

### Current Issues

#### Issue 1: Services/Resources with Different Annotations Score 1.0

**Problem:**
```yaml
# Left: Service with annotation "foo: bar"
# Right: Service with annotation "foo: baz"
# Score: 1.0 (perfect match)
```

Currently, for resources with `spec`, **ALL metadata** is ignored including annotations and labels. This is counter-intuitive.

#### Issue 2: Small ConfigMaps Give Too Much Weight to Metadata

**Problem:**
- 1KB ConfigMap: data=60%, metadata.name=16%, metadata.labels=24%
- A name difference affects the score by 16%, which can prevent matching
- Users have no control over this weighting

#### Issue 3: StringSimilarityThreshold is Confusing

**Problem:**
- Name suggests it controls similarity matching enablement
- Actually controls minimum string length for fuzzy matching (internal detail)
- CLI flag exposes implementation detail
- Current default: 1.0 (100 chars) but stored as float in range 0.0-10.0

## Proposed Changes

### Change 1: Move Fuzzy Matching to Config-Only

**Rationale:**
- Fuzzy string matching is an internal optimization, not a user-facing feature
- Most users should never need to adjust this
- Remove CLI flag to simplify interface

**Config Structure:**
```yaml
diff:
  fuzzyMatching:
    # Enable Levenshtein distance for comparing similar strings
    # When disabled, only exact string matches are considered equal
    enabled: true
    
    # Minimum string length (in characters) to apply fuzzy matching
    # Strings shorter than this use exact comparison
    # Higher values = faster but less accurate for short strings
    # Lower values = slower but more accurate
    minStringLength: 100
```

**Implementation:**
1. Remove `--string-similarity-threshold` CLI flag
2. Add `diff.fuzzyMatching` config section with validation
3. Update `SimilarityScorer` to use config values
4. Keep internal implementation unchanged

**Default Behavior:**
- Enabled: `true`
- MinStringLength: `100` (same as current default)
- **No behavior change for existing users**

---

### Change 2: Add Data Similarity Boost

**Rationale:**
- Give users explicit control over data vs metadata importance
- Help ConfigMaps/Secrets match even with name differences
- Especially useful for small ConfigMaps (< 4KB) where current weighting gives metadata too much influence

**CLI Flag:**
```bash
--data-similarity-boost int
    Boost factor for ConfigMap/Secret data fields (1-10)
    Higher values give more weight to data content vs metadata
    boost=1: no change (original weighting)
    boost=2: data fields count 2x more (default)
    boost=4: data fields count 4x more
    boost=10: data fields heavily prioritized
    (default 2)
```

**Config Structure:**
```yaml
diff:
  options:
    # Boost factor for ConfigMap/Secret data field importance (1-10)
    # Higher values prioritize data content over metadata differences
    # Default: 2 (data fields count twice as much)
    dataSimilarityBoost: 2
```

**Implementation Formula (Additive):**

Current formula:
```go
baseWeight = 0.5 + (dataSize / 10000)
dataWeight = min(0.9, baseWeight)
```

New formula with boost:
```go
baseWeight = 0.5 + (dataSize / 10000)
boostAmount = (boost - 1) * 0.1  // boost=2 adds 0.1, boost=10 adds 0.9
dataWeight = min(0.95, baseWeight + boostAmount)

// Calculate metadata weights from remaining
remainingWeight = 1.0 - dataWeight
metadataNamespaceWeight = remainingWeight * 0.2  // 20% of remaining (NEW)
metadataNameWeight = remainingWeight * 0.3        // 30% of remaining
metadataLabelsWeight = remainingWeight * 0.3      // 30% of remaining (reduced from 50%)
metadataAnnotationsWeight = remainingWeight * 0.2 // 20% of remaining (NEW)
```

**Why Additive (Option C)?**
- **More linear and predictable**: Each boost level adds ~0.1 to data weight
- **Better user control**: Users can understand "boost=4 gives ~0.4 more weight to data"
- **Full range matters**: boost=2 vs boost=10 produces meaningful differences across all ConfigMap sizes
- **Doesn't hit ceiling too early**: Multiplicative approach caps out quickly for medium-sized ConfigMaps

**New Weight Distribution Examples:**

| Data Size | Boost | Data Weight | namespace | name | labels | annotations |
|-----------|-------|-------------|-----------|------|--------|-------------|
| 1KB       | 1     | 0.6         | 0.08      | 0.12 | 0.12   | 0.08        |
| 1KB       | 2     | 0.7         | 0.06      | 0.09 | 0.09   | 0.06        |
| 1KB       | 4     | 0.9         | 0.02      | 0.03 | 0.03   | 0.02        |
| 1KB       | 10    | 0.95        | 0.01      | 0.015| 0.015  | 0.01        |
| 5KB       | 1     | 0.9         | 0.02      | 0.03 | 0.03   | 0.02        |
| 5KB       | 2     | 0.95        | 0.01      | 0.015| 0.015  | 0.01        |
| 5KB       | 4     | 0.95*       | 0.01      | 0.015| 0.015  | 0.01        |
| 20KB      | 1     | 0.9*        | 0.02      | 0.03 | 0.03   | 0.02        |
| 20KB      | 2     | 0.95*       | 0.01      | 0.015| 0.015  | 0.01        |

\* Capped at 0.95 maximum

**Impact Analysis:**

*Scenario: 1KB ConfigMap with name difference (same namespace)*
- **Current (no boost)**: name affects 12% of score
- **boost=2 (default)**: name affects 9% of score → **25% less impact**
- **boost=4**: name affects 3% of score → **75% less impact**

*Scenario: 1KB ConfigMap with namespace AND name difference*
- **boost=2**: namespace affects 6%, name affects 9% = **15% total metadata impact**
- Data content (85%) heavily outweighs metadata differences
- Useful for comparing across environments (prod vs staging)

*Scenario: 5KB ConfigMap*
- Already at 90% data weight
- boost=2 increases to 95% (small but meaningful improvement)

**Validation:**
- Range: 1-10
- Default: 2
- Error if < 1 or > 10

---

### Change 3: Include Annotations in Resource Similarity Scoring

**Rationale:**
- Currently, Service/Deployment with different annotations scores 1.0 (counter-intuitive)
- Annotations often differ between Helm/Kustomize but should reduce similarity score
- Labels already included in ConfigMap/Secret scoring, should do same for annotations

**Changes:**

#### For Resources with `spec` (Services, Deployments, etc.):

Currently:
```go
// Only compares spec, completely ignores metadata
return s.CompareSpecs(aSpecMap, bSpecMap)
```

Proposed:
```go
// Compare spec with high weight, metadata with low weight
specScore := s.CompareSpecs(aSpecMap, bSpecMap)
metadataScore := s.compareMetadata(a, b)

// Weighted combination: spec=90%, metadata=10%
finalScore := (specScore * 0.9) + (metadataScore * 0.1)
return finalScore
```

#### For ConfigMaps/Secrets:

Currently includes only `metadata.name` and `metadata.labels`.

Proposed: Add `metadata.namespace` and `metadata.annotations` with weight distribution:
- metadata.namespace: 20% of metadata weight (NEW - since namespace can differ)
- metadata.name: 30% of metadata weight
- metadata.labels: 30% of metadata weight (reduced from 50%)
- metadata.annotations: 20% of metadata weight (NEW)

**Implementation Details:**

New helper function:
```go
// compareMetadata compares metadata fields with appropriate weights
func (s *SimilarityScorer) compareMetadata(a, b *unstructured.Unstructured) float64 {
    weights := map[string]float64{
        "namespace":   0.2,  // 20% - namespace can differ across environments
        "name":        0.3,  // 30% - name differences matter
        "labels":      0.3,  // 30% - labels important for grouping
        "annotations": 0.2,  // 20% - annotations often differ, less critical
    }
    
    totalScore := 0.0
    totalWeight := 0.0
    
    // Compare namespace (NEW - can differ)
    if a.GetNamespace() == b.GetNamespace() {
        totalScore += weights["namespace"]
    } else {
        totalScore += weights["namespace"] * s.KeyOnlyMatch  // Partial credit
    }
    totalWeight += weights["namespace"]
    
    // Compare name
    if a.GetName() == b.GetName() {
        totalScore += weights["name"]
    } else {
        totalScore += weights["name"] * s.KeyOnlyMatch  // Partial credit
    }
    totalWeight += weights["name"]
    
    // Compare labels
    aLabels := a.GetLabels()
    bLabels := b.GetLabels()
    if len(aLabels) > 0 || len(bLabels) > 0 {
        labelsScore := s.compareObjects(aLabels, bLabels)
        totalScore += labelsScore * weights["labels"]
        totalWeight += weights["labels"]
    }
    
    // Compare annotations (NEW)
    aAnnotations := a.GetAnnotations()
    bAnnotations := b.GetAnnotations()
    if len(aAnnotations) > 0 || len(bAnnotations) > 0 {
        annotationsScore := s.compareObjects(aAnnotations, bAnnotations)
        totalScore += annotationsScore * weights["annotations"]
        totalWeight += weights["annotations"]
    }
    
    if totalWeight == 0 {
        return 1.0
    }
    
    return totalScore / totalWeight
}
```

**Impact Examples:**

*Service with different annotation (same namespace):*
```yaml
# Left:  namespace: prod, annotations: {foo: "bar"}
# Right: namespace: prod, annotations: {foo: "baz"}

# Current: score = 1.0 (metadata ignored)
# New:     score = 0.9 + (0.8 * 0.1 * 0.2) = 0.916
#          (spec matches 100%, annotation differs, 10% metadata weight, 20% annotation weight)
```

*Service in different namespace (NEW scenario):*
```yaml
# Left:  namespace: prod, name: myapp
# Right: namespace: staging, name: myapp

# Current: NOT COMPARED (different namespace groups)
# New:     score = 0.9 + (0.7 * 0.1 * 0.2) = 0.914
#          (spec matches 100%, namespace differs, 10% metadata weight, 20% namespace weight)
```

*ConfigMap with different namespace and labels:*
```yaml
# Left:  namespace: prod, labels: {app: "redis"}
# Right: namespace: staging, labels: {app: "redis-ha"}

# Current weight breakdown (1KB, boost=1):
#   data=60%, name=16%, labels=24%
# New weight breakdown (1KB, boost=1):
#   data=60%, namespace=8%, name=12%, labels=12%, annotations=8%
```

---

### Change 4: Make Similarity Threshold Configurable in Config

**Rationale:**
- Currently hardcoded at 0.7 in matcher
- CLI flag `--similarity-threshold` exists but no config support
- Should be consistent with other options

**Config Addition:**
```yaml
diff:
  options:
    # Minimum similarity score (0.0-1.0) for matching resources
    # Resources with score below this threshold won't be matched
    # Default: 0.7 (70% similarity required)
    similarityThreshold: 0.7
```

**Implementation:**
- Add field to `OptionsConfig`
- Update `Differ.Diff()` to pass threshold from options to matcher
- CLI flag already exists, just add config support
- Validation: 0.0-1.0 range

---

## Implementation Phases

### Phase 0: Change Resource Grouping from GVK+Namespace to GVK Only

**IMPORTANT**: This must be done first as it's a prerequisite for all other changes.

**Files to modify:**
- `pkg/differ/matcher.go`:
  - Rename `GVKNamespace` struct to `GVK` (remove Namespace field)
  - Update `groupByGVKNamespace()` → `groupByGVK()` (remove namespace from grouping key)
  - Update all references to use new `GVK` struct
- `pkg/differ/similarity.go`:
  - Update `compareConfigMapOrSecret()` to include `metadata.namespace` in weight calculation
  - Add namespace comparison (20% of metadata weight)

**Impact:**
- Resources in different namespaces can now be matched (e.g., prod/myapp vs staging/myapp)
- Namespace differences will reduce similarity score but won't prevent matching
- Critical for comparing Helm charts or Kustomize overlays across environments

**Testing:**
- Test ConfigMaps in different namespaces (should be compared and matched if similar)
- Test Services in prod vs staging namespace (should match if specs are identical)
- Verify similarity scores include namespace differences

---

### Phase 1: Move Fuzzy Matching to Config

**Files to modify:**
- `pkg/config/types.go`: Add `FuzzyMatchingConfig` struct
- `pkg/config/validator.go`: Add validation for fuzzy matching config
- `pkg/differ/types.go`: Update `DiffOptions` to include fuzzy matching config
- `pkg/differ/similarity.go`: Read from config instead of threshold
- `cmd/kyt/diff.go`: Remove `--string-similarity-threshold` flag
- `examples/.kyt.yaml`: Add fuzzy matching example

**Testing:**
- Verify default behavior unchanged (enabled=true, minLength=100)
- Test with disabled fuzzy matching
- Test with different minLength values

### Phase 2: Add Data Similarity Boost

**Files to modify:**
- `pkg/config/types.go`: Add `DataSimilarityBoost` to `OptionsConfig`
- `pkg/config/validator.go`: Add validation (range 1-10)
- `pkg/differ/types.go`: Add `DataSimilarityBoost` to `DiffOptions`
- `pkg/differ/similarity.go`: Update `calculateConfigMapWeights()` with additive boost formula
- `cmd/kyt/diff.go`: Add `--data-similarity-boost` flag
- `examples/.kyt.yaml`: Add dataSimilarityBoost example

**Testing:**
- Test with boost=1 (no change from current)
- Test with boost=2 (default)
- Test with boost=4 and boost=10
- Compare scores for small/medium/large ConfigMaps

### Phase 3: Include Annotations in Scoring

**Files to modify:**
- `pkg/differ/similarity.go`:
  - Add `compareMetadata()` helper function
  - Update `CompareResources()` to include metadata for spec-based resources
  - Update `calculateConfigMapWeights()` to include annotations
  - Update `compareConfigMapOrSecret()` to handle annotations

**Testing:**
- Test Services with different annotations
- Test ConfigMaps with different annotations
- Verify weight distribution is correct

### Phase 4: Add Similarity Threshold to Config

**Files to modify:**
- `pkg/config/types.go`: Add `SimilarityThreshold` to `OptionsConfig`
- `pkg/config/validator.go`: Add validation (range 0.0-1.0)
- `pkg/differ/differ.go`: Pass threshold from options to matcher
- `examples/.kyt.yaml`: Add similarityThreshold example

**Testing:**
- Test with different thresholds (0.5, 0.7, 0.9)
- Verify CLI flag overrides config

---

## Validation Rules

### FuzzyMatchingConfig
- `enabled`: boolean (default: true)
- `minStringLength`: int, range 0-10000 (default: 100)
  - 0 = disabled (only exact matches)
  - Higher values = apply fuzzy matching to fewer strings

### DataSimilarityBoost
- Type: int
- Range: 1-10
- Default: 2
- Error if < 1 or > 10

### SimilarityThreshold
- Type: float64
- Range: 0.0-1.0
- Default: 0.7
- Error if < 0.0 or > 1.0

---

## Configuration Examples

### Basic Configuration
```yaml
diff:
  options:
    contextLines: 3
    similarityThreshold: 0.7
    dataSimilarityBoost: 2
  
  fuzzyMatching:
    enabled: true
    minStringLength: 100
```

### Strict Matching (ConfigMap content must be very similar)
```yaml
diff:
  options:
    similarityThreshold: 0.9  # Require 90% similarity
    dataSimilarityBoost: 4    # Heavily favor data content
  
  fuzzyMatching:
    enabled: true
    minStringLength: 50  # Apply fuzzy matching more aggressively
```

### Exact Matching Only (No fuzzy string comparison)
```yaml
diff:
  options:
    similarityThreshold: 0.7
    dataSimilarityBoost: 1  # No boost
  
  fuzzyMatching:
    enabled: false  # Only exact string matches
```

### Lenient Matching (Match more aggressively)
```yaml
diff:
  options:
    similarityThreshold: 0.5  # Accept 50% similarity
    dataSimilarityBoost: 10   # Data is all that matters
  
  fuzzyMatching:
    enabled: true
    minStringLength: 100
```

---

## CLI Examples

```bash
# Default behavior (boost=2, fuzzy matching enabled)
kyt diff left.yaml right.yaml --summary

# Disable similarity matching entirely
kyt diff --exact-match left.yaml right.yaml

# Require higher similarity (90%)
kyt diff --similarity-threshold=0.9 left.yaml right.yaml

# Heavily prioritize data content over metadata
kyt diff --data-similarity-boost=10 left.yaml right.yaml

# Combine flags
kyt diff --similarity-threshold=0.8 --data-similarity-boost=4 left.yaml right.yaml
```

---

## Backward Compatibility

### Breaking Changes

1. **`--string-similarity-threshold` flag removed**
   - **Impact**: Users using this flag will get "unknown flag" error
   - **Migration**: Remove the flag, behavior now controlled by config file
   - **Justification**: This was exposing an internal implementation detail

### Non-Breaking Changes

1. **Default behavior unchanged**
   - Fuzzy matching still enabled with 100 char threshold
   - Similarity threshold still 0.7
   - ConfigMap/Secret scoring logic updated but default boost=2 is close to current behavior

2. **New features are additive**
   - `--data-similarity-boost` is a new flag
   - Config additions don't break existing configs

### Migration Guide

For users currently using `--string-similarity-threshold`:

**Before:**
```bash
kyt diff --string-similarity-threshold=0 left.yaml right.yaml  # Disable fuzzy matching
```

**After (Option 1: Config file):**
```yaml
# .kyt.yaml
diff:
  fuzzyMatching:
    enabled: false
```
```bash
kyt diff -c .kyt.yaml left.yaml right.yaml
```

**After (Option 2: Use --exact-match if goal was strict matching):**
```bash
kyt diff --exact-match left.yaml right.yaml
```

---

## Testing Strategy

### Unit Tests

1. **Fuzzy Matching Tests** (`pkg/differ/similarity_test.go`)
   - Test with enabled=true/false
   - Test with different minStringLength values
   - Test Levenshtein distance calculations

2. **Boost Formula Tests** (`pkg/differ/similarity_test.go`)
   - Test weight calculation with boost=1,2,4,10
   - Test for small/medium/large ConfigMaps
   - Verify additive formula correctness
   - Verify 0.95 cap is enforced

3. **Annotations Tests** (`pkg/differ/similarity_test.go`)
   - Test Services with different annotations
   - Test ConfigMaps with different annotations
   - Verify weight distribution

4. **Config Validation Tests** (`pkg/config/validator_test.go`)
   - Test valid/invalid boost values
   - Test valid/invalid fuzzy matching config
   - Test valid/invalid similarity threshold

### Integration Tests

1. **Test with real ConfigMaps** (use tmp/left.yaml and tmp/right.yaml)
   - Test with boost=1,2,4,10
   - Test with fuzzy matching enabled/disabled
   - Compare scores and verify matching behavior

2. **Test Services with annotations**
   - Create test fixtures with varying annotations
   - Verify scores reflect annotation differences

3. **Test CLI flags**
   - Test `--data-similarity-boost` with various values
   - Test that CLI overrides config
   - Test validation errors

---

## Success Metrics

1. **Fuzzy matching remains effective**
   - ConfigMaps with minor string differences still match (score ≥ 0.7)
   - Default behavior unchanged for existing users

2. **Data boost gives meaningful control**
   - boost=1: Current behavior
   - boost=2: Noticeable reduction in metadata impact (25% less)
   - boost=10: Data dominates scoring (95% weight)

3. **Annotations affect scores appropriately**
   - Services with different annotations score < 1.0
   - ConfigMaps with different annotations have reduced score
   - Impact is proportional (annotations weigh less than labels)

4. **User experience improved**
   - Clear separation between internal (fuzzy matching) and user-facing (boost) controls
   - Config structure is intuitive and well-documented
   - CLI flags are self-explanatory

---

## Documentation Updates

1. **README.md**: Update similarity matching section
2. **docs/diff.md**: Document new flags and config options
3. **examples/.kyt.yaml**: Add commented examples
4. **Flag help text**: Update descriptions for clarity

---

## Future Enhancements

1. **Per-resource-type boost configuration**
   ```yaml
   diff:
     options:
       dataSimilarityBoost:
         configMap: 2
         secret: 4
   ```

2. **Custom weight formulas**
   Allow users to define custom weighting strategies

3. **Metadata field selection**
   Allow users to choose which metadata fields to include/exclude

4. **Alternative string similarity algorithms**
   Support algorithms beyond Levenshtein (e.g., Jaro-Winkler, cosine similarity)

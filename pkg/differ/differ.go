package differ

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aymanbagabas/go-udiff"
	"github.com/nhuray/kyt/pkg/manifest"
	"github.com/nhuray/kyt/pkg/normalizer"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

// Differ performs diffs between two sets of Kubernetes manifests
type Differ struct {
	normalizer *normalizer.Normalizer
	options    *DiffOptions
}

// New creates a new Differ with the given normalizer and options
func New(norm *normalizer.Normalizer, opts *DiffOptions) *Differ {
	if opts == nil {
		opts = NewDefaultDiffOptions()
	}
	return &Differ{
		normalizer: norm,
		options:    opts,
	}
}

// Diff compares two manifest sets and returns the differences
func (d *Differ) Diff(source, target *manifest.ManifestSet) (*DiffResult, error) {
	// Normalize both sets
	normalizedSource := make(map[manifest.ResourceKey]*unstructured.Unstructured)
	for key, obj := range source.Resources {
		normalized, err := d.normalizer.Normalize(obj)
		if err != nil {
			return nil, fmt.Errorf("failed to normalize source resource %s: %w", key.String(), err)
		}
		normalizedSource[key] = normalized
	}

	normalizedTarget := make(map[manifest.ResourceKey]*unstructured.Unstructured)
	for key, obj := range target.Resources {
		normalized, err := d.normalizer.Normalize(obj)
		if err != nil {
			return nil, fmt.Errorf("failed to normalize target resource %s: %w", key.String(), err)
		}
		normalizedTarget[key] = normalized
	}

	// Set up resource cache for similarity matching
	SetResourceCache(normalizedSource, normalizedTarget)

	// Perform resource matching
	// Fuzzy string matching can be enabled/disabled and has a minimum length threshold
	strThreshold := 0
	if d.options.FuzzyStringMatchingEnabled {
		strThreshold = d.options.FuzzyStringMinLength
	}

	matcher := NewResourceMatcherWithOptions(
		d.options.EnableSimilarityMatching, // Enable/disable similarity matching
		d.options.SimilarityThreshold,      // Structural similarity threshold (e.g., 0.7)
		strThreshold,                       // String fuzzy matching threshold (0 = disabled, >0 = character count)
		d.options.DataSimilarityBoost,      // ConfigMap/Secret data boost factor (1-10)
	)

	matches, unmatchedSource, unmatchedTarget := matcher.MatchResources(
		normalizedSource,
		normalizedTarget,
	)

	var changes []ResourceDiff
	identicalCount := 0

	// Process Added resources (in target only)
	for _, key := range unmatchedTarget {
		diff, err := d.generateAddedDiff(key, normalizedTarget[key])
		if err != nil {
			return nil, fmt.Errorf("failed to generate diff for added resource %s: %w", key.String(), err)
		}
		changes = append(changes, diff)
	}

	// Process Removed resources (in source only)
	for _, key := range unmatchedSource {
		diff, err := d.generateRemovedDiff(key, normalizedSource[key])
		if err != nil {
			return nil, fmt.Errorf("failed to generate diff for removed resource %s: %w", key.String(), err)
		}
		changes = append(changes, diff)
	}

	// Process Modified resources
	for _, match := range matches {
		sourceObj := match.SourceResource
		targetObj := match.TargetResource

		// Check if resources are equal
		equal, err := areResourcesEqual(sourceObj, targetObj)
		if err != nil {
			return nil, fmt.Errorf("failed to compare resources for %s: %w", match.SourceKey.String(), err)
		}

		if equal {
			identicalCount++
			continue
		}

		// Generate diff for modified resource
		diff, err := d.generateModifiedDiff(match, sourceObj, targetObj)
		if err != nil {
			return nil, fmt.Errorf("failed to generate diff for modified resource %s: %w", match.SourceKey.String(), err)
		}
		changes = append(changes, diff)
	}

	// Calculate summary
	summary := DiffSummary{
		Added:     len(unmatchedTarget),
		Removed:   len(unmatchedSource),
		Modified:  len(changes) - len(unmatchedTarget) - len(unmatchedSource),
		Identical: identicalCount,
	}

	return &DiffResult{
		Changes: changes,
		Summary: summary,
	}, nil
}

// generateAddedDiff generates a diff for a resource that only exists in target
func (d *Differ) generateAddedDiff(key manifest.ResourceKey, resource *unstructured.Unstructured) (ResourceDiff, error) {
	// Convert to YAML
	targetYAML, err := yaml.Marshal(resource.Object)
	if err != nil {
		return ResourceDiff{}, fmt.Errorf("failed to marshal target: %w", err)
	}

	// Generate unified diff: /dev/null -> b/<key>
	edits := udiff.Strings("", string(targetYAML))
	unified, err := udiff.ToUnified(
		"/dev/null",
		fmt.Sprintf("b/%s", key.String()),
		"",
		edits,
		0, // No context for additions
	)
	if err != nil {
		return ResourceDiff{}, fmt.Errorf("failed to generate unified diff: %w", err)
	}

	// Count insertions (all lines are insertions)
	insertions := countLines(string(targetYAML))

	return ResourceDiff{
		SourceKey:  nil,
		TargetKey:  &key,
		Source:     nil,
		Target:     resource,
		ChangeType: ChangeTypeAdded,
		DiffText:   unified,
		Edits:      edits,
		Insertions: insertions,
		Deletions:  0,
	}, nil
}

// generateRemovedDiff generates a diff for a resource that only exists in source
func (d *Differ) generateRemovedDiff(key manifest.ResourceKey, resource *unstructured.Unstructured) (ResourceDiff, error) {
	// Convert to YAML
	sourceYAML, err := yaml.Marshal(resource.Object)
	if err != nil {
		return ResourceDiff{}, fmt.Errorf("failed to marshal source: %w", err)
	}

	// Generate unified diff: a/<key> -> /dev/null
	edits := udiff.Strings(string(sourceYAML), "")
	unified, err := udiff.ToUnified(
		fmt.Sprintf("a/%s", key.String()),
		"/dev/null",
		string(sourceYAML),
		edits,
		0, // No context for deletions
	)
	if err != nil {
		return ResourceDiff{}, fmt.Errorf("failed to generate unified diff: %w", err)
	}

	// Count deletions (all lines are deletions)
	deletions := countLines(string(sourceYAML))

	return ResourceDiff{
		SourceKey:  &key,
		TargetKey:  nil,
		Source:     resource,
		Target:     nil,
		ChangeType: ChangeTypeRemoved,
		DiffText:   unified,
		Edits:      edits,
		Insertions: 0,
		Deletions:  deletions,
	}, nil
}

// generateModifiedDiff generates a diff for a resource that exists in both but differs
func (d *Differ) generateModifiedDiff(match Match, source, target *unstructured.Unstructured) (ResourceDiff, error) {
	// Convert to YAML
	sourceYAML, err := yaml.Marshal(source.Object)
	if err != nil {
		return ResourceDiff{}, fmt.Errorf("failed to marshal source: %w", err)
	}

	targetYAML, err := yaml.Marshal(target.Object)
	if err != nil {
		return ResourceDiff{}, fmt.Errorf("failed to marshal target: %w", err)
	}

	// Generate edits using go-udiff
	edits := udiff.Strings(string(sourceYAML), string(targetYAML))

	// Generate unified diff with context lines
	unified, err := udiff.ToUnified(
		fmt.Sprintf("a/%s", match.SourceKey.String()),
		fmt.Sprintf("b/%s", match.TargetKey.String()),
		string(sourceYAML),
		edits,
		d.options.ContextLines,
	)
	if err != nil {
		return ResourceDiff{}, fmt.Errorf("failed to generate unified diff: %w", err)
	}

	// Count insertions and deletions from the unified diff
	insertions, deletions := countChangesFromUnifiedDiff(unified)

	return ResourceDiff{
		SourceKey:       &match.SourceKey,
		TargetKey:       &match.TargetKey,
		Source:          source,
		Target:          target,
		ChangeType:      ChangeTypeModified,
		MatchType:       string(match.Type),
		SimilarityScore: match.SimilarityScore,
		DiffText:        unified,
		Edits:           edits,
		Insertions:      insertions,
		Deletions:       deletions,
	}, nil
}

// areResourcesEqual checks if two resources are equal by comparing their JSON representations
func areResourcesEqual(a, b *unstructured.Unstructured) (bool, error) {
	aJSON, err := json.Marshal(a.Object)
	if err != nil {
		return false, err
	}

	bJSON, err := json.Marshal(b.Object)
	if err != nil {
		return false, err
	}

	return bytes.Equal(aJSON, bJSON), nil
}

// countLines counts the number of lines in a string
func countLines(s string) int {
	if s == "" {
		return 0
	}
	count := strings.Count(s, "\n")
	// If no trailing newline, count the last line
	if len(s) > 0 && s[len(s)-1] != '\n' {
		count++
	}
	return count
}

// countChangesFromUnifiedDiff counts insertions and deletions from a unified diff string
// It counts lines that start with '+' (insertions) and '-' (deletions),
// excluding the file header lines (+++ and ---)
func countChangesFromUnifiedDiff(unifiedDiff string) (insertions, deletions int) {
	lines := strings.Split(unifiedDiff, "\n")
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		// Count additions (lines starting with +, but not +++ file headers)
		if line[0] == '+' && !strings.HasPrefix(line, "+++") {
			insertions++
		}
		// Count deletions (lines starting with -, but not --- file headers)
		if line[0] == '-' && !strings.HasPrefix(line, "---") {
			deletions++
		}
	}
	return insertions, deletions
}

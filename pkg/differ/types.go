package differ

import (
	"github.com/aymanbagabas/go-udiff"
	"github.com/nhuray/kyt/pkg/manifest"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ChangeType represents the type of change for a resource
type ChangeType string

const (
	ChangeTypeAdded    ChangeType = "added"
	ChangeTypeRemoved  ChangeType = "removed"
	ChangeTypeModified ChangeType = "modified"
)

// DiffResult contains the complete result of comparing two manifest sets
type DiffResult struct {
	// Changes contains all resource changes (Added, Removed, Modified)
	Changes []ResourceDiff

	// Summary provides aggregate statistics
	Summary DiffSummary
}

// ResourceDiff represents a single resource change with its diff
type ResourceDiff struct {
	// Identification
	SourceKey *manifest.ResourceKey
	TargetKey *manifest.ResourceKey

	// Content
	Source *unstructured.Unstructured
	Target *unstructured.Unstructured

	// Metadata
	ChangeType      ChangeType
	MatchType       string  // "exact" or "similarity" (only for Modified)
	SimilarityScore float64 // 0.0-1.0 (only for similarity matches)

	// Diff output
	DiffText string       // Pre-generated unified diff
	Edits    []udiff.Edit // Edit operations from go-udiff

	// Per-resource stats (for --summary display)
	Insertions int // Lines added in this resource
	Deletions  int // Lines removed in this resource
}

// DiffSummary provides aggregate statistics about the diff
type DiffSummary struct {
	// Resource counts only (not line counts)
	Added     int // Resources only in target
	Removed   int // Resources only in source
	Modified  int // Resources that differ
	Identical int // Resources that are identical (count only, keys not stored)
}

// HasDifferences returns true if there are any differences (added, removed, or modified)
func (r *DiffResult) HasDifferences() bool {
	return len(r.Changes) > 0
}

// GetAdded returns all added resources
func (r *DiffResult) GetAdded() []ResourceDiff {
	var added []ResourceDiff
	for _, change := range r.Changes {
		if change.ChangeType == ChangeTypeAdded {
			added = append(added, change)
		}
	}
	return added
}

// GetRemoved returns all removed resources
func (r *DiffResult) GetRemoved() []ResourceDiff {
	var removed []ResourceDiff
	for _, change := range r.Changes {
		if change.ChangeType == ChangeTypeRemoved {
			removed = append(removed, change)
		}
	}
	return removed
}

// GetModified returns all modified resources
func (r *DiffResult) GetModified() []ResourceDiff {
	var modified []ResourceDiff
	for _, change := range r.Changes {
		if change.ChangeType == ChangeTypeModified {
			modified = append(modified, change)
		}
	}
	return modified
}

// DiffOptions configures how the diff is performed
type DiffOptions struct {
	// ContextLines is the number of context lines for unified diff (default: 3)
	ContextLines int

	// EnableSimilarityMatching enables similarity-based resource matching
	// When enabled, resources with different names but similar structure will be matched
	// Default: true
	EnableSimilarityMatching bool

	// SimilarityThreshold is the minimum similarity score (0.0-1.0) required for matching
	// Only used when EnableSimilarityMatching is true
	// Default: 0.7 (70% similarity)
	SimilarityThreshold float64

	// FuzzyStringMatchingEnabled enables Levenshtein distance for string comparison
	// When enabled, large strings use fuzzy matching instead of exact comparison
	// Default: true
	FuzzyStringMatchingEnabled bool

	// FuzzyStringMinLength is the minimum string length (in characters) for fuzzy matching
	// Strings shorter than this will use exact comparison
	// Default: 100
	FuzzyStringMinLength int

	// DataSimilarityBoost is a boost factor (1-10) for ConfigMap/Secret data field importance
	// Higher values give more weight to data content vs metadata differences
	// Default: 2
	DataSimilarityBoost int
}

// NewDefaultDiffOptions returns DiffOptions with sensible defaults
func NewDefaultDiffOptions() *DiffOptions {
	return &DiffOptions{
		ContextLines:               3,
		EnableSimilarityMatching:   true,
		SimilarityThreshold:        0.7,
		FuzzyStringMatchingEnabled: true,
		FuzzyStringMinLength:       100,
		DataSimilarityBoost:        2,
	}
}

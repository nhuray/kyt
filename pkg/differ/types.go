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

	// StringSimilarityThreshold is the minimum string length (in characters) for fuzzy string matching
	// Used by the similarity scorer when comparing large string fields (e.g., ConfigMap data)
	// Strings longer than this threshold will use Levenshtein distance for better matching
	// This helps match ConfigMaps/Secrets with large data fields that differ slightly
	// Default: 1.0 (100 characters when converted to int)
	// Note: Stored as float64 (0.0-1.0) in config, converted to int (character count) internally
	StringSimilarityThreshold float64
}

// NewDefaultDiffOptions returns DiffOptions with sensible defaults
func NewDefaultDiffOptions() *DiffOptions {
	return &DiffOptions{
		ContextLines:              3,
		EnableSimilarityMatching:  true,
		SimilarityThreshold:       0.7,
		StringSimilarityThreshold: 1.0, // 100 characters (1.0 * 100)
	}
}

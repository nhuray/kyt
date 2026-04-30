package differ

import (
	"github.com/nhuray/k8s-diff/pkg/manifest"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// DiffResult contains the complete result of comparing two manifest sets
type DiffResult struct {
	// Added contains resources that exist in target but not in source
	Added []manifest.ResourceKey

	// Removed contains resources that exist in source but not in target
	Removed []manifest.ResourceKey

	// Modified contains resources that exist in both but have differences
	Modified []ResourceDiff

	// Identical contains resources that exist in both and are identical
	Identical []manifest.ResourceKey

	// Summary provides aggregate statistics
	Summary DiffSummary
}

// ResourceDiff represents a single modified resource with its diff
type ResourceDiff struct {
	// SourceKey uniquely identifies the source resource
	SourceKey manifest.ResourceKey

	// TargetKey uniquely identifies the target resource (may differ from SourceKey for similarity matches)
	TargetKey manifest.ResourceKey

	// Key uniquely identifies the resource (for backward compatibility, equals SourceKey)
	Key manifest.ResourceKey

	// Source is the normalized source resource
	Source *unstructured.Unstructured

	// Target is the normalized target resource
	Target *unstructured.Unstructured

	// DiffText is the formatted diff output (from difftastic or unified diff)
	DiffText string

	// DiffLines contains the number of lines that differ
	DiffLines int

	// MatchType indicates how the resources were matched ("exact" or "similarity")
	MatchType string

	// SimilarityScore is the similarity score (1.0 for exact matches, 0.0-1.0 for similarity matches)
	SimilarityScore float64
}

// DiffSummary provides aggregate statistics about the diff
type DiffSummary struct {
	// TotalResources is the total number of unique resources across both sets
	TotalResources int

	// AddedCount is the number of resources added in target
	AddedCount int

	// RemovedCount is the number of resources removed from source
	RemovedCount int

	// ModifiedCount is the number of resources that differ
	ModifiedCount int

	// IdenticalCount is the number of resources that are identical
	IdenticalCount int
}

// HasDifferences returns true if there are any differences (added, removed, or modified)
func (r *DiffResult) HasDifferences() bool {
	return len(r.Added) > 0 || len(r.Removed) > 0 || len(r.Modified) > 0
}

// DiffOptions configures how the diff is performed
type DiffOptions struct {
	// UseDifftastic enables difftastic for diff generation
	// If false or difftastic is not available, falls back to tree-sitter
	UseDifftastic bool

	// UseTreeSitter enables Go-native tree-sitter diff generation
	// Used as fallback when difftastic is unavailable, or can be forced
	UseTreeSitter bool

	// ColorOutput enables color output (applicable for difftastic and tree-sitter)
	ColorOutput bool

	// ContextLines is the number of context lines for unified diff
	ContextLines int

	// DifftasticDisplay is the display mode for difftastic
	// Options: "side-by-side", "side-by-side-show-both", "inline"
	DifftasticDisplay string

	// TreeSitterWidth is the terminal width for tree-sitter output
	// Default: 120
	TreeSitterWidth int

	// EnableSimilarityMatching enables similarity-based resource matching
	// When enabled, resources with different names but similar structure will be matched
	EnableSimilarityMatching bool

	// SimilarityThreshold is the minimum similarity score (0.0-1.0) required for matching
	// Default: 0.7 (70% similarity)
	SimilarityThreshold float64
}

// NewDefaultDiffOptions returns DiffOptions with sensible defaults
func NewDefaultDiffOptions() *DiffOptions {
	return &DiffOptions{
		UseDifftastic:            true,
		UseTreeSitter:            true, // Enable tree-sitter fallback by default
		ColorOutput:              true,
		ContextLines:             3,
		DifftasticDisplay:        "side-by-side",
		TreeSitterWidth:          120,
		EnableSimilarityMatching: true,
		SimilarityThreshold:      0.7,
	}
}

package differ

import (
	"github.com/nhuray/kyt/pkg/manifest"
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

	// DiffText is the formatted diff output from tree-sitter or unified diff
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
	// UseTreeSitter enables Go-native tree-sitter diff generation (default: true)
	UseTreeSitter bool

	// ColorOutput enables color output for tree-sitter
	ColorOutput bool

	// ContextLines is the number of context lines for unified diff
	ContextLines int

	// DisplayMode is the display mode: side-by-side, inline (default: side-by-side)
	// Only applies when using tree-sitter format
	DisplayMode string

	// OutputFormat controls the diff format: tree-sitter (default), unified
	// tree-sitter: Uses tree-sitter for syntax-aware diffs
	// unified: Uses standard unified diff format
	OutputFormat string

	// TreeSitterWidth is the terminal width for tree-sitter output
	// Default: 120
	TreeSitterWidth int

	// EnableSimilarityMatching enables similarity-based resource matching
	// When enabled, resources with different names but similar structure will be matched
	EnableSimilarityMatching bool

	// SimilarityThreshold is the minimum similarity score (0.0-1.0) required for matching
	// Default: 0.7 (70% similarity)
	SimilarityThreshold float64

	// StringSimilarityThreshold is the minimum string length for fuzzy matching
	// Strings longer than this will use Levenshtein distance for similarity
	// Default: 100 characters
	StringSimilarityThreshold int
}

// NewDefaultDiffOptions returns DiffOptions with sensible defaults
func NewDefaultDiffOptions() *DiffOptions {
	return &DiffOptions{
		UseTreeSitter:             true,
		ColorOutput:               true,
		ContextLines:              3,
		DisplayMode:               "side-by-side",
		OutputFormat:              "tree-sitter",
		TreeSitterWidth:           120,
		EnableSimilarityMatching:  true,
		SimilarityThreshold:       0.7,
		StringSimilarityThreshold: 100, // Enable fuzzy matching for strings > 100 chars
	}
}

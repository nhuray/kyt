package treesitter

import (
	sitter "github.com/smacker/go-tree-sitter"
)

// ChangeType represents the type of change in a diff
type ChangeType int

const (
	// Unchanged means the node is identical in both trees
	Unchanged ChangeType = iota
	// Added means the node exists only in the target
	Added
	// Removed means the node exists only in the source
	Removed
	// Modified means the node exists in both but with different values
	Modified
)

// String returns the string representation of ChangeType
func (c ChangeType) String() string {
	switch c {
	case Unchanged:
		return "unchanged"
	case Added:
		return "added"
	case Removed:
		return "removed"
	case Modified:
		return "modified"
	default:
		return "unknown"
	}
}

// DiffNode represents a node in the diff tree
type DiffNode struct {
	Type       ChangeType  // Type of change
	Key        string      // YAML key (for mappings)
	Index      int         // Array index (for sequences)
	SourceText string      // Text from source
	TargetText string      // Text from target
	Children   []*DiffNode // Child nodes
	LineNumber LineNumbers // Source and target line numbers
}

// LineNumbers contains line number information for both source and target
type LineNumbers struct {
	SourceStart int // Source starting line (1-indexed)
	SourceEnd   int // Source ending line (1-indexed)
	TargetStart int // Target starting line (1-indexed)
	TargetEnd   int // Target ending line (1-indexed)
}

// Differ performs tree-based structural diff on YAML
type Differ struct {
	sourceTree *sitter.Tree
	targetTree *sitter.Tree
	sourceText []byte
	targetText []byte
}

// NewDiffer creates a new Differ with source and target trees
func NewDiffer(sourceTree, targetTree *sitter.Tree, sourceText, targetText []byte) *Differ {
	return &Differ{
		sourceTree: sourceTree,
		targetTree: targetTree,
		sourceText: sourceText,
		targetText: targetText,
	}
}

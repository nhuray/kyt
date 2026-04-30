package treesitter

import (
	"sort"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// Diff computes structural differences between two YAML trees
func (d *Differ) Diff() (*DiffNode, error) {
	sourceRoot := d.sourceTree.RootNode()
	targetRoot := d.targetTree.RootNode()

	return d.diffNodes(sourceRoot, targetRoot), nil
}

// diffNodes recursively compares two tree-sitter nodes
func (d *Differ) diffNodes(source, target *sitter.Node) *DiffNode {
	// Handle nil cases
	if source == nil && target == nil {
		return nil
	}
	if source == nil {
		return d.nodeAdded(target)
	}
	if target == nil {
		return d.nodeRemoved(source)
	}

	// Get node types
	sourceType := source.Type()
	targetType := target.Type()

	// Same node type - recurse into children
	if sourceType == targetType {
		return d.diffSameType(source, target)
	}

	// Different types - mark as modified
	return d.nodeModified(source, target)
}

// diffSameType compares nodes of the same type
func (d *Differ) diffSameType(source, target *sitter.Node) *DiffNode {
	nodeType := source.Type()

	switch nodeType {
	case "stream", "document":
		// Container nodes - recurse into children
		return d.diffChildren(source, target)

	case "block_mapping", "flow_mapping":
		// YAML mappings (key-value pairs)
		return d.diffMapping(source, target)

	case "block_sequence", "flow_sequence":
		// YAML sequences (arrays)
		return d.diffSequence(source, target)

	case "block_mapping_pair", "flow_pair":
		// Key-value pair
		return d.diffPair(source, target)

	case "plain_scalar", "double_quote_scalar", "single_quote_scalar", "block_scalar":
		// Scalar values - compare text
		return d.diffScalar(source, target)

	default:
		// For other types, do simple text comparison
		sourceText := source.Content(d.sourceText)
		targetText := target.Content(d.targetText)

		if sourceText == targetText {
			return &DiffNode{
				Type:       Unchanged,
				SourceText: sourceText,
				TargetText: targetText,
				LineNumber: d.getLineNumbers(source, target),
			}
		}
		return d.nodeModified(source, target)
	}
}

// diffChildren compares container nodes by recursing into children
func (d *Differ) diffChildren(source, target *sitter.Node) *DiffNode {
	sourceChildCount := int(source.ChildCount())
	targetChildCount := int(target.ChildCount())

	var children []*DiffNode
	maxChildren := sourceChildCount
	if targetChildCount > maxChildren {
		maxChildren = targetChildCount
	}

	for i := 0; i < maxChildren; i++ {
		var sourceChild, targetChild *sitter.Node

		if i < sourceChildCount {
			sourceChild = source.Child(i)
		}
		if i < targetChildCount {
			targetChild = target.Child(i)
		}

		childDiff := d.diffNodes(sourceChild, targetChild)
		if childDiff != nil {
			children = append(children, childDiff)
		}
	}

	return &DiffNode{
		Type:       d.aggregateChangeType(children),
		Children:   children,
		LineNumber: d.getLineNumbers(source, target),
	}
}

// diffMapping compares YAML mappings (objects/dicts)
func (d *Differ) diffMapping(source, target *sitter.Node) *DiffNode {
	// Build maps of key -> node for both source and target
	sourceMap := d.buildKeyMap(source)
	targetMap := d.buildKeyMap(target)

	// Get sorted union of all keys
	allKeys := d.getAllKeys(sourceMap, targetMap)

	var children []*DiffNode
	for _, key := range allKeys {
		sourceNode, inSource := sourceMap[key]
		targetNode, inTarget := targetMap[key]

		var childDiff *DiffNode
		if !inSource {
			// Key added in target
			childDiff = d.nodeAdded(targetNode)
		} else if !inTarget {
			// Key removed from source
			childDiff = d.nodeRemoved(sourceNode)
		} else {
			// Key exists in both - recurse
			childDiff = d.diffNodes(sourceNode, targetNode)
		}

		if childDiff != nil {
			childDiff.Key = key
			children = append(children, childDiff)
		}
	}

	return &DiffNode{
		Type:       d.aggregateChangeType(children),
		Children:   children,
		LineNumber: d.getLineNumbers(source, target),
	}
}

// diffSequence compares YAML sequences (arrays)
func (d *Differ) diffSequence(source, target *sitter.Node) *DiffNode {
	sourceElements := d.getSequenceElements(source)
	targetElements := d.getSequenceElements(target)

	maxLen := len(sourceElements)
	if len(targetElements) > maxLen {
		maxLen = len(targetElements)
	}

	var children []*DiffNode
	for i := 0; i < maxLen; i++ {
		var sourceNode, targetNode *sitter.Node

		if i < len(sourceElements) {
			sourceNode = sourceElements[i]
		}
		if i < len(targetElements) {
			targetNode = targetElements[i]
		}

		childDiff := d.diffNodes(sourceNode, targetNode)
		if childDiff != nil {
			childDiff.Index = i
			children = append(children, childDiff)
		}
	}

	return &DiffNode{
		Type:       d.aggregateChangeType(children),
		Children:   children,
		LineNumber: d.getLineNumbers(source, target),
	}
}

// diffPair compares a key-value pair
func (d *Differ) diffPair(source, target *sitter.Node) *DiffNode {
	// Get key and value for both
	sourceKey := d.getPairKey(source)
	_ = d.getPairKey(target) // Keys should match (we match pairs by key in diffMapping)

	sourceValue := d.getPairValue(source)
	targetValue := d.getPairValue(target)

	// Compare values
	valueDiff := d.diffNodes(sourceValue, targetValue)
	if valueDiff != nil {
		valueDiff.Key = sourceKey
	}

	return valueDiff
}

// diffScalar compares scalar values
func (d *Differ) diffScalar(source, target *sitter.Node) *DiffNode {
	sourceText := strings.TrimSpace(source.Content(d.sourceText))
	targetText := strings.TrimSpace(target.Content(d.targetText))

	if sourceText == targetText {
		return &DiffNode{
			Type:       Unchanged,
			SourceText: sourceText,
			TargetText: targetText,
			LineNumber: d.getLineNumbers(source, target),
		}
	}

	return &DiffNode{
		Type:       Modified,
		SourceText: sourceText,
		TargetText: targetText,
		LineNumber: d.getLineNumbers(source, target),
	}
}

// Helper methods

// buildKeyMap extracts key-value pairs from a YAML mapping
func (d *Differ) buildKeyMap(node *sitter.Node) map[string]*sitter.Node {
	result := make(map[string]*sitter.Node)

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		childType := child.Type()
		if childType == "block_mapping_pair" || childType == "flow_pair" {
			key := d.getPairKey(child)
			if key != "" {
				result[key] = child
			}
		}
	}

	return result
}

// getPairKey extracts the key from a key-value pair node
func (d *Differ) getPairKey(pair *sitter.Node) string {
	if pair == nil {
		return ""
	}

	// Look for key field or first child
	keyNode := pair.ChildByFieldName("key")
	if keyNode == nil && pair.ChildCount() > 0 {
		keyNode = pair.Child(0)
	}

	if keyNode != nil {
		return strings.TrimSpace(keyNode.Content(d.sourceText))
	}

	return ""
}

// getPairValue extracts the value from a key-value pair node
func (d *Differ) getPairValue(pair *sitter.Node) *sitter.Node {
	if pair == nil {
		return nil
	}

	// Look for value field or second child
	valueNode := pair.ChildByFieldName("value")
	if valueNode == nil && pair.ChildCount() > 1 {
		valueNode = pair.Child(1)
	}

	return valueNode
}

// getSequenceElements extracts elements from a YAML sequence
func (d *Differ) getSequenceElements(node *sitter.Node) []*sitter.Node {
	var elements []*sitter.Node

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		childType := child.Type()
		// Skip separators and whitespace
		if childType == "block_sequence_item" || childType == "flow_node" {
			elements = append(elements, child)
		}
	}

	return elements
}

// getAllKeys returns sorted union of keys from two maps
func (d *Differ) getAllKeys(m1, m2 map[string]*sitter.Node) []string {
	keySet := make(map[string]bool)

	for k := range m1 {
		keySet[k] = true
	}
	for k := range m2 {
		keySet[k] = true
	}

	keys := make([]string, 0, len(keySet))
	for k := range keySet {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	return keys
}

// aggregateChangeType determines overall change type from children
func (d *Differ) aggregateChangeType(children []*DiffNode) ChangeType {
	if len(children) == 0 {
		return Unchanged
	}

	hasChanges := false
	for _, child := range children {
		if child.Type != Unchanged {
			hasChanges = true
			break
		}
	}

	if hasChanges {
		return Modified
	}
	return Unchanged
}

// getLineNumbers extracts line number information from nodes
func (d *Differ) getLineNumbers(source, target *sitter.Node) LineNumbers {
	ln := LineNumbers{}

	if source != nil {
		ln.SourceStart = int(source.StartPoint().Row) + 1 // 1-indexed
		ln.SourceEnd = int(source.EndPoint().Row) + 1
	}

	if target != nil {
		ln.TargetStart = int(target.StartPoint().Row) + 1 // 1-indexed
		ln.TargetEnd = int(target.EndPoint().Row) + 1
	}

	return ln
}

// nodeAdded creates a DiffNode for an added node
func (d *Differ) nodeAdded(node *sitter.Node) *DiffNode {
	if node == nil {
		return nil
	}

	return &DiffNode{
		Type:       Added,
		TargetText: node.Content(d.targetText),
		LineNumber: LineNumbers{
			TargetStart: int(node.StartPoint().Row) + 1,
			TargetEnd:   int(node.EndPoint().Row) + 1,
		},
	}
}

// nodeRemoved creates a DiffNode for a removed node
func (d *Differ) nodeRemoved(node *sitter.Node) *DiffNode {
	if node == nil {
		return nil
	}

	return &DiffNode{
		Type:       Removed,
		SourceText: node.Content(d.sourceText),
		LineNumber: LineNumbers{
			SourceStart: int(node.StartPoint().Row) + 1,
			SourceEnd:   int(node.EndPoint().Row) + 1,
		},
	}
}

// nodeModified creates a DiffNode for a modified node
func (d *Differ) nodeModified(source, target *sitter.Node) *DiffNode {
	return &DiffNode{
		Type:       Modified,
		SourceText: source.Content(d.sourceText),
		TargetText: target.Content(d.targetText),
		LineNumber: d.getLineNumbers(source, target),
	}
}

package treesitter

import (
	"github.com/fatih/color"
)

// DiffSegment represents a segment of text with its change type
type DiffSegment struct {
	Text string
	Type ChangeType
}

// computeCharDiff computes character-level differences between two strings
// Returns segments with their change types for highlighting
func computeCharDiff(source, target string) ([]DiffSegment, []DiffSegment) {
	if source == target {
		return []DiffSegment{{Text: source, Type: Unchanged}},
			[]DiffSegment{{Text: target, Type: Unchanged}}
	}

	// Use simple word-level diff for better results
	// This is similar to what difftastic does
	sourceSegs, targetSegs := simpleDiff(source, target)

	return sourceSegs, targetSegs
}

// simpleDiff performs a simple character-level diff
func simpleDiff(source, target string) ([]DiffSegment, []DiffSegment) {
	sourceRunes := []rune(source)
	targetRunes := []rune(target)

	// Find common prefix
	commonPrefix := 0
	minLen := len(sourceRunes)
	if len(targetRunes) < minLen {
		minLen = len(targetRunes)
	}
	for commonPrefix < minLen && sourceRunes[commonPrefix] == targetRunes[commonPrefix] {
		commonPrefix++
	}

	// Find common suffix
	commonSuffix := 0
	sourceLen := len(sourceRunes)
	targetLen := len(targetRunes)
	for commonSuffix < (sourceLen-commonPrefix) && commonSuffix < (targetLen-commonPrefix) &&
		sourceRunes[sourceLen-1-commonSuffix] == targetRunes[targetLen-1-commonSuffix] {
		commonSuffix++
	}

	// Build segments
	var sourceSegs []DiffSegment
	var targetSegs []DiffSegment

	// Common prefix
	if commonPrefix > 0 {
		prefix := string(sourceRunes[:commonPrefix])
		sourceSegs = append(sourceSegs, DiffSegment{Text: prefix, Type: Unchanged})
		targetSegs = append(targetSegs, DiffSegment{Text: prefix, Type: Unchanged})
	}

	// Middle (different parts)
	sourceMid := sourceRunes[commonPrefix : sourceLen-commonSuffix]
	targetMid := targetRunes[commonPrefix : targetLen-commonSuffix]

	if len(sourceMid) > 0 {
		sourceSegs = append(sourceSegs, DiffSegment{Text: string(sourceMid), Type: Removed})
	}
	if len(targetMid) > 0 {
		targetSegs = append(targetSegs, DiffSegment{Text: string(targetMid), Type: Added})
	}

	// Common suffix
	if commonSuffix > 0 {
		suffix := string(sourceRunes[sourceLen-commonSuffix:])
		sourceSegs = append(sourceSegs, DiffSegment{Text: suffix, Type: Unchanged})
		targetSegs = append(targetSegs, DiffSegment{Text: suffix, Type: Unchanged})
	}

	// Handle empty cases
	if len(sourceSegs) == 0 {
		sourceSegs = []DiffSegment{{Text: source, Type: Unchanged}}
	}
	if len(targetSegs) == 0 {
		targetSegs = []DiffSegment{{Text: target, Type: Unchanged}}
	}

	return sourceSegs, targetSegs
}

// formatSegments formats segments with appropriate colors and styles
func formatSegments(segments []DiffSegment, useColor bool) string {
	if !useColor {
		var result string
		for _, seg := range segments {
			result += seg.Text
		}
		return result
	}

	var result string
	for _, seg := range segments {
		switch seg.Type {
		case Unchanged:
			result += seg.Text
		case Added:
			// Green with underline for added characters
			result += color.New(color.FgGreen, color.Underline).Sprint(seg.Text)
		case Removed:
			// Red with underline for removed characters
			result += color.New(color.FgRed, color.Underline).Sprint(seg.Text)
		default:
			result += seg.Text
		}
	}
	return result
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

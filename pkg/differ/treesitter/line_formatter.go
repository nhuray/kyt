package treesitter

import (
	"sort"
	"strings"

	"github.com/fatih/color"
)

// LineFormatter formats diffs line-by-line for better side-by-side visualization
type LineFormatter struct {
	width      int
	useColor   bool
	sourceText []byte
	targetText []byte
}

// NewLineFormatter creates a line-based formatter
func NewLineFormatter(width int, useColor bool, sourceText, targetText []byte) *LineFormatter {
	if width <= 0 {
		width = 120
	}
	return &LineFormatter{
		width:      width,
		useColor:   useColor,
		sourceText: sourceText,
		targetText: targetText,
	}
}

// FormatSideBySide formats diff output side-by-side using line numbers from diff tree
func (lf *LineFormatter) FormatSideBySide(root *DiffNode, sourceLabel, targetLabel string) string {
	var buf strings.Builder

	halfWidth := lf.width / 2

	// Header
	buf.WriteString(lf.formatHeader(sourceLabel, targetLabel, halfWidth))
	buf.WriteString("\n")
	buf.WriteString(lf.colorize(strings.Repeat("─", lf.width), color.FgHiBlack))
	buf.WriteString("\n")

	// Extract lines from source and target
	sourceLines := strings.Split(string(lf.sourceText), "\n")
	targetLines := strings.Split(string(lf.targetText), "\n")

	// Build line-based diff from tree
	lineDiffs := lf.buildLineDiffs(root, sourceLines, targetLines)

	// Format each line pair
	for _, ld := range lineDiffs {
		lf.formatLinePair(&buf, ld, halfWidth)
	}

	return buf.String()
}

// LineDiff represents a single line diff
type LineDiff struct {
	SourceLine string
	TargetLine string
	Type       ChangeType
}

// buildLineDiffs converts tree diff to line-based diffs
func (lf *LineFormatter) buildLineDiffs(root *DiffNode, sourceLines, targetLines []string) []LineDiff {
	var result []LineDiff

	// Collect all line ranges from diff tree
	ranges := lf.collectLineRanges(root)

	// Deduplicate ranges - keep only leaf nodes (those with no overlapping children)
	ranges = lf.deduplicateRanges(ranges)

	// Sort by source line number
	sort.Slice(ranges, func(i, j int) bool {
		if ranges[i].SourceStart != ranges[j].SourceStart {
			return ranges[i].SourceStart < ranges[j].SourceStart
		}
		return ranges[i].TargetStart < ranges[j].TargetStart
	})

	// Convert ranges to line diffs
	for _, r := range ranges {
		switch r.Type {
		case Unchanged:
			// Show unchanged lines from source (they're the same in target)
			for i := r.SourceStart; i <= r.SourceEnd && i > 0 && i <= len(sourceLines); i++ {
				result = append(result, LineDiff{
					SourceLine: sourceLines[i-1],
					TargetLine: sourceLines[i-1], // Same line
					Type:       Unchanged,
				})
			}

		case Added:
			// Show empty on left, content on right
			for i := r.TargetStart; i <= r.TargetEnd && i > 0 && i <= len(targetLines); i++ {
				result = append(result, LineDiff{
					SourceLine: "",
					TargetLine: targetLines[i-1],
					Type:       Added,
				})
			}

		case Removed:
			// Show content on left, empty on right
			for i := r.SourceStart; i <= r.SourceEnd && i > 0 && i <= len(sourceLines); i++ {
				result = append(result, LineDiff{
					SourceLine: sourceLines[i-1],
					TargetLine: "",
					Type:       Removed,
				})
			}

		case Modified:
			// Show both sides
			sourceStart := r.SourceStart
			sourceEnd := r.SourceEnd
			targetStart := r.TargetStart
			targetEnd := r.TargetEnd

			// Ensure valid ranges
			if sourceStart <= 0 {
				sourceStart = 1
			}
			if targetStart <= 0 {
				targetStart = 1
			}
			if sourceEnd > len(sourceLines) {
				sourceEnd = len(sourceLines)
			}
			if targetEnd > len(targetLines) {
				targetEnd = len(targetLines)
			}

			// Get max number of lines
			sourceCount := sourceEnd - sourceStart + 1
			targetCount := targetEnd - targetStart + 1
			maxLines := sourceCount
			if targetCount > maxLines {
				maxLines = targetCount
			}

			for i := 0; i < maxLines; i++ {
				srcLine := ""
				tgtLine := ""

				if sourceStart+i <= sourceEnd {
					srcLine = sourceLines[sourceStart+i-1]
				}
				if targetStart+i <= targetEnd {
					tgtLine = targetLines[targetStart+i-1]
				}

				result = append(result, LineDiff{
					SourceLine: srcLine,
					TargetLine: tgtLine,
					Type:       Modified,
				})
			}
		}
	}

	return result
}

// deduplicateRanges removes overlapping ranges, keeping only the most specific (smallest) ones
func (lf *LineFormatter) deduplicateRanges(ranges []LineRange) []LineRange {
	if len(ranges) == 0 {
		return ranges
	}

	// Sort by range size (smaller first) then by start position
	sort.Slice(ranges, func(i, j int) bool {
		sizeI := (ranges[i].SourceEnd - ranges[i].SourceStart) + (ranges[i].TargetEnd - ranges[i].TargetStart)
		sizeJ := (ranges[j].SourceEnd - ranges[j].SourceStart) + (ranges[j].TargetEnd - ranges[j].TargetStart)
		if sizeI != sizeJ {
			return sizeI < sizeJ
		}
		return ranges[i].SourceStart < ranges[j].SourceStart
	})

	// Keep ranges that don't overlap with already-kept ranges
	var result []LineRange
	for _, r := range ranges {
		overlaps := false
		for _, kept := range result {
			if lf.rangesOverlap(r, kept) {
				overlaps = true
				break
			}
		}
		if !overlaps {
			result = append(result, r)
		}
	}

	return result
}

// rangesOverlap checks if two ranges overlap
func (lf *LineFormatter) rangesOverlap(a, b LineRange) bool {
	// Check source overlap
	sourceOverlap := (a.SourceStart >= b.SourceStart && a.SourceStart <= b.SourceEnd) ||
		(a.SourceEnd >= b.SourceStart && a.SourceEnd <= b.SourceEnd) ||
		(b.SourceStart >= a.SourceStart && b.SourceStart <= a.SourceEnd) ||
		(b.SourceEnd >= a.SourceStart && b.SourceEnd <= a.SourceEnd)

	// Check target overlap
	targetOverlap := (a.TargetStart >= b.TargetStart && a.TargetStart <= b.TargetEnd) ||
		(a.TargetEnd >= b.TargetStart && a.TargetEnd <= b.TargetEnd) ||
		(b.TargetStart >= a.TargetStart && b.TargetStart <= a.TargetEnd) ||
		(b.TargetEnd >= a.TargetStart && b.TargetEnd <= a.TargetEnd)

	return sourceOverlap || targetOverlap
}

// LineRange represents a line range with change type
type LineRange struct {
	SourceStart int
	SourceEnd   int
	TargetStart int
	TargetEnd   int
	Type        ChangeType
}

// collectLineRanges extracts all line ranges from diff tree
func (lf *LineFormatter) collectLineRanges(node *DiffNode) []LineRange {
	if node == nil {
		return nil
	}

	var ranges []LineRange

	// Add this node's range if it has line numbers
	if node.LineNumber.SourceStart > 0 || node.LineNumber.TargetStart > 0 {
		ranges = append(ranges, LineRange{
			SourceStart: node.LineNumber.SourceStart,
			SourceEnd:   node.LineNumber.SourceEnd,
			TargetStart: node.LineNumber.TargetStart,
			TargetEnd:   node.LineNumber.TargetEnd,
			Type:        node.Type,
		})
	}

	// Recursively collect from children
	for _, child := range node.Children {
		ranges = append(ranges, lf.collectLineRanges(child)...)
	}

	return ranges
}

// formatLinePair formats a single line pair side-by-side
func (lf *LineFormatter) formatLinePair(buf *strings.Builder, ld LineDiff, halfWidth int) {
	switch ld.Type {
	case Unchanged:
		// Gray on both sides
		left := lf.pad(ld.SourceLine, halfWidth)
		buf.WriteString(lf.colorize(left, color.FgHiBlack))
		buf.WriteString(" │ ")
		buf.WriteString(lf.colorize(ld.TargetLine, color.FgHiBlack))
		buf.WriteString("\n")

	case Added:
		// Empty left, green right
		left := lf.pad("", halfWidth)
		buf.WriteString(left)
		buf.WriteString(" │ ")
		buf.WriteString(lf.colorize(ld.TargetLine, color.FgGreen))
		buf.WriteString("\n")

	case Removed:
		// Red left, empty right
		left := lf.pad(ld.SourceLine, halfWidth)
		buf.WriteString(lf.colorize(left, color.FgRed))
		buf.WriteString(" │ ")
		buf.WriteString("")
		buf.WriteString("\n")

	case Modified:
		// Red left, green right
		left := lf.pad(ld.SourceLine, halfWidth)
		buf.WriteString(lf.colorize(left, color.FgRed))
		buf.WriteString(" │ ")
		buf.WriteString(lf.colorize(ld.TargetLine, color.FgGreen))
		buf.WriteString("\n")
	}
}

// pad pads or truncates string to width
func (lf *LineFormatter) pad(s string, width int) string {
	if len(s) >= width {
		if width > 3 {
			return s[:width-3] + "..."
		}
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

// colorize applies color if enabled
func (lf *LineFormatter) colorize(s string, c color.Attribute) string {
	if !lf.useColor {
		return s
	}
	return color.New(c).Sprint(s)
}

// formatHeader formats the header
func (lf *LineFormatter) formatHeader(left, right string, width int) string {
	leftHeader := lf.colorize(lf.pad(left, width), color.Bold)
	rightHeader := lf.colorize(right, color.Bold)
	return leftHeader + " │ " + rightHeader
}

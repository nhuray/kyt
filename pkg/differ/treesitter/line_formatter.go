package treesitter

import (
	"fmt"
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
	SourceLine   string
	TargetLine   string
	SourceLineNo int // Line number in source (1-based, 0 if not present)
	TargetLineNo int // Line number in target (1-based, 0 if not present)
	Type         ChangeType
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
				targetLineNo := r.TargetStart + (i - r.SourceStart)
				result = append(result, LineDiff{
					SourceLine:   sourceLines[i-1],
					TargetLine:   sourceLines[i-1], // Same line
					SourceLineNo: i,
					TargetLineNo: targetLineNo,
					Type:         Unchanged,
				})
			}

		case Added:
			// Show empty on left, content on right
			for i := r.TargetStart; i <= r.TargetEnd && i > 0 && i <= len(targetLines); i++ {
				result = append(result, LineDiff{
					SourceLine:   "",
					TargetLine:   targetLines[i-1],
					SourceLineNo: 0,
					TargetLineNo: i,
					Type:         Added,
				})
			}

		case Removed:
			// Show content on left, empty on right
			for i := r.SourceStart; i <= r.SourceEnd && i > 0 && i <= len(sourceLines); i++ {
				result = append(result, LineDiff{
					SourceLine:   sourceLines[i-1],
					TargetLine:   "",
					SourceLineNo: i,
					TargetLineNo: 0,
					Type:         Removed,
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
				srcLineNo := 0
				tgtLineNo := 0

				if sourceStart+i <= sourceEnd {
					srcLine = sourceLines[sourceStart+i-1]
					srcLineNo = sourceStart + i
				}
				if targetStart+i <= targetEnd {
					tgtLine = targetLines[targetStart+i-1]
					tgtLineNo = targetStart + i
				}

				// Determine if this specific line changed
				lineType := Modified
				if srcLine == tgtLine && srcLine != "" {
					lineType = Unchanged
				} else if srcLine == "" {
					lineType = Added
				} else if tgtLine == "" {
					lineType = Removed
				}

				result = append(result, LineDiff{
					SourceLine:   srcLine,
					TargetLine:   tgtLine,
					SourceLineNo: srcLineNo,
					TargetLineNo: tgtLineNo,
					Type:         lineType,
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
	// Calculate line number column width (4 chars is usually enough)
	lineNoWidth := 4
	contentWidth := halfWidth - lineNoWidth - 1 // -1 for space after line number

	switch ld.Type {
	case Unchanged:
		// Dim/faint line numbers for unchanged lines
		srcLineNo := lf.formatLineNumber(ld.SourceLineNo, lineNoWidth, color.Faint)
		tgtLineNo := lf.formatLineNumber(ld.TargetLineNo, lineNoWidth, color.Faint)

		// Highlight YAML keys in the content
		srcContent := lf.highlightYAMLKeys(ld.SourceLine, color.Faint)
		tgtContent := lf.highlightYAMLKeys(ld.TargetLine, color.Faint)

		buf.WriteString(srcLineNo)
		buf.WriteString(" ")
		buf.WriteString(lf.padColored(srcContent, ld.SourceLine, contentWidth, color.Faint))
		buf.WriteString(" │ ")
		buf.WriteString(tgtLineNo)
		buf.WriteString(" ")
		buf.WriteString(lf.colorizeIfPlain(tgtContent, ld.TargetLine, color.Faint))
		buf.WriteString("\n")

	case Added:
		// Empty left, bright green and bold right with line number
		srcLineNo := lf.formatLineNumber(0, lineNoWidth, color.Faint)
		tgtLineNo := lf.formatLineNumberBold(ld.TargetLineNo, lineNoWidth, color.FgHiGreen)

		tgtContent := lf.highlightYAMLKeys(ld.TargetLine, color.FgHiGreen)

		buf.WriteString(srcLineNo)
		buf.WriteString(" ")
		buf.WriteString(strings.Repeat(" ", contentWidth))
		buf.WriteString(" │ ")
		buf.WriteString(tgtLineNo)
		buf.WriteString(" ")
		buf.WriteString(lf.colorizeIfPlain(tgtContent, ld.TargetLine, color.FgHiGreen))
		buf.WriteString("\n")

	case Removed:
		// Bright red and bold left with line number, empty right
		srcLineNo := lf.formatLineNumberBold(ld.SourceLineNo, lineNoWidth, color.FgHiRed)
		tgtLineNo := lf.formatLineNumber(0, lineNoWidth, color.Faint)

		srcContent := lf.highlightYAMLKeys(ld.SourceLine, color.FgHiRed)

		buf.WriteString(srcLineNo)
		buf.WriteString(" ")
		buf.WriteString(lf.padColored(srcContent, ld.SourceLine, contentWidth, color.FgHiRed))
		buf.WriteString(" │ ")
		buf.WriteString(tgtLineNo)
		buf.WriteString(" ")
		buf.WriteString("\n")

	case Modified:
		// Use character-level diff for modified lines
		srcLineNo := lf.formatLineNumberBold(ld.SourceLineNo, lineNoWidth, color.FgHiRed)
		tgtLineNo := lf.formatLineNumberBold(ld.TargetLineNo, lineNoWidth, color.FgHiGreen)

		if lf.useColor && ld.SourceLine != "" && ld.TargetLine != "" {
			sourceSegs, targetSegs := computeCharDiff(ld.SourceLine, ld.TargetLine)

			// Format source with character highlighting
			sourceFormatted := lf.formatSegmentsWithPaddingAndKeys(sourceSegs, contentWidth, color.FgHiRed)
			buf.WriteString(srcLineNo)
			buf.WriteString(" ")
			buf.WriteString(sourceFormatted)
			buf.WriteString(" │ ")

			// Format target with character highlighting
			targetFormatted := lf.formatSegmentsWithKeys(targetSegs, color.FgHiGreen)
			buf.WriteString(tgtLineNo)
			buf.WriteString(" ")
			buf.WriteString(targetFormatted)
			buf.WriteString("\n")
		} else {
			// Fallback to line-level coloring
			srcContent := lf.highlightYAMLKeys(ld.SourceLine, color.FgHiRed)
			tgtContent := lf.highlightYAMLKeys(ld.TargetLine, color.FgHiGreen)

			buf.WriteString(srcLineNo)
			buf.WriteString(" ")
			buf.WriteString(lf.padColored(srcContent, ld.SourceLine, contentWidth, color.FgHiRed))
			buf.WriteString(" │ ")
			buf.WriteString(tgtLineNo)
			buf.WriteString(" ")
			buf.WriteString(lf.colorizeIfPlain(tgtContent, ld.TargetLine, color.FgHiGreen))
			buf.WriteString("\n")
		}
	}
}

// formatLineNumber formats a line number with padding and color
func (lf *LineFormatter) formatLineNumber(lineNo, width int, c color.Attribute) string {
	if lineNo == 0 {
		// Empty line number (for added/removed lines)
		return lf.colorize(strings.Repeat(" ", width), c)
	}
	numStr := fmt.Sprintf("%d", lineNo)
	padding := width - len(numStr)
	if padding < 0 {
		padding = 0
	}
	return lf.colorize(strings.Repeat(" ", padding)+numStr, c)
}

// formatLineNumberBold formats a line number with padding, color, and bold
func (lf *LineFormatter) formatLineNumberBold(lineNo, width int, c color.Attribute) string {
	if lineNo == 0 {
		// Empty line number (for added/removed lines)
		return lf.colorize(strings.Repeat(" ", width), c)
	}
	numStr := fmt.Sprintf("%d", lineNo)
	padding := width - len(numStr)
	if padding < 0 {
		padding = 0
	}
	if !lf.useColor {
		return strings.Repeat(" ", padding) + numStr
	}
	col := color.New(c, color.Bold)
	col.EnableColor()
	return col.Sprint(strings.Repeat(" ", padding) + numStr)
}

// highlightYAMLKeys highlights YAML keys (text before colon) in magenta
func (lf *LineFormatter) highlightYAMLKeys(line string, baseColor color.Attribute) string {
	if !lf.useColor {
		return line
	}

	// Find colon position
	colonIdx := strings.Index(line, ":")
	if colonIdx == -1 {
		// No colon, return with base color
		colBase := color.New(baseColor)
		colBase.EnableColor()
		return colBase.Sprint(line)
	}

	// Split into key and value
	key := line[:colonIdx]
	rest := line[colonIdx:]

	// Highlight key in magenta, rest in base color
	colKey := color.New(color.FgMagenta)
	colKey.EnableColor()
	colRest := color.New(baseColor)
	colRest.EnableColor()
	return colKey.Sprint(key) + colRest.Sprint(rest)
}

// colorizeIfPlain applies color only if the text doesn't already have color codes
func (lf *LineFormatter) colorizeIfPlain(text, originalText string, c color.Attribute) string {
	if text != originalText {
		// Already has color codes
		return text
	}
	return lf.colorize(text, c)
}

// padColored pads colored text to the specified width
func (lf *LineFormatter) padColored(coloredText, plainText string, width int, c color.Attribute) string {
	plainLen := len(plainText)
	if plainLen >= width {
		// Truncate if needed
		if width > 3 {
			// This is simplified - ideally we'd truncate the colored text properly
			return lf.colorize(plainText[:width-3]+"...", c)
		}
		return lf.colorize(plainText[:width], c)
	}
	// Add padding
	return coloredText + strings.Repeat(" ", width-plainLen)
}

// formatSegmentsWithPaddingAndKeys formats segments with YAML key highlighting and padding
func (lf *LineFormatter) formatSegmentsWithPaddingAndKeys(segments []DiffSegment, width int, baseColor color.Attribute) string {
	// Calculate plain text length
	plainLen := 0
	for _, seg := range segments {
		plainLen += len(seg.Text)
	}

	// Format segments with key highlighting
	var formatted string
	for _, seg := range segments {
		formatted += lf.formatSegmentWithKeys(seg.Text, seg.Type, baseColor)
	}

	// Add padding
	if plainLen < width {
		formatted += strings.Repeat(" ", width-plainLen)
	} else if plainLen > width && width > 3 {
		// Truncate - simplified version
		plain := plainTextFromSegments(segments)
		formatted = lf.colorize(plain[:width-3]+"...", baseColor)
	}

	return formatted
}

// formatSegmentsWithKeys formats segments with YAML key highlighting
func (lf *LineFormatter) formatSegmentsWithKeys(segments []DiffSegment, baseColor color.Attribute) string {
	var formatted string
	for _, seg := range segments {
		formatted += lf.formatSegmentWithKeys(seg.Text, seg.Type, baseColor)
	}
	return formatted
}

// formatSegmentWithKeys formats a single segment with YAML key highlighting
func (lf *LineFormatter) formatSegmentWithKeys(text string, changeType ChangeType, baseColor color.Attribute) string {
	if !lf.useColor {
		return text
	}

	// Check for YAML key (text before colon)
	colonIdx := strings.Index(text, ":")

	switch changeType {
	case Unchanged:
		if colonIdx != -1 {
			// Has key - highlight key in magenta, rest in base color
			key := text[:colonIdx]
			rest := text[colonIdx:]
			colKey := color.New(color.FgMagenta)
			colKey.EnableColor()
			colRest := color.New(baseColor)
			colRest.EnableColor()
			return colKey.Sprint(key) + colRest.Sprint(rest)
		}
		col := color.New(baseColor)
		col.EnableColor()
		return col.Sprint(text)

	case Added:
		// Bright green, bold, and underline for added parts
		if colonIdx != -1 {
			key := text[:colonIdx]
			rest := text[colonIdx:]
			colKey := color.New(color.FgMagenta)
			colKey.EnableColor()
			colRest := color.New(color.FgHiGreen, color.Bold, color.Underline)
			colRest.EnableColor()
			return colKey.Sprint(key) + colRest.Sprint(rest)
		}
		col := color.New(color.FgHiGreen, color.Bold, color.Underline)
		col.EnableColor()
		return col.Sprint(text)

	case Removed:
		// Bright red, bold, and underline for removed parts
		if colonIdx != -1 {
			key := text[:colonIdx]
			rest := text[colonIdx:]
			colKey := color.New(color.FgMagenta)
			colKey.EnableColor()
			colRest := color.New(color.FgHiRed, color.Bold, color.Underline)
			colRest.EnableColor()
			return colKey.Sprint(key) + colRest.Sprint(rest)
		}
		col := color.New(baseColor, color.Bold, color.Underline)
		col.EnableColor()
		return col.Sprint(text)

	default:
		col := color.New(baseColor)
		col.EnableColor()
		return col.Sprint(text)
	}
}

// formatSegmentsWithPadding formats segments and pads the result
func (lf *LineFormatter) formatSegmentsWithPadding(segments []DiffSegment, width int, baseColor color.Attribute) string {
	// Calculate the plain text length (without ANSI codes)
	plainLen := 0
	for _, seg := range segments {
		plainLen += len(seg.Text)
	}

	// Format segments with appropriate colors
	var formatted string
	for _, seg := range segments {
		switch seg.Type {
		case Unchanged:
			// Use base color for unchanged parts in modified lines
			formatted += color.New(baseColor).Sprint(seg.Text)
		case Removed:
			// Use base color with underline for removed characters
			formatted += color.New(baseColor, color.Underline).Sprint(seg.Text)
		case Added:
			// Added segments don't appear on source side
		default:
			formatted += color.New(baseColor).Sprint(seg.Text)
		}
	}

	// Add padding
	if plainLen < width {
		formatted += strings.Repeat(" ", width-plainLen)
	} else if plainLen > width {
		// Truncate if too long (this is tricky with ANSI codes, so we simplify)
		// For now, just add ellipsis marker
		formatted = lf.colorize(lf.pad(plainTextFromSegments(segments), width), baseColor)
	}

	return formatted
}

// plainTextFromSegments extracts plain text from segments
func plainTextFromSegments(segments []DiffSegment) string {
	var result string
	for _, seg := range segments {
		result += seg.Text
	}
	return result
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
	// Create a color instance and force color output
	col := color.New(c)
	col.EnableColor()
	return col.Sprint(s)
}

// formatHeader formats the header
func (lf *LineFormatter) formatHeader(left, right string, width int) string {
	leftHeader := lf.colorize(lf.pad(left, width), color.Bold)
	rightHeader := lf.colorize(right, color.Bold)
	return leftHeader + " │ " + rightHeader
}

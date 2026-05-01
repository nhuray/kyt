package treesitter

import (
	"strings"

	"github.com/fatih/color"
)

// Formatter formats diff results for display
type Formatter struct {
	width      int  // Terminal width
	useColor   bool // Enable ANSI colors
	indentSize int  // Spaces per indent level
}

// NewFormatter creates a new Formatter with default settings
func NewFormatter(width int, useColor bool, indentSize int) *Formatter {
	if width <= 0 {
		width = 120 // Default terminal width
	}
	if indentSize <= 0 {
		indentSize = 2 // Default indent
	}

	return &Formatter{
		width:      width,
		useColor:   useColor,
		indentSize: indentSize,
	}
}

// FormatSideBySide generates side-by-side diff output
func (f *Formatter) FormatSideBySide(root *DiffNode, sourceLabel, targetLabel string) string {
	var buf strings.Builder

	halfWidth := f.width / 2

	// Header
	buf.WriteString(f.formatHeader(sourceLabel, targetLabel, halfWidth))
	buf.WriteString("\n")
	buf.WriteString(f.colorize(strings.Repeat("─", f.width), color.FgHiBlack))
	buf.WriteString("\n")

	// Body
	f.formatNodeSideBySide(&buf, root, 0)

	return buf.String()
}

// formatNodeSideBySide recursively formats a diff node side-by-side
func (f *Formatter) formatNodeSideBySide(buf *strings.Builder, node *DiffNode, indent int) {
	if node == nil {
		return
	}

	halfWidth := f.width / 2
	padding := strings.Repeat(" ", indent*f.indentSize)

	// Handle based on change type
	switch node.Type {
	case Unchanged:
		// Show on both sides in gray
		left := padding + f.formatNodeContent(node.SourceText, node.Key)
		right := padding + f.formatNodeContent(node.TargetText, node.Key)
		f.writeLine(buf, left, right, color.FgHiBlack, halfWidth)

	case Added:
		// Empty left, green right
		left := ""
		right := padding + f.formatNodeContent(node.TargetText, node.Key)
		f.writeLine(buf, left, right, color.FgGreen, halfWidth)

	case Removed:
		// Red left, empty right
		left := padding + f.formatNodeContent(node.SourceText, node.Key)
		right := ""
		f.writeLine(buf, left, right, color.FgRed, halfWidth)

	case Modified:
		// Red left, green right
		left := padding + f.formatNodeContent(node.SourceText, node.Key)
		right := padding + f.formatNodeContent(node.TargetText, node.Key)

		buf.WriteString(f.colorize(f.pad(left, halfWidth), color.FgRed))
		buf.WriteString(" │ ")
		buf.WriteString(f.colorize(right, color.FgGreen))
		buf.WriteString("\n")
	}

	// Recurse into children
	for _, child := range node.Children {
		f.formatNodeSideBySide(buf, child, indent+1)
	}
}

// formatNodeContent formats the content of a node with its key (if present)
func (f *Formatter) formatNodeContent(text, key string) string {
	if key != "" {
		return key + ": " + text
	}
	return text
}

// writeLine writes a single line with left and right content
func (f *Formatter) writeLine(buf *strings.Builder, left, right string, c color.Attribute, halfWidth int) {
	buf.WriteString(f.colorize(f.pad(left, halfWidth), c))
	buf.WriteString(" │ ")
	buf.WriteString(f.colorize(right, c))
	buf.WriteString("\n")
}

// pad pads or truncates a string to the specified width
func (f *Formatter) pad(s string, width int) string {
	// Remove ANSI color codes for length calculation
	cleanLen := len(s)

	if cleanLen >= width {
		// Truncate with ellipsis
		if width > 3 {
			return s[:width-3] + "..."
		}
		return s[:width]
	}

	// Pad with spaces
	return s + strings.Repeat(" ", width-cleanLen)
}

// colorize applies color to a string if colors are enabled
func (f *Formatter) colorize(s string, c color.Attribute) string {
	if !f.useColor {
		return s
	}
	return color.New(c).Sprint(s)
}

// formatHeader formats the header with source and target labels
func (f *Formatter) formatHeader(left, right string, width int) string {
	leftHeader := f.colorize(f.pad(left, width), color.Bold)
	rightHeader := f.colorize(right, color.Bold)
	return leftHeader + " │ " + rightHeader
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

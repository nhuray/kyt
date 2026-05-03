package diff

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// DiffMode specifies how to render the diff
type DiffMode int

const (
	ModeSideBySide DiffMode = iota
	ModeUnified
)

// Renderer renders parsed diffs with styles
type Renderer struct {
	Width  int
	Mode   DiffMode
	Styles *Styles
}

// Styles defines colors for diff elements
type Styles struct {
	Added      lipgloss.Style
	Removed    lipgloss.Style
	Context    lipgloss.Style
	Header     lipgloss.Style
	LineNumber lipgloss.Style
	Separator  lipgloss.Style
}

// DefaultStyles returns sensible defaults
func DefaultStyles() *Styles {
	return &Styles{
		Added: lipgloss.NewStyle().
			Foreground(lipgloss.Color("34")). // Green
			Background(lipgloss.Color("22")), // Dark green bg
		Removed: lipgloss.NewStyle().
			Foreground(lipgloss.Color("160")). // Red
			Background(lipgloss.Color("52")),  // Dark red bg
		Context: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")), // Light gray
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")), // Cyan
		LineNumber: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")). // Dark gray
			Width(4).
			Align(lipgloss.Right),
		Separator: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
	}
}

// NewRenderer creates a new diff renderer
func NewRenderer(width int, mode DiffMode) *Renderer {
	return &Renderer{
		Width:  width,
		Mode:   mode,
		Styles: DefaultStyles(),
	}
}

// Render renders the parsed diff based on the mode
func (r *Renderer) Render(parsed *ParsedDiff) string {
	switch r.Mode {
	case ModeSideBySide:
		return r.renderSideBySide(parsed)
	case ModeUnified:
		return r.renderUnified(parsed)
	default:
		return r.renderSideBySide(parsed)
	}
}

// renderSideBySide renders a side-by-side diff
func (r *Renderer) renderSideBySide(parsed *ParsedDiff) string {
	var output strings.Builder
	halfWidth := (r.Width - 3) / 2 // -3 for separator " │ "

	// Render header (account for line numbers: 4 digits + 1 space = 5 chars)
	lineNumWidth := 5
	headerWidth := halfWidth - lineNumWidth

	leftHeader := strings.Repeat(" ", lineNumWidth) + r.truncate(parsed.OldLabel, headerWidth)
	rightHeader := strings.Repeat(" ", lineNumWidth) + r.truncate(parsed.NewLabel, headerWidth)

	output.WriteString(r.Styles.Header.Render(leftHeader))
	output.WriteString(" ")
	output.WriteString(r.Styles.Separator.Render("│"))
	output.WriteString(" ")
	output.WriteString(r.Styles.Header.Render(rightHeader))
	output.WriteString("\n")

	// Render separator line
	sep := strings.Repeat("─", halfWidth)
	output.WriteString(r.Styles.Separator.Render(sep))
	output.WriteString(r.Styles.Separator.Render("─┼─"))
	output.WriteString(r.Styles.Separator.Render(sep))
	output.WriteString("\n")

	// Group lines for side-by-side rendering
	for i := 0; i < len(parsed.Lines); i++ {
		line := parsed.Lines[i]

		switch line.Type {
		case LineAdded:
			// Show empty left, green right
			left := r.renderCell("", 0, halfWidth, r.Styles.Context)
			right := r.renderCell(line.Content, line.NewNum, halfWidth, r.Styles.Added)
			output.WriteString(left + r.Styles.Separator.Render(" │ ") + right + "\n")

		case LineRemoved:
			// Show red left, empty right
			left := r.renderCell(line.Content, line.OldNum, halfWidth, r.Styles.Removed)
			right := r.renderCell("", 0, halfWidth, r.Styles.Context)
			output.WriteString(left + r.Styles.Separator.Render(" │ ") + right + "\n")

		case LineContext:
			// Show same content on both sides
			left := r.renderCell(line.Content, line.OldNum, halfWidth, r.Styles.Context)
			right := r.renderCell(line.Content, line.NewNum, halfWidth, r.Styles.Context)
			output.WriteString(left + r.Styles.Separator.Render(" │ ") + right + "\n")

		case LineHunk:
			// Show hunk header across full width
			header := r.Styles.Header.Render(r.truncate(line.Content, r.Width))
			output.WriteString(header + "\n")
		}
	}

	return output.String()
}

// renderCell renders a single cell in side-by-side mode
func (r *Renderer) renderCell(content string, lineNum int, width int, style lipgloss.Style) string {
	lineNumStr := ""
	if lineNum > 0 {
		lineNumStr = r.Styles.LineNumber.Render(fmt.Sprintf("%4d", lineNum))
	} else {
		lineNumStr = r.Styles.LineNumber.Render("    ")
	}

	// Calculate max content width (width - line number - space)
	maxContentWidth := width - 5 // 4 for line number, 1 for space
	if maxContentWidth < 0 {
		maxContentWidth = 0
	}

	// Truncate or pad content
	displayContent := r.truncate(content, maxContentWidth)
	if len(displayContent) < maxContentWidth {
		displayContent = displayContent + strings.Repeat(" ", maxContentWidth-len(displayContent))
	}

	return lineNumStr + " " + style.Render(displayContent)
}

// renderUnified renders a unified diff with line numbers and colors
func (r *Renderer) renderUnified(parsed *ParsedDiff) string {
	var output strings.Builder

	// Render headers
	output.WriteString(r.Styles.Header.Render(parsed.OldLabel) + "\n")
	output.WriteString(r.Styles.Header.Render(parsed.NewLabel) + "\n")
	output.WriteString(r.Styles.Separator.Render(strings.Repeat("─", r.Width)) + "\n")

	for _, line := range parsed.Lines {
		switch line.Type {
		case LineAdded:
			lineNum := r.Styles.LineNumber.Render(fmt.Sprintf("%4d", line.NewNum))
			content := r.Styles.Added.Render("+ " + r.truncate(line.Content, r.Width-7))
			output.WriteString(lineNum + " " + content + "\n")

		case LineRemoved:
			lineNum := r.Styles.LineNumber.Render(fmt.Sprintf("%4d", line.OldNum))
			content := r.Styles.Removed.Render("- " + r.truncate(line.Content, r.Width-7))
			output.WriteString(lineNum + " " + content + "\n")

		case LineContext:
			lineNum := r.Styles.LineNumber.Render(fmt.Sprintf("%4d", line.OldNum))
			content := r.Styles.Context.Render("  " + r.truncate(line.Content, r.Width-7))
			output.WriteString(lineNum + " " + content + "\n")

		case LineHunk:
			output.WriteString(r.Styles.Header.Render(line.Content) + "\n")
		}
	}

	return output.String()
}

// truncate truncates a string to maxLen, adding "..." if needed
func (r *Renderer) truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

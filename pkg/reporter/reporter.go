package reporter

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/nhuray/kyt/pkg/differ"
	"github.com/nhuray/kyt/pkg/manifest"
)

// reportDiff outputs the unified diff for all changes
func (r *Reporter) reportDiff(result *differ.DiffResult, w io.Writer) error {
	for _, change := range result.Changes {
		diffText := change.DiffText

		if r.colorize {
			diffText = colorizeUnifiedDiff(diffText)
		}

		fmt.Fprint(w, diffText)
	}
	return nil
}

// reportSummary outputs a tabular summary of changes
func (r *Reporter) reportSummary(result *differ.DiffResult, w io.Writer) error {
	// Table configuration
	const (
		kindWidth       = 16
		leftWidth       = 21
		rightWidth      = 27
		matchTypeWidth  = 11
		similarityWidth = 17
		changesWidth    = 10
	)

	// Colors
	const (
		cyan   = "\033[36m"
		red    = "\033[31m"
		green  = "\033[32m"
		yellow = "\033[33m"
		gray   = "\033[90m"
		reset  = "\033[0m"
		bold   = "\033[1m"
	)

	// Header
	header := fmt.Sprintf(
		"%-*s │ %-*s │ %-*s │ %-*s │ %-*s │ %s",
		kindWidth, "KIND",
		leftWidth, "LEFT",
		rightWidth, "RIGHT",
		matchTypeWidth, "MATCH TYPE",
		similarityWidth, "SIMILARITY SCORE",
		"CHANGES",
	)

	if r.colorize {
		header = bold + header + reset
	}

	fmt.Fprintln(w, header)

	// Separator
	separator := strings.Repeat("─", kindWidth+1) + "┼" +
		strings.Repeat("─", leftWidth+2) + "┼" +
		strings.Repeat("─", rightWidth+2) + "┼" +
		strings.Repeat("─", matchTypeWidth+2) + "┼" +
		strings.Repeat("─", similarityWidth+2) + "┼" +
		strings.Repeat("─", changesWidth+2)
	fmt.Fprintln(w, separator)

	// Rows
	for _, change := range result.Changes {
		r.printSummaryRow(w, change, kindWidth, leftWidth, rightWidth, matchTypeWidth, similarityWidth)
	}

	// Add identical resources count if any
	if result.Summary.Identical > 0 {
		identicalMsg := fmt.Sprintf("... and %d identical resources (not shown)", result.Summary.Identical)
		if r.colorize {
			identicalMsg = gray + identicalMsg + reset
		}
		fmt.Fprintln(w)
		fmt.Fprintln(w, identicalMsg)
	}

	// Summary line
	fmt.Fprintln(w)
	parts := []string{}
	if result.Summary.Added > 0 {
		text := fmt.Sprintf("%d added", result.Summary.Added)
		if r.colorize {
			text = green + text + reset
		}
		parts = append(parts, text)
	}
	if result.Summary.Removed > 0 {
		text := fmt.Sprintf("%d removed", result.Summary.Removed)
		if r.colorize {
			text = red + text + reset
		}
		parts = append(parts, text)
	}
	if result.Summary.Modified > 0 {
		text := fmt.Sprintf("%d modified", result.Summary.Modified)
		if r.colorize {
			text = yellow + text + reset
		}
		parts = append(parts, text)
	}
	if result.Summary.Identical > 0 {
		text := fmt.Sprintf("%d identical", result.Summary.Identical)
		if r.colorize {
			text = gray + text + reset
		}
		parts = append(parts, text)
	}

	summaryLine := "SUMMARY: " + strings.Join(parts, ", ")
	if r.colorize {
		summaryLine = bold + summaryLine + reset
	}
	fmt.Fprintln(w, summaryLine)

	return nil
}

// printSummaryRow prints a single row in the summary table
func (r *Reporter) printSummaryRow(w io.Writer, change differ.ResourceDiff, kindWidth, leftWidth, rightWidth, matchTypeWidth, similarityWidth int) {
	const (
		red    = "\033[31m"
		green  = "\033[32m"
		yellow = "\033[33m"
		gray   = "\033[90m"
		reset  = "\033[0m"
	)

	// Extract kind and resource names
	kind := ""
	leftName := ""
	rightName := ""

	if change.SourceKey != nil {
		kind = change.SourceKey.Kind
		leftName = formatResourceName(*change.SourceKey)
	}
	if change.TargetKey != nil {
		if kind == "" {
			kind = change.TargetKey.Kind
		}
		rightName = formatResourceName(*change.TargetKey)
	}

	// Highlight name differences
	if r.colorize {
		leftName = highlightDifference(leftName, rightName, change.ChangeType)
		rightName = highlightDifference(rightName, leftName, change.ChangeType)
	}

	// Match type (only show for modified with similarity)
	matchType := ""
	if change.ChangeType == differ.ChangeTypeModified {
		if change.MatchType == "similarity" {
			matchType = "similarity"
		} else {
			matchType = "exact"
		}
	}

	// Similarity score (only for similarity matches)
	similarityScore := ""
	if change.MatchType == "similarity" && change.ChangeType == differ.ChangeTypeModified {
		similarityScore = fmt.Sprintf("%.2f", change.SimilarityScore)
	}

	// Changes (+insertions / -deletions)
	changes := fmt.Sprintf("+%d / -%d", change.Insertions, change.Deletions)
	if r.colorize {
		if change.Insertions > 0 && change.Deletions > 0 {
			changes = yellow + changes + reset
		} else if change.Insertions > 0 {
			changes = green + changes + reset
		} else if change.Deletions > 0 {
			changes = red + changes + reset
		} else {
			changes = gray + changes + reset
		}
	}

	// Format row
	fmt.Fprintf(w, "%-*s │ %-*s │ %-*s │ %-*s │ %-*s │ %s\n",
		kindWidth, truncate(kind, kindWidth),
		leftWidth, truncate(leftName, leftWidth),
		rightWidth, truncate(rightName, rightWidth),
		matchTypeWidth, matchType,
		similarityWidth, similarityScore,
		changes,
	)
}

// formatResourceName formats a resource key as "namespace/name"
func formatResourceName(key manifest.ResourceKey) string {
	if key.Namespace != "" {
		return key.Namespace + "/" + key.Name
	}
	return key.Name
}

// highlightDifference highlights the differing parts between two strings
func highlightDifference(text, other string, changeType differ.ChangeType) string {
	const (
		red    = "\033[31m"
		green  = "\033[32m"
		yellow = "\033[33m"
		reset  = "\033[0m"
	)

	if text == "" || text == other {
		return text
	}

	// Apply color based on change type
	switch changeType {
	case differ.ChangeTypeAdded:
		return green + text + reset
	case differ.ChangeTypeRemoved:
		return red + text + reset
	case differ.ChangeTypeModified:
		// Highlight the difference
		if text != other {
			return yellow + text + reset
		}
	}

	return text
}

// truncate truncates a string to fit within width, adding ellipsis if needed
func truncate(s string, width int) string {
	if utf8.RuneCountInString(s) <= width {
		return s
	}

	// Truncate and add ellipsis
	runes := []rune(s)
	if width > 3 {
		return string(runes[:width-3]) + "..."
	}
	return string(runes[:width])
}

// colorizeUnifiedDiff applies colors to unified diff output
func colorizeUnifiedDiff(diffText string) string {
	const (
		red   = "\033[31m"
		green = "\033[32m"
		cyan  = "\033[36m"
		reset = "\033[0m"
	)

	lines := strings.Split(diffText, "\n")
	var result strings.Builder

	for i, line := range lines {
		if len(line) == 0 {
			if i < len(lines)-1 {
				result.WriteString("\n")
			}
			continue
		}

		switch line[0] {
		case '+':
			if strings.HasPrefix(line, "+++") {
				result.WriteString(cyan + line + reset)
			} else {
				result.WriteString(green + line + reset)
			}
		case '-':
			if strings.HasPrefix(line, "---") {
				result.WriteString(cyan + line + reset)
			} else {
				result.WriteString(red + line + reset)
			}
		case '@':
			if strings.HasPrefix(line, "@@") {
				result.WriteString(cyan + line + reset)
			} else {
				result.WriteString(line)
			}
		default:
			result.WriteString(line)
		}

		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

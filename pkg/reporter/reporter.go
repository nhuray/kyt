package reporter

import (
	"fmt"
	"io"
	"strings"

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

		if _, err := fmt.Fprint(w, diffText); err != nil {
			return fmt.Errorf("failed to write diff: %w", err)
		}
	}
	return nil
}

// reportSummary outputs a tabular summary of changes
func (r *Reporter) reportSummary(result *differ.DiffResult, w io.Writer) error {
	// Table configuration - removed fixed widths for LEFT and RIGHT to avoid truncation
	const (
		changeWidth     = 6 // "CHANGE"
		kindWidth       = 16
		matchTypeWidth  = 11
		similarityWidth = 17
		diffWidth       = 10
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

	// Calculate max widths for LEFT and RIGHT columns by scanning all changes
	leftWidth := len("LEFT")
	rightWidth := len("RIGHT")
	for _, change := range result.Changes {
		if change.SourceKey != nil {
			name := formatResourceName(*change.SourceKey)
			if len(name) > leftWidth {
				leftWidth = len(name)
			}
		}
		if change.TargetKey != nil {
			name := formatResourceName(*change.TargetKey)
			if len(name) > rightWidth {
				rightWidth = len(name)
			}
		}
	}

	// Header
	header := fmt.Sprintf(
		"%-*s │ %-*s │ %-*s │ %-*s │ %-*s │ %-*s │ %s",
		changeWidth, "CHANGE",
		kindWidth, "KIND",
		leftWidth, "LEFT",
		rightWidth, "RIGHT",
		matchTypeWidth, "MATCH TYPE",
		similarityWidth, "SIMILARITY SCORE",
		"DIFF",
	)

	if r.colorize {
		header = bold + header + reset
	}

	if _, err := fmt.Fprintln(w, header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Separator
	separator := strings.Repeat("─", changeWidth+1) + "┼" +
		strings.Repeat("─", kindWidth+2) + "┼" +
		strings.Repeat("─", leftWidth+2) + "┼" +
		strings.Repeat("─", rightWidth+2) + "┼" +
		strings.Repeat("─", matchTypeWidth+2) + "┼" +
		strings.Repeat("─", similarityWidth+2) + "┼" +
		strings.Repeat("─", diffWidth+2)
	if _, err := fmt.Fprintln(w, separator); err != nil {
		return fmt.Errorf("failed to write separator: %w", err)
	}

	// Rows
	for _, change := range result.Changes {
		r.printSummaryRow(w, change, changeWidth, kindWidth, leftWidth, rightWidth, matchTypeWidth, similarityWidth)
	}

	// Add identical resources count if any
	if result.Summary.Identical > 0 {
		identicalMsg := fmt.Sprintf("... and %d identical resources (not shown)", result.Summary.Identical)
		if r.colorize {
			identicalMsg = gray + identicalMsg + reset
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return fmt.Errorf("failed to write newline: %w", err)
		}
		if _, err := fmt.Fprintln(w, identicalMsg); err != nil {
			return fmt.Errorf("failed to write identical message: %w", err)
		}
	}

	// Summary line
	if _, err := fmt.Fprintln(w); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}
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
	if _, err := fmt.Fprintln(w, summaryLine); err != nil {
		return fmt.Errorf("failed to write summary line: %w", err)
	}

	return nil
}

// printSummaryRow prints a single row in the summary table
func (r *Reporter) printSummaryRow(w io.Writer, change differ.ResourceDiff, changeWidth, kindWidth, leftWidth, rightWidth, matchTypeWidth, similarityWidth int) {
	const (
		red       = "\033[31m"
		green     = "\033[32m"
		yellow    = "\033[33m"
		gray      = "\033[90m"
		reset     = "\033[0m"
		underline = "\033[4m"
	)

	// Determine change indicator (A/R/M)
	changeIndicator := ""
	switch change.ChangeType {
	case differ.ChangeTypeAdded:
		changeIndicator = "A"
		if r.colorize {
			changeIndicator = green + changeIndicator + reset
		}
	case differ.ChangeTypeRemoved:
		changeIndicator = "R"
		if r.colorize {
			changeIndicator = red + changeIndicator + reset
		}
	case differ.ChangeTypeModified:
		changeIndicator = "M"
		if r.colorize {
			changeIndicator = yellow + changeIndicator + reset
		}
	}

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

	// Highlight differences in RIGHT column with underline
	if r.colorize && leftName != "" && rightName != "" && leftName != rightName {
		rightName = underlineResourceNameDifferences(leftName, rightName)
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

	// Diff (+insertions / -deletions) - hide +0 / -0 for added/removed
	diff := ""
	switch change.ChangeType {
	case differ.ChangeTypeAdded:
		// Only show insertions for added resources
		diff = fmt.Sprintf("+%d", change.Insertions)
		if r.colorize {
			diff = green + diff + reset
		}
	case differ.ChangeTypeRemoved:
		// Only show deletions for removed resources
		diff = fmt.Sprintf("-%d", change.Deletions)
		if r.colorize {
			diff = red + diff + reset
		}
	case differ.ChangeTypeModified:
		// Show both for modified resources
		diff = fmt.Sprintf("+%d / -%d", change.Insertions, change.Deletions)
		if r.colorize {
			if change.Insertions > 0 && change.Deletions > 0 {
				diff = yellow + diff + reset
			} else if change.Insertions > 0 {
				diff = green + diff + reset
			} else if change.Deletions > 0 {
				diff = red + diff + reset
			} else {
				diff = gray + diff + reset
			}
		}
	}

	// Format row
	// Note: ignoring write error here as this is called in a loop,
	// and the parent function will catch any persistent write failures
	_, _ = fmt.Fprintf(w, "%-*s │ %-*s │ %-*s │ %-*s │ %-*s │ %-*s │ %s\n",
		changeWidth, changeIndicator,
		kindWidth, kind,
		leftWidth, leftName,
		rightWidth, rightName,
		matchTypeWidth, matchType,
		similarityWidth, similarityScore,
		diff,
	)
}

// formatResourceName formats a resource key as "namespace/name"
func formatResourceName(key manifest.ResourceKey) string {
	if key.Namespace != "" {
		return key.Namespace + "/" + key.Name
	}
	return key.Name
}

// underlineResourceNameDifferences underlines the parts of rightName that differ from leftName
func underlineResourceNameDifferences(leftName, rightName string) string {
	const (
		underline = "\033[4m"
		reset     = "\033[0m"
	)

	// If names are identical, no highlighting needed
	if leftName == rightName {
		return rightName
	}

	// Split by "/" to handle namespace/name format
	leftParts := strings.Split(leftName, "/")
	rightParts := strings.Split(rightName, "/")

	var result strings.Builder

	// Handle cases where structure differs (e.g., one has namespace, other doesn't)
	if len(leftParts) != len(rightParts) {
		// Just underline the whole thing if structure is different
		return underline + rightName + reset
	}

	// Compare each part
	for i, rightPart := range rightParts {
		if i > 0 {
			result.WriteString("/")
		}

		leftPart := ""
		if i < len(leftParts) {
			leftPart = leftParts[i]
		}

		if rightPart != leftPart {
			// Underline this part
			result.WriteString(underline + rightPart + reset)
		} else {
			result.WriteString(rightPart)
		}
	}

	return result.String()
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

package reporter

import (
	"fmt"
	"io"
	"sort"
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

	// Sort changes by: CHANGE (A/R/M), then KIND, then LEFT, then RIGHT
	sortedChanges := make([]differ.ResourceDiff, len(result.Changes))
	copy(sortedChanges, result.Changes)
	sort.Slice(sortedChanges, func(i, j int) bool {
		// Compare change types (A < M < R for alphabetical ordering)
		if sortedChanges[i].ChangeType != sortedChanges[j].ChangeType {
			return sortedChanges[i].ChangeType < sortedChanges[j].ChangeType
		}

		// Compare kinds
		kindI := ""
		kindJ := ""
		if sortedChanges[i].SourceKey != nil {
			kindI = sortedChanges[i].SourceKey.Kind
		} else if sortedChanges[i].TargetKey != nil {
			kindI = sortedChanges[i].TargetKey.Kind
		}
		if sortedChanges[j].SourceKey != nil {
			kindJ = sortedChanges[j].SourceKey.Kind
		} else if sortedChanges[j].TargetKey != nil {
			kindJ = sortedChanges[j].TargetKey.Kind
		}
		if kindI != kindJ {
			return kindI < kindJ
		}

		// Compare LEFT (source) names
		leftI := ""
		leftJ := ""
		if sortedChanges[i].SourceKey != nil {
			leftI = formatResourceName(*sortedChanges[i].SourceKey)
		}
		if sortedChanges[j].SourceKey != nil {
			leftJ = formatResourceName(*sortedChanges[j].SourceKey)
		}
		if leftI != leftJ {
			return leftI < leftJ
		}

		// Compare RIGHT (target) names
		rightI := ""
		rightJ := ""
		if sortedChanges[i].TargetKey != nil {
			rightI = formatResourceName(*sortedChanges[i].TargetKey)
		}
		if sortedChanges[j].TargetKey != nil {
			rightJ = formatResourceName(*sortedChanges[j].TargetKey)
		}
		return rightI < rightJ
	})

	// Rows
	for _, change := range sortedChanges {
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

	// Diff (+insertions / -deletions) - hide +0 / -0
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
		// Show both for modified resources, but hide zeros
		if change.Insertions > 0 && change.Deletions > 0 {
			// Both insertions and deletions
			diff = fmt.Sprintf("+%d / -%d", change.Insertions, change.Deletions)
			if r.colorize {
				diff = yellow + diff + reset
			}
		} else if change.Insertions > 0 {
			// Only insertions
			diff = fmt.Sprintf("+%d", change.Insertions)
			if r.colorize {
				diff = green + diff + reset
			}
		} else if change.Deletions > 0 {
			// Only deletions
			diff = fmt.Sprintf("-%d", change.Deletions)
			if r.colorize {
				diff = red + diff + reset
			}
		} else {
			// No changes (shouldn't happen for modified, but handle it)
			diff = ""
		}
	}

	// Format row with proper padding for ANSI codes
	// For fields with ANSI codes, we need to manually pad them because %-*s counts the ANSI bytes
	// Note: ignoring write error here as this is called in a loop,
	// and the parent function will catch any persistent write failures

	// Pad changeIndicator to account for ANSI codes
	changeVisibleLen := visibleLength(changeIndicator)
	changePadding := changeWidth - changeVisibleLen
	if changePadding < 0 {
		changePadding = 0
	}

	// Pad rightName to account for ANSI codes (underline)
	rightVisibleLen := visibleLength(rightName)
	rightPadding := rightWidth - rightVisibleLen
	if rightPadding < 0 {
		rightPadding = 0
	}

	_, _ = fmt.Fprintf(w, "%s%s │ %-*s │ %-*s │ %s%s │ %-*s │ %-*s │ %s\n",
		changeIndicator, strings.Repeat(" ", changePadding),
		kindWidth, kind,
		leftWidth, leftName,
		rightName, strings.Repeat(" ", rightPadding),
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

// visibleLength returns the visible length of a string, excluding ANSI escape codes
func visibleLength(s string) int {
	// Simple ANSI code stripper - matches \033[...m patterns
	inEscape := false
	visibleCount := 0

	for i := 0; i < len(s); i++ {
		if s[i] == '\033' && i+1 < len(s) && s[i+1] == '[' {
			inEscape = true
			i++ // skip the '['
			continue
		}

		if inEscape {
			if s[i] == 'm' {
				inEscape = false
			}
			continue
		}

		visibleCount++
	}

	return visibleCount
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

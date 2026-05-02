package reporter

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
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
	// Colors
	const (
		cyan      = "\033[36m"
		red       = "\033[31m"
		green     = "\033[32m"
		yellow    = "\033[33m"
		gray      = "\033[90m"
		reset     = "\033[0m"
		bold      = "\033[1m"
		underline = "\033[4m"
	)

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

	// Create table
	t := table.NewWriter()
	t.SetOutputMirror(w)

	// Configure table style
	if r.colorize {
		t.SetStyle(table.StyleRounded)
		// Set header style
		t.Style().Color.Header = text.Colors{text.Bold}
	} else {
		t.SetStyle(table.StyleRounded)
	}

	// Add header
	t.AppendHeader(table.Row{"CHANGE", "KIND", "LEFT", "RIGHT", "MATCH TYPE", "SIMILARITY SCORE", "DIFF"})

	// Add rows
	for _, change := range sortedChanges {
		row := r.buildSummaryRow(change)
		t.AppendRow(row)
	}

	// Render table
	t.Render()

	// Add identical resources count if any
	if result.Summary.Identical > 0 {
		identicalMsg := fmt.Sprintf("\n... and %d identical resources (not shown)", result.Summary.Identical)
		if r.colorize {
			identicalMsg = gray + identicalMsg + reset
		}
		if _, err := fmt.Fprintln(w, identicalMsg); err != nil {
			return fmt.Errorf("failed to write identical message: %w", err)
		}
	}

	// Summary line - order: added, modified, removed, identical
	if _, err := fmt.Fprintln(w); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}
	parts := []string{}
	if result.Summary.Added > 0 {
		text := fmt.Sprintf("%d added", result.Summary.Added)
		if r.colorize {
			text = bold + green + text + reset
		}
		parts = append(parts, text)
	}
	if result.Summary.Modified > 0 {
		text := fmt.Sprintf("%d modified", result.Summary.Modified)
		if r.colorize {
			text = bold + yellow + text + reset
		}
		parts = append(parts, text)
	}
	if result.Summary.Removed > 0 {
		text := fmt.Sprintf("%d removed", result.Summary.Removed)
		if r.colorize {
			text = bold + red + text + reset
		}
		parts = append(parts, text)
	}
	if result.Summary.Identical > 0 {
		text := fmt.Sprintf("%d identical", result.Summary.Identical)
		if r.colorize {
			text = bold + gray + text + reset
		}
		parts = append(parts, text)
	}

	summaryLine := "SUMMARY: " + strings.Join(parts, ", ")
	if r.colorize {
		// Make "SUMMARY:" bold as well
		summaryLine = bold + "SUMMARY:" + reset + " " + strings.Join(parts, ", ")
	}
	if _, err := fmt.Fprintln(w, summaryLine); err != nil {
		return fmt.Errorf("failed to write summary line: %w", err)
	}

	return nil
}

// buildSummaryRow builds a row for the summary table
func (r *Reporter) buildSummaryRow(change differ.ResourceDiff) table.Row {
	const (
		red       = "\033[31m"
		green     = "\033[32m"
		yellow    = "\033[33m"
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
			// Both insertions and deletions - color separately
			if r.colorize {
				diff = green + fmt.Sprintf("+%d", change.Insertions) + reset + " / " + red + fmt.Sprintf("-%d", change.Deletions) + reset
			} else {
				diff = fmt.Sprintf("+%d / -%d", change.Insertions, change.Deletions)
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

	return table.Row{changeIndicator, kind, leftName, rightName, matchType, similarityScore, diff}
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

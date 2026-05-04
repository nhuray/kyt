package reporter

import (
	"fmt"
	"io"
	"sort"

	"github.com/nhuray/kyt/pkg/differ"
)

// reportMarkdownSummary outputs a markdown table summary of changes
func (r *Reporter) reportMarkdownSummary(result *differ.DiffResult, w io.Writer) error {
	// If no changes, display nothing (consistent with unified diff)
	if len(result.Changes) == 0 {
		return nil
	}

	// Sort changes by: CHANGE (A/R/M), then KIND, then LEFT, then RIGHT
	sortedChanges := make([]differ.ResourceDiff, len(result.Changes))
	copy(sortedChanges, result.Changes)
	sort.Slice(sortedChanges, func(i, j int) bool {
		// Compare change types (A < M < R for alphabetical ordering)
		if sortedChanges[i].ChangeType != sortedChanges[j].ChangeType {
			return sortedChanges[i].ChangeType < sortedChanges[j].ChangeType
		}

		// Get kind for comparison
		kindI := getKind(sortedChanges[i])
		kindJ := getKind(sortedChanges[j])
		if kindI != kindJ {
			return kindI < kindJ
		}

		// Get left name for comparison
		leftI := getLeftName(sortedChanges[i])
		leftJ := getLeftName(sortedChanges[j])
		if leftI != leftJ {
			return leftI < leftJ
		}

		// Compare right names
		rightI := getRightName(sortedChanges[i])
		rightJ := getRightName(sortedChanges[j])
		return rightI < rightJ
	})

	// Write markdown table header
	if _, err := fmt.Fprintln(w, "## Kyt Diff Summary"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, ""); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "| CHANGE | KIND | LEFT | RIGHT | MATCH TYPE | SIMILARITY | DIFF |"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "|--------|------|------|-------|------------|------------|------|"); err != nil {
		return err
	}

	// Write table rows
	for _, change := range sortedChanges {
		if _, err := fmt.Fprintf(w, "%s\n", buildMarkdownSummaryRow(change)); err != nil {
			return err
		}
	}

	// Write summary line
	if _, err := fmt.Fprintln(w, ""); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "**SUMMARY:** %d added, %d modified, %d removed\n",
		result.Summary.Added,
		result.Summary.Modified,
		result.Summary.Removed,
	); err != nil {
		return err
	}

	return nil
}

// reportMarkdownDiff outputs markdown-formatted diffs grouped by change type
func (r *Reporter) reportMarkdownDiff(result *differ.DiffResult, w io.Writer) error {
	// If no changes, display nothing (consistent with unified diff)
	if len(result.Changes) == 0 {
		return nil
	}

	// Group changes by type
	added := []differ.ResourceDiff{}
	modified := []differ.ResourceDiff{}
	removed := []differ.ResourceDiff{}

	for _, change := range result.Changes {
		switch change.ChangeType {
		case differ.ChangeTypeAdded:
			added = append(added, change)
		case differ.ChangeTypeModified:
			modified = append(modified, change)
		case differ.ChangeTypeRemoved:
			removed = append(removed, change)
		}
	}

	// Sort each group by kind and name
	sortChangesByKindAndName := func(changes []differ.ResourceDiff) {
		sort.Slice(changes, func(i, j int) bool {
			kindI := getKind(changes[i])
			kindJ := getKind(changes[j])
			if kindI != kindJ {
				return kindI < kindJ
			}
			nameI := getLeftName(changes[i])
			if nameI == "" {
				nameI = getRightName(changes[i])
			}
			nameJ := getLeftName(changes[j])
			if nameJ == "" {
				nameJ = getRightName(changes[j])
			}
			return nameI < nameJ
		})
	}

	sortChangesByKindAndName(added)
	sortChangesByKindAndName(modified)
	sortChangesByKindAndName(removed)

	// Write title
	if _, err := fmt.Fprintln(w, "## Kyt Unified Diff"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, ""); err != nil {
		return err
	}

	// Write Added section
	if len(added) > 0 {
		if _, err := fmt.Fprintln(w, "### Added Resources"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, ""); err != nil {
			return err
		}
		for _, change := range added {
			if err := r.writeMarkdownResourceDiff(w, change); err != nil {
				return err
			}
		}
	}

	// Write Modified section
	if len(modified) > 0 {
		if _, err := fmt.Fprintln(w, "### Modified Resources"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, ""); err != nil {
			return err
		}
		for _, change := range modified {
			if err := r.writeMarkdownResourceDiff(w, change); err != nil {
				return err
			}
		}
	}

	// Write Removed section
	if len(removed) > 0 {
		if _, err := fmt.Fprintln(w, "### Removed Resources"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, ""); err != nil {
			return err
		}
		for _, change := range removed {
			if err := r.writeMarkdownResourceDiff(w, change); err != nil {
				return err
			}
		}
	}

	// Write summary
	if _, err := fmt.Fprintln(w, "---"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, ""); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "**Summary:** %d added, %d modified, %d removed\n",
		result.Summary.Added,
		result.Summary.Modified,
		result.Summary.Removed,
	); err != nil {
		return err
	}

	return nil
}

// writeMarkdownResourceDiff writes a single resource diff in markdown format
func (r *Reporter) writeMarkdownResourceDiff(w io.Writer, change differ.ResourceDiff) error {
	// Build subsection header
	kind := getKind(change)
	leftName := getLeftName(change)
	rightName := getRightName(change)

	var header string
	switch change.ChangeType {
	case differ.ChangeTypeAdded:
		header = fmt.Sprintf("#### %s `%s`", kind, rightName)
	case differ.ChangeTypeRemoved:
		header = fmt.Sprintf("#### %s `%s`", kind, leftName)
	case differ.ChangeTypeModified:
		if leftName != rightName {
			header = fmt.Sprintf("#### %s `%s` → `%s`", kind, leftName, rightName)
		} else {
			header = fmt.Sprintf("#### %s `%s`", kind, leftName)
		}
	}

	if _, err := fmt.Fprintln(w, header); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, ""); err != nil {
		return err
	}

	// Add match info for modified resources
	if change.ChangeType == differ.ChangeTypeModified {
		if change.MatchType == "similarity" {
			if _, err := fmt.Fprintf(w, "*Matched by similarity: %.2f*\n\n", change.SimilarityScore); err != nil {
				return err
			}
		}
	}

	// Write diff in markdown code block with diff syntax
	if _, err := fmt.Fprintln(w, "```diff"); err != nil {
		return err
	}
	if _, err := fmt.Fprint(w, change.DiffText); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "```"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, ""); err != nil {
		return err
	}

	return nil
}

// buildMarkdownSummaryRow builds a markdown table row for a resource diff
func buildMarkdownSummaryRow(change differ.ResourceDiff) string {
	// Determine change indicator
	changeIndicator := ""
	switch change.ChangeType {
	case differ.ChangeTypeAdded:
		changeIndicator = "➕ A"
	case differ.ChangeTypeModified:
		changeIndicator = "📝 M"
	case differ.ChangeTypeRemoved:
		changeIndicator = "➖ R"
	}

	// Get kind
	kind := getKind(change)

	// Get left and right names
	leftName := getLeftName(change)
	rightName := getRightName(change)

	// Format match type and similarity
	matchType := ""
	similarity := ""
	if change.ChangeType == differ.ChangeTypeModified {
		matchType = change.MatchType
		if change.MatchType == "similarity" {
			similarity = fmt.Sprintf("%.2f", change.SimilarityScore)
		}
	}

	// Format diff statistics
	diffStats := formatMarkdownDiffStats(change)

	return fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s |",
		changeIndicator,
		kind,
		leftName,
		rightName,
		matchType,
		similarity,
		diffStats,
	)
}

// formatMarkdownDiffStats formats diff statistics for markdown
func formatMarkdownDiffStats(change differ.ResourceDiff) string {
	switch change.ChangeType {
	case differ.ChangeTypeAdded:
		return fmt.Sprintf("+%d", change.Insertions)
	case differ.ChangeTypeRemoved:
		return fmt.Sprintf("-%d", change.Deletions)
	case differ.ChangeTypeModified:
		// Don't show +0 / -0
		if change.Insertions == 0 && change.Deletions == 0 {
			return ""
		}
		return fmt.Sprintf("+%d / -%d", change.Insertions, change.Deletions)
	}
	return ""
}

// Helper functions to get kind and names
func getKind(change differ.ResourceDiff) string {
	if change.SourceKey != nil {
		return change.SourceKey.Kind
	}
	if change.TargetKey != nil {
		return change.TargetKey.Kind
	}
	return ""
}

func getLeftName(change differ.ResourceDiff) string {
	if change.SourceKey == nil {
		return ""
	}
	return formatResourceName(*change.SourceKey)
}

func getRightName(change differ.ResourceDiff) string {
	if change.TargetKey == nil {
		return ""
	}
	return formatResourceName(*change.TargetKey)
}

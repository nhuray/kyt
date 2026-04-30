package reporter

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/nhuray/k8s-diff/pkg/differ"
	"github.com/nhuray/k8s-diff/pkg/manifest"
)

// CLIReporter formats output for human-readable CLI display
type CLIReporter struct {
	options *Options
}

// NewCLIReporter creates a new CLI reporter
func NewCLIReporter(opts *Options) *CLIReporter {
	if opts == nil {
		opts = NewDefaultOptions()
	}
	return &CLIReporter{
		options: opts,
	}
}

// Report formats and writes the diff result for CLI output
func (r *CLIReporter) Report(result *differ.DiffResult, w io.Writer) error {
	// Check if we should use colors
	useColor := r.options.Colorize && isTerminal(w)

	// Print header
	r.printHeader(w, result, useColor)

	// Print added resources
	if len(result.Added) > 0 {
		r.printSection(w, "Added Resources", result.Added, useColor, colorGreen)
	}

	// Print removed resources
	if len(result.Removed) > 0 {
		r.printSection(w, "Removed Resources", result.Removed, useColor, colorRed)
	}

	// Print modified resources with diffs
	if len(result.Modified) > 0 {
		r.printModified(w, result.Modified, useColor)
	}

	// Print identical resources if requested
	if r.options.ShowIdentical && len(result.Identical) > 0 {
		r.printSection(w, "Identical Resources", result.Identical, useColor, colorGray)
	}

	// Print footer summary
	r.printFooter(w, result, useColor)

	return nil
}

// printHeader prints the report header
func (r *CLIReporter) printHeader(w io.Writer, result *differ.DiffResult, useColor bool) {
	separator := "================================================================"

	if useColor {
		_, _ = fmt.Fprintf(w, "%s%s%s\n", colorBold, separator, colorReset)
		_, _ = fmt.Fprintf(w, "%s%s  k8s-diff Report%s\n", colorBold, colorCyan, colorReset)
		_, _ = fmt.Fprintf(w, "%s%s%s\n\n", colorBold, separator, colorReset)
	} else {
		_, _ = fmt.Fprintf(w, "%s\n", separator)
		_, _ = fmt.Fprintf(w, "  k8s-diff Report\n")
		_, _ = fmt.Fprintf(w, "%s\n\n", separator)
	}
}

// printSection prints a section of resources (added/removed/identical)
func (r *CLIReporter) printSection(w io.Writer, title string, keys []manifest.ResourceKey, useColor bool, color string) {
	// Sort keys for consistent output
	sorted := make([]manifest.ResourceKey, len(keys))
	copy(sorted, keys)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].String() < sorted[j].String()
	})

	if useColor {
		_, _ = fmt.Fprintf(w, "%s%s%s (%d):\n", colorBold, title, colorReset, len(sorted))
	} else {
		_, _ = fmt.Fprintf(w, "%s (%d):\n", title, len(sorted))
	}

	for _, key := range sorted {
		if useColor {
			_, _ = fmt.Fprintf(w, "  %s●%s %s\n", color, colorReset, key.String())
		} else {
			_, _ = fmt.Fprintf(w, "  • %s\n", key.String())
		}
	}
	_, _ = fmt.Fprintln(w)
}

// printModified prints modified resources with their diffs
func (r *CLIReporter) printModified(w io.Writer, modified []differ.ResourceDiff, useColor bool) {
	// Sort by resource key for consistent output
	sorted := make([]differ.ResourceDiff, len(modified))
	copy(sorted, modified)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Key.String() < sorted[j].Key.String()
	})

	if useColor {
		_, _ = fmt.Fprintf(w, "%sModified Resources%s (%d):\n\n", colorBold, colorReset, len(sorted))
	} else {
		_, _ = fmt.Fprintf(w, "Modified Resources (%d):\n\n", len(sorted))
	}

	for i, diff := range sorted {
		if i > 0 {
			_, _ = fmt.Fprintln(w)
		}

		// Print resource header
		matchInfo := r.formatMatchInfo(&diff)
		if useColor {
			_, _ = fmt.Fprintf(w, "%s%s───────────────────────────────────────────────────────────────%s\n",
				colorBold, colorYellow, colorReset)
			_, _ = fmt.Fprintf(w, "%s%s● %s%s\n", colorBold, colorYellow, matchInfo, colorReset)
			_, _ = fmt.Fprintf(w, "%s%s───────────────────────────────────────────────────────────────%s\n\n",
				colorBold, colorYellow, colorReset)
		} else {
			_, _ = fmt.Fprintf(w, "───────────────────────────────────────────────────────────────\n")
			_, _ = fmt.Fprintf(w, "• %s\n", matchInfo)
			_, _ = fmt.Fprintf(w, "───────────────────────────────────────────────────────────────\n\n")
		}

		// Print diff (already contains color codes from difftastic if enabled)
		_, _ = fmt.Fprint(w, diff.DiffText)
		_, _ = fmt.Fprintln(w)
	}
}

// printFooter prints the summary footer
func (r *CLIReporter) printFooter(w io.Writer, result *differ.DiffResult, useColor bool) {
	separator := "================================================================"

	if useColor {
		_, _ = fmt.Fprintf(w, "%s%s%s\n", colorBold, separator, colorReset)
		_, _ = fmt.Fprintf(w, "%s%sSummary:%s\n", colorBold, colorCyan, colorReset)
	} else {
		_, _ = fmt.Fprintf(w, "%s\n", separator)
		_, _ = fmt.Fprintf(w, "Summary:\n")
	}

	summary := result.Summary

	// Print summary statistics
	r.printSummaryLine(w, "Total Resources", summary.TotalResources, useColor, "")
	r.printSummaryLine(w, "Added", summary.AddedCount, useColor, colorGreen)
	r.printSummaryLine(w, "Removed", summary.RemovedCount, useColor, colorRed)
	r.printSummaryLine(w, "Modified", summary.ModifiedCount, useColor, colorYellow)
	r.printSummaryLine(w, "Identical", summary.IdenticalCount, useColor, colorGray)

	if useColor {
		_, _ = fmt.Fprintf(w, "%s%s%s\n", colorBold, separator, colorReset)
	} else {
		_, _ = fmt.Fprintf(w, "%s\n", separator)
	}

	// Print status message
	if !result.HasDifferences() {
		if useColor {
			_, _ = fmt.Fprintf(w, "\n%s%s✓ No differences found%s\n", colorBold, colorGreen, colorReset)
		} else {
			_, _ = fmt.Fprintf(w, "\n✓ No differences found\n")
		}
	} else {
		if useColor {
			_, _ = fmt.Fprintf(w, "\n%s%s! Differences detected%s\n", colorBold, colorYellow, colorReset)
		} else {
			_, _ = fmt.Fprintf(w, "\n! Differences detected\n")
		}
	}
}

// printSummaryLine prints a single line of the summary
func (r *CLIReporter) printSummaryLine(w io.Writer, label string, count int, useColor bool, color string) {
	if useColor && color != "" {
		_, _ = fmt.Fprintf(w, "  %-15s %s%d%s\n", label+":", color, count, colorReset)
	} else {
		_, _ = fmt.Fprintf(w, "  %-15s %d\n", label+":", count)
	}
}

// formatMatchInfo formats the match information for display
func (r *CLIReporter) formatMatchInfo(diff *differ.ResourceDiff) string {
	if diff.MatchType == "exact" || diff.MatchType == "" {
		// Exact match or legacy format - just show the key
		return diff.Key.String()
	}

	// Similarity match - show source → target with score
	if diff.SourceKey.String() == diff.TargetKey.String() {
		// Keys are same (shouldn't happen, but handle gracefully)
		return fmt.Sprintf("%s (similarity: %.2f)", diff.Key.String(), diff.SimilarityScore)
	}

	return fmt.Sprintf("%s → %s (similarity: %.2f)",
		diff.SourceKey.String(),
		diff.TargetKey.String(),
		diff.SimilarityScore)
}

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

// isTerminal checks if the writer is a terminal (supports colors)
func isTerminal(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		// Check if it's a terminal
		stat, err := f.Stat()
		if err != nil {
			return false
		}
		// Check if it's a character device (terminal)
		return (stat.Mode() & os.ModeCharDevice) != 0
	}
	return false
}

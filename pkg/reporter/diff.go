package reporter

import (
	"fmt"
	"io"

	"github.com/nhuray/kyt/pkg/differ"
)

// DiffReporter outputs raw unified diff format (no headers)
type DiffReporter struct {
	options *Options
}

// NewDiffReporter creates a new Diff reporter
func NewDiffReporter(opts *Options) *DiffReporter {
	if opts == nil {
		opts = NewDefaultOptions()
	}
	return &DiffReporter{
		options: opts,
	}
}

// Report formats and writes the diff result as unified diffs
func (r *DiffReporter) Report(result *differ.DiffResult, w io.Writer) error {
	// Only output modified resources as diffs
	for _, mod := range result.Modified {
		// Write diff text directly (it's already in unified diff format)
		if _, err := fmt.Fprint(w, mod.DiffText); err != nil {
			return fmt.Errorf("failed to write diff: %w", err)
		}

		// Add separator between resources
		if _, err := fmt.Fprintln(w); err != nil {
			return fmt.Errorf("failed to write separator: %w", err)
		}
	}

	return nil
}

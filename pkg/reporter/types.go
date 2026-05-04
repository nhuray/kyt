package reporter

import (
	"io"

	"github.com/nhuray/kyt/pkg/differ"
)

// Reporter formats and outputs diff results
type Reporter struct {
	showSummary bool
	colorize    bool
	markdown    bool
}

// NewReporter creates a new Reporter with the given options
func NewReporter(showSummary, colorize, markdown bool) *Reporter {
	return &Reporter{
		showSummary: showSummary,
		colorize:    colorize,
		markdown:    markdown,
	}
}

// Report formats and writes the diff result to the writer
func (r *Reporter) Report(result *differ.DiffResult, w io.Writer) error {
	if r.markdown {
		if r.showSummary {
			return r.reportMarkdownSummary(result, w)
		}
		return r.reportMarkdownDiff(result, w)
	}

	if r.showSummary {
		return r.reportSummary(result, w)
	}
	return r.reportDiff(result, w)
}

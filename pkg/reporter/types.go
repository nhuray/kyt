package reporter

import (
	"io"

	"github.com/nicolasleigh/k8s-diff/pkg/differ"
)

// Reporter formats and outputs diff results
type Reporter interface {
	// Report formats and writes the diff result to the writer
	Report(result *differ.DiffResult, w io.Writer) error
}

// Options configures the reporter behavior
type Options struct {
	// Format specifies the output format: cli, json, diff
	Format string

	// Colorize enables color output (only for CLI format)
	Colorize bool

	// ShowIdentical includes identical resources in the output
	ShowIdentical bool

	// Compact produces more compact output (format-specific)
	Compact bool
}

// NewDefaultOptions returns Options with sensible defaults
func NewDefaultOptions() *Options {
	return &Options{
		Format:        "cli",
		Colorize:      true,
		ShowIdentical: false,
		Compact:       false,
	}
}

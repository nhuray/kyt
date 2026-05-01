package reporter

import (
	"fmt"
	"io"

	"github.com/nhuray/kyt/pkg/differ"
	"sigs.k8s.io/yaml"
)

// YAMLReporter formats output as YAML for machine consumption
type YAMLReporter struct {
	options *Options
}

// NewYAMLReporter creates a new YAML reporter
func NewYAMLReporter(opts *Options) *YAMLReporter {
	if opts == nil {
		opts = NewDefaultOptions()
	}
	return &YAMLReporter{
		options: opts,
	}
}

// Report formats and writes the diff result as YAML
func (r *YAMLReporter) Report(result *differ.DiffResult, w io.Writer) error {
	// Use the same structure as JSON output
	jsonReporter := NewJSONReporter(r.options)
	output := jsonReporter.ConvertToJSON(result)

	// Convert to YAML
	yamlBytes, err := yaml.Marshal(output)
	if err != nil {
		return fmt.Errorf("failed to encode YAML: %w", err)
	}

	if _, err := w.Write(yamlBytes); err != nil {
		return fmt.Errorf("failed to write YAML: %w", err)
	}

	return nil
}

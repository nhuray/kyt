package reporter

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/nhuray/k8s-diff/pkg/differ"
	"github.com/nhuray/k8s-diff/pkg/manifest"
)

// JSONReporter formats output as JSON for machine consumption
type JSONReporter struct {
	options *Options
}

// NewJSONReporter creates a new JSON reporter
func NewJSONReporter(opts *Options) *JSONReporter {
	if opts == nil {
		opts = NewDefaultOptions()
	}
	return &JSONReporter{
		options: opts,
	}
}

// JSONOutput represents the JSON structure of the diff output
type JSONOutput struct {
	Summary   SummaryJSON       `json:"summary"`
	Added     []ResourceKeyJSON `json:"added"`
	Removed   []ResourceKeyJSON `json:"removed"`
	Modified  []ModifiedJSON    `json:"modified"`
	Identical []ResourceKeyJSON `json:"identical,omitempty"`
}

// SummaryJSON represents the summary statistics
type SummaryJSON struct {
	TotalResources int `json:"totalResources"`
	Added          int `json:"added"`
	Removed        int `json:"removed"`
	Modified       int `json:"modified"`
	Identical      int `json:"identical"`
}

// ResourceKeyJSON represents a resource identifier in JSON
type ResourceKeyJSON struct {
	Group     string `json:"group"`
	Version   string `json:"version"`
	Kind      string `json:"kind"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name"`
}

// ModifiedJSON represents a modified resource with its diff
type ModifiedJSON struct {
	SourceKey       ResourceKeyJSON `json:"sourceKey"`
	TargetKey       ResourceKeyJSON `json:"targetKey"`
	Key             ResourceKeyJSON `json:"key"` // For backward compatibility
	MatchType       string          `json:"matchType"`
	SimilarityScore float64         `json:"similarityScore"`
	Diff            string          `json:"diff"`
	DiffLines       int             `json:"diffLines"`
}

// Report formats and writes the diff result as JSON
func (r *JSONReporter) Report(result *differ.DiffResult, w io.Writer) error {
	output := r.convertToJSON(result)

	encoder := json.NewEncoder(w)
	if !r.options.Compact {
		encoder.SetIndent("", "  ")
	}

	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// convertToJSON converts DiffResult to JSONOutput
func (r *JSONReporter) convertToJSON(result *differ.DiffResult) *JSONOutput {
	output := &JSONOutput{
		Summary: SummaryJSON{
			TotalResources: result.Summary.TotalResources,
			Added:          result.Summary.AddedCount,
			Removed:        result.Summary.RemovedCount,
			Modified:       result.Summary.ModifiedCount,
			Identical:      result.Summary.IdenticalCount,
		},
		Added:    make([]ResourceKeyJSON, 0, len(result.Added)),
		Removed:  make([]ResourceKeyJSON, 0, len(result.Removed)),
		Modified: make([]ModifiedJSON, 0, len(result.Modified)),
	}

	// Convert added resources
	for _, key := range result.Added {
		output.Added = append(output.Added, convertResourceKey(key))
	}

	// Convert removed resources
	for _, key := range result.Removed {
		output.Removed = append(output.Removed, convertResourceKey(key))
	}

	// Convert modified resources
	for _, diff := range result.Modified {
		output.Modified = append(output.Modified, ModifiedJSON{
			SourceKey:       convertResourceKey(diff.SourceKey),
			TargetKey:       convertResourceKey(diff.TargetKey),
			Key:             convertResourceKey(diff.Key), // For backward compatibility
			MatchType:       diff.MatchType,
			SimilarityScore: diff.SimilarityScore,
			Diff:            diff.DiffText,
			DiffLines:       diff.DiffLines,
		})
	}

	// Include identical resources if requested
	if r.options.ShowIdentical {
		output.Identical = make([]ResourceKeyJSON, 0, len(result.Identical))
		for _, key := range result.Identical {
			output.Identical = append(output.Identical, convertResourceKey(key))
		}
	}

	return output
}

// convertResourceKey converts manifest.ResourceKey to ResourceKeyJSON
func convertResourceKey(key manifest.ResourceKey) ResourceKeyJSON {
	return ResourceKeyJSON{
		Group:     key.Group,
		Version:   key.Version,
		Kind:      key.Kind,
		Namespace: key.Namespace,
		Name:      key.Name,
	}
}

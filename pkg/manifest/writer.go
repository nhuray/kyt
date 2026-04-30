package manifest

import (
	"fmt"
	"io"
	"sort"

	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// WriteYAML writes resources as multi-document YAML to a writer
// Resources are separated by "---" document separator
func WriteYAML(w io.Writer, resources []*unstructured.Unstructured) error {
	if len(resources) == 0 {
		return nil
	}

	// Sort resources for deterministic output
	sorted := make([]*unstructured.Unstructured, len(resources))
	copy(sorted, resources)
	sortResources(sorted)

	// Write each resource as a YAML document
	encoder := yaml.NewEncoder(w)
	encoder.SetIndent(2)
	defer func() {
		_ = encoder.Close() // Best effort close, error already handled below
	}()

	for _, res := range sorted {
		if err := encoder.Encode(res.Object); err != nil {
			key := NewResourceKey(res)
			return fmt.Errorf("failed to encode resource %s: %w", key.String(), err)
		}
	}

	// Explicitly close encoder to catch any errors
	if err := encoder.Close(); err != nil {
		return fmt.Errorf("failed to close encoder: %w", err)
	}

	return nil
}

// WriteYAMLToFile writes resources to a file
func WriteYAMLToFile(filename string, resources []*unstructured.Unstructured) error {
	// Implementation would write to file
	// For now, this is a placeholder
	return fmt.Errorf("WriteYAMLToFile not yet implemented")
}

// sortResources sorts resources by kind, namespace, and name for deterministic output
func sortResources(resources []*unstructured.Unstructured) {
	sort.Slice(resources, func(i, j int) bool {
		keyI := NewResourceKey(resources[i])
		keyJ := NewResourceKey(resources[j])

		// Sort by: Kind -> Namespace -> Name
		if keyI.Kind != keyJ.Kind {
			return keyI.Kind < keyJ.Kind
		}
		if keyI.Namespace != keyJ.Namespace {
			return keyI.Namespace < keyJ.Namespace
		}
		return keyI.Name < keyJ.Name
	})
}

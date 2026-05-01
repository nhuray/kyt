package formatter

import (
	"fmt"
	"sort"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Formatter handles formatting operations on Kubernetes resources
// It only performs formatting (key sorting), not normalization
type Formatter struct{}

// New creates a new Formatter
func New() *Formatter {
	return &Formatter{}
}

// Format applies formatting to a resource (sorts keys alphabetically)
// Returns a formatted copy of the resource
func (f *Formatter) Format(obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	if obj == nil {
		return nil, fmt.Errorf("cannot format nil object")
	}

	// Create a deep copy to avoid modifying the original
	formatted := obj.DeepCopy()

	// Sort all keys recursively
	sorted := SortMapKeys(formatted.Object)
	formatted.Object = sorted

	return formatted, nil
}

// FormatAll applies formatting to multiple resources
func (f *Formatter) FormatAll(objs []*unstructured.Unstructured) ([]*unstructured.Unstructured, error) {
	formatted := make([]*unstructured.Unstructured, 0, len(objs))

	for i, obj := range objs {
		fmtObj, err := f.Format(obj)
		if err != nil {
			return nil, fmt.Errorf("failed to format resource %d: %w", i, err)
		}
		formatted = append(formatted, fmtObj)
	}

	return formatted, nil
}

// SortMapKeys recursively sorts map keys alphabetically
// This function is exported so it can be used by the normalizer package
func SortMapKeys(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Get sorted keys
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build new map with sorted keys
	for _, k := range keys {
		v := m[k]

		// Recursively sort nested maps
		switch val := v.(type) {
		case map[string]interface{}:
			result[k] = SortMapKeys(val)
		case []interface{}:
			result[k] = sortSlice(val)
		default:
			result[k] = v
		}
	}

	return result
}

// sortSlice recursively sorts nested structures in slices
func sortSlice(s []interface{}) []interface{} {
	result := make([]interface{}, len(s))

	for i, v := range s {
		switch val := v.(type) {
		case map[string]interface{}:
			result[i] = SortMapKeys(val)
		case []interface{}:
			result[i] = sortSlice(val)
		default:
			result[i] = v
		}
	}

	return result
}

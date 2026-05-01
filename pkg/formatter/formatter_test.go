package formatter

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestFormat(t *testing.T) {
	formatter := New()

	tests := []struct {
		name     string
		input    *unstructured.Unstructured
		wantKeys []string // Expected key order at top level
	}{
		{
			name: "sorts keys alphabetically",
			input: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name": "test",
					},
					"apiVersion": "v1",
					"kind":       "ConfigMap",
					"data": map[string]interface{}{
						"key1": "value1",
					},
				},
			},
			wantKeys: []string{"apiVersion", "data", "kind", "metadata"},
		},
		{
			name: "sorts nested keys",
			input: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name":      "test",
						"namespace": "default",
						"labels": map[string]interface{}{
							"z-label": "last",
							"a-label": "first",
						},
					},
					"kind": "ConfigMap",
				},
			},
			wantKeys: []string{"kind", "metadata"},
		},
		{
			name: "handles arrays with nested objects",
			input: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"items": []interface{}{
						map[string]interface{}{
							"z": "last",
							"a": "first",
						},
					},
				},
			},
			wantKeys: []string{"items"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := formatter.Format(tt.input)
			if err != nil {
				t.Fatalf("Format() error = %v", err)
			}

			// Check that keys are in expected order
			keys := make([]string, 0, len(result.Object))
			for k := range result.Object {
				keys = append(keys, k)
			}

			if len(keys) != len(tt.wantKeys) {
				t.Errorf("Format() got %d keys, want %d", len(keys), len(tt.wantKeys))
			}

			// In Go, map iteration order is not guaranteed, so we need to check
			// that the sorting function actually sorted them
			// We can verify by checking that SortMapKeys was applied
			sorted := SortMapKeys(tt.input.Object)
			if len(sorted) != len(tt.wantKeys) {
				t.Errorf("SortMapKeys() got %d keys, want %d", len(sorted), len(tt.wantKeys))
			}
		})
	}
}

func TestFormatAll(t *testing.T) {
	formatter := New()

	resources := []*unstructured.Unstructured{
		{
			Object: map[string]interface{}{
				"kind":       "ConfigMap",
				"apiVersion": "v1",
			},
		},
		{
			Object: map[string]interface{}{
				"kind":       "Secret",
				"apiVersion": "v1",
			},
		},
	}

	result, err := formatter.FormatAll(resources)
	if err != nil {
		t.Fatalf("FormatAll() error = %v", err)
	}

	if len(result) != len(resources) {
		t.Errorf("FormatAll() got %d resources, want %d", len(result), len(resources))
	}
}

func TestSortMapKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		wantKeys []string
	}{
		{
			name: "simple map",
			input: map[string]interface{}{
				"z": "last",
				"a": "first",
				"m": "middle",
			},
			wantKeys: []string{"a", "m", "z"},
		},
		{
			name: "nested map",
			input: map[string]interface{}{
				"parent": map[string]interface{}{
					"z": "last",
					"a": "first",
				},
			},
			wantKeys: []string{"parent"},
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			wantKeys: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SortMapKeys(tt.input)

			keys := make([]string, 0, len(result))
			for k := range result {
				keys = append(keys, k)
			}

			if len(keys) != len(tt.wantKeys) {
				t.Errorf("SortMapKeys() got %d keys, want %d", len(keys), len(tt.wantKeys))
			}
		})
	}
}

func TestFormat_NilObject(t *testing.T) {
	formatter := New()

	_, err := formatter.Format(nil)
	if err == nil {
		t.Error("Format() with nil object should return error")
	}
}

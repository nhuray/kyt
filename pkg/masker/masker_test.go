package masker

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestMasker_Mask(t *testing.T) {
	tests := []struct {
		name      string
		keepFirst int
		keepLast  int
		maskChar  string
		input     string
		expected  string
	}{
		{
			name:      "standard base64 value",
			keepFirst: 2,
			keepLast:  2,
			maskChar:  "*",
			input:     "cGFzc3dvcmQxMjM=",
			expected:  "cG************M=",
		},
		{
			name:      "long token",
			keepFirst: 2,
			keepLast:  2,
			maskChar:  "*",
			input:     "dG9rZW4xMjM0NTY3ODkw",
			expected:  "dG****************kw",
		},
		{
			name:      "short value (less than keepFirst+keepLast)",
			keepFirst: 2,
			keepLast:  2,
			maskChar:  "*",
			input:     "abc",
			expected:  "****",
		},
		{
			name:      "exact length (keepFirst+keepLast)",
			keepFirst: 2,
			keepLast:  2,
			maskChar:  "*",
			input:     "abcd",
			expected:  "****",
		},
		{
			name:      "empty string",
			keepFirst: 2,
			keepLast:  2,
			maskChar:  "*",
			input:     "",
			expected:  "<empty>",
		},
		{
			name:      "custom mask character",
			keepFirst: 2,
			keepLast:  2,
			maskChar:  "#",
			input:     "1234567890",
			expected:  "12######90",
		},
		{
			name:      "different keep pattern (3-3)",
			keepFirst: 3,
			keepLast:  3,
			maskChar:  "*",
			input:     "1234567890",
			expected:  "123****890",
		},
		{
			name:      "minimum length to show pattern",
			keepFirst: 2,
			keepLast:  2,
			maskChar:  "*",
			input:     "abcde",
			expected:  "ab*de",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMasker(tt.keepFirst, tt.keepLast, tt.maskChar)
			result := m.Mask(tt.input)
			if result != tt.expected {
				t.Errorf("Mask() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMasker_MaskSecretData(t *testing.T) {
	tests := []struct {
		name     string
		input    *unstructured.Unstructured
		fields   []string
		expected map[string]string
	}{
		{
			name: "mask data field",
			input: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Secret",
					"metadata": map[string]interface{}{
						"name": "test-secret",
					},
					"data": map[string]interface{}{
						"password": "cGFzc3dvcmQxMjM=",
						"token":    "dG9rZW4xMjM0NTY3ODkw",
					},
				},
			},
			fields: []string{"data"},
			expected: map[string]string{
				"password": "cG************M=",
				"token":    "dG****************kw",
			},
		},
		{
			name: "mask stringData field",
			input: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Secret",
					"stringData": map[string]interface{}{
						"api-key": "sk-1234567890abcdef",
					},
				},
			},
			fields: []string{"stringData"},
			expected: map[string]string{
				"api-key": "sk***************ef",
			},
		},
		{
			name: "mask both data and stringData",
			input: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Secret",
					"data": map[string]interface{}{
						"password": "cGFzc3dvcmQxMjM=",
					},
					"stringData": map[string]interface{}{
						"api-key": "sk-1234567890abcdef",
					},
				},
			},
			fields: []string{"data", "stringData"},
			expected: map[string]string{
				"password": "cG************M=",
				"api-key":  "sk***************ef",
			},
		},
		{
			name: "handle missing fields gracefully",
			input: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Secret",
					"data": map[string]interface{}{
						"password": "cGFzc3dvcmQxMjM=",
					},
				},
			},
			fields: []string{"data", "stringData", "nonexistent"},
			expected: map[string]string{
				"password": "cG************M=",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMasker(2, 2, "*")
			result := m.MaskSecretData(tt.input, tt.fields)

			// Verify the original wasn't modified
			if tt.input == result {
				t.Error("MaskSecretData should return a copy, not modify the original")
			}

			// Check masked values
			for field, expectedValues := range tt.expected {
				var fieldData map[string]interface{}
				var found bool
				var err error

				// Try data field first, then stringData
				fieldData, found, err = unstructured.NestedMap(result.Object, "data")
				if !found || err != nil {
					fieldData, found, err = unstructured.NestedMap(result.Object, "stringData")
				}

				if !found || err != nil {
					t.Errorf("Field not found in result")
					continue
				}

				if actualValue, ok := fieldData[field]; ok {
					if actualValue != expectedValues {
						t.Errorf("Field %s = %v, want %v", field, actualValue, expectedValues)
					}
				}
			}
		})
	}
}

func TestMasker_MaskSecretData_NilInput(t *testing.T) {
	m := NewMasker(2, 2, "*")
	result := m.MaskSecretData(nil, []string{"data"})
	if result != nil {
		t.Error("MaskSecretData with nil input should return nil")
	}
}

func TestIsSecret(t *testing.T) {
	tests := []struct {
		name     string
		input    *unstructured.Unstructured
		expected bool
	}{
		{
			name: "is a secret",
			input: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Secret",
				},
			},
			expected: true,
		},
		{
			name: "is not a secret (ConfigMap)",
			input: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
				},
			},
			expected: false,
		},
		{
			name: "is not a secret (Deployment)",
			input: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
				},
			},
			expected: false,
		},
		{
			name:     "nil input",
			input:    nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSecret(tt.input)
			if result != tt.expected {
				t.Errorf("IsSecret() = %v, want %v", result, tt.expected)
			}
		})
	}
}

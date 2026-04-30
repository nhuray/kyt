package differ

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestSimilarityScorer_ExactMatch(t *testing.T) {
	scorer := NewDefaultSimilarityScorer()

	// Create two identical resources
	a := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "test",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"replicas": int64(3),
				"template": map[string]interface{}{
					"image": "nginx:latest",
				},
			},
		},
	}

	b := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "test",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"replicas": int64(3),
				"template": map[string]interface{}{
					"image": "nginx:latest",
				},
			},
		},
	}

	score := scorer.CompareResources(a, b)

	if score != 1.0 {
		t.Errorf("Expected exact match score of 1.0, got %.2f", score)
	}
}

func TestSimilarityScorer_CompletelyDifferent(t *testing.T) {
	scorer := NewDefaultSimilarityScorer()

	// Create two completely different resources
	a := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"spec": map[string]interface{}{
				"replicas": int64(3),
				"image":    "nginx:latest",
				"ports":    []interface{}{int64(80), int64(443)},
			},
		},
	}

	b := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"spec": map[string]interface{}{
				"database": "postgres",
				"version":  "14",
				"storage":  "10Gi",
			},
		},
	}

	score := scorer.CompareResources(a, b)

	// Completely different specs should have very low similarity
	if score > 0.5 {
		t.Errorf("Expected low similarity score for completely different specs, got %.2f", score)
	}
}

func TestSimilarityScorer_PartialMatch(t *testing.T) {
	scorer := NewDefaultSimilarityScorer()

	// Create two partially similar resources
	a := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"spec": map[string]interface{}{
				"replicas": int64(3),
				"image":    "nginx:1.19",
				"port":     int64(80),
			},
		},
	}

	b := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"spec": map[string]interface{}{
				"replicas": int64(5),     // Different value
				"image":    "nginx:1.20", // Different value
				"port":     int64(80),    // Same value
			},
		},
	}

	score := scorer.CompareResources(a, b)

	// Should have intermediate similarity (same keys, some different values)
	if score < 0.3 || score > 0.9 {
		t.Errorf("Expected intermediate similarity score, got %.2f", score)
	}
}

func TestSimilarityScorer_NestedObjectComparison(t *testing.T) {
	scorer := NewDefaultSimilarityScorer()

	// Create resources with nested objects
	a := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"spec": map[string]interface{}{
				"replicas": int64(3),
				"template": map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "nginx",
								"image": "nginx:1.19",
								"resources": map[string]interface{}{
									"limits": map[string]interface{}{
										"cpu":    "500m",
										"memory": "512Mi",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	b := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"spec": map[string]interface{}{
				"replicas": int64(5), // Different
				"template": map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "nginx",
								"image": "nginx:1.20", // Different
								"resources": map[string]interface{}{
									"limits": map[string]interface{}{
										"cpu":    "500m",
										"memory": "512Mi",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	score := scorer.CompareResources(a, b)

	// Should detect structural similarity despite some differences
	// Note: The score will be lower due to weighting of structural matches
	if score < 0.2 || score > 0.5 {
		t.Errorf("Expected moderate structural similarity for nested objects, got %.2f", score)
	}
}

func TestSimilarityScorer_ArrayComparison(t *testing.T) {
	scorer := NewDefaultSimilarityScorer()

	// Create resources with arrays
	a := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"spec": map[string]interface{}{
				"ports": []interface{}{
					map[string]interface{}{
						"port":       int64(80),
						"targetPort": int64(8080),
						"protocol":   "TCP",
					},
					map[string]interface{}{
						"port":       int64(443),
						"targetPort": int64(8443),
						"protocol":   "TCP",
					},
				},
			},
		},
	}

	b := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"spec": map[string]interface{}{
				"ports": []interface{}{
					map[string]interface{}{
						"port":       int64(80),
						"targetPort": int64(8080),
						"protocol":   "TCP",
					},
					map[string]interface{}{
						"port":       int64(443),
						"targetPort": int64(8443),
						"protocol":   "TCP",
					},
				},
			},
		},
	}

	score := scorer.CompareResources(a, b)

	if score != 1.0 {
		t.Errorf("Expected exact match for identical arrays, got %.2f", score)
	}
}

func TestSimilarityScorer_EmptySpecs(t *testing.T) {
	scorer := NewDefaultSimilarityScorer()

	// Create resources with empty specs
	a := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"spec":       map[string]interface{}{},
		},
	}

	b := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"spec":       map[string]interface{}{},
		},
	}

	score := scorer.CompareResources(a, b)

	if score != 1.0 {
		t.Errorf("Expected score of 1.0 for empty specs, got %.2f", score)
	}
}

func TestSimilarityScorer_NoSpec(t *testing.T) {
	scorer := NewDefaultSimilarityScorer()

	// Create resources without spec field (should fall back to full object comparison)
	a := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"data": map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
		},
	}

	b := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"data": map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
		},
	}

	score := scorer.CompareResources(a, b)

	if score != 1.0 {
		t.Errorf("Expected exact match when comparing full objects, got %.2f", score)
	}
}

func TestSimilarityScorer_MissingFields(t *testing.T) {
	scorer := NewDefaultSimilarityScorer()

	// One has field, other doesn't
	a := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"spec": map[string]interface{}{
				"replicas": int64(3),
				"image":    "nginx:latest",
			},
		},
	}

	b := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"spec": map[string]interface{}{
				"replicas": int64(3),
				// Missing "image" field
			},
		},
	}

	score := scorer.CompareResources(a, b)

	// Should have partial similarity (one field matches, one missing)
	if score < 0.3 || score > 0.8 {
		t.Errorf("Expected moderate similarity with missing field, got %.2f", score)
	}
}

func TestSimilarityScorer_DifferentArrayLengths(t *testing.T) {
	scorer := NewDefaultSimilarityScorer()

	// Different array lengths
	a := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"spec": map[string]interface{}{
				"ports": []interface{}{
					int64(80),
					int64(443),
					int64(8080),
				},
			},
		},
	}

	b := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"spec": map[string]interface{}{
				"ports": []interface{}{
					int64(80),
					int64(443),
				},
			},
		},
	}

	score := scorer.CompareResources(a, b)

	// Should have partial similarity (two elements match, one extra)
	if score < 0.4 || score >= 1.0 {
		t.Errorf("Expected partial similarity for different array lengths, got %.2f", score)
	}
}

func TestCompareSpecs_ShortCircuit(t *testing.T) {
	scorer := NewDefaultSimilarityScorer()

	// Create specs with very few overlapping keys (should trigger short-circuit)
	a := map[string]interface{}{
		"field1": "value1",
		"field2": "value2",
		"field3": "value3",
		"field4": "value4",
		"field5": "value5",
	}

	b := map[string]interface{}{
		"field6":  "value6",
		"field7":  "value7",
		"field8":  "value8",
		"field9":  "value9",
		"field10": "value10",
	}

	score := scorer.CompareSpecs(a, b)

	// Should return low score due to no overlapping keys
	if score > 0.3 {
		t.Errorf("Expected very low similarity for non-overlapping keys, got %.2f", score)
	}
}

func TestFormatSimilarity(t *testing.T) {
	tests := []struct {
		score    float64
		expected string
	}{
		{1.0, "1.00"},
		{0.85, "0.85"},
		{0.7, "0.70"},
		{0.123, "0.12"},
		{0.0, "0.00"},
	}

	for _, tt := range tests {
		result := FormatSimilarity(tt.score)
		if result != tt.expected {
			t.Errorf("FormatSimilarity(%.3f) = %s, want %s", tt.score, result, tt.expected)
		}
	}
}

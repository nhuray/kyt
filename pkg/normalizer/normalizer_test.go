package normalizer

import (
	"testing"

	"github.com/nhuray/k8s-diff/pkg/config"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestNormalize(t *testing.T) {
	cfg := &config.Config{
		Normalization: config.NormalizationConfig{
			SortKeys: true,
			RemoveDefaultFields: []string{
				"/status",
				"/metadata/creationTimestamp",
			},
		},
	}

	normalizer := New(cfg)

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name":              "my-service",
				"namespace":         "default",
				"creationTimestamp": "2024-01-01T00:00:00Z",
			},
			"spec": map[string]interface{}{
				"ports": []interface{}{
					map[string]interface{}{
						"port": int64(80),
					},
				},
			},
			"status": map[string]interface{}{
				"loadBalancer": map[string]interface{}{},
			},
		},
	}

	normalized, err := normalizer.Normalize(obj)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	// Check that status was removed
	if _, ok := normalized.Object["status"]; ok {
		t.Error("Expected status to be removed")
	}

	// Check that creationTimestamp was removed
	metadata := normalized.Object["metadata"].(map[string]interface{})
	if _, ok := metadata["creationTimestamp"]; ok {
		t.Error("Expected creationTimestamp to be removed")
	}

	// Check that original object was not modified
	if _, ok := obj.Object["status"]; !ok {
		t.Error("Original object should not be modified")
	}
}

func TestRemoveJSONPointerField(t *testing.T) {
	tests := []struct {
		name        string
		obj         *unstructured.Unstructured
		pointer     string
		shouldExist bool
		expectError bool
	}{
		{
			name: "remove simple field",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"app": "test",
						},
					},
				},
			},
			pointer:     "/metadata/labels",
			shouldExist: false,
			expectError: false,
		},
		{
			name: "remove nested field",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"template": map[string]interface{}{
							"metadata": map[string]interface{}{
								"labels": map[string]interface{}{
									"app": "test",
								},
							},
						},
					},
				},
			},
			pointer:     "/spec/template/metadata/labels",
			shouldExist: false,
			expectError: false,
		},
		{
			name: "field doesn't exist - should not error",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{},
				},
			},
			pointer:     "/metadata/nonexistent",
			shouldExist: false,
			expectError: false, // Changed: not finding a field is not an error in normalization context
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := removeJSONPointerField(tt.obj, tt.pointer)

			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestParseJSONPointer(t *testing.T) {
	tests := []struct {
		name     string
		pointer  string
		expected []string
	}{
		{
			name:     "simple path",
			pointer:  "/metadata/labels",
			expected: []string{"metadata", "labels"},
		},
		{
			name:     "nested path",
			pointer:  "/spec/template/metadata/labels",
			expected: []string{"spec", "template", "metadata", "labels"},
		},
		{
			name:     "with escape for /",
			pointer:  "/metadata/annotations/kubectl.kubernetes.io~1last-applied-configuration",
			expected: []string{"metadata", "annotations", "kubectl.kubernetes.io/last-applied-configuration"},
		},
		{
			name:     "with escape for ~",
			pointer:  "/metadata/labels/app~0version",
			expected: []string{"metadata", "labels", "app~version"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseJSONPointer(tt.pointer)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d parts, got %d", len(tt.expected), len(result))
				return
			}

			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("Part %d: expected %s, got %s", i, tt.expected[i], result[i])
				}
			}
		})
	}
}

func TestSortKeys(t *testing.T) {
	cfg := &config.Config{
		Normalization: config.NormalizationConfig{
			SortKeys: true,
		},
	}

	normalizer := New(cfg)

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"z_field": "value",
			"a_field": "value",
			"m_field": "value",
		},
	}

	normalized, err := normalizer.Normalize(obj)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	// Check that all fields still exist
	if _, ok := normalized.Object["z_field"]; !ok {
		t.Error("Expected z_field to exist")
	}
	if _, ok := normalized.Object["a_field"]; !ok {
		t.Error("Expected a_field to exist")
	}
	if _, ok := normalized.Object["m_field"]; !ok {
		t.Error("Expected m_field to exist")
	}

	// Note: Go maps don't preserve order, so we can't directly test
	// the ordering. The sortKeys function ensures consistent JSON output
	// but the map itself remains unordered in memory.
}

func TestIgnoreRules(t *testing.T) {
	cfg := &config.Config{
		IgnoreDifferences: []config.ResourceIgnoreDifferences{
			{
				Group: "apps",
				Kind:  "Deployment",
				JSONPointers: []string{
					"/spec/replicas",
				},
			},
		},
	}

	normalizer := New(cfg)

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      "test-deployment",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"replicas": int64(3),
				"selector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"app": "test",
					},
				},
			},
		},
	}

	normalized, err := normalizer.Normalize(obj)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	// Check that replicas was removed
	spec := normalized.Object["spec"].(map[string]interface{})
	if _, ok := spec["replicas"]; ok {
		t.Error("Expected replicas to be removed")
	}

	// Check that selector still exists
	if _, ok := spec["selector"]; !ok {
		t.Error("Expected selector to still exist")
	}
}

func TestJQExpression(t *testing.T) {
	cfg := &config.Config{
		IgnoreDifferences: []config.ResourceIgnoreDifferences{
			{
				Group: "apps",
				Kind:  "Deployment",
				JQPathExpressions: []string{
					`.spec.template.spec.containers[] | select(.name == "istio-proxy")`,
				},
			},
		},
	}

	normalizer := New(cfg)

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      "test-deployment",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"template": map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "app",
								"image": "app:latest",
							},
							map[string]interface{}{
								"name":  "istio-proxy",
								"image": "istio/proxyv2:latest",
							},
						},
					},
				},
			},
		},
	}

	normalized, err := normalizer.Normalize(obj)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	// Get containers
	spec := normalized.Object["spec"].(map[string]interface{})
	template := spec["template"].(map[string]interface{})
	templateSpec := template["spec"].(map[string]interface{})
	containers := templateSpec["containers"].([]interface{})

	// Check that istio-proxy was removed
	if len(containers) != 1 {
		t.Errorf("Expected 1 container, got %d", len(containers))
	}

	// Check that the remaining container is "app"
	container := containers[0].(map[string]interface{})
	if container["name"] != "app" {
		t.Errorf("Expected remaining container to be 'app', got %s", container["name"])
	}
}

func TestRemoveManagedFieldsByManagers(t *testing.T) {
	cfg := &config.Config{
		IgnoreDifferences: []config.ResourceIgnoreDifferences{
			{
				Group: "",
				Kind:  "Service",
				ManagedFieldsManagers: []string{
					"kube-controller-manager",
				},
			},
		},
	}

	normalizer := New(cfg)

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name":      "test-service",
				"namespace": "default",
				"managedFields": []interface{}{
					map[string]interface{}{
						"manager":   "kubectl",
						"operation": "Update",
					},
					map[string]interface{}{
						"manager":   "kube-controller-manager",
						"operation": "Update",
					},
				},
			},
		},
	}

	normalized, err := normalizer.Normalize(obj)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	// Get managedFields
	metadata := normalized.Object["metadata"].(map[string]interface{})
	managedFields := metadata["managedFields"].([]interface{})

	// Check that only kubectl manager remains
	if len(managedFields) != 1 {
		t.Errorf("Expected 1 managedField entry, got %d", len(managedFields))
	}

	field := managedFields[0].(map[string]interface{})
	if field["manager"] != "kubectl" {
		t.Errorf("Expected remaining manager to be 'kubectl', got %s", field["manager"])
	}
}

func TestNormalizeAll(t *testing.T) {
	cfg := &config.Config{
		Normalization: config.NormalizationConfig{
			SortKeys: true,
			RemoveDefaultFields: []string{
				"/status",
			},
		},
	}

	normalizer := New(cfg)

	objs := []*unstructured.Unstructured{
		{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Service",
				"metadata": map[string]interface{}{
					"name": "service-1",
				},
				"status": map[string]interface{}{},
			},
		},
		{
			Object: map[string]interface{}{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]interface{}{
					"name": "deployment-1",
				},
				"status": map[string]interface{}{},
			},
		},
	}

	normalized, err := normalizer.NormalizeAll(objs)
	if err != nil {
		t.Fatalf("NormalizeAll failed: %v", err)
	}

	if len(normalized) != 2 {
		t.Errorf("Expected 2 normalized objects, got %d", len(normalized))
	}

	// Check that status was removed from all objects
	for i, obj := range normalized {
		if _, ok := obj.Object["status"]; ok {
			t.Errorf("Object %d: expected status to be removed", i)
		}
	}
}

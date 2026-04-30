package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestNewResourceKey(t *testing.T) {
	tests := []struct {
		name     string
		obj      *unstructured.Unstructured
		expected ResourceKey
	}{
		{
			name: "core resource with namespace",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Service",
					"metadata": map[string]interface{}{
						"name":      "my-service",
						"namespace": "default",
					},
				},
			},
			expected: ResourceKey{
				Group:     "",
				Version:   "v1",
				Kind:      "Service",
				Namespace: "default",
				Name:      "my-service",
			},
		},
		{
			name: "namespaced resource with group",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"metadata": map[string]interface{}{
						"name":      "my-deployment",
						"namespace": "production",
					},
				},
			},
			expected: ResourceKey{
				Group:     "apps",
				Version:   "v1",
				Kind:      "Deployment",
				Namespace: "production",
				Name:      "my-deployment",
			},
		},
		{
			name: "cluster-scoped resource",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Namespace",
					"metadata": map[string]interface{}{
						"name": "my-namespace",
					},
				},
			},
			expected: ResourceKey{
				Group:     "",
				Version:   "v1",
				Kind:      "Namespace",
				Namespace: "",
				Name:      "my-namespace",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := NewResourceKey(tt.obj)

			if key.Group != tt.expected.Group {
				t.Errorf("Group: got %s, want %s", key.Group, tt.expected.Group)
			}
			if key.Version != tt.expected.Version {
				t.Errorf("Version: got %s, want %s", key.Version, tt.expected.Version)
			}
			if key.Kind != tt.expected.Kind {
				t.Errorf("Kind: got %s, want %s", key.Kind, tt.expected.Kind)
			}
			if key.Namespace != tt.expected.Namespace {
				t.Errorf("Namespace: got %s, want %s", key.Namespace, tt.expected.Namespace)
			}
			if key.Name != tt.expected.Name {
				t.Errorf("Name: got %s, want %s", key.Name, tt.expected.Name)
			}
		})
	}
}

func TestResourceKeyString(t *testing.T) {
	tests := []struct {
		name     string
		key      ResourceKey
		expected string
	}{
		{
			name: "core resource with namespace",
			key: ResourceKey{
				Group:     "",
				Version:   "v1",
				Kind:      "Service",
				Namespace: "default",
				Name:      "my-service",
			},
			expected: "Service.core/my-service (namespace: default)",
		},
		{
			name: "resource with group",
			key: ResourceKey{
				Group:     "apps",
				Version:   "v1",
				Kind:      "Deployment",
				Namespace: "production",
				Name:      "my-deployment",
			},
			expected: "Deployment.apps/my-deployment (namespace: production)",
		},
		{
			name: "cluster-scoped resource",
			key: ResourceKey{
				Group:   "",
				Version: "v1",
				Kind:    "Namespace",
				Name:    "my-namespace",
			},
			expected: "Namespace.core/my-namespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.key.String()
			if result != tt.expected {
				t.Errorf("got %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestResourceKeyAPIVersion(t *testing.T) {
	tests := []struct {
		name     string
		key      ResourceKey
		expected string
	}{
		{
			name: "core resource",
			key: ResourceKey{
				Group:   "",
				Version: "v1",
			},
			expected: "v1",
		},
		{
			name: "resource with group",
			key: ResourceKey{
				Group:   "apps",
				Version: "v1",
			},
			expected: "apps/v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.key.APIVersion()
			if result != tt.expected {
				t.Errorf("got %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestManifestSetAdd(t *testing.T) {
	ms := NewManifestSet()

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name":      "my-service",
				"namespace": "default",
			},
		},
	}

	// First add should succeed
	err := ms.Add(obj)
	if err != nil {
		t.Fatalf("First Add failed: %v", err)
	}

	if ms.Len() != 1 {
		t.Errorf("Expected length 1, got %d", ms.Len())
	}

	// Second add of same resource should fail
	err = ms.Add(obj)
	if err == nil {
		t.Error("Expected error when adding duplicate resource, got nil")
	}
}

func TestManifestSetMerge(t *testing.T) {
	ms1 := NewManifestSet()
	ms2 := NewManifestSet()

	obj1 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name":      "service-1",
				"namespace": "default",
			},
		},
	}

	obj2 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name":      "service-2",
				"namespace": "default",
			},
		},
	}

	_ = ms1.Add(obj1)
	_ = ms2.Add(obj2)

	// Merge should succeed
	err := ms1.Merge(ms2)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	if ms1.Len() != 2 {
		t.Errorf("Expected length 2 after merge, got %d", ms1.Len())
	}

	// Merging again should fail due to duplicate
	err = ms1.Merge(ms2)
	if err == nil {
		t.Error("Expected error when merging duplicate resources, got nil")
	}
}

func TestParseBytes(t *testing.T) {
	tests := []struct {
		name          string
		yaml          string
		expectedCount int
		expectError   bool
	}{
		{
			name: "single valid resource",
			yaml: `
apiVersion: v1
kind: Service
metadata:
  name: my-service
  namespace: default
spec:
  ports:
  - port: 80
`,
			expectedCount: 1,
			expectError:   false,
		},
		{
			name: "multi-document YAML",
			yaml: `
apiVersion: v1
kind: Service
metadata:
  name: service-1
  namespace: default
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config-1
  namespace: default
data:
  key: value
`,
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "empty documents ignored",
			yaml: `
---
apiVersion: v1
kind: Service
metadata:
  name: my-service
  namespace: default
---
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
  namespace: default
---
`,
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "missing required field kind",
			yaml: `
apiVersion: v1
metadata:
  name: my-service
  namespace: default
`,
			expectedCount: 0,
			expectError:   true,
		},
		{
			name: "missing required field apiVersion",
			yaml: `
kind: Service
metadata:
  name: my-service
  namespace: default
`,
			expectedCount: 0,
			expectError:   true,
		},
		{
			name: "missing required field metadata.name",
			yaml: `
apiVersion: v1
kind: Service
metadata:
  namespace: default
`,
			expectedCount: 0,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			ms, err := parser.ParseBytes([]byte(tt.yaml))

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if ms.Len() != tt.expectedCount {
				t.Errorf("Expected %d resources, got %d", tt.expectedCount, ms.Len())
			}
		})
	}
}

func TestParseFile(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.yaml")

	content := `
apiVersion: v1
kind: Service
metadata:
  name: test-service
  namespace: default
spec:
  ports:
  - port: 80
`

	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	parser := NewParser()
	ms, err := parser.ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if ms.Len() != 1 {
		t.Errorf("Expected 1 resource, got %d", ms.Len())
	}
}

func TestParseDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	files := map[string]string{
		"service.yaml": `
apiVersion: v1
kind: Service
metadata:
  name: my-service
  namespace: default
`,
		"deployment.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deployment
  namespace: default
spec:
  replicas: 1
`,
		"config.txt": "this should be ignored",
		"subdir/configmap.yml": `
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
  namespace: default
data:
  key: value
`,
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)

		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	parser := NewParser()
	ms, err := parser.ParseDirectory(tmpDir)
	if err != nil {
		t.Fatalf("ParseDirectory failed: %v", err)
	}

	// Should find 3 resources (service, deployment, configmap)
	// config.txt should be ignored
	if ms.Len() != 3 {
		t.Errorf("Expected 3 resources, got %d", ms.Len())
	}
}

func TestParseSkipInvalid(t *testing.T) {
	yaml := `
apiVersion: v1
kind: Service
metadata:
  name: valid-service
  namespace: default
---
apiVersion: v1
metadata:
  name: invalid-no-kind
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: valid-config
  namespace: default
data:
  key: value
`

	// Without SkipInvalid, should fail
	parser := NewParser()
	parser.SkipInvalid = false
	_, err := parser.ParseBytes([]byte(yaml))
	if err == nil {
		t.Error("Expected error without SkipInvalid, got nil")
	}

	// With SkipInvalid, should succeed and skip the invalid resource
	parser.SkipInvalid = true
	ms, err := parser.ParseBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseBytes with SkipInvalid failed: %v", err)
	}

	// Should only have 2 valid resources
	if ms.Len() != 2 {
		t.Errorf("Expected 2 resources with SkipInvalid, got %d", ms.Len())
	}
}

func TestParseReader(t *testing.T) {
	yaml := `
apiVersion: v1
kind: Service
metadata:
  name: test-service
  namespace: default
`

	reader := strings.NewReader(yaml)
	parser := NewParser()

	ms, err := parser.ParseReader(reader)
	if err != nil {
		t.Fatalf("ParseReader failed: %v", err)
	}

	if ms.Len() != 1 {
		t.Errorf("Expected 1 resource, got %d", ms.Len())
	}
}

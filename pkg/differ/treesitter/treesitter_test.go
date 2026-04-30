package treesitter

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestValidateKubernetesResource(t *testing.T) {
	tests := []struct {
		name    string
		obj     *unstructured.Unstructured
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid resource",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
					"metadata": map[string]interface{}{
						"name": "test-config",
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "nil resource",
			obj:     nil,
			wantErr: true,
			errMsg:  "resource is nil",
		},
		{
			name: "missing apiVersion",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "ConfigMap",
					"metadata": map[string]interface{}{
						"name": "test-config",
					},
				},
			},
			wantErr: true,
			errMsg:  "missing required field: apiVersion",
		},
		{
			name: "missing kind",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"metadata": map[string]interface{}{
						"name": "test-config",
					},
				},
			},
			wantErr: true,
			errMsg:  "missing required field: kind",
		},
		{
			name: "missing metadata.name",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
					"metadata":   map[string]interface{}{},
				},
			},
			wantErr: true,
			errMsg:  "missing required field: metadata.name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKubernetesResource(tt.obj)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateKubernetesResource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("ValidateKubernetesResource() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestParser_ParseYAML(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name: "valid YAML",
			content: `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
data:
  key: value`,
			wantErr: false,
		},
		{
			name:    "empty YAML",
			content: "",
			wantErr: false, // Empty is valid YAML
		},
		{
			name: "nested YAML",
			content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
spec:
  replicas: 3
  selector:
    matchLabels:
      app: test`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			defer parser.Close()

			tree, err := parser.ParseYAML([]byte(tt.content))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tree == nil {
				t.Error("ParseYAML() returned nil tree without error")
			}
		})
	}
}

func TestDiffer_Diff_Scalar(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		target     string
		wantChange ChangeType
	}{
		{
			name:       "identical scalars",
			source:     `name: test`,
			target:     `name: test`,
			wantChange: Unchanged,
		},
		{
			name:       "modified scalar",
			source:     `name: test`,
			target:     `name: changed`,
			wantChange: Modified,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			defer parser.Close()

			sourceTree, err := parser.ParseYAML([]byte(tt.source))
			if err != nil {
				t.Fatalf("Failed to parse source: %v", err)
			}

			targetTree, err := parser.ParseYAML([]byte(tt.target))
			if err != nil {
				t.Fatalf("Failed to parse target: %v", err)
			}

			differ := NewDiffer(sourceTree, targetTree, []byte(tt.source), []byte(tt.target))
			result, err := differ.Diff()
			if err != nil {
				t.Fatalf("Diff() error = %v", err)
			}

			if result == nil {
				t.Fatal("Diff() returned nil result")
			}

			// Check the aggregate change type
			if result.Type != tt.wantChange {
				t.Errorf("Diff() change type = %v, want %v", result.Type, tt.wantChange)
			}
		})
	}
}

func TestDiffer_Diff_Mapping(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		target     string
		wantChange ChangeType
	}{
		{
			name: "identical mapping",
			source: `metadata:
  name: test
  namespace: default`,
			target: `metadata:
  name: test
  namespace: default`,
			wantChange: Unchanged,
		},
		{
			name: "added key",
			source: `metadata:
  name: test`,
			target: `metadata:
  name: test
  namespace: default`,
			wantChange: Modified,
		},
		{
			name: "removed key",
			source: `metadata:
  name: test
  namespace: default`,
			target: `metadata:
  name: test`,
			wantChange: Modified,
		},
		{
			name: "modified value",
			source: `metadata:
  name: test
  namespace: default`,
			target: `metadata:
  name: test
  namespace: production`,
			wantChange: Modified,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			defer parser.Close()

			sourceTree, err := parser.ParseYAML([]byte(tt.source))
			if err != nil {
				t.Fatalf("Failed to parse source: %v", err)
			}

			targetTree, err := parser.ParseYAML([]byte(tt.target))
			if err != nil {
				t.Fatalf("Failed to parse target: %v", err)
			}

			differ := NewDiffer(sourceTree, targetTree, []byte(tt.source), []byte(tt.target))
			result, err := differ.Diff()
			if err != nil {
				t.Fatalf("Diff() error = %v", err)
			}

			if result == nil {
				t.Fatal("Diff() returned nil result")
			}

			if result.Type != tt.wantChange {
				t.Errorf("Diff() change type = %v, want %v", result.Type, tt.wantChange)
			}
		})
	}
}

func TestDiffer_Diff_Sequence(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		target     string
		wantChange ChangeType
	}{
		{
			name: "identical sequence",
			source: `items:
  - name: one
  - name: two`,
			target: `items:
  - name: one
  - name: two`,
			wantChange: Unchanged,
		},
		{
			name: "added element",
			source: `items:
  - name: one`,
			target: `items:
  - name: one
  - name: two`,
			wantChange: Modified,
		},
		{
			name: "removed element",
			source: `items:
  - name: one
  - name: two`,
			target: `items:
  - name: one`,
			wantChange: Modified,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			defer parser.Close()

			sourceTree, err := parser.ParseYAML([]byte(tt.source))
			if err != nil {
				t.Fatalf("Failed to parse source: %v", err)
			}

			targetTree, err := parser.ParseYAML([]byte(tt.target))
			if err != nil {
				t.Fatalf("Failed to parse target: %v", err)
			}

			differ := NewDiffer(sourceTree, targetTree, []byte(tt.source), []byte(tt.target))
			result, err := differ.Diff()
			if err != nil {
				t.Fatalf("Diff() error = %v", err)
			}

			if result == nil {
				t.Fatal("Diff() returned nil result")
			}

			if result.Type != tt.wantChange {
				t.Errorf("Diff() change type = %v, want %v", result.Type, tt.wantChange)
			}
		})
	}
}

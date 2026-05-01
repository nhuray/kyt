package treesitter

import (
	"strings"
	"testing"

	"github.com/fatih/color"
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

func TestFormatter_FormatSideBySide(t *testing.T) {
	tests := []struct {
		name        string
		node        *DiffNode
		width       int
		useColor    bool
		wantContent []string // Strings that should be present in output
	}{
		{
			name: "unchanged node",
			node: &DiffNode{
				Type:       Unchanged,
				Key:        "name",
				SourceText: "test",
				TargetText: "test",
			},
			width:    80,
			useColor: false,
			wantContent: []string{
				"name: test",
				"│",
			},
		},
		{
			name: "added node",
			node: &DiffNode{
				Type:       Added,
				Key:        "namespace",
				TargetText: "production",
			},
			width:    80,
			useColor: false,
			wantContent: []string{
				"namespace: production",
				"│",
			},
		},
		{
			name: "removed node",
			node: &DiffNode{
				Type:       Removed,
				Key:        "oldKey",
				SourceText: "oldValue",
			},
			width:    80,
			useColor: false,
			wantContent: []string{
				"oldKey: oldValue",
				"│",
			},
		},
		{
			name: "modified node",
			node: &DiffNode{
				Type:       Modified,
				Key:        "replicas",
				SourceText: "3",
				TargetText: "5",
			},
			width:    80,
			useColor: false,
			wantContent: []string{
				"replicas: 3",
				"replicas: 5",
				"│",
			},
		},
		{
			name: "nested nodes",
			node: &DiffNode{
				Type:       Modified,
				Key:        "metadata",
				SourceText: "",
				TargetText: "",
				Children: []*DiffNode{
					{
						Type:       Unchanged,
						Key:        "name",
						SourceText: "test",
						TargetText: "test",
					},
					{
						Type:       Modified,
						Key:        "namespace",
						SourceText: "default",
						TargetText: "production",
					},
				},
			},
			width:    80,
			useColor: false,
			wantContent: []string{
				"name: test",
				"namespace: default",
				"namespace: production",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewFormatter(tt.width, tt.useColor, 2)
			output := formatter.FormatSideBySide(tt.node, "Source", "Target")

			for _, want := range tt.wantContent {
				if !strings.Contains(output, want) {
					t.Errorf("FormatSideBySide() output missing %q\nGot:\n%s", want, output)
				}
			}

			// Check for header
			if !strings.Contains(output, "Source") || !strings.Contains(output, "Target") {
				t.Error("FormatSideBySide() output missing header labels")
			}
		})
	}
}

func TestFormatter_Colorize(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		useColor bool
	}{
		{
			name:     "color enabled",
			text:     "test",
			useColor: true,
		},
		{
			name:     "color disabled",
			text:     "test",
			useColor: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewFormatter(80, tt.useColor, 2)
			output := formatter.colorize(tt.text, color.FgRed)

			// Text should always be present
			if !strings.Contains(output, tt.text) {
				t.Errorf("colorize() output missing original text %q", tt.text)
			}

			// When color is disabled, output should equal input
			if !tt.useColor && output != tt.text {
				t.Errorf("colorize() with useColor=false should return unmodified text, got %q want %q", output, tt.text)
			}
		})
	}
}

func TestFormatter_Pad(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		width     int
		wantLen   int
		wantTrunc bool
	}{
		{
			name:      "pad short string",
			input:     "test",
			width:     10,
			wantLen:   10,
			wantTrunc: false,
		},
		{
			name:      "exact width",
			input:     "test",
			width:     4,
			wantLen:   4,
			wantTrunc: false,
		},
		{
			name:      "truncate long string",
			input:     "this is a very long string",
			width:     10,
			wantLen:   10,
			wantTrunc: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewFormatter(80, false, 2)
			output := formatter.pad(tt.input, tt.width)

			if len(output) != tt.wantLen {
				t.Errorf("pad() length = %d, want %d", len(output), tt.wantLen)
			}

			if tt.wantTrunc && !strings.Contains(output, "...") {
				t.Error("pad() should truncate with ellipsis")
			}
		})
	}
}

func TestNewFormatter_Defaults(t *testing.T) {
	tests := []struct {
		name           string
		width          int
		indentSize     int
		wantWidth      int
		wantIndentSize int
	}{
		{
			name:           "valid values",
			width:          100,
			indentSize:     4,
			wantWidth:      100,
			wantIndentSize: 4,
		},
		{
			name:           "zero width uses default",
			width:          0,
			indentSize:     4,
			wantWidth:      120,
			wantIndentSize: 4,
		},
		{
			name:           "zero indent uses default",
			width:          100,
			indentSize:     0,
			wantWidth:      100,
			wantIndentSize: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewFormatter(tt.width, false, tt.indentSize)

			if formatter.width != tt.wantWidth {
				t.Errorf("NewFormatter() width = %d, want %d", formatter.width, tt.wantWidth)
			}

			if formatter.indentSize != tt.wantIndentSize {
				t.Errorf("NewFormatter() indentSize = %d, want %d", formatter.indentSize, tt.wantIndentSize)
			}
		})
	}
}

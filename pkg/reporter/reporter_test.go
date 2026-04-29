package reporter

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/nicolasleigh/k8s-diff/pkg/differ"
	"github.com/nicolasleigh/k8s-diff/pkg/manifest"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestCLIReporter_Report(t *testing.T) {
	result := createTestDiffResult()

	tests := []struct {
		name           string
		options        *Options
		expectedStrs   []string
		unexpectedStrs []string
	}{
		{
			name: "basic output without colors",
			options: &Options{
				Format:        "cli",
				Colorize:      false,
				ShowIdentical: false,
			},
			expectedStrs: []string{
				"k8s-diff Report",
				"Added Resources (1):",
				"Removed Resources (1):",
				"Modified Resources (1):",
				"Summary:",
				"Total Resources:",
				"Added:",
				"Removed:",
				"Modified:",
			},
		},
		{
			name: "with identical resources",
			options: &Options{
				Format:        "cli",
				Colorize:      false,
				ShowIdentical: true,
			},
			expectedStrs: []string{
				"Identical Resources (1):",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reporter := NewCLIReporter(tt.options)
			buf := &bytes.Buffer{}

			err := reporter.Report(result, buf)
			if err != nil {
				t.Fatalf("Report failed: %v", err)
			}

			output := buf.String()

			for _, expected := range tt.expectedStrs {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q", expected)
				}
			}

			for _, unexpected := range tt.unexpectedStrs {
				if strings.Contains(output, unexpected) {
					t.Errorf("Expected output to NOT contain %q", unexpected)
				}
			}
		})
	}
}

func TestCLIReporter_NoDifferences(t *testing.T) {
	result := &differ.DiffResult{
		Added:    []manifest.ResourceKey{},
		Removed:  []manifest.ResourceKey{},
		Modified: []differ.ResourceDiff{},
		Identical: []manifest.ResourceKey{
			{Kind: "Service", Name: "test"},
		},
		Summary: differ.DiffSummary{
			TotalResources: 1,
			IdenticalCount: 1,
		},
	}

	reporter := NewCLIReporter(&Options{
		Colorize:      false,
		ShowIdentical: false,
	})
	buf := &bytes.Buffer{}

	err := reporter.Report(result, buf)
	if err != nil {
		t.Fatalf("Report failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "No differences found") {
		t.Error("Expected 'No differences found' message")
	}
}

func TestCLIReporter_Differences(t *testing.T) {
	result := createTestDiffResult()

	reporter := NewCLIReporter(&Options{
		Colorize:      false,
		ShowIdentical: false,
	})
	buf := &bytes.Buffer{}

	err := reporter.Report(result, buf)
	if err != nil {
		t.Fatalf("Report failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "Differences detected") {
		t.Error("Expected 'Differences detected' message")
	}
}

func TestJSONReporter_Report(t *testing.T) {
	result := createTestDiffResult()

	tests := []struct {
		name    string
		options *Options
	}{
		{
			name: "compact JSON",
			options: &Options{
				Format:        "json",
				Compact:       true,
				ShowIdentical: false,
			},
		},
		{
			name: "pretty JSON",
			options: &Options{
				Format:        "json",
				Compact:       false,
				ShowIdentical: false,
			},
		},
		{
			name: "with identical resources",
			options: &Options{
				Format:        "json",
				Compact:       false,
				ShowIdentical: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reporter := NewJSONReporter(tt.options)
			buf := &bytes.Buffer{}

			err := reporter.Report(result, buf)
			if err != nil {
				t.Fatalf("Report failed: %v", err)
			}

			// Parse JSON to verify it's valid
			var output JSONOutput
			if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
				t.Fatalf("Failed to parse JSON output: %v", err)
			}

			// Verify summary
			if output.Summary.TotalResources != 4 {
				t.Errorf("Expected TotalResources=4, got %d", output.Summary.TotalResources)
			}

			if output.Summary.Added != 1 {
				t.Errorf("Expected Added=1, got %d", output.Summary.Added)
			}

			if output.Summary.Removed != 1 {
				t.Errorf("Expected Removed=1, got %d", output.Summary.Removed)
			}

			if output.Summary.Modified != 1 {
				t.Errorf("Expected Modified=1, got %d", output.Summary.Modified)
			}

			if output.Summary.Identical != 1 {
				t.Errorf("Expected Identical=1, got %d", output.Summary.Identical)
			}

			// Verify arrays
			if len(output.Added) != 1 {
				t.Errorf("Expected 1 added resource, got %d", len(output.Added))
			}

			if len(output.Removed) != 1 {
				t.Errorf("Expected 1 removed resource, got %d", len(output.Removed))
			}

			if len(output.Modified) != 1 {
				t.Errorf("Expected 1 modified resource, got %d", len(output.Modified))
			}

			// Check if identical is included based on option
			if tt.options.ShowIdentical {
				if len(output.Identical) != 1 {
					t.Errorf("Expected 1 identical resource, got %d", len(output.Identical))
				}
			} else {
				if output.Identical != nil {
					t.Error("Expected Identical to be omitted when ShowIdentical=false")
				}
			}
		})
	}
}

func TestJSONReporter_ResourceKeyConversion(t *testing.T) {
	key := manifest.ResourceKey{
		Group:     "apps",
		Version:   "v1",
		Kind:      "Deployment",
		Namespace: "default",
		Name:      "test-app",
	}

	jsonKey := convertResourceKey(key)

	if jsonKey.Group != "apps" {
		t.Errorf("Expected Group=apps, got %s", jsonKey.Group)
	}

	if jsonKey.Version != "v1" {
		t.Errorf("Expected Version=v1, got %s", jsonKey.Version)
	}

	if jsonKey.Kind != "Deployment" {
		t.Errorf("Expected Kind=Deployment, got %s", jsonKey.Kind)
	}

	if jsonKey.Namespace != "default" {
		t.Errorf("Expected Namespace=default, got %s", jsonKey.Namespace)
	}

	if jsonKey.Name != "test-app" {
		t.Errorf("Expected Name=test-app, got %s", jsonKey.Name)
	}
}

func TestJSONReporter_EmptyResult(t *testing.T) {
	result := &differ.DiffResult{
		Added:     []manifest.ResourceKey{},
		Removed:   []manifest.ResourceKey{},
		Modified:  []differ.ResourceDiff{},
		Identical: []manifest.ResourceKey{},
		Summary: differ.DiffSummary{
			TotalResources: 0,
		},
	}

	reporter := NewJSONReporter(nil)
	buf := &bytes.Buffer{}

	err := reporter.Report(result, buf)
	if err != nil {
		t.Fatalf("Report failed: %v", err)
	}

	// Parse JSON to verify it's valid
	var output JSONOutput
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if output.Summary.TotalResources != 0 {
		t.Errorf("Expected TotalResources=0, got %d", output.Summary.TotalResources)
	}

	if len(output.Added) != 0 {
		t.Errorf("Expected 0 added resources, got %d", len(output.Added))
	}
}

// Helper function to create a test diff result
func createTestDiffResult() *differ.DiffResult {
	return &differ.DiffResult{
		Added: []manifest.ResourceKey{
			{
				Group:     "",
				Version:   "v1",
				Kind:      "Service",
				Namespace: "default",
				Name:      "new-service",
			},
		},
		Removed: []manifest.ResourceKey{
			{
				Group:     "",
				Version:   "v1",
				Kind:      "Service",
				Namespace: "default",
				Name:      "old-service",
			},
		},
		Modified: []differ.ResourceDiff{
			{
				Key: manifest.ResourceKey{
					Group:     "apps",
					Version:   "v1",
					Kind:      "Deployment",
					Namespace: "default",
					Name:      "my-app",
				},
				Source: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "apps/v1",
						"kind":       "Deployment",
						"metadata": map[string]interface{}{
							"name": "my-app",
						},
					},
				},
				Target: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "apps/v1",
						"kind":       "Deployment",
						"metadata": map[string]interface{}{
							"name": "my-app",
						},
					},
				},
				DiffText:  "--- a/source\n+++ b/target\n@@ -1,1 +1,1 @@\n-old\n+new\n",
				DiffLines: 4,
			},
		},
		Identical: []manifest.ResourceKey{
			{
				Group:     "",
				Version:   "v1",
				Kind:      "ConfigMap",
				Namespace: "default",
				Name:      "my-config",
			},
		},
		Summary: differ.DiffSummary{
			TotalResources: 4,
			AddedCount:     1,
			RemovedCount:   1,
			ModifiedCount:  1,
			IdenticalCount: 1,
		},
	}
}

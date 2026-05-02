package reporter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/nhuray/kyt/pkg/differ"
	"github.com/nhuray/kyt/pkg/manifest"
)

func TestReporter_Report(t *testing.T) {
	result := createTestDiffResult()

	tests := []struct {
		name         string
		showSummary  bool
		colorize     bool
		expectedStrs []string
	}{
		{
			name:        "diff mode without colors",
			showSummary: false,
			colorize:    false,
			expectedStrs: []string{
				"---", // Unified diff markers
				"+++",
			},
		},
		{
			name:        "summary mode",
			showSummary: true,
			colorize:    false,
			expectedStrs: []string{
				"KIND",       // Table header
				"SIMILARITY", // Table column
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reporter := NewReporter(tt.showSummary, tt.colorize)
			buf := &bytes.Buffer{}

			err := reporter.Report(result, buf)
			if err != nil {
				t.Fatalf("Report failed: %v", err)
			}

			output := buf.String()

			for _, expected := range tt.expectedStrs {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, got:\n%s", expected, output)
				}
			}
		})
	}
}

func TestReporter_NoDifferences(t *testing.T) {
	result := &differ.DiffResult{
		Changes: []differ.ResourceDiff{}, // No changes
		Summary: differ.DiffSummary{
			Added:     0,
			Removed:   0,
			Modified:  0,
			Identical: 1,
		},
	}

	reporter := NewReporter(false, false)
	buf := &bytes.Buffer{}

	err := reporter.Report(result, buf)
	if err != nil {
		t.Fatalf("Report failed: %v", err)
	}

	// With no differences, output should be empty (no diffs to show)
	output := buf.String()
	if output != "" {
		t.Logf("Output for no differences: %q", output)
	}
}

func TestNewReporter(t *testing.T) {
	tests := []struct {
		name        string
		showSummary bool
		colorize    bool
	}{
		{"diff mode no color", false, false},
		{"diff mode with color", false, true},
		{"summary mode no color", true, false},
		{"summary mode with color", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reporter := NewReporter(tt.showSummary, tt.colorize)
			if reporter == nil {
				t.Error("Expected non-nil reporter")
			}
			if reporter.showSummary != tt.showSummary {
				t.Errorf("Expected showSummary=%v, got %v", tt.showSummary, reporter.showSummary)
			}
			if reporter.colorize != tt.colorize {
				t.Errorf("Expected colorize=%v, got %v", tt.colorize, reporter.colorize)
			}
		})
	}
}

// createTestDiffResult creates a test DiffResult with sample data
func createTestDiffResult() *differ.DiffResult {
	sourceKey := &manifest.ResourceKey{
		Group:     "",
		Version:   "v1",
		Kind:      "ConfigMap",
		Namespace: "default",
		Name:      "test-cm",
	}

	targetKey := &manifest.ResourceKey{
		Group:     "",
		Version:   "v1",
		Kind:      "ConfigMap",
		Namespace: "default",
		Name:      "test-cm",
	}

	return &differ.DiffResult{
		Changes: []differ.ResourceDiff{
			{
				SourceKey:       sourceKey,
				TargetKey:       targetKey,
				ChangeType:      differ.ChangeTypeModified,
				MatchType:       "exact",
				SimilarityScore: 1.0,
				DiffText: `--- a/ConfigMap/default/test-cm
+++ b/ConfigMap/default/test-cm
@@ -1,3 +1,3 @@
 apiVersion: v1
 kind: ConfigMap
-data: old
+data: new
`,
				Insertions: 1,
				Deletions:  1,
			},
		},
		Summary: differ.DiffSummary{
			Added:     0,
			Removed:   0,
			Modified:  1,
			Identical: 0,
		},
	}
}

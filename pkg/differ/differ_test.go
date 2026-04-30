package differ

import (
	"testing"

	"github.com/nhuray/k8s-diff/pkg/config"
	"github.com/nhuray/k8s-diff/pkg/manifest"
	"github.com/nhuray/k8s-diff/pkg/normalizer"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestDiff_AddedResources(t *testing.T) {
	cfg := config.NewDefaultConfig()
	norm := normalizer.New(cfg)
	differ := New(norm, NewDefaultDiffOptions())

	source := manifest.NewManifestSet()
	target := manifest.NewManifestSet()

	// Add a resource only to target
	targetRes := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name":      "new-service",
				"namespace": "default",
			},
		},
	}
	_ = target.Add(targetRes)

	result, err := differ.Diff(source, target)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	if len(result.Added) != 1 {
		t.Errorf("Expected 1 added resource, got %d", len(result.Added))
	}

	if len(result.Removed) != 0 {
		t.Errorf("Expected 0 removed resources, got %d", len(result.Removed))
	}

	if len(result.Modified) != 0 {
		t.Errorf("Expected 0 modified resources, got %d", len(result.Modified))
	}

	if result.Summary.AddedCount != 1 {
		t.Errorf("Expected AddedCount=1, got %d", result.Summary.AddedCount)
	}
}

func TestDiff_RemovedResources(t *testing.T) {
	cfg := config.NewDefaultConfig()
	norm := normalizer.New(cfg)
	differ := New(norm, NewDefaultDiffOptions())

	source := manifest.NewManifestSet()
	target := manifest.NewManifestSet()

	// Add a resource only to source
	sourceRes := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name":      "old-service",
				"namespace": "default",
			},
		},
	}
	_ = source.Add(sourceRes)

	result, err := differ.Diff(source, target)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	if len(result.Added) != 0 {
		t.Errorf("Expected 0 added resources, got %d", len(result.Added))
	}

	if len(result.Removed) != 1 {
		t.Errorf("Expected 1 removed resource, got %d", len(result.Removed))
	}

	if len(result.Modified) != 0 {
		t.Errorf("Expected 0 modified resources, got %d", len(result.Modified))
	}

	if result.Summary.RemovedCount != 1 {
		t.Errorf("Expected RemovedCount=1, got %d", result.Summary.RemovedCount)
	}
}

func TestDiff_ModifiedResources(t *testing.T) {
	cfg := config.NewDefaultConfig()
	norm := normalizer.New(cfg)

	// Disable difftastic for predictable test output
	opts := NewDefaultDiffOptions()
	opts.UseDifftastic = false

	differ := New(norm, opts)

	source := manifest.NewManifestSet()
	target := manifest.NewManifestSet()

	// Add same resource with different specs to both
	sourceRes := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name":      "my-service",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"ports": []interface{}{
					map[string]interface{}{
						"port": int64(80),
					},
				},
			},
		},
	}

	targetRes := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name":      "my-service",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"ports": []interface{}{
					map[string]interface{}{
						"port": int64(443),
					},
				},
			},
		},
	}

	_ = source.Add(sourceRes)
	_ = target.Add(targetRes)

	result, err := differ.Diff(source, target)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	if len(result.Added) != 0 {
		t.Errorf("Expected 0 added resources, got %d", len(result.Added))
	}

	if len(result.Removed) != 0 {
		t.Errorf("Expected 0 removed resources, got %d", len(result.Removed))
	}

	if len(result.Modified) != 1 {
		t.Errorf("Expected 1 modified resource, got %d", len(result.Modified))
	}

	if result.Summary.ModifiedCount != 1 {
		t.Errorf("Expected ModifiedCount=1, got %d", result.Summary.ModifiedCount)
	}

	// Check that diff text is not empty
	if len(result.Modified) > 0 && result.Modified[0].DiffText == "" {
		t.Error("Expected non-empty diff text")
	}
}

func TestDiff_IdenticalResources(t *testing.T) {
	cfg := config.NewDefaultConfig()
	norm := normalizer.New(cfg)
	differ := New(norm, NewDefaultDiffOptions())

	source := manifest.NewManifestSet()
	target := manifest.NewManifestSet()

	// Add identical resources to both
	res := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name":      "my-service",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"ports": []interface{}{
					map[string]interface{}{
						"port": int64(80),
					},
				},
			},
		},
	}

	_ = source.Add(res.DeepCopy())
	_ = target.Add(res.DeepCopy())

	result, err := differ.Diff(source, target)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	if len(result.Added) != 0 {
		t.Errorf("Expected 0 added resources, got %d", len(result.Added))
	}

	if len(result.Removed) != 0 {
		t.Errorf("Expected 0 removed resources, got %d", len(result.Removed))
	}

	if len(result.Modified) != 0 {
		t.Errorf("Expected 0 modified resources, got %d", len(result.Modified))
	}

	if len(result.Identical) != 1 {
		t.Errorf("Expected 1 identical resource, got %d", len(result.Identical))
	}

	if result.Summary.IdenticalCount != 1 {
		t.Errorf("Expected IdenticalCount=1, got %d", result.Summary.IdenticalCount)
	}
}

func TestDiff_MixedChanges(t *testing.T) {
	cfg := config.NewDefaultConfig()
	norm := normalizer.New(cfg)

	opts := NewDefaultDiffOptions()
	opts.UseDifftastic = false
	opts.EnableSimilarityMatching = false // Disable similarity matching for this test

	differ := New(norm, opts)

	source := manifest.NewManifestSet()
	target := manifest.NewManifestSet()

	// Add resource only in source (removed)
	removedRes := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name":      "removed-service",
				"namespace": "default",
			},
		},
	}
	_ = source.Add(removedRes)

	// Add resource only in target (added)
	addedRes := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name":      "added-service",
				"namespace": "default",
			},
		},
	}
	_ = target.Add(addedRes)

	// Add identical resource to both
	identicalRes := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "my-config",
				"namespace": "default",
			},
		},
	}
	_ = source.Add(identicalRes.DeepCopy())
	_ = target.Add(identicalRes.DeepCopy())

	// Add modified resource to both
	modifiedSource := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      "my-app",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"replicas": int64(2),
			},
		},
	}

	modifiedTarget := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      "my-app",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"replicas": int64(3),
			},
		},
	}

	_ = source.Add(modifiedSource)
	_ = target.Add(modifiedTarget)

	result, err := differ.Diff(source, target)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	if len(result.Added) != 1 {
		t.Errorf("Expected 1 added resource, got %d", len(result.Added))
	}

	if len(result.Removed) != 1 {
		t.Errorf("Expected 1 removed resource, got %d", len(result.Removed))
	}

	if len(result.Modified) != 1 {
		t.Errorf("Expected 1 modified resource, got %d", len(result.Modified))
	}

	if len(result.Identical) != 1 {
		t.Errorf("Expected 1 identical resource, got %d", len(result.Identical))
	}

	// Check summary
	if result.Summary.TotalResources != 4 {
		t.Errorf("Expected TotalResources=4, got %d", result.Summary.TotalResources)
	}

	if result.Summary.AddedCount != 1 {
		t.Errorf("Expected AddedCount=1, got %d", result.Summary.AddedCount)
	}

	if result.Summary.RemovedCount != 1 {
		t.Errorf("Expected RemovedCount=1, got %d", result.Summary.RemovedCount)
	}

	if result.Summary.ModifiedCount != 1 {
		t.Errorf("Expected ModifiedCount=1, got %d", result.Summary.ModifiedCount)
	}

	if result.Summary.IdenticalCount != 1 {
		t.Errorf("Expected IdenticalCount=1, got %d", result.Summary.IdenticalCount)
	}
}

func TestDiffResult_HasDifferences(t *testing.T) {
	tests := []struct {
		name     string
		result   *DiffResult
		expected bool
	}{
		{
			name: "no differences",
			result: &DiffResult{
				Added:     []manifest.ResourceKey{},
				Removed:   []manifest.ResourceKey{},
				Modified:  []ResourceDiff{},
				Identical: []manifest.ResourceKey{{Kind: "Service", Name: "test"}},
			},
			expected: false,
		},
		{
			name: "has added",
			result: &DiffResult{
				Added:    []manifest.ResourceKey{{Kind: "Service", Name: "new"}},
				Removed:  []manifest.ResourceKey{},
				Modified: []ResourceDiff{},
			},
			expected: true,
		},
		{
			name: "has removed",
			result: &DiffResult{
				Added:    []manifest.ResourceKey{},
				Removed:  []manifest.ResourceKey{{Kind: "Service", Name: "old"}},
				Modified: []ResourceDiff{},
			},
			expected: true,
		},
		{
			name: "has modified",
			result: &DiffResult{
				Added:   []manifest.ResourceKey{},
				Removed: []manifest.ResourceKey{},
				Modified: []ResourceDiff{
					{Key: manifest.ResourceKey{Kind: "Service", Name: "changed"}},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.result.HasDifferences()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestAreResourcesEqual(t *testing.T) {
	res1 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name": "test",
			},
		},
	}

	res2 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name": "test",
			},
		},
	}

	res3 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name": "different",
			},
		},
	}

	equal, err := areResourcesEqual(res1, res2)
	if err != nil {
		t.Fatalf("areResourcesEqual failed: %v", err)
	}
	if !equal {
		t.Error("Expected res1 and res2 to be equal")
	}

	equal, err = areResourcesEqual(res1, res3)
	if err != nil {
		t.Fatalf("areResourcesEqual failed: %v", err)
	}
	if equal {
		t.Error("Expected res1 and res3 to be different")
	}
}

func TestDiff_SimilarityMatching(t *testing.T) {
	cfg := config.NewDefaultConfig()
	norm := normalizer.New(cfg)

	opts := NewDefaultDiffOptions()
	opts.UseDifftastic = false
	opts.EnableSimilarityMatching = true
	opts.SimilarityThreshold = 0.6 // Threshold appropriate for normalized resources

	differ := New(norm, opts)

	source := manifest.NewManifestSet()
	target := manifest.NewManifestSet()

	// Add Deployment with name v1 in source
	deployV1 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      "my-app-v1",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"replicas": int64(3),
				"template": map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "nginx",
								"image": "nginx:1.19",
							},
						},
					},
				},
			},
		},
	}
	_ = source.Add(deployV1)

	// Add Deployment with name v2 in target (similar structure, different name and image)
	deployV2 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      "my-app-v2",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"replicas": int64(3),
				"template": map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "nginx",
								"image": "nginx:1.20", // Updated image
							},
						},
					},
				},
			},
		},
	}
	_ = target.Add(deployV2)

	result, err := differ.Diff(source, target)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	// Debug: Print what we got
	t.Logf("Added: %d, Removed: %d, Modified: %d", len(result.Added), len(result.Removed), len(result.Modified))
	if len(result.Modified) > 0 {
		t.Logf("Modified[0]: MatchType=%s, Score=%.2f, SourceKey=%s, TargetKey=%s",
			result.Modified[0].MatchType, result.Modified[0].SimilarityScore,
			result.Modified[0].SourceKey.Name, result.Modified[0].TargetKey.Name)
	}

	// With similarity matching enabled, these should be matched as modified
	if len(result.Added) != 0 {
		t.Errorf("Expected 0 added resources with similarity matching, got %d", len(result.Added))
	}

	if len(result.Removed) != 0 {
		t.Errorf("Expected 0 removed resources with similarity matching, got %d", len(result.Removed))
	}

	if len(result.Modified) != 1 {
		t.Errorf("Expected 1 modified resource with similarity matching, got %d", len(result.Modified))
	}

	// Check match metadata
	if len(result.Modified) > 0 {
		mod := result.Modified[0]
		if mod.MatchType != "similarity" {
			t.Errorf("Expected MatchType='similarity', got '%s'", mod.MatchType)
		}
		if mod.SimilarityScore < 0.5 || mod.SimilarityScore > 0.7 {
			t.Errorf("Expected SimilarityScore between 0.5 and 0.7 (normalized resources), got %.2f", mod.SimilarityScore)
		}
		if mod.SourceKey.Name != "my-app-v1" {
			t.Errorf("Expected SourceKey name 'my-app-v1', got '%s'", mod.SourceKey.Name)
		}
		if mod.TargetKey.Name != "my-app-v2" {
			t.Errorf("Expected TargetKey name 'my-app-v2', got '%s'", mod.TargetKey.Name)
		}
	}
}

func TestDiff_TreeSitterFallback(t *testing.T) {
	cfg := config.NewDefaultConfig()
	norm := normalizer.New(cfg)

	// Configure differ to skip difftastic and use tree-sitter
	opts := NewDefaultDiffOptions()
	opts.UseDifftastic = false
	opts.UseTreeSitter = true
	opts.ColorOutput = false // Disable colors for easier testing

	differ := New(norm, opts)

	source := manifest.NewManifestSet()
	target := manifest.NewManifestSet()

	// Add ConfigMap to source
	sourceRes := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "test-config",
				"namespace": "default",
			},
			"data": map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
		},
	}
	_ = source.Add(sourceRes)

	// Add modified version to target
	targetRes := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "test-config",
				"namespace": "default",
			},
			"data": map[string]interface{}{
				"key1": "value1-modified",
				"key2": "value2",
				"key3": "value3", // Added key
			},
		},
	}
	_ = target.Add(targetRes)

	result, err := differ.Diff(source, target)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	if len(result.Modified) != 1 {
		t.Fatalf("Expected 1 modified resource, got %d", len(result.Modified))
	}

	// Verify diff text is generated
	diffText := result.Modified[0].DiffText
	if diffText == "" {
		t.Error("Expected non-empty diff text")
	}

	// Verify diff contains expected changes
	if !contains(diffText, "key1") {
		t.Error("Diff should contain modified key 'key1'")
	}
	if !contains(diffText, "key3") {
		t.Error("Diff should contain added key 'key3'")
	}

	// Verify side-by-side separator is present
	if !contains(diffText, "│") {
		t.Error("Tree-sitter diff should contain side-by-side separator '│'")
	}
}

func TestDiff_TreeSitterWithDeployment(t *testing.T) {
	cfg := config.NewDefaultConfig()
	norm := normalizer.New(cfg)

	opts := NewDefaultDiffOptions()
	opts.UseDifftastic = false
	opts.UseTreeSitter = true
	opts.ColorOutput = false

	differ := New(norm, opts)

	source := manifest.NewManifestSet()
	target := manifest.NewManifestSet()

	// Add Deployment to source
	sourceRes := &unstructured.Unstructured{
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
	_ = source.Add(sourceRes)

	// Add modified version with different replicas
	targetRes := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      "test-deployment",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"replicas": int64(5), // Changed from 3 to 5
				"selector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"app": "test",
					},
				},
			},
		},
	}
	_ = target.Add(targetRes)

	result, err := differ.Diff(source, target)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	if len(result.Modified) != 1 {
		t.Fatalf("Expected 1 modified resource, got %d", len(result.Modified))
	}

	// Verify diff text contains replicas change
	diffText := result.Modified[0].DiffText
	if !contains(diffText, "replicas") {
		t.Error("Diff should show replicas change")
	}
}

func TestDiff_FallbackChain(t *testing.T) {
	cfg := config.NewDefaultConfig()
	norm := normalizer.New(cfg)

	// Test with default options (auto mode)
	opts := NewDefaultDiffOptions()
	opts.ColorOutput = false

	differ := New(norm, opts)

	source := manifest.NewManifestSet()
	target := manifest.NewManifestSet()

	// Add simple ConfigMap
	sourceRes := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "test",
				"namespace": "default",
			},
			"data": map[string]interface{}{
				"key": "value",
			},
		},
	}
	_ = source.Add(sourceRes)

	targetRes := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "test",
				"namespace": "default",
			},
			"data": map[string]interface{}{
				"key": "new-value",
			},
		},
	}
	_ = target.Add(targetRes)

	// Should not fail regardless of which diff tool is available
	result, err := differ.Diff(source, target)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	if len(result.Modified) != 1 {
		t.Fatalf("Expected 1 modified resource, got %d", len(result.Modified))
	}

	// Some diff should be generated
	if result.Modified[0].DiffText == "" {
		t.Error("Expected non-empty diff text")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) &&
			findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestDiff_SimilarityMatching_Disabled(t *testing.T) {
	cfg := config.NewDefaultConfig()
	norm := normalizer.New(cfg)

	opts := NewDefaultDiffOptions()
	opts.UseDifftastic = false
	opts.EnableSimilarityMatching = false // Disabled

	differ := New(norm, opts)

	source := manifest.NewManifestSet()
	target := manifest.NewManifestSet()

	// Add ConfigMap with name v1 in source
	configV1 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "app-config-v1",
				"namespace": "default",
			},
			"data": map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
		},
	}
	_ = source.Add(configV1)

	// Add ConfigMap with name v2 in target (similar structure, different name)
	configV2 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "app-config-v2",
				"namespace": "default",
			},
			"data": map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
		},
	}
	_ = target.Add(configV2)

	result, err := differ.Diff(source, target)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	// With similarity matching disabled, these should be reported as added/removed
	if len(result.Added) != 1 {
		t.Errorf("Expected 1 added resource without similarity matching, got %d", len(result.Added))
	}

	if len(result.Removed) != 1 {
		t.Errorf("Expected 1 removed resource without similarity matching, got %d", len(result.Removed))
	}

	if len(result.Modified) != 0 {
		t.Errorf("Expected 0 modified resources without similarity matching, got %d", len(result.Modified))
	}
}

func TestDiff_SimilarityMatching_BelowThreshold(t *testing.T) {
	cfg := config.NewDefaultConfig()
	norm := normalizer.New(cfg)

	opts := NewDefaultDiffOptions()
	opts.UseDifftastic = false
	opts.EnableSimilarityMatching = true
	opts.SimilarityThreshold = 0.9 // High threshold

	differ := New(norm, opts)

	source := manifest.NewManifestSet()
	target := manifest.NewManifestSet()

	// Add ConfigMap with some data in source
	config1 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "config1",
				"namespace": "default",
			},
			"data": map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
		},
	}
	_ = source.Add(config1)

	// Add ConfigMap with mostly different data in target
	config2 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "config2",
				"namespace": "default",
			},
			"data": map[string]interface{}{
				"key3": "value3",
				"key4": "value4",
				"key5": "value5",
			},
		},
	}
	_ = target.Add(config2)

	result, err := differ.Diff(source, target)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	// Resources are too different to match (below threshold)
	// Should be reported as added/removed
	if len(result.Added) != 1 {
		t.Errorf("Expected 1 added resource (below threshold), got %d", len(result.Added))
	}

	if len(result.Removed) != 1 {
		t.Errorf("Expected 1 removed resource (below threshold), got %d", len(result.Removed))
	}
}

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
	target.Add(targetRes)

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
	source.Add(sourceRes)

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

	source.Add(sourceRes)
	target.Add(targetRes)

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

	source.Add(res.DeepCopy())
	target.Add(res.DeepCopy())

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
	source.Add(removedRes)

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
	target.Add(addedRes)

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
	source.Add(identicalRes.DeepCopy())
	target.Add(identicalRes.DeepCopy())

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

	source.Add(modifiedSource)
	target.Add(modifiedTarget)

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

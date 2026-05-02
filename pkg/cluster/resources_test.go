package cluster

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestCommonResourceTypes(t *testing.T) {
	resourceTypes := CommonResourceTypes()

	// Should return a non-empty list
	if len(resourceTypes) == 0 {
		t.Fatal("CommonResourceTypes() returned empty list")
	}

	// Check for some expected resource types
	expectedResources := map[string]schema.GroupVersionResource{
		"pods": {
			Group:    "",
			Version:  "v1",
			Resource: "pods",
		},
		"services": {
			Group:    "",
			Version:  "v1",
			Resource: "services",
		},
		"deployments": {
			Group:    "apps",
			Version:  "v1",
			Resource: "deployments",
		},
		"configmaps": {
			Group:    "",
			Version:  "v1",
			Resource: "configmaps",
		},
		"secrets": {
			Group:    "",
			Version:  "v1",
			Resource: "secrets",
		},
	}

	// Create a map for easier lookup
	resourceMap := make(map[string]schema.GroupVersionResource)
	for _, gvr := range resourceTypes {
		resourceMap[gvr.Resource] = gvr
	}

	// Verify expected resources are present
	for resourceName, expectedGVR := range expectedResources {
		actualGVR, found := resourceMap[resourceName]
		if !found {
			t.Errorf("Expected resource %q not found in CommonResourceTypes()", resourceName)
			continue
		}

		if actualGVR.Group != expectedGVR.Group {
			t.Errorf("Resource %q: expected group %q, got %q", resourceName, expectedGVR.Group, actualGVR.Group)
		}
		if actualGVR.Version != expectedGVR.Version {
			t.Errorf("Resource %q: expected version %q, got %q", resourceName, expectedGVR.Version, actualGVR.Version)
		}
		if actualGVR.Resource != expectedGVR.Resource {
			t.Errorf("Resource %q: expected resource %q, got %q", resourceName, expectedGVR.Resource, actualGVR.Resource)
		}
	}

	// Should include approximately 15 resource types (as documented)
	if len(resourceTypes) < 10 || len(resourceTypes) > 20 {
		t.Errorf("Expected approximately 15 resource types, got %d", len(resourceTypes))
	}
}

func TestCommonResourceTypesConsistency(t *testing.T) {
	// Call multiple times to ensure consistency
	types1 := CommonResourceTypes()
	types2 := CommonResourceTypes()

	if len(types1) != len(types2) {
		t.Errorf("CommonResourceTypes() returned different lengths: %d vs %d", len(types1), len(types2))
	}

	// Verify same resources are returned
	for i := range types1 {
		if types1[i] != types2[i] {
			t.Errorf("CommonResourceTypes()[%d] differs: %v vs %v", i, types1[i], types2[i])
		}
	}
}

package treesitter

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ValidateKubernetesResource checks if the object is a valid Kubernetes resource
// Required fields: apiVersion, kind, metadata.name
func ValidateKubernetesResource(obj *unstructured.Unstructured) error {
	if obj == nil {
		return fmt.Errorf("resource is nil")
	}

	// Check apiVersion
	if obj.GetAPIVersion() == "" {
		return fmt.Errorf("missing required field: apiVersion")
	}

	// Check kind
	if obj.GetKind() == "" {
		return fmt.Errorf("missing required field: kind")
	}

	// Check metadata.name
	if obj.GetName() == "" {
		return fmt.Errorf("missing required field: metadata.name")
	}

	return nil
}

// ValidateManifestPair validates both source and target resources
func ValidateManifestPair(source, target *unstructured.Unstructured) error {
	if err := ValidateKubernetesResource(source); err != nil {
		return fmt.Errorf("source validation failed: %w", err)
	}

	if err := ValidateKubernetesResource(target); err != nil {
		return fmt.Errorf("target validation failed: %w", err)
	}

	return nil
}

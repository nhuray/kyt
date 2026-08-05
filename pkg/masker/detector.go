package masker

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

// IsSecret checks if an unstructured object is a Kubernetes Secret
func IsSecret(obj *unstructured.Unstructured) bool {
	if obj == nil {
		return false
	}
	return obj.GetKind() == "Secret"
}

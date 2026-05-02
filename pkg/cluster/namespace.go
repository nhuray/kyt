package cluster

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NamespaceNotFoundError represents an error when a namespace doesn't exist
type NamespaceNotFoundError struct {
	Namespace string
	Context   string
}

func (e *NamespaceNotFoundError) Error() string {
	return fmt.Sprintf("namespace %q does not exist in cluster (context: %s)", e.Namespace, e.Context)
}

// ValidateNamespace checks if the specified namespace exists in the cluster
// Returns NamespaceNotFoundError if the namespace doesn't exist
func (c *ClusterClient) ValidateNamespace(namespace string) error {
	if namespace == "" {
		// Empty namespace means cluster-scoped or all namespaces, which is always valid
		return nil
	}

	ctx := context.Background()

	// Use the core/v1 namespaces resource to check if namespace exists
	namespaceGVR := schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "namespaces",
	}

	_, err := c.client.Resource(namespaceGVR).Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return &NamespaceNotFoundError{
			Namespace: namespace,
			Context:   c.context,
		}
	}

	return nil
}

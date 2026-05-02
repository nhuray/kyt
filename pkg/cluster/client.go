package cluster

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

// ClusterClient wraps the Kubernetes dynamic client for resource operations
type ClusterClient struct {
	client  dynamic.Interface
	context string
}

// NewClusterClient creates a new client for accessing Kubernetes resources
// If kubeconfigPath is empty, it uses the default kubeconfig location
// If contextName is empty, it uses the current context from kubeconfig
func NewClusterClient(kubeconfigPath, contextName string) (*ClusterClient, error) {
	// Determine kubeconfig path
	if kubeconfigPath == "" {
		// Use default location
		if env := os.Getenv("KUBECONFIG"); env != "" {
			kubeconfigPath = env
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get home directory: %w", err)
			}
			kubeconfigPath = filepath.Join(home, ".kube", "config")
		}
	}

	// Check if kubeconfig file exists
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("kubeconfig not found at %s", kubeconfigPath)
	}

	// Load kubeconfig
	loadingRules := &clientcmd.ClientConfigLoadingRules{
		ExplicitPath: kubeconfigPath,
	}

	configOverrides := &clientcmd.ConfigOverrides{}
	if contextName != "" {
		configOverrides.CurrentContext = contextName
	}

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		configOverrides,
	)

	// Get the current context name for error messages
	rawConfig, err := kubeConfig.RawConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	effectiveContext := rawConfig.CurrentContext
	if contextName != "" {
		effectiveContext = contextName
		// Validate that the context exists
		if _, exists := rawConfig.Contexts[contextName]; !exists {
			return nil, fmt.Errorf("context %q not found in kubeconfig", contextName)
		}
	}

	// Build REST config
	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to build REST config: %w", err)
	}

	// Create dynamic client
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	return &ClusterClient{
		client:  client,
		context: effectiveContext,
	}, nil
}

// GetResourcesInNamespace fetches all resources of the specified types from a namespace
// Returns a slice of unstructured resources
func (c *ClusterClient) GetResourcesInNamespace(
	namespace string,
	resourceTypes []schema.GroupVersionResource,
) ([]*unstructured.Unstructured, error) {
	ctx := context.Background()
	var allResources []*unstructured.Unstructured

	for _, gvr := range resourceTypes {
		// Get the resource interface for this type
		var resourceInterface dynamic.ResourceInterface
		if namespace == "" {
			// Cluster-scoped resources
			resourceInterface = c.client.Resource(gvr)
		} else {
			// Namespace-scoped resources
			resourceInterface = c.client.Resource(gvr).Namespace(namespace)
		}

		// List the resources
		list, err := resourceInterface.List(ctx, metav1.ListOptions{})
		if err != nil {
			// Skip resources we can't access (e.g., CRDs not installed, permission denied)
			// We'll handle this gracefully by continuing to the next resource type
			continue
		}

		// Add each resource to our collection
		for i := range list.Items {
			allResources = append(allResources, &list.Items[i])
		}
	}

	return allResources, nil
}

// GetContext returns the context name this client is using
func (c *ClusterClient) GetContext() string {
	return c.context
}

package cluster

import (
	"context"
	"fmt"
	"io"
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
	verbose io.Writer // Writer for verbose output (typically os.Stderr, or nil to disable)
}

// NewClusterClient creates a new client for accessing Kubernetes resources
// If kubeconfigPath is empty, it uses the default kubeconfig location
// If contextName is empty, it uses the current context from kubeconfig
func NewClusterClient(kubeconfigPath, contextName string) (*ClusterClient, error) {
	return NewClusterClientWithVerbose(kubeconfigPath, contextName, nil)
}

// NewClusterClientWithVerbose creates a new client with verbose logging
// verboseWriter can be os.Stderr for verbose output, or nil to disable
func NewClusterClientWithVerbose(kubeconfigPath, contextName string, verboseWriter io.Writer) (*ClusterClient, error) {
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
		return nil, &KubeconfigNotFoundError{Path: kubeconfigPath}
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
			return nil, &ContextNotFoundError{
				Context:        contextName,
				KubeconfigPath: kubeconfigPath,
			}
		}
	}

	if effectiveContext == "" {
		return nil, &NoContextError{KubeconfigPath: kubeconfigPath}
	}

	// Build REST config
	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, &ConnectionError{
			Context: effectiveContext,
			Err:     err,
		}
	}

	// Create dynamic client
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, &ConnectionError{
			Context: effectiveContext,
			Err:     err,
		}
	}

	if verboseWriter != nil {
		fmt.Fprintf(verboseWriter, "  Connected to cluster (context: %s)\n", effectiveContext)
	}

	return &ClusterClient{
		client:  client,
		context: effectiveContext,
		verbose: verboseWriter,
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
	var skippedCount int
	var fetchedCount int

	if c.verbose != nil {
		fmt.Fprintf(c.verbose, "  Fetching resources from namespace %q...\n", namespace)
	}

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
			skippedCount++
			if c.verbose != nil {
				fmt.Fprintf(c.verbose, "  Skipped %s/%s: %v\n", gvr.Group, gvr.Resource, err)
			}
			continue
		}

		// Add each resource to our collection
		resourceCount := len(list.Items)
		if resourceCount > 0 {
			fetchedCount++
			if c.verbose != nil {
				fmt.Fprintf(c.verbose, "  Found %d %s\n", resourceCount, gvr.Resource)
			}
		}

		for i := range list.Items {
			allResources = append(allResources, &list.Items[i])
		}
	}

	if c.verbose != nil {
		fmt.Fprintf(c.verbose, "  Total: %d resources from %d resource types (%d types skipped)\n", len(allResources), fetchedCount, skippedCount)
	}

	return allResources, nil
}

// GetContext returns the context name this client is using
func (c *ClusterClient) GetContext() string {
	return c.context
}

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/nhuray/kyt/pkg/cluster"
	"github.com/nhuray/kyt/pkg/manifest"
	"k8s.io/client-go/tools/clientcmd"
)

// inputType represents the type of input source
type inputType string

const (
	inputTypeFile      inputType = "file"
	inputTypeNamespace inputType = "namespace"
)

// inputSource represents a parsed input source
type inputSource struct {
	Type  inputType
	Value string // file path or namespace name
}

// parseInput parses an input argument and determines if it's a file/directory or namespace reference
func parseInput(arg string) inputSource {
	// Check for namespace prefix
	if strings.HasPrefix(arg, "ns:") {
		namespace := strings.TrimPrefix(arg, "ns:")
		return inputSource{
			Type:  inputTypeNamespace,
			Value: namespace,
		}
	}

	// Default to file/directory
	return inputSource{
		Type:  inputTypeFile,
		Value: arg,
	}
}

// loadManifests loads manifests from either a file/directory or a Kubernetes namespace
// contextName should already be resolved (either from --context flag or current context)
// verboseWriter is used for verbose logging (typically os.Stderr, or nil to disable)
func loadManifests(input inputSource, contextName string, verboseWriter io.Writer) (*manifest.ManifestSet, error) {
	switch input.Type {
	case inputTypeFile:
		return loadManifestsFromFile(input.Value)
	case inputTypeNamespace:
		return loadManifestsFromNamespace(input.Value, contextName, verboseWriter)
	default:
		return nil, fmt.Errorf("unknown input type: %s", input.Type)
	}
}

// getCurrentContext retrieves the current context from kubeconfig
func getCurrentContext() (string, error) {
	// Determine kubeconfig path
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		kubeconfigPath = filepath.Join(home, ".kube", "config")
	}

	// Check if kubeconfig file exists
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		return "", fmt.Errorf("kubeconfig not found at %s", kubeconfigPath)
	}

	// Load kubeconfig
	loadingRules := &clientcmd.ClientConfigLoadingRules{
		ExplicitPath: kubeconfigPath,
	}

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		&clientcmd.ConfigOverrides{},
	)

	rawConfig, err := kubeConfig.RawConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	if rawConfig.CurrentContext == "" {
		return "", fmt.Errorf("no current context set in kubeconfig")
	}

	return rawConfig.CurrentContext, nil
}

// loadManifestsFromFile loads manifests from a file or directory
func loadManifestsFromFile(path string) (*manifest.ManifestSet, error) {
	parser := manifest.NewParser()

	// Check if path is a file or directory
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	if info.IsDir() {
		return parser.ParseDirectory(path)
	}

	return parser.ParseFile(path)
}

// loadManifestsFromNamespace loads manifests from a Kubernetes namespace
func loadManifestsFromNamespace(namespace, contextName string, verboseWriter io.Writer) (*manifest.ManifestSet, error) {
	// Create cluster client (kubeconfigPath="", contextName=contextName)
	client, err := cluster.NewClusterClientWithVerbose("", contextName, verboseWriter)
	if err != nil {
		return nil, err // Error already has helpful message from cluster package
	}

	// Validate namespace exists
	if err := client.ValidateNamespace(namespace); err != nil {
		return nil, err // Error already has helpful message
	}

	// Get common resource types
	resourceTypes := cluster.CommonResourceTypes()

	// Fetch resources from namespace
	resources, err := client.GetResourcesInNamespace(namespace, resourceTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to get resources from namespace %s: %w", namespace, err)
	}

	// Convert to ManifestSet
	manifestSet := manifest.NewManifestSet()
	for _, resource := range resources {
		key := manifest.NewResourceKey(resource)
		manifestSet.Resources[key] = resource
		// Set source file to indicate cluster origin
		manifestSet.SourceFile[key] = fmt.Sprintf("%s/%s/%s", contextName, namespace, resource.GetKind())
	}

	return manifestSet, nil
}

// formatInputForDisplay returns a human-readable representation of the input
func formatInputForDisplay(input inputSource, contextName string) string {
	switch input.Type {
	case inputTypeFile:
		return input.Value
	case inputTypeNamespace:
		if contextName != "" {
			return fmt.Sprintf("ns:%s (context: %s)", input.Value, contextName)
		}
		return fmt.Sprintf("ns:%s", input.Value)
	default:
		return input.Value
	}
}

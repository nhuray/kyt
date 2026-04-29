package manifest

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ResourceKey uniquely identifies a Kubernetes resource
// Format: apiVersion.kind.namespace.name
type ResourceKey struct {
	Group     string // API group (empty for core resources like Pod, Service)
	Version   string // API version (e.g., "v1", "v1beta1")
	Kind      string // Resource kind (e.g., "Deployment", "Service")
	Namespace string // Namespace (empty for cluster-scoped resources)
	Name      string // Resource name
}

// NewResourceKey creates a ResourceKey from an unstructured Kubernetes object
func NewResourceKey(obj *unstructured.Unstructured) ResourceKey {
	gvk := obj.GroupVersionKind()
	return ResourceKey{
		Group:     gvk.Group,
		Version:   gvk.Version,
		Kind:      gvk.Kind,
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
}

// String returns a human-readable representation of the ResourceKey
// Format: kind.group/name (namespace: namespace)
func (k ResourceKey) String() string {
	group := k.Group
	if group == "" {
		group = "core"
	}

	if k.Namespace == "" {
		return fmt.Sprintf("%s.%s/%s", k.Kind, group, k.Name)
	}
	return fmt.Sprintf("%s.%s/%s (namespace: %s)", k.Kind, group, k.Name, k.Namespace)
}

// APIVersion returns the full apiVersion string (group/version or just version for core)
func (k ResourceKey) APIVersion() string {
	if k.Group == "" {
		return k.Version
	}
	return fmt.Sprintf("%s/%s", k.Group, k.Version)
}

// ManifestSet represents a collection of Kubernetes resources indexed by ResourceKey
type ManifestSet struct {
	Resources map[ResourceKey]*unstructured.Unstructured
}

// NewManifestSet creates an empty ManifestSet
func NewManifestSet() *ManifestSet {
	return &ManifestSet{
		Resources: make(map[ResourceKey]*unstructured.Unstructured),
	}
}

// Add adds a resource to the ManifestSet
// Returns an error if a resource with the same key already exists
func (m *ManifestSet) Add(obj *unstructured.Unstructured) error {
	key := NewResourceKey(obj)

	if _, exists := m.Resources[key]; exists {
		return fmt.Errorf("duplicate resource: %s", key.String())
	}

	m.Resources[key] = obj
	return nil
}

// Get retrieves a resource by its key
func (m *ManifestSet) Get(key ResourceKey) (*unstructured.Unstructured, bool) {
	obj, ok := m.Resources[key]
	return obj, ok
}

// Keys returns all ResourceKeys in the ManifestSet
func (m *ManifestSet) Keys() []ResourceKey {
	keys := make([]ResourceKey, 0, len(m.Resources))
	for k := range m.Resources {
		keys = append(keys, k)
	}
	return keys
}

// Len returns the number of resources in the ManifestSet
func (m *ManifestSet) Len() int {
	return len(m.Resources)
}

// Merge adds all resources from another ManifestSet
// Returns an error if any duplicate keys are found
func (m *ManifestSet) Merge(other *ManifestSet) error {
	for key, obj := range other.Resources {
		if _, exists := m.Resources[key]; exists {
			return fmt.Errorf("duplicate resource during merge: %s", key.String())
		}
		m.Resources[key] = obj
	}
	return nil
}

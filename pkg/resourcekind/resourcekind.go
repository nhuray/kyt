package resourcekind

import (
	"strings"
)

// ResourceKind represents a Kubernetes resource with its various name forms
type ResourceKind struct {
	Kind       string   // Canonical kind name (e.g., "Deployment")
	Singular   string   // Singular form (e.g., "deployment")
	Plural     string   // Plural form (e.g., "deployments")
	ShortNames []string // Short names (e.g., ["deploy"])
}

// Matcher provides functionality to match and resolve Kubernetes resource kinds
type Matcher struct {
	kinds map[string]ResourceKind // Map from all name forms (lowercase) to ResourceKind
}

// NewMatcher creates a new Matcher with common Kubernetes resource kinds
func NewMatcher() *Matcher {
	m := &Matcher{
		kinds: make(map[string]ResourceKind),
	}

	// Register all common Kubernetes resource kinds
	m.registerKinds()

	return m
}

// registerKinds registers all common Kubernetes resource types
func (m *Matcher) registerKinds() {
	// Core resources
	m.register("Pod", "pod", "pods", []string{"po"})
	m.register("Service", "service", "services", []string{"svc"})
	m.register("ConfigMap", "configmap", "configmaps", []string{"cm"})
	m.register("Secret", "secret", "secrets", []string{"sec"})
	m.register("Namespace", "namespace", "namespaces", []string{"ns"})
	m.register("Node", "node", "nodes", []string{"no"})
	m.register("PersistentVolume", "persistentvolume", "persistentvolumes", []string{"pv"})
	m.register("PersistentVolumeClaim", "persistentvolumeclaim", "persistentvolumeclaims", []string{"pvc"})
	m.register("ServiceAccount", "serviceaccount", "serviceaccounts", []string{"sa"})
	m.register("Endpoints", "endpoints", "endpoints", []string{"ep"})
	m.register("Event", "event", "events", []string{"ev"})
	m.register("LimitRange", "limitrange", "limitranges", []string{"limits"})
	m.register("ResourceQuota", "resourcequota", "resourcequotas", []string{"quota"})

	// Apps resources
	m.register("Deployment", "deployment", "deployments", []string{"deploy", "dep"})
	m.register("StatefulSet", "statefulset", "statefulsets", []string{"sts"})
	m.register("DaemonSet", "daemonset", "daemonsets", []string{"ds"})
	m.register("ReplicaSet", "replicaset", "replicasets", []string{"rs"})

	// Batch resources
	m.register("Job", "job", "jobs", []string{"jo"})
	m.register("CronJob", "cronjob", "cronjobs", []string{"cj"})

	// Networking resources
	m.register("Ingress", "ingress", "ingresses", []string{"ing"})
	m.register("IngressClass", "ingressclass", "ingressclasses", []string{})
	m.register("NetworkPolicy", "networkpolicy", "networkpolicies", []string{"netpol", "np"})

	// Storage resources
	m.register("StorageClass", "storageclass", "storageclasses", []string{"sc"})
	m.register("VolumeAttachment", "volumeattachment", "volumeattachments", []string{})

	// RBAC resources
	m.register("Role", "role", "roles", []string{"ro"})
	m.register("RoleBinding", "rolebinding", "rolebindings", []string{"rb"})
	m.register("ClusterRole", "clusterrole", "clusterroles", []string{"cr"})
	m.register("ClusterRoleBinding", "clusterrolebinding", "clusterrolebindings", []string{"crb"})

	// Policy resources
	m.register("PodDisruptionBudget", "poddisruptionbudget", "poddisruptionbudgets", []string{"pdb"})
	m.register("PodSecurityPolicy", "podsecuritypolicy", "podsecuritypolicies", []string{"psp"})

	// Autoscaling resources
	m.register("HorizontalPodAutoscaler", "horizontalpodautoscaler", "horizontalpodautoscalers", []string{"hpa"})

	// Custom Resource Definitions
	m.register("CustomResourceDefinition", "customresourcedefinition", "customresourcedefinitions", []string{"crd", "crds"})

	// API resources
	m.register("APIService", "apiservice", "apiservices", []string{})

	// Certificates
	m.register("CertificateSigningRequest", "certificatesigningrequest", "certificatesigningrequests", []string{"csr"})

	// Admission
	m.register("MutatingWebhookConfiguration", "mutatingwebhookconfiguration", "mutatingwebhookconfigurations", []string{})
	m.register("ValidatingWebhookConfiguration", "validatingwebhookconfiguration", "validatingwebhookconfigurations", []string{})

	// Priority
	m.register("PriorityClass", "priorityclass", "priorityclasses", []string{"pc"})

	// Runtime
	m.register("RuntimeClass", "runtimeclass", "runtimeclasses", []string{})
}

// register adds a resource kind with all its name forms
func (m *Matcher) register(kind, singular, plural string, shortNames []string) {
	rk := ResourceKind{
		Kind:       kind,
		Singular:   singular,
		Plural:     plural,
		ShortNames: shortNames,
	}

	// Map all forms to the same ResourceKind
	m.kinds[strings.ToLower(kind)] = rk
	m.kinds[strings.ToLower(singular)] = rk
	m.kinds[strings.ToLower(plural)] = rk

	for _, short := range shortNames {
		m.kinds[strings.ToLower(short)] = rk
	}
}

// Resolve resolves a resource name (short, singular, or plural) to its canonical Kind
// Returns the canonical Kind name and true if found, empty string and false otherwise
func (m *Matcher) Resolve(name string) (string, bool) {
	rk, ok := m.kinds[strings.ToLower(name)]
	if !ok {
		return "", false
	}
	return rk.Kind, true
}

// Match checks if a resource kind matches a given filter
// The filter can be a short name, singular, or plural form
func (m *Matcher) Match(kind, filter string) bool {
	// Resolve both to canonical forms
	canonicalKind, ok := m.Resolve(kind)
	if !ok {
		// If we can't resolve the kind, do a direct case-insensitive comparison
		canonicalKind = kind
	}

	canonicalFilter, ok := m.Resolve(filter)
	if !ok {
		// If we can't resolve the filter, do a direct case-insensitive comparison
		canonicalFilter = filter
	}

	return strings.EqualFold(canonicalKind, canonicalFilter)
}

// ParseList parses a comma-separated list of resource kinds
// Returns a slice of canonical Kind names
func (m *Matcher) ParseList(list string) []string {
	if list == "" {
		return nil
	}

	parts := strings.Split(list, ",")
	var result []string

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Try to resolve to canonical form
		if canonical, ok := m.Resolve(part); ok {
			result = append(result, canonical)
		} else {
			// If we can't resolve, keep the original (might be a custom resource)
			result = append(result, part)
		}
	}

	return result
}

// MatchesAny checks if a resource kind matches any of the filters
func (m *Matcher) MatchesAny(kind string, filters []string) bool {
	for _, filter := range filters {
		if m.Match(kind, filter) {
			return true
		}
	}
	return false
}

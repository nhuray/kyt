package config

// Config represents the complete configuration for kyt
type Config struct {
	IgnoreDifferences []ResourceIgnoreDifferences `yaml:"ignoreDifferences"`
	Normalization     NormalizationConfig         `yaml:"normalization"`
	Output            OutputConfig                `yaml:"output"`
}

// ResourceIgnoreDifferences defines ignore rules for specific resource types
// This matches ArgoCD's ignoreDifferences format for compatibility
type ResourceIgnoreDifferences struct {
	// Group is the API group (empty string for core resources like Pod, Service)
	Group string `yaml:"group"`

	// Kind is the resource kind (e.g., "Deployment", "Service")
	// Use "*" to match all kinds
	Kind string `yaml:"kind"`

	// Name is the resource name (optional, empty matches all)
	// Supports glob patterns (e.g., "redis-*")
	Name string `yaml:"name,omitempty"`

	// Namespace is the resource namespace (optional, empty matches all)
	// Supports glob patterns (e.g., "prod-*")
	Namespace string `yaml:"namespace,omitempty"`

	// JSONPointers is a list of JSON Pointer (RFC 6901) paths to ignore
	// Example: "/metadata/labels", "/spec/replicas"
	JSONPointers []string `yaml:"jsonPointers,omitempty"`

	// JQPathExpressions is a list of JQ path expressions to ignore
	// More powerful than JSON Pointers, allows complex filtering
	// Example: ".spec.template.spec.containers[] | select(.name == \"istio-proxy\")"
	JQPathExpressions []string `yaml:"jqPathExpressions,omitempty"`

	// ManagedFieldsManagers is a list of field managers to ignore
	// Used in server-side apply scenarios
	// Example: ["kube-controller-manager", "kubectl-client-side-apply"]
	ManagedFieldsManagers []string `yaml:"managedFieldsManagers,omitempty"`
}

// NormalizationConfig controls how resources are normalized before comparison
type NormalizationConfig struct {
	// SortKeys sorts object keys alphabetically for consistent diffs
	SortKeys bool `yaml:"sortKeys"`

	// SortArrays defines which arrays should be sorted before comparison
	// Useful for arrays where order doesn't matter (e.g., env vars, ports)
	SortArrays []ArraySortConfig `yaml:"sortArrays,omitempty"`

	// RemoveDefaultFields removes fields with default values
	// Example: removeDefaultFields: ["status", "metadata.managedFields"]
	RemoveDefaultFields []string `yaml:"removeDefaultFields,omitempty"`
}

// ArraySortConfig defines how to sort a specific array
type ArraySortConfig struct {
	// Path is a JQ-style path to the array
	// Example: ".spec.template.spec.containers[].ports"
	Path string `yaml:"path"`

	// SortBy is the field name to sort by
	// Example: "containerPort", "name"
	SortBy string `yaml:"sortBy"`
}

// OutputConfig controls the output format and styling
type OutputConfig struct {
	// Format is the output format: "cli", "json", "yaml", "diff"
	Format string `yaml:"format"`

	// Colorize enables colored output (only for cli/diff formats)
	Colorize bool `yaml:"colorize"`

	// ShowUnchanged shows resources that have no differences
	ShowUnchanged bool `yaml:"showUnchanged"`

	// ContextLines is the number of context lines for unified diff (default: 3)
	ContextLines int `yaml:"contextLines,omitempty"`

	// StringSimilarityThreshold is the minimum string length to trigger fuzzy matching
	// Strings longer than this will use Levenshtein distance for similarity
	// Default: 100 characters
	StringSimilarityThreshold int `yaml:"stringSimilarityThreshold,omitempty"`
}

// NewDefaultConfig returns a Config with sensible defaults
func NewDefaultConfig() *Config {
	return &Config{
		IgnoreDifferences: []ResourceIgnoreDifferences{},
		Normalization: NormalizationConfig{
			SortKeys: true,
			RemoveDefaultFields: []string{
				"/status",
				"/metadata/managedFields",
				"/metadata/creationTimestamp",
				"/metadata/generation",
				"/metadata/resourceVersion",
				"/metadata/uid",
			},
		},
		Output: OutputConfig{
			Format:                    "cli",
			Colorize:                  true,
			ShowUnchanged:             false,
			ContextLines:              3,
			StringSimilarityThreshold: 100, // Enable fuzzy matching for strings > 100 chars
		},
	}
}

// Merge merges another config into this one
// Rules from the other config are appended (not replaced)
func (c *Config) Merge(other *Config) {
	// Append ignore rules
	c.IgnoreDifferences = append(c.IgnoreDifferences, other.IgnoreDifferences...)

	// Merge normalization (other takes precedence for boolean fields)
	if other.Normalization.SortKeys {
		c.Normalization.SortKeys = true
	}
	c.Normalization.SortArrays = append(c.Normalization.SortArrays, other.Normalization.SortArrays...)
	c.Normalization.RemoveDefaultFields = append(c.Normalization.RemoveDefaultFields, other.Normalization.RemoveDefaultFields...)

	// Output config: other takes precedence
	if other.Output.Format != "" {
		c.Output.Format = other.Output.Format
	}
	if other.Output.Colorize {
		c.Output.Colorize = true
	}
	if other.Output.ShowUnchanged {
		c.Output.ShowUnchanged = true
	}
	if other.Output.ContextLines > 0 {
		c.Output.ContextLines = other.Output.ContextLines
	}
	if other.Output.StringSimilarityThreshold > 0 {
		c.Output.StringSimilarityThreshold = other.Output.StringSimilarityThreshold
	}
}

// MatchesResource checks if a ResourceIgnoreDifferences matches a given resource
// Supports glob patterns in name and namespace fields
func (r *ResourceIgnoreDifferences) MatchesResource(group, kind, namespace, name string) bool {
	// Match group (exact match, empty matches core resources)
	if r.Group != group {
		return false
	}

	// Match kind (exact match or wildcard)
	if r.Kind != "*" && r.Kind != kind {
		return false
	}

	// Match namespace (empty matches all, otherwise check glob)
	if r.Namespace != "" && !matchGlob(r.Namespace, namespace) {
		return false
	}

	// Match name (empty matches all, otherwise check glob)
	if r.Name != "" && !matchGlob(r.Name, name) {
		return false
	}

	return true
}

// matchGlob performs simple glob matching (* and ? wildcards)
func matchGlob(pattern, str string) bool {
	// For MVP, use simple string matching
	// TODO: Implement proper glob matching with * and ? support
	if pattern == "*" {
		return true
	}
	return pattern == str
}

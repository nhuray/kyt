package config

// Config represents the complete configuration for kyt
type Config struct {
	Diff DiffConfig `yaml:"diff"`
}

// DiffConfig contains all configuration for the diff command
type DiffConfig struct {
	IgnoreDifferences []ResourceIgnoreDifferences `yaml:"ignoreDifferences"`
	Normalization     NormalizationConfig         `yaml:"normalization"`
	Options           OptionsConfig               `yaml:"options"`
	FuzzyMatching     FuzzyMatchingConfig         `yaml:"fuzzyMatching"`
	Pager             string                      `yaml:"pager,omitempty"`
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

// FuzzyMatchingConfig controls fuzzy string matching behavior
type FuzzyMatchingConfig struct {
	// Enabled enables Levenshtein distance for comparing similar strings
	// When disabled, only exact string matches are considered equal
	// Default: true
	Enabled bool `yaml:"enabled"`

	// MinStringLength is the minimum string length (in characters) to apply fuzzy matching
	// Strings shorter than this use exact comparison
	// Higher values = faster but less accurate for short strings
	// Lower values = slower but more accurate
	// Default: 100
	MinStringLength int `yaml:"minStringLength,omitempty"`
}

// OptionsConfig controls diff generation options
type OptionsConfig struct {
	// ContextLines is the number of context lines for unified diff (default: 3)
	ContextLines int `yaml:"contextLines,omitempty"`

	// SimilarityThreshold is the minimum similarity score (0.0-1.0) for matching resources
	// Only used when similarity matching is enabled
	// Default: 0.7 (70% similarity)
	SimilarityThreshold float64 `yaml:"similarityThreshold,omitempty"`

	// DataSimilarityBoost is a boost factor (1-10) for ConfigMap/Secret data field importance
	// Higher values give more weight to data content vs metadata differences
	// boost=1: no boost (original weighting)
	// boost=2: data fields count 2x more (default)
	// boost=4: data fields count 4x more
	// boost=10: data fields heavily prioritized
	// Default: 2
	DataSimilarityBoost int `yaml:"dataSimilarityBoost,omitempty"`
}

// NewDefaultConfig returns a Config with sensible defaults
func NewDefaultConfig() *Config {
	return &Config{
		Diff: DiffConfig{
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
			Options: OptionsConfig{
				ContextLines:        3,
				SimilarityThreshold: 0.7,
				DataSimilarityBoost: 2,
			},
			FuzzyMatching: FuzzyMatchingConfig{
				Enabled:         true,
				MinStringLength: 100,
			},
			Pager: "", // Use $PAGER by default
		},
	}
}

// Merge merges another config into this one
// Rules from the other config are appended (not replaced)
func (c *Config) Merge(other *Config) {
	// Append ignore rules
	c.Diff.IgnoreDifferences = append(c.Diff.IgnoreDifferences, other.Diff.IgnoreDifferences...)

	// Merge normalization (other takes precedence for boolean fields)
	if other.Diff.Normalization.SortKeys {
		c.Diff.Normalization.SortKeys = true
	}
	c.Diff.Normalization.SortArrays = append(c.Diff.Normalization.SortArrays, other.Diff.Normalization.SortArrays...)
	c.Diff.Normalization.RemoveDefaultFields = append(c.Diff.Normalization.RemoveDefaultFields, other.Diff.Normalization.RemoveDefaultFields...)

	// Options config: other takes precedence
	if other.Diff.Options.ContextLines > 0 {
		c.Diff.Options.ContextLines = other.Diff.Options.ContextLines
	}
	if other.Diff.Options.SimilarityThreshold > 0 {
		c.Diff.Options.SimilarityThreshold = other.Diff.Options.SimilarityThreshold
	}
	if other.Diff.Options.DataSimilarityBoost > 0 {
		c.Diff.Options.DataSimilarityBoost = other.Diff.Options.DataSimilarityBoost
	}

	// FuzzyMatching config: other takes precedence
	if !other.Diff.FuzzyMatching.Enabled {
		c.Diff.FuzzyMatching.Enabled = false
	}
	if other.Diff.FuzzyMatching.MinStringLength > 0 {
		c.Diff.FuzzyMatching.MinStringLength = other.Diff.FuzzyMatching.MinStringLength
	}

	// Pager: other takes precedence if non-empty
	if other.Diff.Pager != "" {
		c.Diff.Pager = other.Diff.Pager
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

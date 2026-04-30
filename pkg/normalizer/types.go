package normalizer

import (
	"github.com/nhuray/kyt/pkg/config"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Normalizer applies normalization and ignore rules to Kubernetes resources
type Normalizer struct {
	config *config.Config
}

// New creates a new Normalizer with the given configuration
func New(cfg *config.Config) *Normalizer {
	if cfg == nil {
		cfg = config.NewDefaultConfig()
	}
	return &Normalizer{
		config: cfg,
	}
}

// NormalizeResult contains the result of normalizing a resource
type NormalizeResult struct {
	// Original is the original unmodified resource
	Original *unstructured.Unstructured

	// Normalized is the normalized resource with ignore rules applied
	Normalized *unstructured.Unstructured

	// Ignored is true if the entire resource should be ignored
	Ignored bool

	// IgnoredPaths contains the JSON paths that were ignored
	IgnoredPaths []string
}

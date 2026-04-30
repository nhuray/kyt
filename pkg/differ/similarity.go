package differ

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// SimilarityScorer computes structural similarity between Kubernetes resources
type SimilarityScorer struct {
	// Weights for different match types
	ExactMatchScore   float64
	StructuralMatch   float64
	KeyOnlyMatch      float64
	ArrayMatch        float64
	MissingFieldScore float64
}

// NewDefaultSimilarityScorer returns a scorer with default weights
func NewDefaultSimilarityScorer() *SimilarityScorer {
	return &SimilarityScorer{
		ExactMatchScore:   1.0,
		StructuralMatch:   0.8,
		KeyOnlyMatch:      0.3,
		ArrayMatch:        0.7,
		MissingFieldScore: 0.0,
	}
}

// CompareResources computes similarity between two Kubernetes resources
// Returns a score between 0.0 (completely different) and 1.0 (identical)
func (s *SimilarityScorer) CompareResources(a, b *unstructured.Unstructured) float64 {
	aObj := a.Object
	bObj := b.Object

	// Try to compare spec field first (most relevant for K8s resources)
	aSpec, aHasSpec := aObj["spec"]
	bSpec, bHasSpec := bObj["spec"]

	if aHasSpec && bHasSpec {
		// Both have spec - compare spec fields
		aSpecMap, aOk := aSpec.(map[string]interface{})
		bSpecMap, bOk := bSpec.(map[string]interface{})
		if aOk && bOk {
			return s.CompareSpecs(aSpecMap, bSpecMap)
		}
	}

	// Fallback: compare full object excluding metadata
	// Create copies without metadata for comparison
	aFiltered := make(map[string]interface{})
	bFiltered := make(map[string]interface{})

	for k, v := range aObj {
		if k != "metadata" && k != "apiVersion" && k != "kind" {
			aFiltered[k] = v
		}
	}

	for k, v := range bObj {
		if k != "metadata" && k != "apiVersion" && k != "kind" {
			bFiltered[k] = v
		}
	}

	return s.CompareSpecs(aFiltered, bFiltered)
}

// CompareSpecs compares two spec maps and returns similarity score
func (s *SimilarityScorer) CompareSpecs(a, b map[string]interface{}) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1.0 // Both empty
	}

	if len(a) == 0 || len(b) == 0 {
		return 0.0 // One empty, one not
	}

	// Short-circuit: check top-level key overlap
	// If very few keys match, skip deep comparison
	aKeys := make(map[string]bool)
	for k := range a {
		aKeys[k] = true
	}

	commonKeys := 0
	for k := range b {
		if aKeys[k] {
			commonKeys++
		}
	}

	allKeys := len(a)
	if len(b) > allKeys {
		allKeys = len(b)
	}

	keyOverlap := float64(commonKeys) / float64(allKeys)
	if keyOverlap < 0.3 {
		// Less than 30% key overlap - likely very different
		return keyOverlap * s.KeyOnlyMatch
	}

	// Deep comparison of all fields
	return s.compareObjects(a, b)
}

// compareObjects recursively compares two objects (maps)
func (s *SimilarityScorer) compareObjects(a, b map[string]interface{}) float64 {
	if len(a) == 0 && len(b) == 0 {
		return s.ExactMatchScore
	}

	// Collect all unique keys
	allKeys := make(map[string]bool)
	for k := range a {
		allKeys[k] = true
	}
	for k := range b {
		allKeys[k] = true
	}

	if len(allKeys) == 0 {
		return s.ExactMatchScore
	}

	totalScore := 0.0

	for key := range allKeys {
		aVal, aExists := a[key]
		bVal, bExists := b[key]

		if !aExists || !bExists {
			// Key missing in one side
			totalScore += s.MissingFieldScore
		} else {
			// Both exist - compare values
			totalScore += s.compareValues(aVal, bVal)
		}
	}

	return totalScore / float64(len(allKeys))
}

// compareValues compares two values of any type
func (s *SimilarityScorer) compareValues(a, b interface{}) float64 {
	if a == nil && b == nil {
		return s.ExactMatchScore
	}

	if a == nil || b == nil {
		return s.MissingFieldScore
	}

	// Check if exact match using reflect.DeepEqual
	if reflect.DeepEqual(a, b) {
		return s.ExactMatchScore
	}

	// Type-specific comparison
	aVal := reflect.ValueOf(a)
	bVal := reflect.ValueOf(b)

	// Must be same kind to compare structurally
	if aVal.Kind() != bVal.Kind() {
		return s.KeyOnlyMatch // Different types but key exists
	}

	switch aVal.Kind() {
	case reflect.Map:
		aMap, aOk := a.(map[string]interface{})
		bMap, bOk := b.(map[string]interface{})
		if aOk && bOk {
			// Recursive object comparison
			score := s.compareObjects(aMap, bMap)
			// Weight structural matches
			return score * s.StructuralMatch
		}
		return s.KeyOnlyMatch

	case reflect.Slice, reflect.Array:
		return s.compareArrays(a, b)

	default:
		// Primitives (string, int, bool, etc.) - already checked equality
		return s.KeyOnlyMatch
	}
}

// compareArrays compares two arrays/slices
// Assumes arrays are sorted by normalization
func (s *SimilarityScorer) compareArrays(a, b interface{}) float64 {
	aVal := reflect.ValueOf(a)
	bVal := reflect.ValueOf(b)

	if aVal.Len() == 0 && bVal.Len() == 0 {
		return s.ExactMatchScore
	}

	if aVal.Len() == 0 || bVal.Len() == 0 {
		return s.MissingFieldScore
	}

	// Element-by-element comparison
	maxLen := aVal.Len()
	if bVal.Len() > maxLen {
		maxLen = bVal.Len()
	}

	totalScore := 0.0
	matchCount := 0

	// Compare elements at same positions (arrays are normalized/sorted)
	minLen := aVal.Len()
	if bVal.Len() < minLen {
		minLen = bVal.Len()
	}

	for i := 0; i < minLen; i++ {
		aElem := aVal.Index(i).Interface()
		bElem := bVal.Index(i).Interface()

		if reflect.DeepEqual(aElem, bElem) {
			matchCount++
			totalScore += s.ExactMatchScore
		} else {
			// Try structural comparison for nested elements
			aMap, aOk := aElem.(map[string]interface{})
			bMap, bOk := bElem.(map[string]interface{})
			if aOk && bOk {
				totalScore += s.compareObjects(aMap, bMap) * s.StructuralMatch
			} else {
				totalScore += s.KeyOnlyMatch // Different values
			}
		}
	}

	// Penalize for length differences
	lengthDiff := aVal.Len() - bVal.Len()
	if lengthDiff < 0 {
		lengthDiff = -lengthDiff
	}

	// Add penalty for missing elements in longer array
	totalScore += float64(lengthDiff) * s.MissingFieldScore

	avgScore := totalScore / float64(maxLen)

	// Weight array matches
	return avgScore * s.ArrayMatch
}

// FormatSimilarity formats a similarity score for display
func FormatSimilarity(score float64) string {
	return fmt.Sprintf("%.2f", score)
}

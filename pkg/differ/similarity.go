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

	// StringSimilarityThreshold is the minimum string length for fuzzy matching
	// Strings longer than this will use Levenshtein distance
	StringSimilarityThreshold int
}

// NewDefaultSimilarityScorer returns a scorer with default weights
func NewDefaultSimilarityScorer() *SimilarityScorer {
	return &SimilarityScorer{
		ExactMatchScore:           1.0,
		StructuralMatch:           0.8,
		KeyOnlyMatch:              0.3,
		ArrayMatch:                0.7,
		MissingFieldScore:         0.0,
		StringSimilarityThreshold: 100, // Enable fuzzy matching for strings > 100 chars
	}
}

// NewSimilarityScorerWithThreshold returns a scorer with custom string similarity threshold
func NewSimilarityScorerWithThreshold(threshold int) *SimilarityScorer {
	scorer := NewDefaultSimilarityScorer()
	scorer.StringSimilarityThreshold = threshold
	return scorer
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

	// For ConfigMaps and Secrets, use weighted comparison including metadata
	// This gives more weight to data fields than to metadata differences
	kind := a.GetKind()
	if kind == "ConfigMap" || kind == "Secret" {
		return s.compareConfigMapOrSecret(aObj, bObj, kind)
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

// compareConfigMapOrSecret compares ConfigMaps/Secrets with weighted comparison
// Includes metadata fields but gives higher weight to data fields based on size
func (s *SimilarityScorer) compareConfigMapOrSecret(a, b map[string]interface{}, kind string) float64 {
	// Calculate weights for all fields including metadata
	weights := s.calculateConfigMapWeights(a, b, kind)

	totalScore := 0.0
	totalWeight := 0.0

	// Collect all unique keys (including metadata subfields)
	allKeys := make(map[string]bool)
	for k := range weights {
		allKeys[k] = true
	}

	for key := range allKeys {
		weight := weights[key]
		totalWeight += weight

		var score float64
		var exists bool

		// Special handling for metadata subfields
		switch key {
		case "metadata.name":
			aName, aExists, _ := unstructured.NestedString(a, "metadata", "name")
			bName, bExists, _ := unstructured.NestedString(b, "metadata", "name")

			if !aExists || !bExists {
				score = s.MissingFieldScore
			} else if aName == bName {
				score = s.ExactMatchScore
			} else {
				score = s.KeyOnlyMatch // Names different but key exists
			}
			exists = true

		case "metadata.labels":
			aLabels, aExists, _ := unstructured.NestedMap(a, "metadata", "labels")
			bLabels, bExists, _ := unstructured.NestedMap(b, "metadata", "labels")

			if !aExists || !bExists {
				score = s.MissingFieldScore
			} else {
				// Compare labels as maps
				score = s.compareObjects(aLabels, bLabels) * s.StructuralMatch
			}
			exists = true

		default:
			// Top-level fields (data, immutable, etc.)
			aVal, aExists := a[key]
			bVal, bExists := b[key]

			if !aExists || !bExists {
				score = s.MissingFieldScore
				exists = aExists || bExists
			} else {
				// Both exist - compare values
				// For data fields, compare directly without structural weight discount
				if key == "data" || key == "binaryData" || key == "stringData" {
					// Data fields - compare as maps but don't apply structural discount
					if aMap, aOk := aVal.(map[string]interface{}); aOk {
						if bMap, bOk := bVal.(map[string]interface{}); bOk {
							score = s.compareObjects(aMap, bMap)
						} else {
							score = s.KeyOnlyMatch
						}
					} else {
						score = s.compareValues(aVal, bVal)
					}
				} else {
					score = s.compareValues(aVal, bVal)
				}
				exists = true
			}
		}

		if exists {
			totalScore += score * weight
		}
	}

	if totalWeight == 0 {
		return s.ExactMatchScore
	}

	return totalScore / totalWeight
}

// calculateConfigMapWeights computes weights for ConfigMap/Secret fields
// Gives higher weight to data fields based on size, lower weight to metadata
func (s *SimilarityScorer) calculateConfigMapWeights(a, b map[string]interface{}, kind string) map[string]float64 {
	weights := make(map[string]float64)

	// Data field names based on kind
	dataFieldCandidates := make(map[string]bool)
	switch kind {
	case "ConfigMap":
		dataFieldCandidates["data"] = true
		dataFieldCandidates["binaryData"] = true
	case "Secret":
		dataFieldCandidates["data"] = true
		dataFieldCandidates["stringData"] = true
	}

	// Determine which data fields actually exist
	dataFields := make(map[string]bool)
	for field := range dataFieldCandidates {
		if _, existsA := a[field]; existsA {
			dataFields[field] = true
		}
		if _, existsB := b[field]; existsB {
			dataFields[field] = true
		}
	}

	// Calculate total size of data fields
	dataSize := 0
	for field := range dataFields {
		if val, exists := a[field]; exists {
			dataSize += calculateMapSize(val)
		}
		if val, exists := b[field]; exists {
			dataSize += calculateMapSize(val)
		}
	}

	// Average the sizes from both objects
	if dataSize > 0 {
		dataSize = dataSize / 2
	}

	// Calculate dynamic weight for data fields
	// Formula: min(0.9, 0.5 + (dataSize / 10000))
	dataWeight := 0.5
	if dataSize > 0 {
		dataWeight = 0.5 + (float64(dataSize) / 10000.0)
		if dataWeight > 0.9 {
			dataWeight = 0.9
		}
	}

	// Assign weights ONLY to data fields that exist
	for field := range dataFields {
		weights[field] = dataWeight
	}

	// Check if metadata fields exist
	hasMetadata := false
	if _, exists := a["metadata"]; exists {
		hasMetadata = true
	}
	if _, exists := b["metadata"]; exists {
		hasMetadata = true
	}

	// Only include metadata fields if metadata exists
	if hasMetadata {
		remainingWeight := 1.0 - dataWeight
		weights["metadata.name"] = remainingWeight * 0.4   // 40% of remaining
		weights["metadata.labels"] = remainingWeight * 0.6 // 60% of remaining
	}

	return weights
}

// calculateMapSize calculates the total size of strings in a map
func calculateMapSize(val interface{}) int {
	size := 0

	switch v := val.(type) {
	case map[string]interface{}:
		for _, value := range v {
			if str, ok := value.(string); ok {
				size += len(str)
			} else if bytes, ok := value.([]byte); ok {
				size += len(bytes)
			}
		}
	}

	return size
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

	case reflect.String:
		// Use fuzzy string matching for long strings
		aStr, aOk := a.(string)
		bStr, bOk := b.(string)
		if aOk && bOk {
			// Check if strings are long enough for fuzzy matching
			// Note: threshold of 0 disables fuzzy matching
			if s.StringSimilarityThreshold > 0 && (len(aStr) >= s.StringSimilarityThreshold || len(bStr) >= s.StringSimilarityThreshold) {
				// Use Levenshtein-based similarity
				similarity := stringSimilarity(aStr, bStr)
				// Scale by structural match weight
				return similarity * s.StructuralMatch
			}
		}
		// Short strings or type mismatch - treat as key-only match
		return s.KeyOnlyMatch

	default:
		// Primitives (int, bool, etc.) - already checked equality
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

package differ

import (
	"math"
)

// levenshteinDistance computes the Levenshtein distance between two strings
// Returns the minimum number of single-character edits (insertions, deletions, substitutions)
// required to change one string into the other
func levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Create a matrix to store distances
	// matrix[i][j] represents the distance between a[0:i] and b[0:j]
	matrix := make([][]int, len(a)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(b)+1)
	}

	// Initialize first row and column
	for i := 0; i <= len(a); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(b); j++ {
		matrix[0][j] = j
	}

	// Fill in the rest of the matrix
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(a)][len(b)]
}

// stringSimilarity computes similarity between two strings as a ratio
// Returns a value between 0.0 (completely different) and 1.0 (identical)
// Uses normalized Levenshtein distance: 1 - (distance / maxLength)
func stringSimilarity(a, b string) float64 {
	if a == b {
		return 1.0
	}

	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}

	distance := levenshteinDistance(a, b)
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}

	// Normalize: similarity = 1 - (distance / maxLength)
	similarity := 1.0 - (float64(distance) / float64(maxLen))

	// Ensure non-negative (shouldn't happen, but be safe)
	if similarity < 0 {
		similarity = 0
	}

	return similarity
}

// stringSimilarityOptimized computes string similarity with early exit
// For very different strings, stops early to avoid expensive computation
// threshold: if quick check suggests similarity < threshold, return early
func stringSimilarityOptimized(a, b string, threshold float64) float64 {
	if a == b {
		return 1.0
	}

	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}

	// Quick length check: if lengths differ by more than threshold allows, skip
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}

	lengthDiff := maxLen - minLen
	maxPossibleSimilarity := 1.0 - (float64(lengthDiff) / float64(maxLen))

	if maxPossibleSimilarity < threshold {
		// Even if all common characters match, similarity would be below threshold
		return maxPossibleSimilarity
	}

	// Full Levenshtein computation
	return stringSimilarity(a, b)
}

// min returns the minimum of three integers
func min(a, b, c int) int {
	result := a
	if b < result {
		result = b
	}
	if c < result {
		result = c
	}
	return result
}

// minInt returns the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// maxInt returns the maximum of two integers
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// jaroWinklerSimilarity computes Jaro-Winkler similarity (alternative to Levenshtein)
// Generally faster but less accurate for large edits
// Not currently used, but kept for reference
func jaroWinklerSimilarity(a, b string) float64 {
	if a == b {
		return 1.0
	}

	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}

	// Compute Jaro similarity
	maxDist := int(math.Max(float64(len(a)), float64(len(b)))/2) - 1
	if maxDist < 0 {
		maxDist = 0
	}

	aMatches := make([]bool, len(a))
	bMatches := make([]bool, len(b))
	matches := 0
	transpositions := 0

	// Find matches
	for i := 0; i < len(a); i++ {
		start := maxInt(0, i-maxDist)
		end := minInt(i+maxDist+1, len(b))

		for j := start; j < end; j++ {
			if bMatches[j] || a[i] != b[j] {
				continue
			}
			aMatches[i] = true
			bMatches[j] = true
			matches++
			break
		}
	}

	if matches == 0 {
		return 0.0
	}

	// Count transpositions
	k := 0
	for i := 0; i < len(a); i++ {
		if !aMatches[i] {
			continue
		}
		for !bMatches[k] {
			k++
		}
		if a[i] != b[k] {
			transpositions++
		}
		k++
	}

	jaro := (float64(matches)/float64(len(a)) +
		float64(matches)/float64(len(b)) +
		float64(matches-transpositions/2)/float64(matches)) / 3.0

	// Jaro-Winkler adds prefix bonus
	prefixLen := 0
	maxPrefix := minInt(minInt(len(a), len(b)), 4)
	for i := 0; i < maxPrefix; i++ {
		if a[i] == b[i] {
			prefixLen++
		} else {
			break
		}
	}

	return jaro + float64(prefixLen)*0.1*(1.0-jaro)
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

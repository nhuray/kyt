package differ

import (
	"testing"
)

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		name     string
		s1       string
		s2       string
		expected int
	}{
		{
			name:     "identical strings",
			s1:       "hello",
			s2:       "hello",
			expected: 0,
		},
		{
			name:     "empty strings",
			s1:       "",
			s2:       "",
			expected: 0,
		},
		{
			name:     "one empty string",
			s1:       "hello",
			s2:       "",
			expected: 5,
		},
		{
			name:     "one character difference",
			s1:       "hello",
			s2:       "hallo",
			expected: 1,
		},
		{
			name:     "insertion",
			s1:       "hello",
			s2:       "hellow",
			expected: 1,
		},
		{
			name:     "deletion",
			s1:       "hello",
			s2:       "hell",
			expected: 1,
		},
		{
			name:     "multiple differences",
			s1:       "kitten",
			s2:       "sitting",
			expected: 3,
		},
		{
			name:     "completely different",
			s1:       "abc",
			s2:       "xyz",
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := levenshteinDistance(tt.s1, tt.s2)
			if result != tt.expected {
				t.Errorf("levenshteinDistance(%q, %q) = %d, want %d", tt.s1, tt.s2, result, tt.expected)
			}
		})
	}
}

func TestStringSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		s1       string
		s2       string
		expected float64
		delta    float64 // tolerance for float comparison
	}{
		{
			name:     "identical strings",
			s1:       "hello world",
			s2:       "hello world",
			expected: 1.0,
			delta:    0.0001,
		},
		{
			name:     "completely different",
			s1:       "abc",
			s2:       "xyz",
			expected: 0.0,
			delta:    0.0001,
		},
		{
			name:     "one character difference",
			s1:       "hello",
			s2:       "hallo",
			expected: 0.8, // 4/5 characters match
			delta:    0.0001,
		},
		{
			name:     "empty strings",
			s1:       "",
			s2:       "",
			expected: 1.0,
			delta:    0.0001,
		},
		{
			name:     "one empty string",
			s1:       "hello",
			s2:       "",
			expected: 0.0,
			delta:    0.0001,
		},
		{
			name:     "very similar long strings",
			s1:       "This is a very long string that should have high similarity",
			s2:       "This is a very long string that should have high similarity!",
			expected: 0.9836, // 60 out of 61 characters
			delta:    0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stringSimilarity(tt.s1, tt.s2)
			if !floatEquals(result, tt.expected, tt.delta) {
				t.Errorf("stringSimilarity(%q, %q) = %f, want %f (±%f)", tt.s1, tt.s2, result, tt.expected, tt.delta)
			}
		})
	}
}

func TestStringSimilarityOptimized(t *testing.T) {
	tests := []struct {
		name            string
		s1              string
		s2              string
		threshold       float64
		expectedMin     float64 // minimum expected value
		shouldEarlyExit bool    // whether early exit optimization should apply
	}{
		{
			name:            "above threshold - full computation",
			s1:              "hello world",
			s2:              "hello world!",
			threshold:       0.9,
			expectedMin:     0.9,
			shouldEarlyExit: false,
		},
		{
			name:            "below threshold - early exit",
			s1:              "completely different",
			s2:              "xyz",
			threshold:       0.8,
			expectedMin:     0.0,
			shouldEarlyExit: true,
		},
		{
			name:            "identical strings",
			s1:              "test",
			s2:              "test",
			threshold:       0.95,
			expectedMin:     1.0,
			shouldEarlyExit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stringSimilarityOptimized(tt.s1, tt.s2, tt.threshold)

			// For early exit cases, result should be < threshold
			if tt.shouldEarlyExit && result >= tt.threshold {
				t.Errorf("stringSimilarityOptimized(%q, %q, %f) = %f, expected early exit with result < %f",
					tt.s1, tt.s2, tt.threshold, result, tt.threshold)
			}

			// Check minimum expected value
			if result < tt.expectedMin {
				t.Errorf("stringSimilarityOptimized(%q, %q, %f) = %f, want >= %f",
					tt.s1, tt.s2, tt.threshold, result, tt.expectedMin)
			}
		})
	}
}

func TestJaroWinklerSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		s1       string
		s2       string
		expected float64
		delta    float64
	}{
		{
			name:     "identical strings",
			s1:       "hello",
			s2:       "hello",
			expected: 1.0,
			delta:    0.0001,
		},
		{
			name:     "completely different",
			s1:       "abc",
			s2:       "xyz",
			expected: 0.0,
			delta:    0.0001,
		},
		{
			name:     "similar strings",
			s1:       "martha",
			s2:       "marhta",
			expected: 0.961, // Jaro-Winkler favors strings with common prefix
			delta:    0.01,
		},
		{
			name:     "common prefix",
			s1:       "test",
			s2:       "testing",
			expected: 0.90, // high score due to common prefix
			delta:    0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := jaroWinklerSimilarity(tt.s1, tt.s2)
			if !floatEquals(result, tt.expected, tt.delta) {
				t.Errorf("jaroWinklerSimilarity(%q, %q) = %f, want %f (±%f)",
					tt.s1, tt.s2, result, tt.expected, tt.delta)
			}
		})
	}
}

func TestSimilarityScorerWithStringThreshold(t *testing.T) {
	tests := []struct {
		name      string
		threshold int
		str1      string
		str2      string
		wantFuzzy bool // whether fuzzy matching should be used
	}{
		{
			name:      "strings above threshold - should use fuzzy",
			threshold: 10,
			str1:      "this is a long string with many characters",
			str2:      "this is a long string with some characters",
			wantFuzzy: true,
		},
		{
			name:      "strings below threshold - should use exact",
			threshold: 100,
			str1:      "short",
			str2:      "shirt",
			wantFuzzy: false,
		},
		{
			name:      "threshold 0 disables fuzzy matching",
			threshold: 0,
			str1:      "any string no matter how long it is",
			str2:      "any string no matter how long it was",
			wantFuzzy: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scorer := NewSimilarityScorerWithThreshold(tt.threshold)

			// Test by comparing the score with what we'd expect
			score := scorer.compareValues(tt.str1, tt.str2)

			if tt.wantFuzzy {
				// Fuzzy matching should give a high score for similar strings
				// and scale by StructuralMatch (0.8)
				if score < 0.5 {
					t.Errorf("Expected fuzzy matching with high similarity, got score %f", score)
				}
			} else {
				// Without fuzzy matching, non-identical strings get KeyOnlyMatch (0.3)
				if score != scorer.KeyOnlyMatch {
					t.Errorf("Expected exact matching (KeyOnlyMatch=%f), got score %f", scorer.KeyOnlyMatch, score)
				}
			}
		})
	}
}

// floatEquals checks if two floats are equal within a delta
func floatEquals(a, b, delta float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff <= delta
}

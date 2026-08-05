package masker

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/nhuray/kyt/pkg/config"
)

// PatternMatcher handles pattern-based secret masking with regex caching
type PatternMatcher struct {
	rules       []config.SecretMaskingRule
	regexCache  sync.Map // map[string]*regexp.Regexp
	globalMask  string
}

// NewPatternMatcher creates a new pattern matcher
func NewPatternMatcher(rules []config.SecretMaskingRule, globalMaskChar string) *PatternMatcher {
	if globalMaskChar == "" {
		globalMaskChar = "*"
	}
	return &PatternMatcher{
		rules:      rules,
		globalMask: globalMaskChar,
	}
}

// getRegex retrieves or compiles and caches a regex pattern
func (pm *PatternMatcher) getRegex(pattern string) (*regexp.Regexp, error) {
	// Check cache first
	if cached, ok := pm.regexCache.Load(pattern); ok {
		return cached.(*regexp.Regexp), nil
	}

	// Compile and cache
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to compile pattern: %w", err)
	}

	pm.regexCache.Store(pattern, compiled)
	return compiled, nil
}

// Mask applies pattern-based masking to a value
func (pm *PatternMatcher) Mask(value string) (string, error) {
	// Try each rule in order (first match wins)
	for _, rule := range pm.rules {
		re, err := pm.getRegex(rule.Pattern)
		if err != nil {
			return "", fmt.Errorf("rule '%s': %w", rule.Name, err)
		}

		// Check if pattern matches
		if !re.MatchString(value) {
			continue
		}

		// Pattern matched, apply masking
		return pm.applyRule(value, rule, re)
	}

	// No rule matched, mask everything (fallback)
	return MaskKeepFirstLast(value, 0, 0, pm.globalMask), nil
}

// applyRule applies a specific rule to mask the value
func (pm *PatternMatcher) applyRule(value string, rule config.SecretMaskingRule, re *regexp.Regexp) (string, error) {
	// Find all submatches with their positions
	matchIndexes := re.FindStringSubmatchIndex(value)
	if matchIndexes == nil {
		// Should not happen since we checked MatchString, but handle gracefully
		return MaskKeepFirstLast(value, 0, 0, pm.globalMask), nil
	}

	// Get capture group names
	names := re.SubexpNames()

	// Determine mask character (rule-specific or global)
	maskChar := rule.MaskChar
	if maskChar == "" {
		maskChar = pm.globalMask
	}

	// Build result by reconstructing the string with masked captures
	var result []byte
	lastEnd := 0

	for i := 1; i < len(names); i++ {
		if names[i] == "" {
			continue // Skip unnamed groups
		}

		start := matchIndexes[2*i]
		end := matchIndexes[2*i+1]

		if start < 0 || end < 0 {
			continue // Group didn't match
		}

		// Append everything between last capture and this one
		result = append(result, value[lastEnd:start]...)

		// Get masking strategy for this capture
		strategyStr, exists := rule.Masks[names[i]]
		if !exists {
			// This should not happen due to validation, but handle gracefully
			result = append(result, value[start:end]...)
			lastEnd = end
			continue
		}

		// Parse and apply strategy
		strategy, err := ParseStrategy(strategyStr)
		if err != nil {
			return "", fmt.Errorf("rule '%s', capture '%s': %w", rule.Name, names[i], err)
		}

		captureValue := value[start:end]
		masked := strategy.Execute(captureValue, maskChar)
		result = append(result, masked...)

		lastEnd = end
	}

	// Append any remaining text after the last capture
	result = append(result, value[lastEnd:]...)

	return string(result), nil
}

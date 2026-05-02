package config

import (
	"fmt"
	"strings"

	"github.com/itchyny/gojq"
)

// Validator validates configuration
type Validator struct{}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{}
}

// Validate validates a configuration
func (v *Validator) Validate(cfg *Config) error {
	// Validate ignore differences
	for i, rule := range cfg.Diff.IgnoreDifferences {
		if err := v.validateIgnoreRule(&rule, i); err != nil {
			return err
		}
	}

	// Validate options config
	if err := v.validateOptionsConfig(&cfg.Diff.Options); err != nil {
		return err
	}

	// Validate fuzzy matching config
	if err := v.validateFuzzyMatchingConfig(&cfg.Diff.FuzzyMatching); err != nil {
		return err
	}

	return nil
}

// validateIgnoreRule validates a single ignore rule
func (v *Validator) validateIgnoreRule(rule *ResourceIgnoreDifferences, index int) error {
	// Kind is required
	if rule.Kind == "" {
		return fmt.Errorf("ignoreDifferences[%d]: kind is required", index)
	}

	// At least one ignore method must be specified
	if len(rule.JSONPointers) == 0 && len(rule.JQPathExpressions) == 0 && len(rule.ManagedFieldsManagers) == 0 {
		return fmt.Errorf("ignoreDifferences[%d]: must specify at least one of: jsonPointers, jqPathExpressions, or managedFieldsManagers", index)
	}

	// Validate JSON Pointers
	for j, pointer := range rule.JSONPointers {
		if err := v.validateJSONPointer(pointer); err != nil {
			return fmt.Errorf("ignoreDifferences[%d].jsonPointers[%d]: %w", index, j, err)
		}
	}

	// Validate JQ expressions
	for j, expr := range rule.JQPathExpressions {
		if err := v.validateJQExpression(expr); err != nil {
			return fmt.Errorf("ignoreDifferences[%d].jqPathExpressions[%d]: %w", index, j, err)
		}
	}

	return nil
}

// validateJSONPointer validates a JSON Pointer (RFC 6901)
func (v *Validator) validateJSONPointer(pointer string) error {
	// JSON Pointers must start with "/"
	if !strings.HasPrefix(pointer, "/") {
		return fmt.Errorf("JSON Pointer must start with '/': %s", pointer)
	}

	// Check for invalid escape sequences
	// In JSON Pointer, ~ must be followed by 0 (for ~) or 1 (for /)
	parts := strings.Split(pointer, "~")
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) == 0 || (parts[i][0] != '0' && parts[i][0] != '1') {
			return fmt.Errorf("invalid JSON Pointer escape sequence in: %s (~ must be followed by 0 or 1)", pointer)
		}
	}

	return nil
}

// validateJQExpression validates a JQ path expression
func (v *Validator) validateJQExpression(expr string) error {
	if expr == "" {
		return fmt.Errorf("JQ expression cannot be empty")
	}

	// Try to parse the expression with gojq
	_, err := gojq.Parse(expr)
	if err != nil {
		return fmt.Errorf("invalid JQ expression: %w", err)
	}

	return nil
}

// validateOptionsConfig validates options configuration
func (v *Validator) validateOptionsConfig(opts *OptionsConfig) error {
	// Context lines must be non-negative
	if opts.ContextLines < 0 {
		return fmt.Errorf("contextLines must be non-negative, got: %d", opts.ContextLines)
	}

	// Similarity threshold must be between 0 and 1
	if opts.SimilarityThreshold < 0 || opts.SimilarityThreshold > 1 {
		return fmt.Errorf("similarityThreshold must be between 0 and 1, got: %f", opts.SimilarityThreshold)
	}

	// Data similarity boost must be between 1 and 10
	if opts.DataSimilarityBoost < 0 {
		return fmt.Errorf("dataSimilarityBoost must be non-negative, got: %d", opts.DataSimilarityBoost)
	}
	if opts.DataSimilarityBoost > 10 {
		return fmt.Errorf("dataSimilarityBoost must be between 1 and 10, got: %d", opts.DataSimilarityBoost)
	}

	return nil
}

// validateFuzzyMatchingConfig validates fuzzy matching configuration
func (v *Validator) validateFuzzyMatchingConfig(cfg *FuzzyMatchingConfig) error {
	// MinStringLength must be non-negative
	if cfg.MinStringLength < 0 {
		return fmt.Errorf("fuzzyMatching.minStringLength must be non-negative, got: %d", cfg.MinStringLength)
	}

	// Practical upper limit to prevent performance issues
	if cfg.MinStringLength > 10000 {
		return fmt.Errorf("fuzzyMatching.minStringLength must be <= 10000, got: %d", cfg.MinStringLength)
	}

	return nil
}

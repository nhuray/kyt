package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ValidateSecretMaskingRules validates all rules in the config
func ValidateSecretMaskingRules(rules []SecretMaskingRule) error {
	if len(rules) == 0 {
		return fmt.Errorf("at least one masking rule is required (including default)")
	}

	names := make(map[string]bool)

	for i, rule := range rules {
		// Validate rule name uniqueness
		if rule.Name == "" {
			return fmt.Errorf("rule #%d: name is required", i)
		}
		if names[rule.Name] {
			return fmt.Errorf("rule '%s': duplicate rule name", rule.Name)
		}
		names[rule.Name] = true

		// Validate pattern compiles
		compiled, err := regexp.Compile(rule.Pattern)
		if err != nil {
			return fmt.Errorf("rule '%s': invalid regex pattern: %w", rule.Name, err)
		}

		// Validate capture groups exist in pattern
		captureNames := compiled.SubexpNames()[1:] // Skip index 0 (full match)
		captureMap := make(map[string]bool)
		for _, name := range captureNames {
			if name != "" {
				captureMap[name] = true
			}
		}

		// Validate masks reference valid capture groups
		if len(rule.Masks) == 0 {
			return fmt.Errorf("rule '%s': must define at least one mask", rule.Name)
		}

		for captureName, strategy := range rule.Masks {
			// Check capture group exists
			if !captureMap[captureName] {
				return fmt.Errorf("rule '%s': mask references non-existent capture group '%s'", rule.Name, captureName)
			}

			// Validate strategy syntax
			if err := ValidateStrategy(strategy); err != nil {
				return fmt.Errorf("rule '%s', capture '%s': %w", rule.Name, captureName, err)
			}
		}

		// Check for unmapped capture groups (fail-fast for security)
		for captureName := range captureMap {
			if _, exists := rule.Masks[captureName]; !exists {
				return fmt.Errorf("rule '%s': capture group '%s' is not mapped to a masking strategy", rule.Name, captureName)
			}
		}
	}

	return nil
}

// ValidateStrategy validates a masking strategy string
func ValidateStrategy(strategy string) error {
	// Expected formats: "mask-F-E" or "mask-F-E-S-M"
	parts := strings.Split(strategy, "-")

	if len(parts) < 3 || parts[0] != "mask" {
		return fmt.Errorf("invalid strategy format '%s': expected 'mask-F-E' or 'mask-F-E-S-M'", strategy)
	}

	// Parse numbers
	var nums []int
	for i := 1; i < len(parts); i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			return fmt.Errorf("invalid strategy '%s': '%s' is not a number", strategy, parts[i])
		}
		nums = append(nums, n)
	}

	// Validate based on format
	if len(nums) == 2 {
		// mask-F-E
		F, E := nums[0], nums[1]
		if F < 0 || E < 0 {
			return fmt.Errorf("invalid strategy '%s': F and E must be >= 0", strategy)
		}
	} else if len(nums) == 4 {
		// mask-F-E-S-M
		F, E, S, M := nums[0], nums[1], nums[2], nums[3]
		if F < 0 || E < 0 {
			return fmt.Errorf("invalid strategy '%s': F and E must be >= 0", strategy)
		}
		if S <= 0 {
			return fmt.Errorf("invalid strategy '%s': S must be > 0", strategy)
		}
		if M < 0 {
			return fmt.Errorf("invalid strategy '%s': M must be >= 0", strategy)
		}
	} else {
		return fmt.Errorf("invalid strategy '%s': expected 2 or 4 numbers, got %d", strategy, len(nums))
	}

	return nil
}

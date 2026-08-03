package filter

import (
	"fmt"
	"strings"
)

// Expression represents a parsed filter expression with include/exclude rules
type Expression struct {
	// Includes contains items to explicitly include
	// Empty means "include all" (unless Excludes is non-empty)
	Includes []string

	// Excludes contains items to explicitly exclude
	Excludes []string
}

// Parse parses a comma-separated filter expression
// Supports both include and exclude syntax with "-" prefix
// Examples:
//   - "dep,sts" -> includes: [dep, sts], excludes: []
//   - "-cm,-sec" -> includes: [], excludes: [cm, sec]
//   - "dep,sts,-cm" -> includes: [dep, sts], excludes: [cm]
func Parse(expr string) (*Expression, error) {
	if expr == "" {
		return &Expression{}, nil
	}

	parts := strings.Split(expr, ",")
	var includes, excludes []string

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.HasPrefix(part, "-") {
			// Exclude
			excludeName := strings.TrimPrefix(part, "-")
			if excludeName == "" {
				return nil, fmt.Errorf("invalid exclude syntax: empty name after '-'")
			}
			excludes = append(excludes, excludeName)
		} else {
			// Include
			includes = append(includes, part)
		}
	}

	return &Expression{
		Includes: includes,
		Excludes: excludes,
	}, nil
}

// ParseList parses a list of filter expressions (from config file)
// Each item can have a "-" prefix for exclusion
// Examples:
//   - ["Deployment", "StatefulSet"] -> includes
//   - ["-ConfigMap", "-Secret"] -> excludes
//   - ["Deployment", "-Secret"] -> mixed
func ParseList(items []string) (*Expression, error) {
	var includes, excludes []string

	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		if strings.HasPrefix(item, "-") {
			// Exclude
			excludeName := strings.TrimPrefix(item, "-")
			if excludeName == "" {
				return nil, fmt.Errorf("invalid exclude syntax: empty name after '-'")
			}
			excludes = append(excludes, excludeName)
		} else {
			// Include
			includes = append(includes, item)
		}
	}

	return &Expression{
		Includes: includes,
		Excludes: excludes,
	}, nil
}

// ShouldInclude determines if a value should be included based on the filter expression
// Logic:
//   1. If there are includes specified, value must match one of them
//   2. If there are excludes specified, value must NOT match any of them
//   3. Process includes first, then apply excludes
//   4. If both are empty, include everything
func (e *Expression) ShouldInclude(value string, matcher func(string, string) bool) bool {
	if e == nil || (len(e.Includes) == 0 && len(e.Excludes) == 0) {
		// No filters specified, include everything
		return true
	}

	// If includes are specified, value must match one of them
	if len(e.Includes) > 0 {
		matchedInclude := false
		for _, include := range e.Includes {
			if matcher(value, include) {
				matchedInclude = true
				break
			}
		}
		if !matchedInclude {
			// Value doesn't match any include filter
			return false
		}
	}

	// Check excludes (these take precedence over includes)
	for _, exclude := range e.Excludes {
		if matcher(value, exclude) {
			// Value matches an exclude filter
			return false
		}
	}

	// Value passed all filters
	return true
}

// Merge merges another expression into this one
// CLI expressions take precedence over config expressions
// If CLI has any filters, they completely override config filters
func (e *Expression) Merge(other *Expression) *Expression {
	if other == nil {
		return e
	}

	// If other (CLI) has any filters, it completely overrides this (config)
	if len(other.Includes) > 0 || len(other.Excludes) > 0 {
		return &Expression{
			Includes: other.Includes,
			Excludes: other.Excludes,
		}
	}

	// CLI has no filters, use config
	return e
}

// IsEmpty returns true if the expression has no filters
func (e *Expression) IsEmpty() bool {
	return len(e.Includes) == 0 && len(e.Excludes) == 0
}

// String returns a human-readable representation of the filter
func (e *Expression) String() string {
	if e.IsEmpty() {
		return "all"
	}

	var parts []string
	if len(e.Includes) > 0 {
		parts = append(parts, fmt.Sprintf("include: %v", e.Includes))
	}
	if len(e.Excludes) > 0 {
		parts = append(parts, fmt.Sprintf("exclude: %v", e.Excludes))
	}

	return strings.Join(parts, ", ")
}

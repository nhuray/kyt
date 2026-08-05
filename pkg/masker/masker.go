package masker

import (
	"strings"

	"github.com/nhuray/kyt/pkg/config"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Masker handles masking of sensitive data
type Masker struct {
	// Legacy simple mode
	keepFirst int
	keepLast  int
	maskChar  string

	// New rule-based mode
	patternMatcher *PatternMatcher
	multiline      bool // Enable per-line masking for multiline values
}

// NewMasker creates a new Masker with simple keep-first-last pattern (legacy mode)
func NewMasker(keepFirst, keepLast int, maskChar string) *Masker {
	return &Masker{
		keepFirst: keepFirst,
		keepLast:  keepLast,
		maskChar:  maskChar,
		multiline: true, // Default to true
	}
}

// NewMaskerWithRules creates a new Masker with rule-based masking
func NewMaskerWithRules(rules []config.SecretMaskingRule, maskChar string, multiline bool) *Masker {
	return &Masker{
		patternMatcher: NewPatternMatcher(rules, maskChar),
		maskChar:       maskChar,
		multiline:      multiline,
	}
}

// Mask applies masking to a string value
// Uses rule-based masking if configured, otherwise falls back to simple keep-first-last
func (m *Masker) Mask(value string) string {
	// Handle empty values
	if len(value) == 0 {
		return "<empty>"
	}

	// Check if we should apply per-line masking for multiline values
	if m.multiline && m.isMultiline(value) {
		return m.maskMultiline(value)
	}

	// Single-line masking
	return m.maskSingleLine(value)
}

// isMultiline checks if a value should be treated as multiline
func (m *Masker) isMultiline(value string) bool {
	lines := strings.Split(value, "\n")
	// Must have at least 2 lines (not just a trailing newline)
	return len(lines) > 1 && value != lines[0]+"\n"
}

// maskMultiline masks each line independently
func (m *Masker) maskMultiline(value string) string {
	lines := strings.Split(value, "\n")
	maskedLines := make([]string, len(lines))
	for i, line := range lines {
		maskedLines[i] = m.maskSingleLine(line)
	}
	return strings.Join(maskedLines, "\n")
}

// maskSingleLine masks a single line value
func (m *Masker) maskSingleLine(value string) string {
	// Use rule-based masking if available
	if m.patternMatcher != nil {
		masked, err := m.patternMatcher.Mask(value)
		if err != nil {
			// Log error and fall back to masking everything
			// In production, you might want to use a logger here
			return strings.Repeat(m.maskChar, len(value))
		}
		return masked
	}

	// Legacy simple mode: keep-first-last pattern
	// If value is too short to mask meaningfully, show minimum asterisks
	if len(value) <= m.keepFirst+m.keepLast {
		return strings.Repeat(m.maskChar, 4) // minimum "****"
	}

	// Apply keep-first-last pattern
	front := value[:m.keepFirst]
	back := value[len(value)-m.keepLast:]
	maskedLength := len(value) - m.keepFirst - m.keepLast
	masked := strings.Repeat(m.maskChar, maskedLength)

	return front + masked + back
}

// MaskSecretData masks specified fields in a Secret object
// This function creates a deep copy and masks the values in the specified fields
func (m *Masker) MaskSecretData(secretObj *unstructured.Unstructured, fields []string) *unstructured.Unstructured {
	if secretObj == nil {
		return nil
	}

	// Create a deep copy to avoid modifying the original
	masked := secretObj.DeepCopy()
	obj := masked.Object

	// Mask each specified field
	for _, field := range fields {
		if fieldData, found, err := unstructured.NestedMap(obj, field); found && err == nil {
			maskedFieldData := m.maskMap(fieldData)
			_ = unstructured.SetNestedMap(obj, maskedFieldData, field)
		}
	}

	return masked
}

// maskMap masks all values in a map (typically Secret data or stringData)
func (m *Masker) maskMap(data map[string]interface{}) map[string]interface{} {
	if data == nil {
		return nil
	}

	masked := make(map[string]interface{}, len(data))
	for key, value := range data {
		// Convert value to string and mask it
		if strValue, ok := value.(string); ok {
			masked[key] = m.Mask(strValue)
		} else {
			// If not a string, keep as-is (shouldn't happen for Secret data/stringData)
			masked[key] = value
		}
	}
	return masked
}

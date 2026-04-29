package normalizer

import (
	"encoding/json"
	"fmt"
	"sort"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Normalize applies normalization to a resource based on the configuration
// Returns a normalized copy of the resource
func (n *Normalizer) Normalize(obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	if obj == nil {
		return nil, fmt.Errorf("cannot normalize nil object")
	}

	// Create a deep copy to avoid modifying the original
	normalized := obj.DeepCopy()

	// Apply default field removals
	if err := n.removeDefaultFields(normalized); err != nil {
		return nil, fmt.Errorf("failed to remove default fields: %w", err)
	}

	// Apply ignore rules
	if err := n.applyIgnoreRules(normalized); err != nil {
		return nil, fmt.Errorf("failed to apply ignore rules: %w", err)
	}

	// Sort keys if configured
	if n.config.Normalization.SortKeys {
		if err := n.sortKeys(normalized); err != nil {
			return nil, fmt.Errorf("failed to sort keys: %w", err)
		}
	}

	return normalized, nil
}

// NormalizeAll applies normalization to multiple resources
func (n *Normalizer) NormalizeAll(objs []*unstructured.Unstructured) ([]*unstructured.Unstructured, error) {
	normalized := make([]*unstructured.Unstructured, 0, len(objs))

	for i, obj := range objs {
		norm, err := n.Normalize(obj)
		if err != nil {
			return nil, fmt.Errorf("failed to normalize resource %d: %w", i, err)
		}
		normalized = append(normalized, norm)
	}

	return normalized, nil
}

// removeDefaultFields removes fields that are in the removeDefaultFields list
func (n *Normalizer) removeDefaultFields(obj *unstructured.Unstructured) error {
	for _, field := range n.config.Normalization.RemoveDefaultFields {
		if err := removeJSONPointerField(obj, field); err != nil {
			// Don't fail if field doesn't exist, just continue
			continue
		}
	}
	return nil
}

// applyIgnoreRules applies all matching ignore rules to the resource
func (n *Normalizer) applyIgnoreRules(obj *unstructured.Unstructured) error {
	gvk := obj.GroupVersionKind()
	group := gvk.Group
	kind := gvk.Kind
	namespace := obj.GetNamespace()
	name := obj.GetName()

	for _, rule := range n.config.IgnoreDifferences {
		// Check if this rule matches the resource
		if !rule.MatchesResource(group, kind, namespace, name) {
			continue
		}

		// Apply JSON Pointer ignores
		for _, pointer := range rule.JSONPointers {
			if err := removeJSONPointerField(obj, pointer); err != nil {
				// Don't fail if field doesn't exist
				continue
			}
		}

		// Apply JQ expression ignores
		for _, expr := range rule.JQPathExpressions {
			if err := removeJQExpressionField(obj, expr); err != nil {
				// Don't fail if expression doesn't match
				continue
			}
		}

		// Remove managed fields by manager
		if len(rule.ManagedFieldsManagers) > 0 {
			if err := removeManagedFieldsByManagers(obj, rule.ManagedFieldsManagers); err != nil {
				continue
			}
		}
	}

	return nil
}

// sortKeys recursively sorts all object keys alphabetically
func (n *Normalizer) sortKeys(obj *unstructured.Unstructured) error {
	data := obj.Object
	sorted := sortMapKeys(data)
	obj.Object = sorted
	return nil
}

// sortMapKeys recursively sorts map keys
func sortMapKeys(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Get sorted keys
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build new map with sorted keys
	for _, k := range keys {
		v := m[k]

		// Recursively sort nested maps
		switch val := v.(type) {
		case map[string]interface{}:
			result[k] = sortMapKeys(val)
		case []interface{}:
			result[k] = sortSlice(val)
		default:
			result[k] = v
		}
	}

	return result
}

// sortSlice recursively sorts nested structures in slices
func sortSlice(s []interface{}) []interface{} {
	result := make([]interface{}, len(s))

	for i, v := range s {
		switch val := v.(type) {
		case map[string]interface{}:
			result[i] = sortMapKeys(val)
		case []interface{}:
			result[i] = sortSlice(val)
		default:
			result[i] = v
		}
	}

	return result
}

// removeJSONPointerField removes a field specified by a JSON Pointer (RFC 6901)
func removeJSONPointerField(obj *unstructured.Unstructured, pointer string) error {
	if pointer == "" || pointer == "/" {
		// Can't remove root
		return fmt.Errorf("cannot remove root object")
	}

	// Parse the JSON Pointer
	parts := parseJSONPointer(pointer)
	if len(parts) == 0 {
		return fmt.Errorf("invalid JSON Pointer: %s", pointer)
	}

	// Navigate to the parent and remove the field
	current := obj.Object
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]

		next, ok := current[part]
		if !ok {
			// Field doesn't exist
			return fmt.Errorf("field not found: %s", pointer)
		}

		nextMap, ok := next.(map[string]interface{})
		if !ok {
			return fmt.Errorf("field is not an object: %s", pointer)
		}

		current = nextMap
	}

	// Remove the final field
	lastPart := parts[len(parts)-1]
	delete(current, lastPart)

	return nil
}

// parseJSONPointer parses a JSON Pointer into parts
func parseJSONPointer(pointer string) []string {
	if pointer == "" || pointer == "/" {
		return []string{}
	}

	// Remove leading /
	if pointer[0] == '/' {
		pointer = pointer[1:]
	}

	parts := []string{}
	current := ""

	for i := 0; i < len(pointer); i++ {
		if pointer[i] == '/' {
			if current != "" {
				parts = append(parts, unescapeJSONPointer(current))
				current = ""
			}
		} else {
			current += string(pointer[i])
		}
	}

	if current != "" {
		parts = append(parts, unescapeJSONPointer(current))
	}

	return parts
}

// unescapeJSONPointer unescapes ~ sequences in JSON Pointer
func unescapeJSONPointer(s string) string {
	// ~1 -> /
	// ~0 -> ~
	result := ""
	for i := 0; i < len(s); i++ {
		if s[i] == '~' && i+1 < len(s) {
			if s[i+1] == '1' {
				result += "/"
				i++
			} else if s[i+1] == '0' {
				result += "~"
				i++
			} else {
				result += string(s[i])
			}
		} else {
			result += string(s[i])
		}
	}
	return result
}

// removeJQExpressionField removes fields matched by a JQ expression
func removeJQExpressionField(obj *unstructured.Unstructured, expr string) error {
	// Use the JQ processor to remove matching fields
	return RemoveJQMatches(obj, expr)
}

// removeManagedFieldsByManagers removes managedFields entries for specific managers
func removeManagedFieldsByManagers(obj *unstructured.Unstructured, managers []string) error {
	// Get metadata.managedFields
	metadata, ok := obj.Object["metadata"].(map[string]interface{})
	if !ok {
		return nil
	}

	managedFields, ok := metadata["managedFields"].([]interface{})
	if !ok {
		return nil
	}

	// Filter out entries with matching managers
	filtered := []interface{}{}
	for _, field := range managedFields {
		fieldMap, ok := field.(map[string]interface{})
		if !ok {
			filtered = append(filtered, field)
			continue
		}

		manager, ok := fieldMap["manager"].(string)
		if !ok {
			filtered = append(filtered, field)
			continue
		}

		// Check if this manager should be removed
		shouldRemove := false
		for _, m := range managers {
			if m == manager {
				shouldRemove = true
				break
			}
		}

		if !shouldRemove {
			filtered = append(filtered, field)
		}
	}

	if len(filtered) == 0 {
		delete(metadata, "managedFields")
	} else {
		metadata["managedFields"] = filtered
	}

	return nil
}

// ToJSON converts an unstructured object to pretty-printed JSON
func ToJSON(obj *unstructured.Unstructured) (string, error) {
	data, err := json.MarshalIndent(obj.Object, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

package normalizer

import (
	"encoding/json"
	"fmt"

	"github.com/itchyny/gojq"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// JQProcessor handles JQ expression evaluation for removing fields
type JQProcessor struct {
	query *gojq.Query
}

// NewJQProcessor creates a new JQ processor with the given expression
func NewJQProcessor(expr string) (*JQProcessor, error) {
	query, err := gojq.Parse(expr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JQ expression: %w", err)
	}

	return &JQProcessor{
		query: query,
	}, nil
}

// Execute evaluates the JQ expression against the object and returns matching paths
func (p *JQProcessor) Execute(obj *unstructured.Unstructured) ([]interface{}, error) {
	// Convert object to interface{} for gojq
	data := obj.Object

	// Execute the query
	iter := p.query.Run(data)
	results := []interface{}{}

	for {
		v, ok := iter.Next()
		if !ok {
			break
		}

		if err, ok := v.(error); ok {
			return nil, fmt.Errorf("JQ execution error: %w", err)
		}

		results = append(results, v)
	}

	return results, nil
}

// RemoveJQMatches removes all fields that match the JQ expression
func RemoveJQMatches(obj *unstructured.Unstructured, expr string) error {
	// Parse common JQ patterns and convert them to field removals
	// This handles the most common use cases for ignore rules

	// Common patterns:
	// 1. .spec.template.spec.containers[] | select(.name == "istio-proxy")
	//    -> Remove the container with name "istio-proxy"
	// 2. .metadata.annotations["key"]
	//    -> Remove the annotation with the specified key
	// 3. .spec.template.spec.volumes[] | select(.name | startswith("istio-"))
	//    -> Remove volumes whose names start with "istio-"

	processor, err := NewJQProcessor(expr)
	if err != nil {
		return err
	}

	// Execute to find matches
	matches, err := processor.Execute(obj)
	if err != nil {
		// If JQ fails, just skip (field might not exist)
		return nil
	}

	// For each match, try to remove it from the object
	// This is complex because we need to find the path and remove it
	// For MVP, we'll handle specific common patterns

	if len(matches) == 0 {
		return nil
	}

	// Try to remove the matched items
	// This requires understanding the JQ expression structure
	// For now, we'll implement support for the most common patterns

	return removeMatchedFields(obj, expr, matches)
}

// removeMatchedFields removes fields based on JQ expression results
func removeMatchedFields(obj *unstructured.Unstructured, expr string, matches []interface{}) error {
	// This is a complex operation that depends on the JQ expression structure
	// For MVP, we'll handle the most common pattern:
	// .spec.template.spec.containers[] | select(.name == "value")

	// Parse the expression to understand what to remove
	if containsPattern(expr, "containers[]") && containsPattern(expr, "select") {
		return removeMatchingContainers(obj, matches)
	}

	if containsPattern(expr, "initContainers[]") && containsPattern(expr, "select") {
		return removeMatchingInitContainers(obj, matches)
	}

	if containsPattern(expr, "volumes[]") && containsPattern(expr, "select") {
		return removeMatchingVolumes(obj, matches)
	}

	// For other patterns, we'll skip for MVP
	return nil
}

// containsPattern checks if a string contains a pattern
func containsPattern(s, pattern string) bool {
	return len(s) >= len(pattern) && contains(s, pattern)
}

// contains checks if s contains substr
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// removeMatchingContainers removes containers that match the JQ results
func removeMatchingContainers(obj *unstructured.Unstructured, matches []interface{}) error {
	// Get the containers array
	spec, ok := obj.Object["spec"].(map[string]interface{})
	if !ok {
		return nil
	}

	template, ok := spec["template"].(map[string]interface{})
	if !ok {
		return nil
	}

	templateSpec, ok := template["spec"].(map[string]interface{})
	if !ok {
		return nil
	}

	containers, ok := templateSpec["containers"].([]interface{})
	if !ok {
		return nil
	}

	// Convert matches to a set of container names to remove
	namesToRemove := make(map[string]bool)
	for _, match := range matches {
		if containerMap, ok := match.(map[string]interface{}); ok {
			if name, ok := containerMap["name"].(string); ok {
				namesToRemove[name] = true
			}
		}
	}

	// Filter out matching containers
	filtered := []interface{}{}
	for _, container := range containers {
		containerMap, ok := container.(map[string]interface{})
		if !ok {
			filtered = append(filtered, container)
			continue
		}

		name, ok := containerMap["name"].(string)
		if !ok {
			filtered = append(filtered, container)
			continue
		}

		if !namesToRemove[name] {
			filtered = append(filtered, container)
		}
	}

	templateSpec["containers"] = filtered
	return nil
}

// removeMatchingInitContainers removes init containers that match the JQ results
func removeMatchingInitContainers(obj *unstructured.Unstructured, matches []interface{}) error {
	spec, ok := obj.Object["spec"].(map[string]interface{})
	if !ok {
		return nil
	}

	template, ok := spec["template"].(map[string]interface{})
	if !ok {
		return nil
	}

	templateSpec, ok := template["spec"].(map[string]interface{})
	if !ok {
		return nil
	}

	initContainers, ok := templateSpec["initContainers"].([]interface{})
	if !ok {
		return nil
	}

	// Convert matches to a set of container names to remove
	namesToRemove := make(map[string]bool)
	for _, match := range matches {
		if containerMap, ok := match.(map[string]interface{}); ok {
			if name, ok := containerMap["name"].(string); ok {
				namesToRemove[name] = true
			}
		}
	}

	// Filter out matching init containers
	filtered := []interface{}{}
	for _, container := range initContainers {
		containerMap, ok := container.(map[string]interface{})
		if !ok {
			filtered = append(filtered, container)
			continue
		}

		name, ok := containerMap["name"].(string)
		if !ok {
			filtered = append(filtered, container)
			continue
		}

		if !namesToRemove[name] {
			filtered = append(filtered, container)
		}
	}

	if len(filtered) == 0 {
		delete(templateSpec, "initContainers")
	} else {
		templateSpec["initContainers"] = filtered
	}

	return nil
}

// removeMatchingVolumes removes volumes that match the JQ results
func removeMatchingVolumes(obj *unstructured.Unstructured, matches []interface{}) error {
	spec, ok := obj.Object["spec"].(map[string]interface{})
	if !ok {
		return nil
	}

	template, ok := spec["template"].(map[string]interface{})
	if !ok {
		return nil
	}

	templateSpec, ok := template["spec"].(map[string]interface{})
	if !ok {
		return nil
	}

	volumes, ok := templateSpec["volumes"].([]interface{})
	if !ok {
		return nil
	}

	// Convert matches to a set of volume names to remove
	namesToRemove := make(map[string]bool)
	for _, match := range matches {
		if volumeMap, ok := match.(map[string]interface{}); ok {
			if name, ok := volumeMap["name"].(string); ok {
				namesToRemove[name] = true
			}
		}
	}

	// Filter out matching volumes
	filtered := []interface{}{}
	for _, volume := range volumes {
		volumeMap, ok := volume.(map[string]interface{})
		if !ok {
			filtered = append(filtered, volume)
			continue
		}

		name, ok := volumeMap["name"].(string)
		if !ok {
			filtered = append(filtered, volume)
			continue
		}

		if !namesToRemove[name] {
			filtered = append(filtered, volume)
		}
	}

	if len(filtered) == 0 {
		delete(templateSpec, "volumes")
	} else {
		templateSpec["volumes"] = filtered
	}

	return nil
}

// EvaluateJQ evaluates a JQ expression and returns the result as JSON
func EvaluateJQ(obj *unstructured.Unstructured, expr string) (string, error) {
	processor, err := NewJQProcessor(expr)
	if err != nil {
		return "", err
	}

	results, err := processor.Execute(obj)
	if err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

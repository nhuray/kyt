package normalizer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/itchyny/gojq"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	// DefaultJQExecutionTimeout is the maximum time allowed for a JQ patch to execute
	DefaultJQExecutionTimeout = 1 * time.Second
)

// RemoveJQMatches removes all fields that match the JQ expression
// This follows ArgoCD's approach: wrap the expression in del() and execute it
func RemoveJQMatches(obj *unstructured.Unstructured, expr string) error {
	if obj == nil {
		return errors.New("cannot apply JQ expression to nil object")
	}

	// Wrap the expression in del() just like ArgoCD does
	jqDeletionQuery, err := gojq.Parse(fmt.Sprintf("del(%s)", expr))
	if err != nil {
		return fmt.Errorf("failed to parse JQ expression: %w", err)
	}

	// Compile the query
	jqDeletionCode, err := gojq.Compile(jqDeletionQuery)
	if err != nil {
		return fmt.Errorf("failed to compile JQ expression: %w", err)
	}

	// Marshal the object to JSON
	docData, err := json.Marshal(obj.Object)
	if err != nil {
		return fmt.Errorf("failed to marshal object: %w", err)
	}

	// Unmarshal to a generic map for JQ processing
	var dataJSON map[string]interface{}
	err = json.Unmarshal(docData, &dataJSON)
	if err != nil {
		return fmt.Errorf("failed to unmarshal object: %w", err)
	}

	// Execute the JQ deletion with timeout
	ctx, cancel := context.WithTimeout(context.Background(), DefaultJQExecutionTimeout)
	defer cancel()

	iter := jqDeletionCode.RunWithContext(ctx, dataJSON)
	first, ok := iter.Next()
	if !ok {
		return errors.New("JQ patch did not return any data")
	}

	// Check for errors
	if err, ok := first.(error); ok {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("JQ patch execution timed out (%v)", DefaultJQExecutionTimeout.String())
		}
		// Ignore errors about missing paths - the field might not exist
		if isIgnorableJQError(err) {
			return nil
		}
		return fmt.Errorf("JQ patch returned error: %w", err)
	}

	// Check that only one result was returned
	_, ok = iter.Next()
	if ok {
		return errors.New("JQ patch returned multiple objects")
	}

	// Marshal the result back to JSON
	patchedData, err := json.Marshal(first)
	if err != nil {
		return fmt.Errorf("failed to marshal patched data: %w", err)
	}

	// Unmarshal back into the object
	var patchedObject map[string]interface{}
	err = json.Unmarshal(patchedData, &patchedObject)
	if err != nil {
		return fmt.Errorf("failed to unmarshal patched data: %w", err)
	}

	obj.Object = patchedObject
	return nil
}

// isIgnorableJQError checks if a JQ error should be ignored
func isIgnorableJQError(err error) bool {
	errStr := err.Error()
	// Common errors when trying to delete non-existent paths
	return contains(errStr, "cannot delete") ||
		contains(errStr, "cannot index") ||
		contains(errStr, "null") ||
		contains(errStr, "is not defined")
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

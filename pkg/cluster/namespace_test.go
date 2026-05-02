package cluster

import (
	"testing"
)

func TestNamespaceNotFoundError(t *testing.T) {
	err := &NamespaceNotFoundError{
		Namespace: "nonexistent",
		Context:   "prod",
	}

	errorMsg := err.Error()

	// Should mention the namespace
	if errorMsg == "" {
		t.Fatal("Error message should not be empty")
	}

	// Basic check that it mentions the namespace and context
	expectedParts := []string{"nonexistent", "prod"}
	for _, part := range expectedParts {
		found := false
		for i := 0; i < len(errorMsg)-len(part)+1; i++ {
			if errorMsg[i:i+len(part)] == part {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Error message should contain %q, got: %s", part, errorMsg)
		}
	}
}

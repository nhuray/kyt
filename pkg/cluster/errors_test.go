package cluster

import (
	"strings"
	"testing"
)

func TestKubeconfigNotFoundError(t *testing.T) {
	err := &KubeconfigNotFoundError{
		Path: "/home/user/.kube/config",
	}

	errorMsg := err.Error()

	// Should mention the path
	if !strings.Contains(errorMsg, "/home/user/.kube/config") {
		t.Errorf("Error message should contain path, got: %s", errorMsg)
	}

	// Should contain troubleshooting hints
	if !strings.Contains(errorMsg, "Troubleshooting") {
		t.Errorf("Error message should contain troubleshooting section, got: %s", errorMsg)
	}

	if !strings.Contains(errorMsg, "kubectl") {
		t.Errorf("Error message should mention kubectl, got: %s", errorMsg)
	}
}

func TestContextNotFoundError(t *testing.T) {
	err := &ContextNotFoundError{
		Context:        "prod",
		KubeconfigPath: "/home/user/.kube/config",
	}

	errorMsg := err.Error()

	// Should mention the context name
	if !strings.Contains(errorMsg, "prod") {
		t.Errorf("Error message should contain context name, got: %s", errorMsg)
	}

	// Should mention the kubeconfig path
	if !strings.Contains(errorMsg, "/home/user/.kube/config") {
		t.Errorf("Error message should contain kubeconfig path, got: %s", errorMsg)
	}

	// Should contain troubleshooting hints
	if !strings.Contains(errorMsg, "kubectl config get-contexts") {
		t.Errorf("Error message should mention kubectl config get-contexts, got: %s", errorMsg)
	}
}

func TestNoContextError(t *testing.T) {
	err := &NoContextError{
		KubeconfigPath: "/home/user/.kube/config",
	}

	errorMsg := err.Error()

	// Should mention no context is set
	if !strings.Contains(errorMsg, "no current context") {
		t.Errorf("Error message should mention no current context, got: %s", errorMsg)
	}

	// Should suggest setting a context
	if !strings.Contains(errorMsg, "kubectl config use-context") {
		t.Errorf("Error message should suggest setting context, got: %s", errorMsg)
	}

	// Should suggest using --context flag
	if !strings.Contains(errorMsg, "--context flag") {
		t.Errorf("Error message should mention --context flag, got: %s", errorMsg)
	}
}

func TestConnectionError(t *testing.T) {
	originalErr := &testError{msg: "connection refused"}
	err := &ConnectionError{
		Context: "prod",
		Err:     originalErr,
	}

	errorMsg := err.Error()

	// Should mention the context
	if !strings.Contains(errorMsg, "prod") {
		t.Errorf("Error message should contain context name, got: %s", errorMsg)
	}

	// Should contain original error message
	if !strings.Contains(errorMsg, "connection refused") {
		t.Errorf("Error message should contain original error, got: %s", errorMsg)
	}

	// Should suggest kubectl cluster-info
	if !strings.Contains(errorMsg, "kubectl cluster-info") {
		t.Errorf("Error message should suggest kubectl cluster-info, got: %s", errorMsg)
	}

	// Test Unwrap
	if err.Unwrap() != originalErr {
		t.Errorf("Unwrap should return original error")
	}
}

func TestPermissionDeniedErrorWithNamespace(t *testing.T) {
	err := &PermissionDeniedError{
		Context:   "prod",
		Namespace: "default",
		Resource:  "pods",
	}

	errorMsg := err.Error()

	// Should mention all fields
	if !strings.Contains(errorMsg, "pods") {
		t.Errorf("Error message should contain resource name, got: %s", errorMsg)
	}
	if !strings.Contains(errorMsg, "default") {
		t.Errorf("Error message should contain namespace, got: %s", errorMsg)
	}
	if !strings.Contains(errorMsg, "prod") {
		t.Errorf("Error message should contain context, got: %s", errorMsg)
	}

	// Should suggest kubectl auth can-i
	if !strings.Contains(errorMsg, "kubectl auth can-i") {
		t.Errorf("Error message should suggest kubectl auth can-i, got: %s", errorMsg)
	}
}

func TestPermissionDeniedErrorWithoutNamespace(t *testing.T) {
	err := &PermissionDeniedError{
		Context:  "prod",
		Resource: "nodes",
	}

	errorMsg := err.Error()

	// Should mention resource and context
	if !strings.Contains(errorMsg, "nodes") {
		t.Errorf("Error message should contain resource name, got: %s", errorMsg)
	}
	if !strings.Contains(errorMsg, "prod") {
		t.Errorf("Error message should contain context, got: %s", errorMsg)
	}

	// Should suggest kubectl auth can-i
	if !strings.Contains(errorMsg, "kubectl auth can-i") {
		t.Errorf("Error message should suggest kubectl auth can-i, got: %s", errorMsg)
	}
}

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

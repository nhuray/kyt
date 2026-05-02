package cluster

import "fmt"

// KubeconfigNotFoundError is returned when the kubeconfig file cannot be found
type KubeconfigNotFoundError struct {
	Path string
}

func (e *KubeconfigNotFoundError) Error() string {
	return fmt.Sprintf("kubeconfig not found at %s\n\nTroubleshooting:\n- Check if KUBECONFIG environment variable is set correctly\n- Ensure kubectl is configured: run 'kubectl config view'\n- Default location is ~/.kube/config", e.Path)
}

// ContextNotFoundError is returned when a specified context doesn't exist
type ContextNotFoundError struct {
	Context        string
	KubeconfigPath string
}

func (e *ContextNotFoundError) Error() string {
	return fmt.Sprintf("context %q not found in kubeconfig\n\nTroubleshooting:\n- List available contexts: kubectl config get-contexts\n- Check kubeconfig file: %s\n- Ensure the context name is spelled correctly", e.Context, e.KubeconfigPath)
}

// NoContextError is returned when no current context is set
type NoContextError struct {
	KubeconfigPath string
}

func (e *NoContextError) Error() string {
	return fmt.Sprintf("no current context set in kubeconfig\n\nTroubleshooting:\n- Set a current context: kubectl config use-context <context-name>\n- Or specify a context with --context flag\n- List available contexts: kubectl config get-contexts")
}

// ConnectionError is returned when connection to the cluster fails
type ConnectionError struct {
	Context string
	Err     error
}

func (e *ConnectionError) Error() string {
	return fmt.Sprintf("failed to connect to cluster (context: %s): %v\n\nTroubleshooting:\n- Verify cluster is accessible: kubectl cluster-info --context %s\n- Check network connectivity and VPN status\n- Verify credentials are valid and not expired", e.Context, e.Err, e.Context)
}

func (e *ConnectionError) Unwrap() error {
	return e.Err
}

// PermissionDeniedError is returned when the user lacks RBAC permissions
type PermissionDeniedError struct {
	Context   string
	Namespace string
	Resource  string
}

func (e *PermissionDeniedError) Error() string {
	if e.Namespace != "" {
		return fmt.Sprintf("permission denied: cannot list %s in namespace %q (context: %s)\n\nTroubleshooting:\n- Verify your RBAC permissions: kubectl auth can-i list %s -n %s --context %s\n- Contact your cluster administrator for access", e.Resource, e.Namespace, e.Context, e.Resource, e.Namespace, e.Context)
	}
	return fmt.Sprintf("permission denied: cannot list %s (context: %s)\n\nTroubleshooting:\n- Verify your RBAC permissions: kubectl auth can-i list %s --context %s\n- Contact your cluster administrator for access", e.Resource, e.Context, e.Resource, e.Context)
}

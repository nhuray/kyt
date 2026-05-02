package cluster

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// CommonResourceTypes returns the resource types typically included in "kubectl get all"
// These are the most common workload and service resources in a namespace
func CommonResourceTypes() []schema.GroupVersionResource {
	return []schema.GroupVersionResource{
		// Core/v1 resources
		{Group: "", Version: "v1", Resource: "pods"},
		{Group: "", Version: "v1", Resource: "services"},
		{Group: "", Version: "v1", Resource: "replicationcontrollers"},
		{Group: "", Version: "v1", Resource: "configmaps"},
		{Group: "", Version: "v1", Resource: "secrets"},
		{Group: "", Version: "v1", Resource: "persistentvolumeclaims"},

		// apps/v1 resources
		{Group: "apps", Version: "v1", Resource: "deployments"},
		{Group: "apps", Version: "v1", Resource: "daemonsets"},
		{Group: "apps", Version: "v1", Resource: "replicasets"},
		{Group: "apps", Version: "v1", Resource: "statefulsets"},

		// batch/v1 resources
		{Group: "batch", Version: "v1", Resource: "jobs"},
		{Group: "batch", Version: "v1", Resource: "cronjobs"},

		// networking.k8s.io/v1 resources
		{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},

		// autoscaling/v1 resources
		{Group: "autoscaling", Version: "v1", Resource: "horizontalpodautoscalers"},
	}
}

package resourcekind

import (
	"testing"
)

func TestResolve(t *testing.T) {
	m := NewMatcher()

	tests := []struct {
		name   string
		input  string
		want   string
		wantOk bool
	}{
		// ConfigMap tests
		{"configmap plural", "configmaps", "ConfigMap", true},
		{"configmap singular", "configmap", "ConfigMap", true},
		{"configmap short", "cm", "ConfigMap", true},
		{"configmap kind", "ConfigMap", "ConfigMap", true},
		{"configmap case insensitive", "CONFIGMAP", "ConfigMap", true},

		// Deployment tests
		{"deployment plural", "deployments", "Deployment", true},
		{"deployment singular", "deployment", "Deployment", true},
		{"deployment short", "deploy", "Deployment", true},
		{"deployment kind", "Deployment", "Deployment", true},

		// Service tests
		{"service plural", "services", "Service", true},
		{"service singular", "service", "Service", true},
		{"service short", "svc", "Service", true},

		// Secret tests
		{"secret plural", "secrets", "Secret", true},
		{"secret singular", "secret", "Secret", true},

		// Pod tests
		{"pod plural", "pods", "Pod", true},
		{"pod singular", "pod", "Pod", true},
		{"pod short", "po", "Pod", true},

		// StatefulSet tests
		{"statefulset plural", "statefulsets", "StatefulSet", true},
		{"statefulset short", "sts", "StatefulSet", true},

		// DaemonSet tests
		{"daemonset short", "ds", "DaemonSet", true},

		// Ingress tests
		{"ingress plural", "ingresses", "Ingress", true},
		{"ingress short", "ing", "Ingress", true},

		// Not found
		{"unknown", "foobar", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := m.Resolve(tt.input)
			if ok != tt.wantOk {
				t.Errorf("Resolve(%q) ok = %v, want %v", tt.input, ok, tt.wantOk)
			}
			if got != tt.want {
				t.Errorf("Resolve(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMatch(t *testing.T) {
	m := NewMatcher()

	tests := []struct {
		name   string
		kind   string
		filter string
		want   bool
	}{
		// Exact matches
		{"exact kind match", "ConfigMap", "ConfigMap", true},
		{"exact kind match lowercase", "configmap", "configmap", true},

		// Short name matches
		{"kind vs short", "ConfigMap", "cm", true},
		{"short vs kind", "cm", "ConfigMap", true},
		{"short vs short", "cm", "cm", true},

		// Singular/plural matches
		{"kind vs plural", "ConfigMap", "configmaps", true},
		{"kind vs singular", "ConfigMap", "configmap", true},
		{"plural vs singular", "configmaps", "configmap", true},

		// Case insensitive
		{"case insensitive", "CONFIGMAP", "cm", true},

		// Deployment
		{"deployment short", "Deployment", "deploy", true},
		{"deployment plural", "Deployment", "deployments", true},

		// Service
		{"service short", "Service", "svc", true},

		// No match
		{"no match", "ConfigMap", "Secret", false},
		{"no match unknown", "ConfigMap", "foobar", false},

		// Custom resources (unknown types - direct comparison)
		{"custom resource exact", "MyCustomResource", "MyCustomResource", true},
		{"custom resource no match", "MyCustomResource", "OtherResource", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.Match(tt.kind, tt.filter)
			if got != tt.want {
				t.Errorf("Match(%q, %q) = %v, want %v", tt.kind, tt.filter, got, tt.want)
			}
		})
	}
}

func TestParseList(t *testing.T) {
	m := NewMatcher()

	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "empty",
			input: "",
			want:  nil,
		},
		{
			name:  "single short name",
			input: "cm",
			want:  []string{"ConfigMap"},
		},
		{
			name:  "multiple short names",
			input: "cm,svc,deploy",
			want:  []string{"ConfigMap", "Service", "Deployment"},
		},
		{
			name:  "mixed forms",
			input: "cm,services,deploy",
			want:  []string{"ConfigMap", "Service", "Deployment"},
		},
		{
			name:  "with spaces",
			input: "cm, svc, deploy",
			want:  []string{"ConfigMap", "Service", "Deployment"},
		},
		{
			name:  "singular and plural",
			input: "configmap,secrets,deployment",
			want:  []string{"ConfigMap", "Secret", "Deployment"},
		},
		{
			name:  "kind names",
			input: "ConfigMap,Service,Deployment",
			want:  []string{"ConfigMap", "Service", "Deployment"},
		},
		{
			name:  "with unknown type",
			input: "cm,UnknownType,svc",
			want:  []string{"ConfigMap", "UnknownType", "Service"},
		},
		{
			name:  "trailing comma",
			input: "cm,svc,",
			want:  []string{"ConfigMap", "Service"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.ParseList(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("ParseList(%q) length = %d, want %d", tt.input, len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ParseList(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestMatchesAny(t *testing.T) {
	m := NewMatcher()

	tests := []struct {
		name    string
		kind    string
		filters []string
		want    bool
	}{
		{
			name:    "matches first",
			kind:    "ConfigMap",
			filters: []string{"cm", "svc"},
			want:    true,
		},
		{
			name:    "matches second",
			kind:    "Service",
			filters: []string{"cm", "svc"},
			want:    true,
		},
		{
			name:    "no match",
			kind:    "Deployment",
			filters: []string{"cm", "svc"},
			want:    false,
		},
		{
			name:    "empty filters",
			kind:    "ConfigMap",
			filters: []string{},
			want:    false,
		},
		{
			name:    "matches with kind name",
			kind:    "ConfigMap",
			filters: []string{"ConfigMap", "Service"},
			want:    true,
		},
		{
			name:    "matches with mixed forms",
			kind:    "Deployment",
			filters: []string{"cm", "deploy", "svc"},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.MatchesAny(tt.kind, tt.filters)
			if got != tt.want {
				t.Errorf("MatchesAny(%q, %v) = %v, want %v", tt.kind, tt.filters, got, tt.want)
			}
		})
	}
}

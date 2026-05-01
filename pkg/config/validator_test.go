package config

import (
	"testing"
)

func TestValidatorValidateJSONPointer(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name        string
		pointer     string
		expectError bool
	}{
		{
			name:        "valid simple pointer",
			pointer:     "/metadata/labels",
			expectError: false,
		},
		{
			name:        "valid nested pointer",
			pointer:     "/spec/template/metadata/labels",
			expectError: false,
		},
		{
			name:        "valid with tilde escape for /",
			pointer:     "/metadata/annotations/kubectl.kubernetes.io~1last-applied-configuration",
			expectError: false,
		},
		{
			name:        "valid with tilde escape for ~",
			pointer:     "/metadata/labels/app~0version",
			expectError: false,
		},
		{
			name:        "valid root pointer",
			pointer:     "/",
			expectError: false,
		},
		{
			name:        "invalid - doesn't start with /",
			pointer:     "metadata/labels",
			expectError: true,
		},
		{
			name:        "invalid - wrong escape sequence",
			pointer:     "/metadata/labels~2",
			expectError: true,
		},
		{
			name:        "invalid - incomplete escape",
			pointer:     "/metadata/labels~",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateJSONPointer(tt.pointer)
			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestValidatorValidateJQExpression(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name        string
		expression  string
		expectError bool
	}{
		{
			name:        "valid simple path",
			expression:  ".metadata.labels",
			expectError: false,
		},
		{
			name:        "valid with array filter",
			expression:  ".spec.containers[] | select(.name == \"nginx\")",
			expectError: false,
		},
		{
			name:        "valid complex expression",
			expression:  ".spec.template.spec.containers[] | select(.name == \"istio-proxy\")",
			expectError: false,
		},
		{
			name:        "valid with multiple filters",
			expression:  ".items[] | select(.kind == \"Deployment\") | .metadata.name",
			expectError: false,
		},
		{
			name:        "empty expression",
			expression:  "",
			expectError: true,
		},
		{
			name:        "invalid syntax",
			expression:  ".metadata.labels[",
			expectError: true,
		},
		{
			name:        "invalid - unclosed string",
			expression:  ".metadata.labels | select(.app == \"nginx)",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateJQExpression(tt.expression)
			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestValidatorValidateIgnoreRule(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name        string
		rule        ResourceIgnoreDifferences
		expectError bool
	}{
		{
			name: "valid with JSON pointers",
			rule: ResourceIgnoreDifferences{
				Group: "apps",
				Kind:  "Deployment",
				JSONPointers: []string{
					"/spec/replicas",
				},
			},
			expectError: false,
		},
		{
			name: "valid with JQ expressions",
			rule: ResourceIgnoreDifferences{
				Group: "apps",
				Kind:  "Deployment",
				JQPathExpressions: []string{
					".spec.template.spec.containers[] | select(.name == \"nginx\")",
				},
			},
			expectError: false,
		},
		{
			name: "valid with managed fields managers",
			rule: ResourceIgnoreDifferences{
				Group: "apps",
				Kind:  "Deployment",
				ManagedFieldsManagers: []string{
					"kube-controller-manager",
				},
			},
			expectError: false,
		},
		{
			name: "invalid - missing kind",
			rule: ResourceIgnoreDifferences{
				Group: "apps",
				JSONPointers: []string{
					"/spec/replicas",
				},
			},
			expectError: true,
		},
		{
			name: "invalid - no ignore methods specified",
			rule: ResourceIgnoreDifferences{
				Group: "apps",
				Kind:  "Deployment",
			},
			expectError: true,
		},
		{
			name: "invalid - bad JSON pointer",
			rule: ResourceIgnoreDifferences{
				Group: "apps",
				Kind:  "Deployment",
				JSONPointers: []string{
					"spec/replicas", // missing leading /
				},
			},
			expectError: true,
		},
		{
			name: "invalid - bad JQ expression",
			rule: ResourceIgnoreDifferences{
				Group: "apps",
				Kind:  "Deployment",
				JQPathExpressions: []string{
					".spec.containers[", // unclosed bracket
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateIgnoreRule(&tt.rule, 0)
			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestValidatorValidateOutputConfig(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name        string
		output      OutputConfig
		expectError bool
	}{
		{
			name: "valid cli format",
			output: OutputConfig{
				Format: "cli",
			},
			expectError: false,
		},
		{
			name: "valid json format",
			output: OutputConfig{
				Format: "json",
			},
			expectError: false,
		},
		{
			name: "valid with context lines",
			output: OutputConfig{
				Format:       "diff",
				ContextLines: 5,
			},
			expectError: false,
		},
		{
			name: "invalid format",
			output: OutputConfig{
				Format: "invalid",
			},
			expectError: true,
		},
		{
			name: "invalid negative context lines",
			output: OutputConfig{
				Format:       "diff",
				ContextLines: -1,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateOutputConfig(&tt.output)
			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestValidatorValidateConfig(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name: "valid complete config",
			config: Config{
				IgnoreDifferences: []ResourceIgnoreDifferences{
					{
						Group: "apps",
						Kind:  "Deployment",
						JSONPointers: []string{
							"/spec/replicas",
						},
					},
				},
				Output: OutputConfig{
					Format: "cli",
				},
			},
			expectError: false,
		},
		{
			name: "valid with multiple ignore rules",
			config: Config{
				IgnoreDifferences: []ResourceIgnoreDifferences{
					{
						Group: "",
						Kind:  "Service",
						JSONPointers: []string{
							"/metadata/labels",
						},
					},
					{
						Group: "apps",
						Kind:  "Deployment",
						JQPathExpressions: []string{
							".spec.template.spec.containers[]",
						},
					},
				},
				Output: OutputConfig{
					Format: "json",
				},
			},
			expectError: false,
		},
		{
			name: "invalid - bad ignore rule",
			config: Config{
				IgnoreDifferences: []ResourceIgnoreDifferences{
					{
						Group: "apps",
						// Missing Kind
						JSONPointers: []string{
							"/spec/replicas",
						},
					},
				},
				Output: OutputConfig{
					Format: "cli",
				},
			},
			expectError: true,
		},
		{
			name: "invalid - bad output config",
			config: Config{
				IgnoreDifferences: []ResourceIgnoreDifferences{
					{
						Group: "apps",
						Kind:  "Deployment",
						JSONPointers: []string{
							"/spec/replicas",
						},
					},
				},
				Output: OutputConfig{
					Format: "invalid-format",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(&tt.config)
			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

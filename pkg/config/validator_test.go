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

func TestValidatorValidateOptionsConfig(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name        string
		options     OptionsConfig
		expectError bool
	}{
		{
			name: "valid options",
			options: OptionsConfig{
				ContextLines:        3,
				SimilarityThreshold: 0.7,
				DataSimilarityBoost: 2,
			},
			expectError: false,
		},
		{
			name: "negative context lines",
			options: OptionsConfig{
				ContextLines: -1,
			},
			expectError: true,
		},
		{
			name: "similarity threshold too low",
			options: OptionsConfig{
				SimilarityThreshold: -0.1,
			},
			expectError: true,
		},
		{
			name: "similarity threshold too high",
			options: OptionsConfig{
				SimilarityThreshold: 1.5,
			},
			expectError: true,
		},
		{
			name: "data similarity boost too high",
			options: OptionsConfig{
				DataSimilarityBoost: 11,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateOptionsConfig(&tt.options)
			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestValidatorValidateFuzzyMatchingConfig(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name        string
		fuzzy       FuzzyMatchingConfig
		expectError bool
	}{
		{
			name: "valid fuzzy matching",
			fuzzy: FuzzyMatchingConfig{
				Enabled:         true,
				MinStringLength: 100,
			},
			expectError: false,
		},
		{
			name: "negative min string length",
			fuzzy: FuzzyMatchingConfig{
				MinStringLength: -1,
			},
			expectError: true,
		},
		{
			name: "min string length too high",
			fuzzy: FuzzyMatchingConfig{
				MinStringLength: 20000,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateFuzzyMatchingConfig(&tt.fuzzy)
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
				Diff: DiffConfig{
					IgnoreDifferences: []ResourceIgnoreDifferences{
						{
							Group: "apps",
							Kind:  "Deployment",
							JSONPointers: []string{
								"/spec/replicas",
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid with multiple ignore rules",
			config: Config{
				Diff: DiffConfig{
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
				},
			},
			expectError: false,
		},
		{
			name: "invalid - bad ignore rule",
			config: Config{
				Diff: DiffConfig{
					IgnoreDifferences: []ResourceIgnoreDifferences{
						{
							Group: "apps",
							// Missing Kind
							JSONPointers: []string{
								"/spec/replicas",
							},
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "invalid - bad options config",
			config: Config{
				Diff: DiffConfig{
					IgnoreDifferences: []ResourceIgnoreDifferences{
						{
							Group: "apps",
							Kind:  "Deployment",
							JSONPointers: []string{
								"/spec/replicas",
							},
						},
					},
					Options: OptionsConfig{
						SimilarityThreshold: 1.5, // Invalid: > 1.0
					},
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

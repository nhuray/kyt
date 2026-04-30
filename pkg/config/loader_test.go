package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()

	// Check defaults
	if !cfg.Normalization.SortKeys {
		t.Error("Expected sortKeys to be true by default")
	}

	if cfg.Output.Format != "cli" {
		t.Errorf("Expected default format to be 'cli', got %s", cfg.Output.Format)
	}

	if cfg.Output.DiffTool != "auto" {
		t.Errorf("Expected default diffTool to be 'auto', got %s", cfg.Output.DiffTool)
	}

	if !cfg.Output.Colorize {
		t.Error("Expected colorize to be true by default")
	}
}

func TestConfigMerge(t *testing.T) {
	cfg1 := &Config{
		IgnoreDifferences: []ResourceIgnoreDifferences{
			{
				Group: "",
				Kind:  "Service",
				JSONPointers: []string{
					"/metadata/labels",
				},
			},
		},
		Output: OutputConfig{
			Format:   "cli",
			DiffTool: "difft",
		},
	}

	cfg2 := &Config{
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
			Format:   "json",
			Colorize: true,
		},
	}

	cfg1.Merge(cfg2)

	// Check that ignore rules were appended
	if len(cfg1.IgnoreDifferences) != 2 {
		t.Errorf("Expected 2 ignore rules after merge, got %d", len(cfg1.IgnoreDifferences))
	}

	// Check that output config was overridden
	if cfg1.Output.Format != "json" {
		t.Errorf("Expected format to be 'json' after merge, got %s", cfg1.Output.Format)
	}

	if !cfg1.Output.Colorize {
		t.Error("Expected colorize to be true after merge")
	}
}

func TestResourceIgnoreDifferencesMatchesResource(t *testing.T) {
	tests := []struct {
		name      string
		rule      ResourceIgnoreDifferences
		group     string
		kind      string
		namespace string
		resName   string
		expected  bool
	}{
		{
			name: "exact match",
			rule: ResourceIgnoreDifferences{
				Group:     "apps",
				Kind:      "Deployment",
				Namespace: "default",
				Name:      "my-app",
			},
			group:     "apps",
			kind:      "Deployment",
			namespace: "default",
			resName:   "my-app",
			expected:  true,
		},
		{
			name: "wildcard kind",
			rule: ResourceIgnoreDifferences{
				Group: "",
				Kind:  "*",
			},
			group:     "",
			kind:      "Service",
			namespace: "default",
			resName:   "my-service",
			expected:  true,
		},
		{
			name: "empty namespace matches all",
			rule: ResourceIgnoreDifferences{
				Group: "apps",
				Kind:  "Deployment",
			},
			group:     "apps",
			kind:      "Deployment",
			namespace: "production",
			resName:   "my-app",
			expected:  true,
		},
		{
			name: "empty name matches all",
			rule: ResourceIgnoreDifferences{
				Group:     "apps",
				Kind:      "Deployment",
				Namespace: "default",
			},
			group:     "apps",
			kind:      "Deployment",
			namespace: "default",
			resName:   "any-deployment",
			expected:  true,
		},
		{
			name: "group mismatch",
			rule: ResourceIgnoreDifferences{
				Group: "apps",
				Kind:  "Deployment",
			},
			group:     "batch",
			kind:      "Deployment",
			namespace: "default",
			resName:   "my-app",
			expected:  false,
		},
		{
			name: "kind mismatch",
			rule: ResourceIgnoreDifferences{
				Group: "apps",
				Kind:  "Deployment",
			},
			group:     "apps",
			kind:      "StatefulSet",
			namespace: "default",
			resName:   "my-app",
			expected:  false,
		},
		{
			name: "name mismatch",
			rule: ResourceIgnoreDifferences{
				Group: "apps",
				Kind:  "Deployment",
				Name:  "specific-app",
			},
			group:     "apps",
			kind:      "Deployment",
			namespace: "default",
			resName:   "other-app",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.rule.MatchesResource(tt.group, tt.kind, tt.namespace, tt.resName)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestLoaderLoadBytes(t *testing.T) {
	yaml := `
ignoreDifferences:
  - group: "apps"
    kind: "Deployment"
    jsonPointers:
      - /spec/replicas

output:
  format: json
  diffTool: diff
`

	loader := NewLoader()
	cfg, err := loader.LoadBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if len(cfg.IgnoreDifferences) != 1 {
		t.Errorf("Expected 1 ignore rule, got %d", len(cfg.IgnoreDifferences))
	}

	if cfg.IgnoreDifferences[0].Group != "apps" {
		t.Errorf("Expected group 'apps', got %s", cfg.IgnoreDifferences[0].Group)
	}

	if cfg.Output.Format != "json" {
		t.Errorf("Expected format 'json', got %s", cfg.Output.Format)
	}
}

func TestLoaderLoadFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".k8s-diff.yaml")

	content := `
ignoreDifferences:
  - group: ""
    kind: "Service"
    jsonPointers:
      - /metadata/labels
`

	err := os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	loader := NewLoader()
	cfg, err := loader.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	if len(cfg.IgnoreDifferences) != 1 {
		t.Errorf("Expected 1 ignore rule, got %d", len(cfg.IgnoreDifferences))
	}
}

func TestLoaderLoadDefault(t *testing.T) {
	tmpDir := t.TempDir()

	// Test with no config file (should return defaults)
	loader := NewLoader()
	cfg, err := loader.LoadDefaultFromDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load default config: %v", err)
	}

	if cfg.Output.Format != "cli" {
		t.Errorf("Expected default format 'cli', got %s", cfg.Output.Format)
	}

	// Test with existing config file
	configPath := filepath.Join(tmpDir, DefaultConfigFileName)
	content := `
output:
  format: json
`
	err = os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cfg, err = loader.LoadDefaultFromDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load default config: %v", err)
	}

	if cfg.Output.Format != "json" {
		t.Errorf("Expected format 'json', got %s", cfg.Output.Format)
	}
}

func TestLoaderLoadMultiple(t *testing.T) {
	tmpDir := t.TempDir()

	config1Path := filepath.Join(tmpDir, "config1.yaml")
	config1 := `
ignoreDifferences:
  - group: ""
    kind: "Service"
    jsonPointers:
      - /metadata/labels
`

	config2Path := filepath.Join(tmpDir, "config2.yaml")
	config2 := `
ignoreDifferences:
  - group: "apps"
    kind: "Deployment"
    jsonPointers:
      - /spec/replicas
`

	err := os.WriteFile(config1Path, []byte(config1), 0644)
	if err != nil {
		t.Fatalf("Failed to create config1: %v", err)
	}

	err = os.WriteFile(config2Path, []byte(config2), 0644)
	if err != nil {
		t.Fatalf("Failed to create config2: %v", err)
	}

	loader := NewLoader()
	cfg, err := loader.LoadMultiple([]string{config1Path, config2Path})
	if err != nil {
		t.Fatalf("Failed to load multiple configs: %v", err)
	}

	// Should have both ignore rules
	if len(cfg.IgnoreDifferences) != 2 {
		t.Errorf("Expected 2 ignore rules, got %d", len(cfg.IgnoreDifferences))
	}
}

func TestLoaderSearchConfig(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	// Create config in parent directory
	configPath := filepath.Join(tmpDir, DefaultConfigFileName)
	content := `
output:
  format: json
`
	err = os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	loader := NewLoader()
	cfg, foundPath, err := loader.SearchConfig(subDir)
	if err != nil {
		t.Fatalf("Failed to search config: %v", err)
	}

	if foundPath != configPath {
		t.Errorf("Expected to find config at %s, found at %s", configPath, foundPath)
	}

	if cfg.Output.Format != "json" {
		t.Errorf("Expected format 'json', got %s", cfg.Output.Format)
	}
}

func TestLoaderSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	cfg := &Config{
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
			Format:   "json",
			DiffTool: "diff",
		},
	}

	loader := NewLoader()
	err := loader.Save(cfg, configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Try to load it back
	loadedCfg, err := loader.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if loadedCfg.Output.Format != "json" {
		t.Errorf("Expected format 'json', got %s", loadedCfg.Output.Format)
	}
}

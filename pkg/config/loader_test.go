package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()

	// Check defaults
	if !cfg.Diff.Normalization.SortKeys {
		t.Error("Expected sortKeys to be true by default")
	}

	if cfg.Diff.CLI.Display != "side-by-side" {
		t.Errorf("Expected default display to be 'side-by-side', got %s", cfg.Diff.CLI.Display)
	}

	if !cfg.Diff.CLI.Colorize {
		t.Error("Expected colorize to be true by default")
	}
}

func TestConfigMerge(t *testing.T) {
	cfg1 := &Config{
		Diff: DiffConfig{
			IgnoreDifferences: []ResourceIgnoreDifferences{
				{
					Group: "",
					Kind:  "Service",
					JSONPointers: []string{
						"/metadata/labels",
					},
				},
			},
			CLI: CLIConfig{
				Display: "side-by-side",
			},
		},
	}

	cfg2 := &Config{
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
			CLI: CLIConfig{
				Display:  "inline",
				Colorize: true,
			},
		},
	}

	cfg1.Merge(cfg2)

	// Check that ignore rules were appended
	if len(cfg1.Diff.IgnoreDifferences) != 2 {
		t.Errorf("Expected 2 ignore rules after merge, got %d", len(cfg1.Diff.IgnoreDifferences))
	}

	// Check that CLI config was overridden
	if cfg1.Diff.CLI.Display != "inline" {
		t.Errorf("Expected display to be 'inline' after merge, got %s", cfg1.Diff.CLI.Display)
	}

	if !cfg1.Diff.CLI.Colorize {
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
diff:
  diff:
  ignoreDifferences:
    - group: "apps"
      kind: "Deployment"
      jsonPointers:
        - /spec/replicas

  cli:
    display: inline
`

	loader := NewLoader()
	cfg, err := loader.LoadBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if len(cfg.Diff.IgnoreDifferences) != 1 {
		t.Errorf("Expected 1 ignore rule, got %d", len(cfg.Diff.IgnoreDifferences))
	}

	if cfg.Diff.IgnoreDifferences[0].Group != "apps" {
		t.Errorf("Expected group 'apps', got %s", cfg.Diff.IgnoreDifferences[0].Group)
	}

	if cfg.Diff.CLI.Display != "inline" {
		t.Errorf("Expected display 'inline', got %s", cfg.Diff.CLI.Display)
	}
}

func TestLoaderLoadFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".kyt.yaml")

	content := `
diff:
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

	if len(cfg.Diff.IgnoreDifferences) != 1 {
		t.Errorf("Expected 1 ignore rule, got %d", len(cfg.Diff.IgnoreDifferences))
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

	if cfg.Diff.CLI.Display != "side-by-side" {
		t.Errorf("Expected default display 'side-by-side', got %s", cfg.Diff.CLI.Display)
	}

	// Test with existing config file
	configPath := filepath.Join(tmpDir, DefaultConfigFileName)
	content := `
diff:
  cli:
    display: inline
`
	err = os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cfg, err = loader.LoadDefaultFromDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load default config: %v", err)
	}

	if cfg.Diff.CLI.Display != "inline" {
		t.Errorf("Expected display 'inline', got %s", cfg.Diff.CLI.Display)
	}
}

func TestLoaderLoadMultiple(t *testing.T) {
	tmpDir := t.TempDir()

	config1Path := filepath.Join(tmpDir, "config1.yaml")
	config1 := `
diff:
  ignoreDifferences:
  - group: ""
    kind: "Service"
    jsonPointers:
      - /metadata/labels
`

	config2Path := filepath.Join(tmpDir, "config2.yaml")
	config2 := `
diff:
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
	if len(cfg.Diff.IgnoreDifferences) != 2 {
		t.Errorf("Expected 2 ignore rules, got %d", len(cfg.Diff.IgnoreDifferences))
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
diff:
  cli:
    display: inline
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

	if cfg.Diff.CLI.Display != "inline" {
		t.Errorf("Expected display 'inline', got %s", cfg.Diff.CLI.Display)
	}
}

func TestLoaderSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	cfg := &Config{
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
			CLI: CLIConfig{
				Display: "inline",
			},
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

	if loadedCfg.Diff.CLI.Display != "inline" {
		t.Errorf("Expected display 'inline', got %s", loadedCfg.Diff.CLI.Display)
	}
}

package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultConfigFileName is the default config file name
	DefaultConfigFileName = ".kyt.yaml"
	// LegacyConfigFileName is the legacy config file name (for backward compatibility)
	LegacyConfigFileName = ".k8s-diff.yaml"
)

// Loader handles loading and validating configuration
type Loader struct {
	// Validator validates configuration
	Validator *Validator
}

// NewLoader creates a new config loader
func NewLoader() *Loader {
	return &Loader{
		Validator: NewValidator(),
	}
}

// Load loads configuration from a file path
func (l *Loader) Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	return l.LoadBytes(data)
}

// LoadBytes loads configuration from a byte slice
func (l *Loader) LoadBytes(data []byte) (*Config, error) {
	var cfg Config

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	// Validate the configuration
	if err := l.Validator.Validate(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// LoadDefault attempts to load the default config file from the current directory
// Returns a default config if the file doesn't exist
func (l *Loader) LoadDefault() (*Config, error) {
	return l.LoadDefaultFromDir(".")
}

// LoadDefaultFromDir attempts to load the default config file from a specific directory
// Returns a default config if the file doesn't exist
func (l *Loader) LoadDefaultFromDir(dir string) (*Config, error) {
	configPath := filepath.Join(dir, DefaultConfigFileName)

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return default config if file doesn't exist
		return NewDefaultConfig(), nil
	}

	return l.Load(configPath)
}

// LoadMultiple loads and merges multiple config files
// Later configs override earlier ones
func (l *Loader) LoadMultiple(paths []string) (*Config, error) {
	if len(paths) == 0 {
		return NewDefaultConfig(), nil
	}

	// Load first config
	cfg, err := l.Load(paths[0])
	if err != nil {
		return nil, fmt.Errorf("failed to load config %s: %w", paths[0], err)
	}

	// Merge additional configs
	for i := 1; i < len(paths); i++ {
		additional, err := l.Load(paths[i])
		if err != nil {
			return nil, fmt.Errorf("failed to load config %s: %w", paths[i], err)
		}
		cfg.Merge(additional)
	}

	return cfg, nil
}

// LoadWithDefaults loads a config file and merges it with default config
// User config overrides defaults
func (l *Loader) LoadWithDefaults(path string) (*Config, error) {
	defaultCfg := NewDefaultConfig()

	// If path is empty, return defaults
	if path == "" {
		return defaultCfg, nil
	}

	userCfg, err := l.Load(path)
	if err != nil {
		return nil, err
	}

	defaultCfg.Merge(userCfg)
	return defaultCfg, nil
}

// SearchConfig searches for a config file in the current directory and parent directories
// This mimics behavior of tools like git, searching upwards for config files
// Looks for .kyt.yaml first, then falls back to .k8s-diff.yaml for backward compatibility
func (l *Loader) SearchConfig(startDir string) (*Config, string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	for {
		// Try new config file name first
		configPath := filepath.Join(dir, DefaultConfigFileName)
		if _, err := os.Stat(configPath); err == nil {
			// Found config file
			cfg, err := l.Load(configPath)
			if err != nil {
				return nil, "", err
			}
			return cfg, configPath, nil
		}

		// Try legacy config file name for backward compatibility
		legacyConfigPath := filepath.Join(dir, LegacyConfigFileName)
		if _, err := os.Stat(legacyConfigPath); err == nil {
			// Found legacy config file
			cfg, err := l.Load(legacyConfigPath)
			if err != nil {
				return nil, "", err
			}
			// Note: We could add a deprecation warning here if desired
			return cfg, legacyConfigPath, nil
		}

		// Move to parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root directory, no config found
			break
		}
		dir = parent
	}

	// No config found, return defaults
	return NewDefaultConfig(), "", nil
}

// Save writes a config to a file
func (l *Loader) Save(cfg *Config, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

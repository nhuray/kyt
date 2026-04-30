package main

import (
	"fmt"
	"os"

	"github.com/nhuray/k8s-diff/pkg/config"
	"github.com/spf13/cobra"
)

var (
	// Version information (set by build flags)
	version = "dev"
	commit  = "none"
	date    = "unknown"

	// Global flags
	rootConfigFile string
	rootVerbose    bool
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		// Check if this is an exitError with a specific code
		if exitErr, ok := err.(*exitError); ok {
			os.Exit(exitErr.code)
		}
		// Print error to stderr and exit with code 2
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}
}

var rootCmd = &cobra.Command{
	Use:   "ky",
	Short: "ky - Kubernetes YAML toolkit",
	Long: `ky normalizes and compares Kubernetes manifests.

A powerful toolkit for working with Kubernetes YAML files:
- Normalize manifests with configurable ignore rules
- Compare manifests with smart similarity matching
- Lint and format YAML files
- Pipe-friendly for use with kubectl, kustomize, helm

Use 'ky <command> --help' for more information about a command.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&rootConfigFile, "config", "c", "", "config file (default: .ky.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&rootVerbose, "verbose", "v", false, "verbose output to stderr")

	// Override cobra's error handling to use our exit codes
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		cmd.Println(err)
		cmd.Println(cmd.UsageString())
		return &exitError{code: 2}
	})
}

// loadConfig loads the configuration file
func loadConfig() (*config.Config, error) {
	loader := config.NewLoader()

	if rootConfigFile != "" {
		// Load specified config file
		return loader.Load(rootConfigFile)
	}

	// Search for config file in current directory and parents
	cfg, foundPath, err := loader.SearchConfig(".")
	if err != nil {
		return nil, err
	}

	if foundPath != "" && rootVerbose {
		fmt.Fprintf(os.Stderr, "Using config: %s\n", foundPath)
	}

	return cfg, nil
}

// exitError is a custom error that includes an exit code
type exitError struct {
	code int
}

func (e *exitError) Error() string {
	return fmt.Sprintf("exit code %d", e.code)
}

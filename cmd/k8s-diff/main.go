package main

import (
	"fmt"
	"os"

	"github.com/nicolasleigh/k8s-diff/pkg/config"
	"github.com/nicolasleigh/k8s-diff/pkg/differ"
	"github.com/nicolasleigh/k8s-diff/pkg/manifest"
	"github.com/nicolasleigh/k8s-diff/pkg/normalizer"
	"github.com/nicolasleigh/k8s-diff/pkg/reporter"
	"github.com/spf13/cobra"
)

var (
	// Version information (set by build flags)
	version = "dev"
	commit  = "none"
	date    = "unknown"

	// Command-line flags
	configFile     string
	outputFormat   string
	noColor        bool
	showIdentical  bool
	difftasticMode string
	diffTool       string
	skipNormalize  bool
	verbose        bool
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(2)
	}
}

var rootCmd = &cobra.Command{
	Use:   "k8s-diff <source> <target>",
	Short: "Compare Kubernetes manifests with ArgoCD-compatible ignore rules",
	Long: `k8s-diff compares Kubernetes manifests with configurable ignore rules.

It supports:
- JSON Pointer ignore rules (RFC 6901)
- JQ path expression ignore rules
- Difftastic or unified diff output
- Multiple output formats (CLI, JSON)

Examples:
  # Compare two directories
  k8s-diff ./source-manifests ./target-manifests

  # Compare with custom config
  k8s-diff -c .k8s-diff.yaml ./source ./target

  # Output as JSON
  k8s-diff -o json ./source ./target

  # Compare Helm vs Kustomize
  helm template my-chart > /tmp/helm.yaml
  kustomize build ./overlay > /tmp/kustomize.yaml
  k8s-diff /tmp/helm.yaml /tmp/kustomize.yaml
`,
	Args:          cobra.ExactArgs(2),
	RunE:          runDiff,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "config file path (default: search for .k8s-diff.yaml)")
	rootCmd.Flags().StringVarP(&outputFormat, "output", "o", "cli", "output format: cli, json")
	rootCmd.Flags().BoolVar(&noColor, "no-color", false, "disable colored output")
	rootCmd.Flags().BoolVar(&showIdentical, "show-identical", false, "show identical resources in output")
	rootCmd.Flags().StringVar(&difftasticMode, "display", "side-by-side", "difftastic display mode: side-by-side, inline")
	rootCmd.Flags().StringVar(&diffTool, "diff-tool", "difft", "diff tool: difft, diff")
	rootCmd.Flags().BoolVar(&skipNormalize, "skip-normalize", false, "skip normalization (use raw manifests)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Add version command
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("k8s-diff %s\n", version)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  date:   %s\n", date)
	},
}

func runDiff(cmd *cobra.Command, args []string) error {
	sourcePath := args[0]
	targetPath := args[1]

	if verbose {
		fmt.Fprintf(os.Stderr, "Comparing:\n")
		fmt.Fprintf(os.Stderr, "  Source: %s\n", sourcePath)
		fmt.Fprintf(os.Stderr, "  Target: %s\n", targetPath)
	}

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if verbose && configFile != "" {
		fmt.Fprintf(os.Stderr, "  Config: %s\n", configFile)
	}

	// Parse source manifests
	if verbose {
		fmt.Fprintf(os.Stderr, "\nParsing source manifests...\n")
	}
	sourceManifests, err := parseManifests(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to parse source manifests: %w", err)
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "  Found %d resources in source\n", sourceManifests.Len())
	}

	// Parse target manifests
	if verbose {
		fmt.Fprintf(os.Stderr, "Parsing target manifests...\n")
	}
	targetManifests, err := parseManifests(targetPath)
	if err != nil {
		return fmt.Errorf("failed to parse target manifests: %w", err)
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "  Found %d resources in target\n", targetManifests.Len())
	}

	// Create normalizer
	norm := normalizer.New(cfg)

	// Create differ
	diffOpts := &differ.DiffOptions{
		UseDifftastic:     diffTool == "difft",
		ColorOutput:       !noColor,
		ContextLines:      3,
		DifftasticDisplay: difftasticMode,
	}
	diff := differ.New(norm, diffOpts)

	// Perform diff
	if verbose {
		fmt.Fprintf(os.Stderr, "\nComparing manifests...\n")
	}
	result, err := diff.Diff(sourceManifests, targetManifests)
	if err != nil {
		return fmt.Errorf("failed to diff manifests: %w", err)
	}

	// Create reporter
	reporterOpts := &reporter.Options{
		Format:        outputFormat,
		Colorize:      !noColor,
		ShowIdentical: showIdentical,
		Compact:       false,
	}

	var rep reporter.Reporter
	switch outputFormat {
	case "json":
		rep = reporter.NewJSONReporter(reporterOpts)
	case "cli":
		rep = reporter.NewCLIReporter(reporterOpts)
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}

	// Generate output
	if verbose {
		fmt.Fprintf(os.Stderr, "Generating output...\n\n")
	}
	if err := rep.Report(result, os.Stdout); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	// Exit with appropriate code
	if result.HasDifferences() {
		return &exitError{code: 1}
	}

	return nil
}

// loadConfig loads the configuration file
func loadConfig() (*config.Config, error) {
	loader := config.NewLoader()

	if configFile != "" {
		// Load specified config file
		return loader.Load(configFile)
	}

	// Search for config file in current directory and parents
	cfg, foundPath, err := loader.SearchConfig(".")
	if err != nil {
		return nil, err
	}

	if foundPath != "" && verbose {
		fmt.Fprintf(os.Stderr, "  Found config: %s\n", foundPath)
	}

	return cfg, nil
}

// parseManifests parses manifests from a file or directory
func parseManifests(path string) (*manifest.ManifestSet, error) {
	parser := manifest.NewParser()

	// Check if path is a file or directory
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	if info.IsDir() {
		return parser.ParseDirectory(path)
	}

	return parser.ParseFile(path)
}

// exitError is a custom error that includes an exit code
type exitError struct {
	code int
}

func (e *exitError) Error() string {
	return fmt.Sprintf("exit code %d", e.code)
}

func init() {
	// Override cobra's error handling to use our exit codes
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		cmd.Println(err)
		cmd.Println(cmd.UsageString())
		return &exitError{code: 2}
	})

	// Handle exit errors
	cobra.OnFinalize(func() {
		if r := recover(); r != nil {
			if e, ok := r.(*exitError); ok {
				os.Exit(e.code)
			}
			panic(r)
		}
	})
}

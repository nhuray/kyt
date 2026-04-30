package main

import (
	"fmt"
	"os"

	"github.com/nhuray/kyt/pkg/differ"
	"github.com/nhuray/kyt/pkg/manifest"
	"github.com/nhuray/kyt/pkg/normalizer"
	"github.com/nhuray/kyt/pkg/reporter"
	"github.com/spf13/cobra"
)

var (
	// Diff command flags
	diffOutputFormat              string
	diffNoColor                   bool
	diffShowIdentical             bool
	diffDifftasticMode            string
	diffDiffTool                  string
	diffSkipNormalize             bool
	diffExactMatch                bool
	diffSimilarityThreshold       float64
	diffWidth                     int
	diffStringSimilarityThreshold int
)

var diffCmd = &cobra.Command{
	Use:   "diff <source> <target>",
	Short: "Compare Kubernetes manifests",
	Long: `Compare Kubernetes manifests with configurable ignore rules.

Supports:
- JSON Pointer ignore rules (RFC 6901)
- JQ path expression ignore rules
- Multiple diff tools: difftastic, tree-sitter, unified diff
- Multiple output formats (CLI, JSON)
- Smart similarity matching for renamed resources

Diff Tools:
- auto: Try difftastic first, fall back to tree-sitter, then unified (default)
- difft: Use difftastic only (external tool, best quality)
- treesitter: Use Go-native tree-sitter (built-in, good quality)
- diff: Use standard unified diff (built-in, basic quality)

Examples:
  # Compare two directories (auto mode - tries all diff tools)
  kyt diff ./source-manifests ./target-manifests

  # Force tree-sitter diff (no external dependencies)
  kyt diff --diff-tool treesitter ./source ./target

  # Compare with custom config
  kyt diff -c .kyt.yaml ./source ./target

  # Output as JSON
  kyt diff -o json ./source ./target

  # Compare Helm vs Kustomize
  helm template my-chart > /tmp/helm.yaml
  kustomize build ./overlay > /tmp/kustomize.yaml
  kyt diff /tmp/helm.yaml /tmp/kustomize.yaml

  # Disable similarity matching (exact name match only)
  kyt diff --exact-match ./source ./target
`,
	Args:          cobra.ExactArgs(2),
	RunE:          runDiff,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	diffCmd.Flags().StringVarP(&diffOutputFormat, "output", "o", "cli", "output format: cli, json")
	diffCmd.Flags().BoolVar(&diffNoColor, "no-color", false, "disable colored output")
	diffCmd.Flags().BoolVar(&diffShowIdentical, "show-identical", false, "show identical resources in output")
	diffCmd.Flags().StringVar(&diffDifftasticMode, "display", "side-by-side", "difftastic display mode: side-by-side, inline")
	diffCmd.Flags().StringVar(&diffDiffTool, "diff-tool", "auto", "diff tool: auto (try all), difft (difftastic only), treesitter (Go native), diff (unified)")
	diffCmd.Flags().IntVar(&diffWidth, "width", 0, "terminal width for diff output (0 = auto-detect)")
	diffCmd.Flags().BoolVar(&diffSkipNormalize, "skip-normalize", false, "skip normalization (use raw manifests)")
	diffCmd.Flags().BoolVar(&diffExactMatch, "exact-match", false, "disable similarity matching (only exact name matches)")
	diffCmd.Flags().Float64Var(&diffSimilarityThreshold, "similarity-threshold", 0.7, "minimum similarity score (0.0-1.0) for matching resources")
	diffCmd.Flags().IntVar(&diffStringSimilarityThreshold, "string-similarity-threshold", 100, "minimum string length for fuzzy matching (0 = disable)")

	rootCmd.AddCommand(diffCmd)
}

func runDiff(cmd *cobra.Command, args []string) error {
	sourcePath := args[0]
	targetPath := args[1]

	if rootVerbose {
		fmt.Fprintf(os.Stderr, "Comparing:\n")
		fmt.Fprintf(os.Stderr, "  Source: %s\n", sourcePath)
		fmt.Fprintf(os.Stderr, "  Target: %s\n", targetPath)
	}

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if rootVerbose && rootConfigFile != "" {
		fmt.Fprintf(os.Stderr, "  Config: %s\n", rootConfigFile)
	}

	// Parse source manifests
	if rootVerbose {
		fmt.Fprintf(os.Stderr, "\nParsing source manifests...\n")
	}
	sourceManifests, err := parseManifests(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to parse source manifests: %w", err)
	}
	if rootVerbose {
		fmt.Fprintf(os.Stderr, "  Found %d resources in source\n", sourceManifests.Len())
	}

	// Parse target manifests
	if rootVerbose {
		fmt.Fprintf(os.Stderr, "Parsing target manifests...\n")
	}
	targetManifests, err := parseManifests(targetPath)
	if err != nil {
		return fmt.Errorf("failed to parse target manifests: %w", err)
	}
	if rootVerbose {
		fmt.Fprintf(os.Stderr, "  Found %d resources in target\n", targetManifests.Len())
	}

	// Create normalizer
	norm := normalizer.New(cfg)

	// Determine string similarity threshold (flag takes precedence over config)
	stringSimilarityThreshold := diffStringSimilarityThreshold
	if stringSimilarityThreshold == 100 && cfg.Output.StringSimilarityThreshold > 0 {
		// If flag is at default value, use config value
		stringSimilarityThreshold = cfg.Output.StringSimilarityThreshold
	}

	// Create differ
	diffOpts := &differ.DiffOptions{
		UseDifftastic:             diffDiffTool == "auto" || diffDiffTool == "difft",
		UseTreeSitter:             diffDiffTool == "auto" || diffDiffTool == "treesitter",
		ColorOutput:               !diffNoColor,
		ContextLines:              3,
		DifftasticDisplay:         diffDifftasticMode,
		DifftasticWidth:           diffWidth,
		TreeSitterWidth:           120,
		EnableSimilarityMatching:  !diffExactMatch,
		SimilarityThreshold:       diffSimilarityThreshold,
		StringSimilarityThreshold: stringSimilarityThreshold,
	}
	// If "diff" is explicitly specified, disable both difftastic and tree-sitter
	if diffDiffTool == "diff" {
		diffOpts.UseDifftastic = false
		diffOpts.UseTreeSitter = false
	}
	diff := differ.New(norm, diffOpts)

	// Perform diff
	if rootVerbose {
		fmt.Fprintf(os.Stderr, "\nComparing manifests...\n")
	}
	result, err := diff.Diff(sourceManifests, targetManifests)
	if err != nil {
		return fmt.Errorf("failed to diff manifests: %w", err)
	}

	// Create reporter
	reporterOpts := &reporter.Options{
		Format:        diffOutputFormat,
		Colorize:      !diffNoColor,
		ShowIdentical: diffShowIdentical,
		Compact:       false,
	}

	var rep reporter.Reporter
	switch diffOutputFormat {
	case "json":
		rep = reporter.NewJSONReporter(reporterOpts)
	case "cli":
		rep = reporter.NewCLIReporter(reporterOpts)
	default:
		return fmt.Errorf("unsupported output format: %s", diffOutputFormat)
	}

	// Generate output
	if rootVerbose {
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

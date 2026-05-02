package main

import (
	"fmt"
	"io"
	"os"

	"github.com/nhuray/kyt/pkg/config"
	"github.com/nhuray/kyt/pkg/differ"
	"github.com/nhuray/kyt/pkg/manifest"
	"github.com/nhuray/kyt/pkg/normalizer"
	"github.com/nhuray/kyt/pkg/pager"
	"github.com/nhuray/kyt/pkg/reporter"
	"github.com/nhuray/kyt/pkg/resourcekind"
	"github.com/spf13/cobra"
)

var (
	// Diff command flags
	diffOutput              string
	diffSummary             bool
	diffUnified             int
	diffColor               string
	diffExactMatch          bool
	diffSimilarityThreshold float64
	diffDataSimilarityBoost int
	diffIncludeKinds        string
	diffExcludeKinds        string
	diffContext             string
)

var diffCmd = &cobra.Command{
	Use:   "diff <source> <target>",
	Short: "Compare Kubernetes manifests",
	Long: `Compare Kubernetes manifests with configurable ignore rules.

Supports:
- Unified diff format (git-style)
- Configurable pager support (with $PAGER fallback)
- Tabular summary with --summary flag
- Smart similarity matching for renamed resources
- Resource filtering by kind (include/exclude)
- Live cluster comparison using namespace syntax (ns:namespace)

Exit Codes:
- 0: No differences found
- 1: Differences found
- 2+: Error occurred

Resource Filtering:
- Use --include to only compare specific resource kinds
- Use --exclude to skip specific resource kinds
- Supports short names (cm, svc, deploy), singular (configmap, service), and plural (configmaps, services)
- Both flags accept comma-separated lists

Cluster Comparison:
- Use ns:namespace syntax to compare resources from a Kubernetes namespace
- Requires --context flag to specify the cluster context
- Example: kyt diff --context prod ns:default ns:staging
- Fetches ~15 common resource types (pods, deployments, services, etc.)
- Use --include/--exclude to filter resource types

Examples:
  # Compare two directories
  kyt diff ./source-manifests ./target-manifests

  # Compare two namespaces in the same cluster
  kyt diff --context prod ns:default ns:staging

  # Compare local manifests against live cluster
  kyt diff ./manifests --context prod ns:production

  # Compare live cluster against local manifests
  kyt diff --context prod ns:production ./manifests

  # Compare specific resource types from cluster
  kyt diff --context prod --include deploy,svc ns:default ns:staging

  # Show tabular summary instead of full diff
  kyt diff --summary ./source ./target

  # Compare with custom config
  kyt diff -c .kyt.yaml ./source ./target

  # Write output to file
  kyt diff -o diff.txt ./source ./target

  # Use 5 lines of context (default is 3)
  kyt diff -U5 ./source ./target

  # Compare only ConfigMaps and Secrets
  kyt diff --include cm,secrets ./source ./target

  # Compare all except Secrets
  kyt diff --exclude secrets ./source ./target

  # Compare Deployments and Services only (multiple forms supported)
  kyt diff --include deploy,svc ./source ./target
  kyt diff --include deployments,services ./source ./target
  kyt diff --include Deployment,Service ./source ./target

  # Compare Helm vs Kustomize
  helm template my-chart > /tmp/helm.yaml
  kustomize build ./overlay > /tmp/kustomize.yaml
  kyt diff /tmp/helm.yaml /tmp/kustomize.yaml

  # Control color output
  kyt diff --color=always ./source ./target  # Always colorize
  kyt diff --color=never ./source ./target   # Never colorize
  kyt diff --color=auto ./source ./target    # Auto (default, based on TTY)
`,
	Args:          cobra.ExactArgs(2),
	RunE:          runDiff,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	diffCmd.Flags().StringVarP(&diffOutput, "output", "o", "", "write diff to file instead of stdout")
	diffCmd.Flags().BoolVar(&diffSummary, "summary", false, "show tabular summary of resource changes")
	diffCmd.Flags().IntVarP(&diffUnified, "unified", "U", 3, "generate diff with <n> lines of context")
	diffCmd.Flags().StringVar(&diffColor, "color", "auto", "colorize output: auto, always, never")
	diffCmd.Flags().BoolVar(&diffExactMatch, "exact-match", false, "disable similarity matching (only exact name matches)")
	diffCmd.Flags().Float64Var(&diffSimilarityThreshold, "similarity-threshold", 0.7, "minimum similarity score (0.0-1.0) for matching resources")
	diffCmd.Flags().IntVar(&diffDataSimilarityBoost, "data-similarity-boost", 2, "boost factor for ConfigMap/Secret data fields (1-10, higher = more weight on data)")
	diffCmd.Flags().StringVar(&diffIncludeKinds, "include", "", "comma-separated list of resource kinds to include (e.g., 'cm,svc,deploy')")
	diffCmd.Flags().StringVar(&diffExcludeKinds, "exclude", "", "comma-separated list of resource kinds to exclude (e.g., 'secrets,configmaps')")
	diffCmd.Flags().StringVar(&diffContext, "context", "", "Kubernetes context to use for namespace inputs (ns:namespace)")

	rootCmd.AddCommand(diffCmd)
}

func runDiff(cmd *cobra.Command, args []string) error {
	// Parse input sources
	sourceInput := parseInput(args[0])
	targetInput := parseInput(args[1])

	if rootVerbose {
		fmt.Fprintf(os.Stderr, "Comparing:\n")
		fmt.Fprintf(os.Stderr, "  Source: %s\n", formatInputForDisplay(sourceInput, diffContext))
		fmt.Fprintf(os.Stderr, "  Target: %s\n", formatInputForDisplay(targetInput, diffContext))
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
		fmt.Fprintf(os.Stderr, "\nLoading source manifests...\n")
	}
	sourceManifests, err := loadManifests(sourceInput, diffContext)
	if err != nil {
		return fmt.Errorf("failed to load source manifests: %w", err)
	}
	if rootVerbose {
		fmt.Fprintf(os.Stderr, "  Found %d resources in source\n", sourceManifests.Len())
	}

	// Parse target manifests
	if rootVerbose {
		fmt.Fprintf(os.Stderr, "Loading target manifests...\n")
	}
	targetManifests, err := loadManifests(targetInput, diffContext)
	if err != nil {
		return fmt.Errorf("failed to load target manifests: %w", err)
	}
	if rootVerbose {
		fmt.Fprintf(os.Stderr, "  Found %d resources in target\n", targetManifests.Len())
	}

	// Apply resource kind filtering
	if diffIncludeKinds != "" || diffExcludeKinds != "" {
		if rootVerbose {
			fmt.Fprintf(os.Stderr, "Applying resource kind filters...\n")
		}

		matcher := resourcekind.NewMatcher()
		var includeFilters, excludeFilters []string

		if diffIncludeKinds != "" {
			includeFilters = matcher.ParseList(diffIncludeKinds)
			if rootVerbose {
				fmt.Fprintf(os.Stderr, "  Include: %v\n", includeFilters)
			}
		}

		if diffExcludeKinds != "" {
			excludeFilters = matcher.ParseList(diffExcludeKinds)
			if rootVerbose {
				fmt.Fprintf(os.Stderr, "  Exclude: %v\n", excludeFilters)
			}
		}

		sourceManifests = filterManifests(sourceManifests, matcher, includeFilters, excludeFilters)
		targetManifests = filterManifests(targetManifests, matcher, includeFilters, excludeFilters)

		if rootVerbose {
			fmt.Fprintf(os.Stderr, "  After filtering: %d source, %d target\n", sourceManifests.Len(), targetManifests.Len())
		}
	}

	// Create normalizer
	norm := normalizer.New(cfg)

	// Determine output destination and pager usage
	var outputWriter io.WriteCloser
	var usePager bool

	if diffOutput != "" {
		// Writing to file
		file, err := os.Create(diffOutput)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer func() {
			if closeErr := file.Close(); closeErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to close output file: %v\n", closeErr)
			}
		}()
		outputWriter = file
		usePager = false
	} else {
		// Writing to stdout - check for pager
		pagerCmd := getPagerCommand(cfg)
		p := pager.NewPager(pagerCmd)

		if p.ShouldPage(true) {
			pagerWriter, err := p.Pipe()
			if err != nil {
				// Fallback to stdout
				fmt.Fprintf(os.Stderr, "Warning: pager failed: %v\n", err)
				outputWriter = nopWriteCloser{os.Stdout}
				usePager = false
			} else {
				outputWriter = pagerWriter
				usePager = true
			}
		} else {
			outputWriter = nopWriteCloser{os.Stdout}
			usePager = false
		}
	}
	defer func() {
		if closeErr := outputWriter.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close output: %v\n", closeErr)
		}
	}()

	// Determine colorization
	colorize := shouldColorize(diffColor, !usePager)

	// Get context lines (CLI overrides config)
	contextLines := diffUnified
	if !cmd.Flags().Changed("unified") && cfg.Diff.Options.ContextLines > 0 {
		contextLines = cfg.Diff.Options.ContextLines
	}

	// Get similarity threshold (CLI overrides config)
	similarityThreshold := diffSimilarityThreshold
	if !cmd.Flags().Changed("similarity-threshold") && cfg.Diff.Options.SimilarityThreshold > 0 {
		similarityThreshold = cfg.Diff.Options.SimilarityThreshold
	}

	// Get data similarity boost (CLI overrides config)
	dataSimilarityBoost := diffDataSimilarityBoost
	if !cmd.Flags().Changed("data-similarity-boost") && cfg.Diff.Options.DataSimilarityBoost > 0 {
		dataSimilarityBoost = cfg.Diff.Options.DataSimilarityBoost
	}

	// Create differ
	diffOpts := &differ.DiffOptions{
		ContextLines:               contextLines,
		EnableSimilarityMatching:   !diffExactMatch,
		SimilarityThreshold:        similarityThreshold,
		FuzzyStringMatchingEnabled: cfg.Diff.FuzzyMatching.Enabled,
		FuzzyStringMinLength:       cfg.Diff.FuzzyMatching.MinStringLength,
		DataSimilarityBoost:        dataSimilarityBoost,
	}
	d := differ.New(norm, diffOpts)

	// Perform diff
	if rootVerbose {
		fmt.Fprintf(os.Stderr, "\nComparing manifests...\n")
	}
	result, err := d.Diff(sourceManifests, targetManifests)
	if err != nil {
		return fmt.Errorf("failed to diff manifests: %w", err)
	}

	// Create reporter
	rep := reporter.NewReporter(diffSummary, colorize)

	// Generate output
	if rootVerbose {
		fmt.Fprintf(os.Stderr, "Generating output...\n\n")
	}
	if err := rep.Report(result, outputWriter); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	// Git-style exit code: 1 if changes exist, 0 if no changes
	if hasChanges(result) {
		return &exitError{code: 1}
	}

	return nil
}

// getPagerCommand returns the pager command from config or environment
func getPagerCommand(cfg *config.Config) string {
	// Priority: config > $PAGER
	if cfg.Diff.Pager != "" {
		return cfg.Diff.Pager
	}
	return os.Getenv("PAGER")
}

// shouldColorize determines whether to colorize output based on flag and context
func shouldColorize(colorFlag string, notUsingPager bool) bool {
	switch colorFlag {
	case "always":
		return true
	case "never":
		return false
	case "auto":
		// Don't colorize if using pager (let pager handle it)
		if !notUsingPager {
			return false
		}
		// Check if stdout is TTY
		fileInfo, _ := os.Stdout.Stat()
		return (fileInfo.Mode() & os.ModeCharDevice) != 0
	}
	return false
}

// hasChanges checks if the diff result contains any changes
func hasChanges(result *differ.DiffResult) bool {
	return len(result.Changes) > 0
}

// nopWriteCloser wraps an io.Writer with a no-op Close method
type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error {
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

// filterManifests filters a ManifestSet based on include/exclude filters
func filterManifests(manifestSet *manifest.ManifestSet, matcher *resourcekind.Matcher, includeFilters, excludeFilters []string) *manifest.ManifestSet {
	filtered := manifest.NewManifestSet()

	for key, obj := range manifestSet.Resources {
		kind := obj.GetKind()

		// If include filters are specified, only include matching kinds
		if len(includeFilters) > 0 {
			if !matcher.MatchesAny(kind, includeFilters) {
				continue
			}
		}

		// If exclude filters are specified, skip matching kinds
		if len(excludeFilters) > 0 {
			if matcher.MatchesAny(kind, excludeFilters) {
				continue
			}
		}

		// Add to filtered set
		filtered.Resources[key] = obj
		// Preserve source file information if available
		if sourcePath, ok := manifestSet.GetSourceFile(key); ok {
			filtered.SourceFile[key] = sourcePath
		}
	}

	return filtered
}

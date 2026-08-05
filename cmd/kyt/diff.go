package main

import (
	"fmt"
	"io"
	"os"

	"github.com/nhuray/kyt/pkg/config"
	"github.com/nhuray/kyt/pkg/differ"
	"github.com/nhuray/kyt/pkg/filter"
	"github.com/nhuray/kyt/pkg/manifest"
	"github.com/nhuray/kyt/pkg/normalizer"
	"github.com/nhuray/kyt/pkg/pager"
	"github.com/nhuray/kyt/pkg/reporter"
	"github.com/nhuray/kyt/pkg/resourcekind"
	"github.com/nhuray/kyt/pkg/tui"
	"github.com/spf13/cobra"
)

var (
	// Diff command flags
	diffOutput              string
	diffSummary             bool
	diffMarkdown            bool
	diffTUI                 bool
	diffUnified             int
	diffColor               string
	diffExactMatch          bool
	diffSimilarityThreshold float64
	diffDataSimilarityBoost int
	diffKinds               string
	diffNamespaces          string
	diffContext             string
	diffMaskSecrets         *bool // Use pointer to detect if flag was explicitly set
)

var diffCmd = &cobra.Command{
	Use:   "diff <left> <right>",
	Short: "Compare Kubernetes manifests",
	Long: `Compare Kubernetes manifests with configurable ignore rules.

Supports:
- Unified diff format (git-style)
- Configurable pager support (with $PAGER fallback)
- Tabular summary with --summary flag
- Smart similarity matching for renamed resources
- Resource filtering by kind and namespace
- Live cluster comparison using namespace syntax (ns:namespace)

Exit Codes:
- 0: No differences found
- 1: Differences found
- 2+: Error occurred

Resource Filtering:
- Use --kind/-k to filter resource kinds (supports include/exclude syntax)
- Use --namespace/-n/--ns to filter namespaces (supports include/exclude syntax)
- Supports short names (cm, svc, deploy), singular (configmap, service), and plural (configmaps, services)
- Use "-" prefix for exclusions
- Examples:
  --kind dep,sts           # include only Deployments and StatefulSets
  --kind -cm,-sec          # exclude ConfigMaps and Secrets
  --namespace prod,staging # include only prod and staging namespaces
  --namespace -kube-system # exclude kube-system namespace
- Both flags accept comma-separated lists
- Cluster-scoped resources (ClusterRole, Node, etc.) are excluded when namespace filters are active

Cluster Comparison:
- Use ns:namespace syntax to compare resources from a Kubernetes namespace
- Use --context flag to specify cluster context (defaults to current context)
- Example: kyt diff ns:default ns:staging
- Example: kyt diff --context prod ns:default ns:staging
- Fetches ~15 common resource types (pods, deployments, services, etc.)
- Use --kind to filter resource types

Secret Masking:
- Kubernetes Secrets are masked by default using pattern-based rules to prevent credential leaks
- Masking examples:
  * Connection strings: "postgres://admin:pass@host:5432" → "postgres://ad***:pa**@*****:5432"
  * API keys: "sk_live_abc123def456" → "********abc1****f456"
  * Base64: "cGFzc3dvcmQxMjM=" → "cGFz********wM="
- Configure masking rules in .kyt.yaml:
  diff:
    secretMasking:
      enabled: true
      maskChar: "*"
      fields: ["data", "stringData"]
      rules:
        - name: "postgres"
          pattern: '^postgres://(?<user>[^:]+):(?<password>[^@]+)@(?<host>.+)$'
          masks:
            user: "mask-2-0"
            password: "mask-2-2"
            host: "mask-0-10"
        - name: "default"
          pattern: '^(?<value>.+)$'
          masks:
            value: "mask-2-2"
- Strategies: "mask-F-E" (keep first F, last E chars) or "mask-F-E-S-M" (sequenced)
- To disable masking temporarily: kyt diff --mask-secrets=false ./left ./right
- Alternative: exclude Secrets entirely with --kind=-secrets
- See docs/diff.md#secret-masking for full documentation

Examples:
  # Compare two directories
  kyt diff ./left-manifests ./right-manifests

  # Compare two namespaces (uses current context)
  kyt diff ns:default ns:staging

  # Compare two namespaces in a specific cluster
  kyt diff --context prod ns:default ns:staging

  # Compare local manifests against live cluster
  kyt diff ./manifests ns:production

  # Compare live cluster against local manifests
  kyt diff ns:production ./manifests

  # Compare specific resource types from cluster
  kyt diff --kind deploy,svc ns:default ns:staging

  # Exclude specific resource types
  kyt diff --kind -cm,-sec ./left ./right

  # Filter by namespace
  kyt diff --namespace production,staging ./left ./right

  # Combine kind and namespace filters
  kyt diff --kind dep,sts --namespace prod ./left ./right

  # Show tabular summary instead of full diff
  kyt diff --summary ./left ./right

  # Compare with custom config
  kyt diff -c .kyt.yaml ./left ./right

  # Write output to file
  kyt diff -o diff.txt ./left ./right

  # Use 5 lines of context (default is 3)
  kyt diff -U5 ./left ./right

  # Compare only Deployments and StatefulSets
  kyt diff --kind dep,sts ./left ./right
  kyt diff --kind deployments,statefulsets ./left ./right
  kyt diff --kind Deployment,StatefulSet ./left ./right

  # Compare all except Secrets and ConfigMaps
  kyt diff --kind -secrets,-configmaps ./left ./right

  # Compare Helm vs Kustomize
  helm template my-chart > /tmp/helm.yaml
  kustomize build ./overlay > /tmp/kustomize.yaml
  kyt diff /tmp/helm.yaml /tmp/kustomize.yaml

  # Control color output
  kyt diff --color=always ./left ./right  # Always colorize
  kyt diff --color=never ./left ./right   # Never colorize
  kyt diff --color=auto ./left ./right    # Auto (default, based on TTY)

  # Secret masking (enabled by default)
  kyt diff ./left ./right                 # Secrets are masked by default
  kyt diff --mask-secrets=false ./left ./right  # Disable masking to see full Secret values
`,
	Args:          cobra.ExactArgs(2),
	RunE:          runDiff,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	diffCmd.Flags().StringVarP(&diffOutput, "output", "o", "", "write diff to file instead of stdout")
	diffCmd.Flags().BoolVar(&diffSummary, "summary", false, "show tabular summary of resource changes")
	diffCmd.Flags().BoolVar(&diffMarkdown, "markdown", false, "output in markdown format (suitable for GitHub PR comments)")
	diffCmd.Flags().BoolVar(&diffTUI, "tui", false, "show interactive terminal UI for exploring diffs")
	diffCmd.Flags().IntVarP(&diffUnified, "unified", "U", 3, "generate diff with <n> lines of context")
	diffCmd.Flags().StringVar(&diffColor, "color", "auto", "colorize output: auto, always, never")
	diffCmd.Flags().BoolVar(&diffExactMatch, "exact-match", false, "disable similarity matching (only exact name matches)")
	diffCmd.Flags().Float64Var(&diffSimilarityThreshold, "similarity-threshold", 0.7, "minimum similarity score (0.0-1.0) for matching resources")
	diffCmd.Flags().IntVar(&diffDataSimilarityBoost, "data-similarity-boost", 2, "boost factor for ConfigMap/Secret data fields (1-10, higher = more weight on data)")
	diffCmd.Flags().StringVarP(&diffKinds, "kind", "k", "", "filter resource kinds (e.g., 'dep,sts' or '-cm,-sec')")
	diffCmd.Flags().StringVar(&diffKinds, "kinds", "", "filter resource kinds (alias for --kind)")
	diffCmd.Flags().StringVarP(&diffNamespaces, "namespace", "n", "", "filter namespaces (e.g., 'prod,staging' or '-kube-system')")
	diffCmd.Flags().StringVar(&diffNamespaces, "namespaces", "", "filter namespaces (alias for --namespace)")
	diffCmd.Flags().StringVar(&diffNamespaces, "ns", "", "filter namespaces (alias for --namespace)")
	diffCmd.Flags().StringVar(&diffContext, "context", "", "Kubernetes context to use for namespace inputs (defaults to current context)")
	
	// Secret masking flag - default true
	defaultMaskSecrets := true
	diffMaskSecrets = &defaultMaskSecrets
	diffCmd.Flags().BoolVar(diffMaskSecrets, "mask-secrets", true, "mask Secret data/stringData values in output")

	rootCmd.AddCommand(diffCmd)
}

func runDiff(cmd *cobra.Command, args []string) error {
	// Parse input sources
	sourceInput := parseInput(args[0])
	targetInput := parseInput(args[1])

	// Determine the effective context to use
	// If either input is a namespace and no context is specified, get the current context
	effectiveContext := diffContext
	if effectiveContext == "" && (sourceInput.Type == inputTypeNamespace || targetInput.Type == inputTypeNamespace) {
		currentContext, err := getCurrentContext()
		if err != nil {
			return fmt.Errorf("no --context flag provided and failed to get current context from kubeconfig: %w", err)
		}
		effectiveContext = currentContext
		if rootVerbose {
			fmt.Fprintf(os.Stderr, "Using current context: %s\n\n", effectiveContext)
		}
	}

	if rootVerbose {
		fmt.Fprintf(os.Stderr, "Comparing:\n")
		fmt.Fprintf(os.Stderr, "  Left: %s\n", formatInputForDisplay(sourceInput, effectiveContext))
		fmt.Fprintf(os.Stderr, "  Right: %s\n", formatInputForDisplay(targetInput, effectiveContext))
	}

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if rootVerbose && rootConfigFile != "" {
		fmt.Fprintf(os.Stderr, "  Config: %s\n", rootConfigFile)
	}

	// Determine verbose writer for cluster operations
	var verboseWriter io.Writer
	if rootVerbose {
		verboseWriter = os.Stderr
	}

	// Parse left manifests
	if rootVerbose {
		fmt.Fprintf(os.Stderr, "\nLoading left manifests...\n")
	}
	sourceManifests, err := loadManifests(sourceInput, effectiveContext, verboseWriter)
	if err != nil {
		return fmt.Errorf("failed to load left manifests: %w", err)
	}
	if rootVerbose {
		fmt.Fprintf(os.Stderr, "  Found %d resources in left\n", sourceManifests.Len())
	}

	// Parse right manifests
	if rootVerbose {
		fmt.Fprintf(os.Stderr, "Loading right manifests...\n")
	}
	targetManifests, err := loadManifests(targetInput, effectiveContext, verboseWriter)
	if err != nil {
		return fmt.Errorf("failed to load right manifests: %w", err)
	}
	if rootVerbose {
		fmt.Fprintf(os.Stderr, "  Found %d resources in right\n", targetManifests.Len())
	}

	// Apply resource kind filtering
	var kindFilter, namespaceFilter *filter.Expression
	
	// Parse CLI filters
	if diffKinds != "" {
		var err error
		kindFilter, err = filter.Parse(diffKinds)
		if err != nil {
			return fmt.Errorf("failed to parse --kind filter: %w", err)
		}
	}
	
	if diffNamespaces != "" {
		var err error
		namespaceFilter, err = filter.Parse(diffNamespaces)
		if err != nil {
			return fmt.Errorf("failed to parse --namespace filter: %w", err)
		}
	}
	
	// Parse config file filters and merge with CLI (CLI takes precedence)
	if len(cfg.Diff.Filters.Kinds) > 0 || len(cfg.Diff.Filters.Namespaces) > 0 {
		configKindFilter, err := filter.ParseList(cfg.Diff.Filters.Kinds)
		if err != nil {
			return fmt.Errorf("failed to parse config kinds filter: %w", err)
		}
		
		configNamespaceFilter, err := filter.ParseList(cfg.Diff.Filters.Namespaces)
		if err != nil {
			return fmt.Errorf("failed to parse config namespaces filter: %w", err)
		}
		
		// Merge: CLI overrides config
		if kindFilter == nil || kindFilter.IsEmpty() {
			kindFilter = configKindFilter
		}
		if namespaceFilter == nil || namespaceFilter.IsEmpty() {
			namespaceFilter = configNamespaceFilter
		}
	}
	
	// Apply filters if any are specified
	if (kindFilter != nil && !kindFilter.IsEmpty()) || (namespaceFilter != nil && !namespaceFilter.IsEmpty()) {
		if rootVerbose {
			fmt.Fprintf(os.Stderr, "Applying resource filters...\n")
			if kindFilter != nil && !kindFilter.IsEmpty() {
				fmt.Fprintf(os.Stderr, "  Kinds: %s\n", kindFilter.String())
			}
			if namespaceFilter != nil && !namespaceFilter.IsEmpty() {
				fmt.Fprintf(os.Stderr, "  Namespaces: %s\n", namespaceFilter.String())
			}
		}

		matcher := resourcekind.NewMatcher()
		sourceManifests = filterManifests(sourceManifests, matcher, kindFilter, namespaceFilter)
		targetManifests = filterManifests(targetManifests, matcher, kindFilter, namespaceFilter)

		if rootVerbose {
			fmt.Fprintf(os.Stderr, "  After filtering: %d left, %d right\n", sourceManifests.Len(), targetManifests.Len())
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
		// Only close if using pager (stdout doesn't need closing)
		if usePager {
			defer func() {
				if closeErr := outputWriter.Close(); closeErr != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to close pager: %v\n", closeErr)
				}
			}()
		}
	}

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

	// Override Secret masking from CLI if provided
	maskSecretsConfig := cfg.Diff.SecretMasking
	if diffMaskSecrets != nil && cmd.Flags().Changed("mask-secrets") {
		// CLI flag overrides config
		maskSecretsConfig.Enabled = diffMaskSecrets
	}

	// Create differ with Secret masking support
	diffOpts := &differ.DiffOptions{
		ContextLines:               contextLines,
		EnableSimilarityMatching:   !diffExactMatch,
		SimilarityThreshold:        similarityThreshold,
		FuzzyStringMatchingEnabled: cfg.Diff.FuzzyMatching.Enabled,
		FuzzyStringMinLength:       cfg.Diff.FuzzyMatching.MinStringLength,
		DataSimilarityBoost:        dataSimilarityBoost,
	}
	d := differ.NewWithMasking(norm, diffOpts, maskSecretsConfig)

	// Perform diff
	if rootVerbose {
		fmt.Fprintf(os.Stderr, "\nComparing manifests...\n")
	}
	result, err := d.Diff(sourceManifests, targetManifests)
	if err != nil {
		return fmt.Errorf("failed to diff manifests: %w", err)
	}

	// Check for conflicting flags
	if diffTUI && diffOutput != "" {
		return fmt.Errorf("--tui and --output flags are mutually exclusive")
	}
	if diffTUI && diffSummary {
		return fmt.Errorf("--tui and --summary flags are mutually exclusive")
	}
	if diffTUI && diffMarkdown {
		return fmt.Errorf("--tui and --markdown flags are mutually exclusive")
	}
	if diffMarkdown && diffColor != "auto" {
		return fmt.Errorf("--markdown and --color flags are mutually exclusive (markdown is always uncolored)")
	}

	// If TUI mode is requested, launch the interactive interface
	if diffTUI {
		leftSource := formatInputForDisplay(sourceInput, effectiveContext)
		rightSource := formatInputForDisplay(targetInput, effectiveContext)
		return tui.Run(result, leftSource, rightSource)
	}

	// Markdown format disables colorization
	if diffMarkdown {
		colorize = false
	}

	// Create reporter
	rep := reporter.NewReporter(diffSummary, colorize, diffMarkdown)

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

// filterManifests filters a ManifestSet based on kind and namespace filters
func filterManifests(
	manifestSet *manifest.ManifestSet,
	matcher *resourcekind.Matcher,
	kindFilter *filter.Expression,
	namespaceFilter *filter.Expression,
) *manifest.ManifestSet {
	filtered := manifest.NewManifestSet()

	for key, obj := range manifestSet.Resources {
		kind := obj.GetKind()
		namespace := obj.GetNamespace()

		// Apply kind filter
		if kindFilter != nil && !kindFilter.IsEmpty() {
			// Use a matcher function that handles resource kind aliases
			kindMatcher := func(objKind, filterKind string) bool {
				return matcher.Match(objKind, filterKind)
			}
			if !kindFilter.ShouldInclude(kind, kindMatcher) {
				continue
			}
		}

		// Apply namespace filter
		if namespaceFilter != nil && !namespaceFilter.IsEmpty() {
			// Cluster-scoped resources (empty namespace) are excluded when namespace filters are active
			if namespace == "" {
				continue
			}

			// Simple string matcher for namespaces (case-sensitive)
			namespaceMatcher := func(objNs, filterNs string) bool {
				return objNs == filterNs
			}
			if !namespaceFilter.ShouldInclude(namespace, namespaceMatcher) {
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

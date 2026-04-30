package differ

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/nhuray/k8s-diff/pkg/differ/treesitter"
	"github.com/nhuray/k8s-diff/pkg/manifest"
	"github.com/nhuray/k8s-diff/pkg/normalizer"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

// Differ performs diffs between two sets of Kubernetes manifests
type Differ struct {
	normalizer *normalizer.Normalizer
	options    *DiffOptions
}

// New creates a new Differ with the given normalizer and options
func New(norm *normalizer.Normalizer, opts *DiffOptions) *Differ {
	if opts == nil {
		opts = NewDefaultDiffOptions()
	}
	return &Differ{
		normalizer: norm,
		options:    opts,
	}
}

// Diff compares two manifest sets and returns the differences
func (d *Differ) Diff(source, target *manifest.ManifestSet) (*DiffResult, error) {
	result := &DiffResult{
		Added:     []manifest.ResourceKey{},
		Removed:   []manifest.ResourceKey{},
		Modified:  []ResourceDiff{},
		Identical: []manifest.ResourceKey{},
	}

	// Normalize both sets
	normalizedSource := make(map[manifest.ResourceKey]*unstructured.Unstructured)
	for key, obj := range source.Resources {
		normalized, err := d.normalizer.Normalize(obj)
		if err != nil {
			return nil, fmt.Errorf("failed to normalize source resource %s: %w", key.String(), err)
		}
		normalizedSource[key] = normalized
	}

	normalizedTarget := make(map[manifest.ResourceKey]*unstructured.Unstructured)
	for key, obj := range target.Resources {
		normalized, err := d.normalizer.Normalize(obj)
		if err != nil {
			return nil, fmt.Errorf("failed to normalize target resource %s: %w", key.String(), err)
		}
		normalizedTarget[key] = normalized
	}

	// Set up resource cache for similarity matching
	SetResourceCache(normalizedSource, normalizedTarget)

	// Perform 2-stage matching
	matcher := NewResourceMatcher(
		d.options.EnableSimilarityMatching,
		d.options.SimilarityThreshold,
	)

	matches, unmatchedSource, unmatchedTarget := matcher.MatchResources(
		normalizedSource,
		normalizedTarget,
	)

	// Process matched pairs
	for _, match := range matches {
		sourceObj := match.SourceResource
		targetObj := match.TargetResource

		// Check if resources are equal
		equal, err := areResourcesEqual(sourceObj, targetObj)
		if err != nil {
			return nil, fmt.Errorf("failed to compare resources for %s: %w", match.SourceKey.String(), err)
		}

		if equal {
			result.Identical = append(result.Identical, match.SourceKey)
		} else {
			// Generate diff
			diffText, diffLines, err := d.generateDiff(match.SourceKey, sourceObj, targetObj)
			if err != nil {
				return nil, fmt.Errorf("failed to generate diff for %s: %w", match.SourceKey.String(), err)
			}

			result.Modified = append(result.Modified, ResourceDiff{
				SourceKey:       match.SourceKey,
				TargetKey:       match.TargetKey,
				Key:             match.SourceKey, // For backward compatibility
				Source:          sourceObj,
				Target:          targetObj,
				DiffText:        diffText,
				DiffLines:       diffLines,
				MatchType:       string(match.Type),
				SimilarityScore: match.SimilarityScore,
			})
		}
	}

	// Process unmatched resources
	result.Removed = unmatchedSource
	result.Added = unmatchedTarget

	// Calculate summary
	allKeys := make(map[manifest.ResourceKey]bool)
	for key := range normalizedSource {
		allKeys[key] = true
	}
	for key := range normalizedTarget {
		allKeys[key] = true
	}

	result.Summary = DiffSummary{
		TotalResources: len(allKeys),
		AddedCount:     len(result.Added),
		RemovedCount:   len(result.Removed),
		ModifiedCount:  len(result.Modified),
		IdenticalCount: len(result.Identical),
	}

	return result, nil
}

// areResourcesEqual checks if two resources are equal by comparing their JSON representations
func areResourcesEqual(a, b *unstructured.Unstructured) (bool, error) {
	aJSON, err := json.Marshal(a.Object)
	if err != nil {
		return false, err
	}

	bJSON, err := json.Marshal(b.Object)
	if err != nil {
		return false, err
	}

	return bytes.Equal(aJSON, bJSON), nil
}

// generateDiff generates a diff between two resources
func (d *Differ) generateDiff(key manifest.ResourceKey, source, target *unstructured.Unstructured) (string, int, error) {
	// Convert resources to pretty-printed JSON
	sourceJSON, err := json.MarshalIndent(source.Object, "", "  ")
	if err != nil {
		return "", 0, fmt.Errorf("failed to marshal source: %w", err)
	}

	targetJSON, err := json.MarshalIndent(target.Object, "", "  ")
	if err != nil {
		return "", 0, fmt.Errorf("failed to marshal target: %w", err)
	}

	// Try difftastic first if enabled
	if d.options.UseDifftastic {
		diffText, diffLines, err := d.generateDifftasticDiff(key, sourceJSON, targetJSON)
		if err == nil {
			return diffText, diffLines, nil
		}
		// Fall through to try tree-sitter
	}

	// Try tree-sitter diff as fallback if enabled
	if d.options.UseTreeSitter {
		diffText, diffLines, err := d.generateTreeSitterDiff(key, source, target)
		if err == nil {
			return diffText, diffLines, nil
		}
		// Fall through to unified diff
	}

	// Generate unified diff as final fallback
	return d.generateUnifiedDiff(key, sourceJSON, targetJSON)
}

// generateDifftasticDiff generates a diff using difftastic
func (d *Differ) generateDifftasticDiff(key manifest.ResourceKey, sourceJSON, targetJSON []byte) (string, int, error) {
	// Check if difftastic is available
	if !isDifftasticAvailable() {
		return "", 0, fmt.Errorf("difftastic not available")
	}

	// Create temp files
	tmpDir, err := os.MkdirTemp("", "k8s-diff-*")
	if err != nil {
		return "", 0, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	sourceFile := filepath.Join(tmpDir, "source.json")
	targetFile := filepath.Join(tmpDir, "target.json")

	if err := os.WriteFile(sourceFile, sourceJSON, 0644); err != nil {
		return "", 0, fmt.Errorf("failed to write source file: %w", err)
	}

	if err := os.WriteFile(targetFile, targetJSON, 0644); err != nil {
		return "", 0, fmt.Errorf("failed to write target file: %w", err)
	}

	// Build difftastic command
	args := []string{sourceFile, targetFile}

	// Add display mode
	if d.options.DifftasticDisplay != "" {
		args = append(args, "--display", d.options.DifftasticDisplay)
	}

	// Add color option
	// We need to explicitly set color mode because difftastic auto-detects TTY
	// and won't use colors when output is captured
	if d.options.ColorOutput {
		args = append(args, "--color", "always")
	} else {
		args = append(args, "--color", "never")
	}

	// Execute difftastic
	cmd := exec.Command("difft", args...)
	output, err := cmd.CombinedOutput()

	// difftastic returns exit code 1 when there are differences, which is expected
	// Only treat it as an error if the output is empty and there's an error
	if err != nil && len(output) == 0 {
		return "", 0, fmt.Errorf("difftastic failed: %w", err)
	}

	// Count diff lines (approximate)
	diffLines := bytes.Count(output, []byte("\n"))

	return string(output), diffLines, nil
}

// generateUnifiedDiff generates a unified diff
func (d *Differ) generateUnifiedDiff(key manifest.ResourceKey, sourceJSON, targetJSON []byte) (string, int, error) {
	// Create temp files
	tmpDir, err := os.MkdirTemp("", "k8s-diff-*")
	if err != nil {
		return "", 0, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	sourceFile := filepath.Join(tmpDir, "source.json")
	targetFile := filepath.Join(tmpDir, "target.json")

	if err := os.WriteFile(sourceFile, sourceJSON, 0644); err != nil {
		return "", 0, fmt.Errorf("failed to write source file: %w", err)
	}

	if err := os.WriteFile(targetFile, targetJSON, 0644); err != nil {
		return "", 0, fmt.Errorf("failed to write target file: %w", err)
	}

	// Use diff command
	args := []string{
		"-u",
		fmt.Sprintf("-U%d", d.options.ContextLines),
		"--label", fmt.Sprintf("a/%s", key.String()),
		"--label", fmt.Sprintf("b/%s", key.String()),
		sourceFile,
		targetFile,
	}

	cmd := exec.Command("diff", args...)
	output, err := cmd.CombinedOutput()

	// diff returns exit code 1 when there are differences, which is expected
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				// This is expected - means there are differences
				err = nil
			}
		}
	}

	if err != nil && len(output) == 0 {
		return "", 0, fmt.Errorf("diff command failed: %w", err)
	}

	// Count diff lines (lines starting with +, -, or @@)
	lines := bytes.Split(output, []byte("\n"))
	diffLines := 0
	for _, line := range lines {
		if len(line) > 0 && (line[0] == '+' || line[0] == '-' || line[0] == '@') {
			diffLines++
		}
	}

	return string(output), diffLines, nil
}

// isDifftasticAvailable checks if difftastic is available in PATH
func isDifftasticAvailable() bool {
	_, err := exec.LookPath("difft")
	return err == nil
}

// generateTreeSitterDiff generates a diff using Go-native tree-sitter parser
func (d *Differ) generateTreeSitterDiff(key manifest.ResourceKey, source, target *unstructured.Unstructured) (string, int, error) {
	// Validate that both resources are valid Kubernetes resources
	if err := treesitter.ValidateKubernetesResource(source); err != nil {
		return "", 0, fmt.Errorf("invalid source resource: %w", err)
	}
	if err := treesitter.ValidateKubernetesResource(target); err != nil {
		return "", 0, fmt.Errorf("invalid target resource: %w", err)
	}

	// Convert resources to YAML for tree-sitter parsing
	sourceYAML, err := yaml.Marshal(source.Object)
	if err != nil {
		return "", 0, fmt.Errorf("failed to marshal source to YAML: %w", err)
	}

	targetYAML, err := yaml.Marshal(target.Object)
	if err != nil {
		return "", 0, fmt.Errorf("failed to marshal target to YAML: %w", err)
	}

	// Parse YAML with tree-sitter
	parser := treesitter.NewParser()
	defer parser.Close()

	sourceTree, err := parser.ParseYAML(sourceYAML)
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse source YAML: %w", err)
	}

	targetTree, err := parser.ParseYAML(targetYAML)
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse target YAML: %w", err)
	}

	// Perform diff
	differ := treesitter.NewDiffer(sourceTree, targetTree, sourceYAML, targetYAML)
	diffResult, err := differ.Diff()
	if err != nil {
		return "", 0, fmt.Errorf("failed to generate tree-sitter diff: %w", err)
	}

	// Format output
	formatter := treesitter.NewFormatter(d.options.TreeSitterWidth, d.options.ColorOutput, 2)
	sourceLabel := fmt.Sprintf("a/%s", key.String())
	targetLabel := fmt.Sprintf("b/%s", key.String())
	diffText := formatter.FormatSideBySide(diffResult, sourceLabel, targetLabel)

	// Count diff lines (approximate - count newlines)
	diffLines := bytes.Count([]byte(diffText), []byte("\n"))

	return diffText, diffLines, nil
}

# Implementation Plan: Git-Style Diff for kyt with Pager Support

## Summary of Decisions

✅ Use `go-udiff` library for unified diff generation
✅ Remove tree-sitter entirely
✅ Remove JSON/YAML reporters - only unified diff
✅ Add configurable pager support (simple string config)
✅ Respect `$PAGER` environment variable
✅ Rename `diff.cli` → `diff.options`
✅ Default: no pager, colorized unified diff to terminal
✅ Add `--stat` flag for summary
✅ Add `-U<n>` flag for context lines
✅ Add `--color=auto|always|never` flag
✅ Git-style exit codes (0=no diff, 1=has diff)
✅ Remove `--display`, `--show-identical` flags

---

## Phase 1: Dependencies & Cleanup

### 1.1 Add go-udiff
```bash
go get github.com/aymanbagabas/go-udiff@latest
```

### 1.2 Remove tree-sitter
- Delete `pkg/differ/treesitter/` directory
- Run `go mod tidy`

---

## Phase 2: Type System Updates

### 2.1 `pkg/differ/types.go`

```go
type ChangeType string

const (
    ChangeTypeAdded    ChangeType = "added"
    ChangeTypeRemoved  ChangeType = "removed"
    ChangeTypeModified ChangeType = "modified"
)

type ResourceDiff struct {
    // Identification
    SourceKey   *manifest.ResourceKey
    TargetKey   *manifest.ResourceKey
    
    // Content
    Source      *unstructured.Unstructured
    Target      *unstructured.Unstructured
    
    // Metadata
    ChangeType      ChangeType
    MatchType       string
    SimilarityScore float64
    
    // Diff output
    DiffText    string
    Edits       []udiff.Edit
    
    // Per-resource stats (for --stat)
    Insertions  int  // Lines added in this resource
    Deletions   int  // Lines deleted in this resource
}

type DiffResult struct {
    Changes []ResourceDiff  // Only Added, Removed, Modified
    Summary DiffSummary
}

type DiffSummary struct {
    // Resource counts only (not line counts)
    Added     int  // Resources only in target
    Removed   int  // Resources only in source
    Modified  int  // Resources that differ
    Identical int  // Resources that are identical (count only, keys not stored)
}

type DiffOptions struct {
    ContextLines               int
    StringSimilarityThreshold  float64
}
```

### 2.2 `pkg/config/types.go`

```go
type DiffConfig struct {
    IgnoreDifferences []ResourceIgnoreDifferences `yaml:"ignoreDifferences"`
    Normalization     NormalizationConfig         `yaml:"normalization"`
    Options           OptionsConfig               `yaml:"options"`
    Pager             string                      `yaml:"pager"`
}

type OptionsConfig struct {
    ContextLines              int     `yaml:"contextLines"`
    StringSimilarityThreshold float64 `yaml:"stringSimilarityThreshold"`
}

func NewDefaultOptionsConfig() OptionsConfig {
    return OptionsConfig{
        ContextLines:              3,
        StringSimilarityThreshold: 0.6,
    }
}

func NewDefaultDiffConfig() DiffConfig {
    return DiffConfig{
        Options: NewDefaultOptionsConfig(),
        Pager:   "", // No pager by default
    }
}
```

---

## Phase 3: Pager Implementation

### 3.1 Create `pkg/pager/pager.go`

```go
package pager

import (
    "io"
    "os"
    "os/exec"
    "strings"
)

type Pager struct {
    command string
    args    []string
    enabled bool
}

func NewPager(command string) *Pager {
    if command == "" {
        return &Pager{enabled: false}
    }
    
    parts := strings.Fields(command)
    return &Pager{
        command: parts[0],
        args:    parts[1:],
        enabled: true,
    }
}

func (p *Pager) ShouldPage(isStdout bool) bool {
    if !p.enabled || !isStdout {
        return false
    }
    
    fileInfo, err := os.Stdout.Stat()
    if err != nil {
        return false
    }
    
    return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

func (p *Pager) Pipe() (io.WriteCloser, error) {
    cmd := exec.Command(p.command, p.args...)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    
    stdin, err := cmd.StdinPipe()
    if err != nil {
        return nil, err
    }
    
    if err := cmd.Start(); err != nil {
        return nil, err
    }
    
    return &pagerWriter{
        WriteCloser: stdin,
        cmd:         cmd,
    }, nil
}

type pagerWriter struct {
    io.WriteCloser
    cmd *exec.Cmd
}

func (pw *pagerWriter) Close() error {
    if err := pw.WriteCloser.Close(); err != nil {
        return err
    }
    return pw.cmd.Wait()
}
```

---

## Phase 4: Diff Generation with go-udiff

### 4.1 `pkg/differ/differ.go` - Main Logic

```go
func (d *Differ) Diff(sourceManifests, targetManifests []manifest.Manifest) (*DiffResult, error) {
    // Normalize
    normalizedSource, _ := d.normalizer.Normalize(sourceManifests)
    normalizedTarget, _ := d.normalizer.Normalize(targetManifests)
    
    // Build maps
    sourceMap := buildResourceMap(normalizedSource)
    targetMap := buildResourceMap(normalizedTarget)
    
    // Match resources
    matches, unmatchedSource, unmatchedTarget := d.matcher.Match(
        sourceMap, targetMap, d.options.StringSimilarityThreshold,
    )
    
    var changes []ResourceDiff
    identicalCount := 0  // Count only, don't store keys
    
    // Process Added
    for _, key := range unmatchedTarget {
        diff, _ := d.generateAddedDiff(key, targetMap[key])
        changes = append(changes, diff)
    }
    
    // Process Removed
    for _, key := range unmatchedSource {
        diff, _ := d.generateRemovedDiff(key, sourceMap[key])
        changes = append(changes, diff)
    }
    
    // Process Modified
    modifiedCount := 0
    for _, match := range matches {
        sourceRes := sourceMap[match.SourceKey]
        targetRes := targetMap[match.TargetKey]
        
        if d.areEqual(sourceRes, targetRes) {
            identicalCount++  // Just count, don't store
            continue
        }
        
        diff, _ := d.generateModifiedDiff(match, sourceRes, targetRes)
        changes = append(changes, diff)
        modifiedCount++
    }
    
    summary := DiffSummary{
        Added:     len(unmatchedTarget),
        Removed:   len(unmatchedSource),
        Modified:  modifiedCount,
        Identical: identicalCount,
    }
    
    return &DiffResult{
        Changes: changes,
        Summary: summary,
    }, nil
}
```

### 4.2 Diff Generation Methods

```go
func (d *Differ) generateAddedDiff(key manifest.ResourceKey, resource *unstructured.Unstructured) (ResourceDiff, error) {
    targetYAML, _ := d.convertToYAML(resource)
    
    edits := udiff.Strings("", targetYAML)
    unified, _ := udiff.ToUnified(
        "/dev/null",
        fmt.Sprintf("b/%s", key.String()),
        "",
        edits,
        0,
    )
    
    insertions := countLines(targetYAML)
    
    return ResourceDiff{
        TargetKey:  &key,
        Target:     resource,
        ChangeType: ChangeTypeAdded,
        DiffText:   unified,
        Edits:      edits,
        Insertions: insertions,
    }, nil
}

func (d *Differ) generateRemovedDiff(key manifest.ResourceKey, resource *unstructured.Unstructured) (ResourceDiff, error) {
    sourceYAML, _ := d.convertToYAML(resource)
    
    edits := udiff.Strings(sourceYAML, "")
    unified, _ := udiff.ToUnified(
        fmt.Sprintf("a/%s", key.String()),
        "/dev/null",
        sourceYAML,
        edits,
        0,
    )
    
    deletions := countLines(sourceYAML)
    
    return ResourceDiff{
        SourceKey:  &key,
        Source:     resource,
        ChangeType: ChangeTypeRemoved,
        DiffText:   unified,
        Edits:      edits,
        Deletions:  deletions,
    }, nil
}

func (d *Differ) generateModifiedDiff(match Match, source, target *unstructured.Unstructured) (ResourceDiff, error) {
    sourceYAML, _ := d.convertToYAML(source)
    targetYAML, _ := d.convertToYAML(target)
    
    edits := udiff.Strings(sourceYAML, targetYAML)
    unified, _ := udiff.ToUnified(
        fmt.Sprintf("a/%s", match.SourceKey.String()),
        fmt.Sprintf("b/%s", match.TargetKey.String()),
        sourceYAML,
        edits,
        d.options.ContextLines,
    )
    
    insertions, deletions := countChanges(edits, sourceYAML)
    
    return ResourceDiff{
        SourceKey:       &match.SourceKey,
        TargetKey:       &match.TargetKey,
        Source:          source,
        Target:          target,
        ChangeType:      ChangeTypeModified,
        MatchType:       match.Type,
        SimilarityScore: match.Score,
        DiffText:        unified,
        Edits:           edits,
        Insertions:      insertions,
        Deletions:       deletions,
    }, nil
}

// Helper functions
func countLines(s string) int {
    if s == "" {
        return 0
    }
    count := strings.Count(s, "\n")
    if s[len(s)-1] != '\n' {
        count++
    }
    return count
}

func countChanges(edits []udiff.Edit, source string) (insertions, deletions int) {
    for _, edit := range edits {
        if edit.End > edit.Start {
            deleted := source[edit.Start:edit.End]
            deletions += countLines(deleted)
        }
        if edit.New != "" {
            insertions += countLines(edit.New)
        }
    }
    return
}
```

---

## Phase 5: Reporter Implementation

### 5.1 Delete Old Reporters
- Remove `pkg/reporter/json.go`
- Remove `pkg/reporter/yaml.go`
- Remove `pkg/reporter/cli.go`
- Remove `pkg/reporter/diff.go`

### 5.2 Create `pkg/reporter/reporter.go`

```go
package reporter

type Reporter struct {
    showStat bool
    colorize bool
}

func NewReporter(showStat, colorize bool) *Reporter {
    return &Reporter{
        showStat: showStat,
        colorize: colorize,
    }
}

func (r *Reporter) Report(result *differ.DiffResult, w io.Writer) error {
    if r.showStat {
        return r.reportStat(result, w)
    }
    return r.reportDiff(result, w)
}

func (r *Reporter) reportDiff(result *differ.DiffResult, w io.Writer) error {
    for _, change := range result.Changes {
        diffText := change.DiffText
        
        if r.colorize {
            diffText = colorizeUnifiedDiff(diffText)
        }
        
        fmt.Fprint(w, diffText)
    }
    return nil
}

func (r *Reporter) reportStat(result *differ.DiffResult, w io.Writer) error {
    maxKeyLength := 0
    for _, change := range result.Changes {
        keyLength := len(r.getResourceDisplayName(change))
        if keyLength > maxKeyLength {
            maxKeyLength = keyLength
        }
    }
    
    // Print per-resource stats with line changes
    for _, change := range result.Changes {
        r.printResourceStat(w, change, maxKeyLength)
    }
    
    // Print summary line with resource counts (not line counts)
    parts := []string{}
    if result.Summary.Added > 0 {
        parts = append(parts, fmt.Sprintf("%d added", result.Summary.Added))
    }
    if result.Summary.Removed > 0 {
        parts = append(parts, fmt.Sprintf("%d removed", result.Summary.Removed))
    }
    if result.Summary.Modified > 0 {
        parts = append(parts, fmt.Sprintf("%d modified", result.Summary.Modified))
    }
    if result.Summary.Identical > 0 {
        parts = append(parts, fmt.Sprintf("%d identical", result.Summary.Identical))
    }
    
    fmt.Fprintf(w, " %s\n", strings.Join(parts, ", "))
    
    return nil
}

func (r *Reporter) printResourceStat(w io.Writer, change differ.ResourceDiff, maxKeyLength int) error {
    displayName := r.getResourceDisplayName(change)
    total := change.Insertions + change.Deletions
    bar := makeBar(change.Insertions, change.Deletions, 40)
    
    if r.colorize {
        bar = colorizeBar(bar)
    }
    
    // Show per-resource line change stats
    fmt.Fprintf(w, " %-*s | %4d %s\n", maxKeyLength, displayName, total, bar)
    return nil
}

func colorizeUnifiedDiff(diffText string) string {
    const (
        red   = "\033[31m"
        green = "\033[32m"
        cyan  = "\033[36m"
        reset = "\033[0m"
    )
    
    lines := strings.Split(diffText, "\n")
    var result strings.Builder
    
    for i, line := range lines {
        if len(line) == 0 {
            if i < len(lines)-1 {
                result.WriteString("\n")
            }
            continue
        }
        
        switch line[0] {
        case '+':
            if strings.HasPrefix(line, "+++") {
                result.WriteString(cyan + line + reset)
            } else {
                result.WriteString(green + line + reset)
            }
        case '-':
            if strings.HasPrefix(line, "---") {
                result.WriteString(cyan + line + reset)
            } else {
                result.WriteString(red + line + reset)
            }
        case '@':
            if strings.HasPrefix(line, "@@") {
                result.WriteString(cyan + line + reset)
            } else {
                result.WriteString(line)
            }
        default:
            result.WriteString(line)
        }
        
        if i < len(lines)-1 {
            result.WriteString("\n")
        }
    }
    
    return result.String()
}

// Additional helper functions: getResourceDisplayName, makeBar, colorizeBar, pluralize
```

---

## Phase 6: CLI Updates

### 6.1 `cmd/kyt/diff.go` - Flags

```go
var (
    diffOutput                    string
    diffStat                      bool
    diffUnified                   int
    diffColor                     string
    diffStringSimilarityThreshold float64
    diffConfigFiles               []string
)

func init() {
    diffCmd.Flags().StringVarP(&diffOutput, "output", "o", "",
        "Write diff to file instead of stdout")
    
    diffCmd.Flags().BoolVar(&diffStat, "stat", false,
        "Show diffstat summary (like git diff --stat)")
    
    diffCmd.Flags().IntVarP(&diffUnified, "unified", "U", 3,
        "Generate diff with <n> lines of context")
    
    diffCmd.Flags().StringVar(&diffColor, "color", "auto",
        "Colorize output: auto, always, never")
    
    diffCmd.Flags().Float64Var(&diffStringSimilarityThreshold,
        "string-similarity-threshold", 0.0,
        "Similarity threshold (0.0-1.0, 0.0 disables)")
    
    diffCmd.Flags().StringSliceVarP(&diffConfigFiles, "config", "c", nil,
        "Config file(s)")
}
```

### 6.2 Main Diff Logic

```go
func runDiff(cmd *cobra.Command, args []string) error {
    // ... load config, parse manifests ...
    
    // Determine output destination
    var outputWriter io.WriteCloser
    var usePager bool
    
    if diffOutput != "" {
        // Writing to file
        file, _ := os.Create(diffOutput)
        defer file.Close()
        outputWriter = file
        usePager = false
    } else {
        // Writing to stdout - check for pager
        pagerCmd := getPagerCommand(cfg)
        pager := pager.NewPager(pagerCmd)
        
        if pager.ShouldPage(true) {
            pagerWriter, err := pager.Pipe()
            if err != nil {
                // Fallback to stdout
                fmt.Fprintf(os.Stderr, "Warning: pager failed: %v\n", err)
                outputWriter = os.Stdout
                usePager = false
            } else {
                outputWriter = pagerWriter
                usePager = true
            }
        } else {
            outputWriter = os.Stdout
            usePager = false
        }
    }
    defer outputWriter.Close()
    
    // Determine colorization
    colorize := shouldColorize(diffColor, !usePager)
    
    // Get context lines (CLI overrides config)
    contextLines := diffUnified
    if !cmd.Flags().Changed("unified") {
        contextLines = cfg.Diff.Options.ContextLines
    }
    
    // Get similarity threshold (CLI overrides config)
    similarityThreshold := diffStringSimilarityThreshold
    if similarityThreshold == 0.0 {
        similarityThreshold = cfg.Diff.Options.StringSimilarityThreshold
    }
    
    // Create differ
    diffOpts := &differ.DiffOptions{
        ContextLines:              contextLines,
        StringSimilarityThreshold: similarityThreshold,
    }
    differ := differ.NewDiffer(normalizer, diffOpts)
    
    // Perform diff
    result, _ := differ.Diff(sourceManifests, targetManifests)
    
    // Create reporter
    reporter := reporter.NewReporter(diffStat, colorize)
    
    // Generate output
    reporter.Report(result, outputWriter)
    
    // Git-style exit code
    if hasChanges(result) {
        os.Exit(1)
    }
    
    return nil
}

func getPagerCommand(cfg *config.Config) string {
    // Priority: config > $PAGER
    if cfg.Diff.Pager != "" {
        return cfg.Diff.Pager
    }
    return os.Getenv("PAGER")
}

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

func hasChanges(result *differ.DiffResult) bool {
    return len(result.Changes) > 0
}
```

---

## Phase 7: Config & Validation Updates

### 7.1 Update `pkg/config/validator.go`

```go
func (v *Validator) ValidateOptionsConfig(cfg OptionsConfig) error {
    if cfg.ContextLines < 0 {
        return fmt.Errorf("contextLines must be non-negative, got %d", cfg.ContextLines)
    }
    
    if cfg.StringSimilarityThreshold < 0 || cfg.StringSimilarityThreshold > 1 {
        return fmt.Errorf("stringSimilarityThreshold must be between 0 and 1, got %f",
            cfg.StringSimilarityThreshold)
    }
    
    return nil
}
```

### 7.2 Update Example Configs

```yaml
# examples/.kyt.yaml
diff:
  ignoreDifferences:
    - group: ""
      kind: "Service"
      jsonPointers:
        - /spec/clusterIP
  
  normalization:
    sortKeys: true
    removeTimestamps: true
  
  options:
    contextLines: 3
    stringSimilarityThreshold: 0.6
  
  # Optional: pipe output through external diff viewer
  # Only used when stdout is a terminal
  pager: ""  # Examples: "delta --side-by-side", "bat --language=diff"
```

---

## Phase 8: Documentation

### 8.1 README.md

```markdown
## Diff Command

Show differences between Kubernetes manifests in git-style unified diff format.

### Basic Usage

```bash
kyt diff v1/deployment.yaml v2/deployment.yaml
```

Output:
```diff
--- a/apps/v1/Deployment/default/myapp
+++ b/apps/v1/Deployment/default/myapp
@@ -5,7 +5,7 @@
   namespace: default
 spec:
-  replicas: 2
+  replicas: 3
```

### Summary Statistics

```bash
kyt diff --stat v1/ v2/
```

Output:
```
 apps/v1/Deployment/default/frontend        |   12 +++--
 apps/v1/Deployment/default/backend         |   45 ++++++++++----
 v1/Service/default/api (new)               |   67 ++++++++++++++++++++
 v1/ConfigMap/default/old-config (deleted)  |   23 --------
 2 added, 1 removed, 3 modified, 7 identical
```

### Output Options

```bash
# Write to file
kyt diff -o changes.diff v1/ v2/

# Control context lines
kyt diff -U5 v1/ v2/

# Disable colors
kyt diff --color=never v1/ v2/
```

### Enhanced Visualization with Pagers

Configure a pager in `.kyt.yaml`:

```yaml
diff:
  pager: "delta --side-by-side"
```

Popular pagers:
- **delta**: Side-by-side diffs with syntax highlighting
- **diff-so-fancy**: Enhanced unified diffs
- **bat**: Syntax-highlighted viewing

### Exit Codes

- `0` - No differences
- `1` - Differences found
- `>1` - Error occurred
```

### 8.2 Migration Guide

```markdown
## Breaking Changes

### Removed Flags
- `--display` → Removed (use pager for visualization)
- `--output json|yaml` → Use `-o file.diff` or pipe
- `--show-identical` → Removed
- `--no-color` → Use `--color=never`

### Config Changes
```yaml
# Before
diff:
  cli:
    display: side-by-side
    colorize: true

# After
diff:
  options:
    contextLines: 3
  pager: "delta --side-by-side"  # Optional
```

### Exit Codes
Now returns `1` when differences found (was `0`)
```

---

## Phase 9: Testing

### 9.1 Unit Tests
- `pkg/differ/differ_test.go` - Test Added/Removed/Modified diff generation
- `pkg/reporter/reporter_test.go` - Test full diff and --stat output
- `pkg/pager/pager_test.go` - Test pager logic and TTY detection

### 9.2 Integration Tests
- `cmd/kyt/diff_test.go` - Test exit codes, flag combinations, pager integration

---

## Implementation Estimate

| Phase | Component | Time |
|-------|-----------|------|
| 1 | Dependencies & cleanup | 30 min |
| 2 | Type system updates | 1 hour |
| 3 | Pager implementation | 2 hours |
| 4 | Diff generation with go-udiff | 3 hours |
| 5 | Reporter implementation | 2 hours |
| 6 | CLI updates | 2 hours |
| 7 | Config & validation | 1 hour |
| 8 | Documentation | 2 hours |
| 9 | Testing | 3 hours |
| **Total** | | **~16 hours** |

---

## Design Rationale

### Why go-udiff?
- **Cross-platform**: No dependency on system `diff` command
- **Performance**: No process spawning, no temp files
- **Structured data**: Get hunks, lines, and OpKind directly
- **Zero dependencies**: Pure Go implementation

### Why Remove Tree-Sitter?
- **Normalization**: kyt already sorts keys and removes transient fields
- **Simplicity**: Line-by-line diff is sufficient after normalization
- **Tooling**: Unified diff works with all existing diff tools
- **Maintainability**: Less code to maintain

### Why Pager Support?
- **Git-like UX**: Users expect this behavior
- **Flexibility**: Users can choose their preferred viewer
- **Simple**: ~2 hours implementation for significant UX improvement
- **Optional**: Disabled by default, opt-in via config

### Why Remove JSON/YAML Output?
- **Standard format**: Unified diff is universal
- **Tool ecosystem**: Can pipe to diff2html, delta, etc.
- **Less complexity**: One output format to maintain
- **Git alignment**: Matches git's philosophy

### Resource-Level vs Line-Level Stats
- **Per-resource stats**: Show line insertions/deletions for each resource (in `--stat` output)
- **Summary stats**: Show resource counts only (added/removed/modified/identical)
- **Rationale**: Kubernetes resources are distinct units (like database records), not accumulations of line changes
- **Git comparison**: Git shows line changes across files; kyt shows line changes per resource but counts resources in summary
- **User mental model**: "I added 2 Services, modified 1 Deployment" is clearer than "45 insertions across resources"

---

## Next Steps

1. Review and approve this plan
2. Create feature branch: `feature/git-style-diff`
3. Implement phases 1-9 in order
4. Create PR with comprehensive testing
5. Update CHANGELOG with breaking changes

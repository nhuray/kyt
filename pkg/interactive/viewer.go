package interactive

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

// Viewer provides interactive diff viewing with fzf and delta
type Viewer struct {
	fzfPath   string
	deltaPath string
}

// NewViewer creates a new interactive viewer
// Returns an error if fzf or delta are not available
func NewViewer() (*Viewer, error) {
	fzfPath, err := exec.LookPath("fzf")
	if err != nil {
		return nil, fmt.Errorf("fzf not found in PATH. Install with: brew install fzf")
	}

	deltaPath, err := exec.LookPath("delta")
	if err != nil {
		return nil, fmt.Errorf("delta not found in PATH. Install with: brew install git-delta")
	}

	return &Viewer{
		fzfPath:   fzfPath,
		deltaPath: deltaPath,
	}, nil
}

// Resource represents a single resource in the diff
type Resource struct {
	Name      string // e.g., "ConfigMap.core/redis-configuration (namespace: redis-ha)"
	StartLine int
	EndLine   int
	Content   string
}

// Show displays the diff interactively using fzf for selection and delta for viewing
func (v *Viewer) Show(diffOutput []byte) error {
	if len(diffOutput) == 0 {
		fmt.Println("No differences found")
		return nil
	}

	// Parse the diff to extract resources
	resources, err := v.parseResources(diffOutput)
	if err != nil {
		return fmt.Errorf("failed to parse diff: %w", err)
	}

	if len(resources) == 0 {
		// No structured resources, just show the whole diff
		return v.showWithDelta(diffOutput)
	}

	// Show interactive selector
	return v.showInteractive(resources, diffOutput)
}

// parseResources extracts individual resources from the diff output
func (v *Viewer) parseResources(diffOutput []byte) ([]Resource, error) {
	scanner := bufio.NewScanner(bytes.NewReader(diffOutput))

	// Regex to match resource headers: "--- a/Kind.group/name (namespace: ns)"
	resourceRegex := regexp.MustCompile(`^(---|\+\+\+) (a/|b/)([^[:space:]]+) \(namespace: ([^)]+)\)`)

	var resources []Resource
	resourceMap := make(map[string]*Resource)
	lineNum := 0
	var currentBuf bytes.Buffer

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		currentBuf.WriteString(line)
		currentBuf.WriteByte('\n')

		// Check if this is a resource header
		matches := resourceRegex.FindStringSubmatch(line)
		if matches != nil {
			resourceID := matches[3] + " (namespace: " + matches[4] + ")"

			// If we already have this resource, update end line
			if res, exists := resourceMap[resourceID]; exists {
				res.EndLine = lineNum
			} else {
				// New resource
				res := &Resource{
					Name:      resourceID,
					StartLine: lineNum,
					EndLine:   lineNum,
				}
				resourceMap[resourceID] = res
				resources = append(resources, *res)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Extract content for each resource
	lines := strings.Split(string(diffOutput), "\n")
	for i := range resources {
		res := &resources[i]
		if res.StartLine > 0 && res.EndLine <= len(lines) {
			res.Content = strings.Join(lines[res.StartLine-1:res.EndLine], "\n")
		}
	}

	// Sort by name
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Name < resources[j].Name
	})

	return resources, nil
}

// showInteractive shows the fzf selector with delta preview
func (v *Viewer) showInteractive(resources []Resource, fullDiff []byte) error {
	// Create a temp file for the full diff
	tmpFile, err := os.CreateTemp("", "kyt-diff-*.txt")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write(fullDiff); err != nil {
		return fmt.Errorf("failed to write diff to temp file: %w", err)
	}

	tmpFile.Close()

	// Prepare resource names for fzf
	resourceNames := make([]string, len(resources))
	for i, res := range resources {
		resourceNames[i] = res.Name
	}

	// Build fzf command with delta preview
	fzfCmd := exec.Command(v.fzfPath,
		"--prompt=Select resource (Enter=view all, Esc=quit): ",
		"--preview="+v.buildPreviewCommand(tmpFile.Name()),
		"--preview-window=right:70%:wrap",
		"--bind=ctrl-/:toggle-preview",
		"--bind=enter:execute("+v.deltaPath+" --side-by-side --line-numbers --paging=always < "+tmpFile.Name()+")+abort",
		"--header=Enter=view all | Ctrl-/=toggle preview | Esc=quit",
		"--height=100%",
		"--ansi",
	)

	// Pipe resource names to fzf
	stdin, err := fzfCmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	fzfCmd.Stdout = os.Stdout
	fzfCmd.Stderr = os.Stderr

	if err := fzfCmd.Start(); err != nil {
		return fmt.Errorf("failed to start fzf: %w", err)
	}

	// Write resource names to fzf
	go func() {
		defer stdin.Close()
		for _, name := range resourceNames {
			fmt.Fprintln(stdin, name)
		}
	}()

	if err := fzfCmd.Wait(); err != nil {
		// Exit code 130 means user pressed Esc, which is fine
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			return nil
		}
		// Exit code 0 means user pressed Enter (view all), which triggers the bind action
		return nil
	}

	return nil
}

// buildPreviewCommand creates the preview command for fzf
func (v *Viewer) buildPreviewCommand(tmpFile string) string {
	// Use grep to find the resource section and pipe to delta
	return fmt.Sprintf("grep -A 50 '{}' %s | %s --side-by-side --line-numbers --paging=never --width $FZF_PREVIEW_COLUMNS",
		tmpFile, v.deltaPath)
}

// showWithDelta displays content through delta
func (v *Viewer) showWithDelta(content []byte) error {
	cmd := exec.Command(v.deltaPath, "--side-by-side", "--line-numbers", "--paging=always")
	cmd.Stdin = bytes.NewReader(content)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

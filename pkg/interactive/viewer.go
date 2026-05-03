package interactive

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/nhuray/kyt/pkg/differ"
)

// Viewer provides interactive diff viewing with tmux, fzf, and delta
type Viewer struct {
	fzfPath   string
	deltaPath string
	tmuxPath  string
}

// NewViewer creates a new interactive viewer
// Returns an error if fzf, delta, or tmux are not available
func NewViewer() (*Viewer, error) {
	fzfPath, err := exec.LookPath("fzf")
	if err != nil {
		return nil, fmt.Errorf("fzf not found in PATH. Install with: brew install fzf")
	}

	deltaPath, err := exec.LookPath("delta")
	if err != nil {
		return nil, fmt.Errorf("delta not found in PATH. Install with: brew install git-delta")
	}

	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return nil, fmt.Errorf("tmux not found in PATH. Install with: brew install tmux")
	}

	return &Viewer{
		fzfPath:   fzfPath,
		deltaPath: deltaPath,
		tmuxPath:  tmuxPath,
	}, nil
}

// Show displays the diff interactively in tmux windows with fzf and delta
func (v *Viewer) Show(result *differ.DiffResult) error {
	if !result.HasDifferences() {
		fmt.Println("No differences found")
		return nil
	}

	// Categorize resources
	added := result.GetAdded()
	modified := result.GetModified()
	removed := result.GetRemoved()

	// Sort each category by resource name
	sortByName := func(resources []differ.ResourceDiff) {
		sort.Slice(resources, func(i, j int) bool {
			return getResourceName(resources[i]) < getResourceName(resources[j])
		})
	}
	sortByName(added)
	sortByName(modified)
	sortByName(removed)

	// Show summary
	fmt.Fprintf(os.Stderr, "Starting interactive mode:\n")
	fmt.Fprintf(os.Stderr, "  Modified: %d resources\n", len(modified))
	fmt.Fprintf(os.Stderr, "  Added:    %d resources\n", len(added))
	fmt.Fprintf(os.Stderr, "  Removed:  %d resources\n", len(removed))
	fmt.Fprintf(os.Stderr, "\nNavigation:\n")
	fmt.Fprintf(os.Stderr, "  Ctrl-b 0/1/2: Switch between Modified/Added/Removed windows\n")
	fmt.Fprintf(os.Stderr, "  Ctrl-t:       Toggle preview\n")
	fmt.Fprintf(os.Stderr, "  Esc:          Quit\n\n")

	// Create and launch tmux session
	return v.createTmuxSession(added, modified, removed)
}

// getResourceName returns a display name for a resource
func getResourceName(rd differ.ResourceDiff) string {
	// Use TargetKey for added resources, SourceKey for removed, either for modified
	if rd.ChangeType == differ.ChangeTypeAdded && rd.TargetKey != nil {
		return fmt.Sprintf("%s.%s/%s (namespace: %s)",
			rd.TargetKey.Kind, rd.TargetKey.Group, rd.TargetKey.Name, rd.TargetKey.Namespace)
	} else if rd.ChangeType == differ.ChangeTypeRemoved && rd.SourceKey != nil {
		return fmt.Sprintf("%s.%s/%s (namespace: %s)",
			rd.SourceKey.Kind, rd.SourceKey.Group, rd.SourceKey.Name, rd.SourceKey.Namespace)
	} else if rd.TargetKey != nil {
		return fmt.Sprintf("%s.%s/%s (namespace: %s)",
			rd.TargetKey.Kind, rd.TargetKey.Group, rd.TargetKey.Name, rd.TargetKey.Namespace)
	} else if rd.SourceKey != nil {
		return fmt.Sprintf("%s.%s/%s (namespace: %s)",
			rd.SourceKey.Kind, rd.SourceKey.Group, rd.SourceKey.Name, rd.SourceKey.Namespace)
	}
	return "unknown"
}

// generateCleanDiff creates clean YAML (no +/-) for added/removed, keeps diff for modified
func (v *Viewer) generateCleanDiff(rd differ.ResourceDiff) string {
	if rd.ChangeType == differ.ChangeTypeModified {
		// Keep diff as-is for modified resources
		return rd.DiffText
	}

	// For added/removed, strip +/- prefixes to show clean YAML
	lines := strings.Split(rd.DiffText, "\n")
	var cleaned []string

	for _, line := range lines {
		// Skip diff headers (---, +++, @@)
		if strings.HasPrefix(line, "---") ||
			strings.HasPrefix(line, "+++") ||
			strings.HasPrefix(line, "@@") {
			continue
		}

		// Remove leading +/- but keep the rest
		if len(line) > 0 && (line[0] == '+' || line[0] == '-') {
			cleaned = append(cleaned, line[1:])
		} else {
			cleaned = append(cleaned, line)
		}
	}

	return strings.Join(cleaned, "\n")
}

// createTmuxSession creates a tmux session with 3 windows (Modified, Added, Removed)
func (v *Viewer) createTmuxSession(added, modified, removed []differ.ResourceDiff) error {
	sessionName := fmt.Sprintf("kyt-diff-%d", time.Now().Unix())

	// Create detached session with first window (Modified - most important)
	cmd := exec.Command(v.tmuxPath, "new-session", "-d", "-s", sessionName, "-n", "Modified")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Setup window 0: Modified Resources
	if err := v.launchWindowWithFzf(sessionName, 0, "Modified", modified, differ.ChangeTypeModified); err != nil {
		return err
	}

	// Create window 1: Added Resources
	cmd = exec.Command(v.tmuxPath, "new-window", "-t", sessionName, "-n", "Added")
	if err := cmd.Run(); err != nil {
		return err
	}
	if err := v.launchWindowWithFzf(sessionName, 1, "Added", added, differ.ChangeTypeAdded); err != nil {
		return err
	}

	// Create window 2: Removed Resources
	cmd = exec.Command(v.tmuxPath, "new-window", "-t", sessionName, "-n", "Removed")
	if err := cmd.Run(); err != nil {
		return err
	}
	if err := v.launchWindowWithFzf(sessionName, 2, "Removed", removed, differ.ChangeTypeRemoved); err != nil {
		return err
	}

	// Select window 0 (Modified) as default
	cmd = exec.Command(v.tmuxPath, "select-window", "-t", fmt.Sprintf("%s:0", sessionName))
	if err := cmd.Run(); err != nil {
		return err
	}

	// Attach to the session (this blocks until user exits)
	cmd = exec.Command(v.tmuxPath, "attach-session", "-t", sessionName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Cleanup session even on error
		exec.Command(v.tmuxPath, "kill-session", "-t", sessionName).Run()
		return fmt.Errorf("tmux session error: %w", err)
	}

	// Cleanup: kill session after detach
	exec.Command(v.tmuxPath, "kill-session", "-t", sessionName).Run()

	return nil
}

// launchWindowWithFzf sets up a tmux window with fzf (left pane) and delta preview (right pane)
func (v *Viewer) launchWindowWithFzf(sessionName string, windowIndex int, windowName string, resources []differ.ResourceDiff, changeType differ.ChangeType) error {
	target := fmt.Sprintf("%s:%d", sessionName, windowIndex)

	if len(resources) == 0 {
		// Show a message in empty windows
		msg := fmt.Sprintf("No %s resources", strings.ToLower(string(changeType)))
		cmd := exec.Command(v.tmuxPath, "send-keys", "-t", target,
			fmt.Sprintf("echo '%s' && echo '' && echo 'Press Ctrl-b 0/1/2 to switch windows, or Ctrl-d to exit' && bash", msg), "C-m")
		return cmd.Run()
	}

	// Create temp directory for this window's files
	tmpDir, err := os.MkdirTemp("", fmt.Sprintf("kyt-%s-*", changeType))
	if err != nil {
		return err
	}

	// Write resource list file and individual diff files
	resourceListFile := filepath.Join(tmpDir, "resources.txt")
	f, err := os.Create(resourceListFile)
	if err != nil {
		return err
	}

	for i, rd := range resources {
		// Write resource name to list
		name := getResourceName(rd)
		fmt.Fprintln(f, name)

		// Write individual diff file
		diffFile := filepath.Join(tmpDir, fmt.Sprintf("resource_%d.diff", i+1))
		cleanDiff := v.generateCleanDiff(rd)
		if err := os.WriteFile(diffFile, []byte(cleanDiff), 0644); err != nil {
			f.Close()
			return err
		}
	}
	f.Close()

	// Split window vertically: left 30% for fzf, right 70% for delta
	cmd := exec.Command(v.tmuxPath, "split-window", "-t", target, "-h", "-p", "70")
	if err := cmd.Run(); err != nil {
		return err
	}

	// Build fzf command with delta preview
	previewCmd := fmt.Sprintf("%s --side-by-side --line-numbers --paging=never --width $COLUMNS %s/resource_{n}.diff",
		v.deltaPath, tmpDir)

	header := fmt.Sprintf("%s: %d resources | Ctrl-t=toggle preview | Esc=quit | Ctrl-b 0/1/2=switch window",
		windowName, len(resources))

	fzfCmd := fmt.Sprintf("%s --prompt='%s: ' --header='%s' --preview='%s' --preview-window=right:70%%:wrap --height=100%% --bind=ctrl-t:toggle-preview < %s",
		v.fzfPath, windowName, header, previewCmd, resourceListFile)

	// Send fzf command to left pane (pane 0)
	cmd = exec.Command(v.tmuxPath, "send-keys", "-t",
		fmt.Sprintf("%s.0", target), // .0 is left pane
		fzfCmd, "C-m")

	return cmd.Run()
}

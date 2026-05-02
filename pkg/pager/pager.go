package pager

import (
	"io"
	"os"
	"os/exec"
	"strings"
)

// Pager handles piping output through an external pager command
type Pager struct {
	command string
	args    []string
	enabled bool
}

// NewPager creates a new Pager from a command string
// Example: "delta --side-by-side" or "less -R"
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

// ShouldPage determines if output should be piped through the pager
// Only pages when:
// - Pager is enabled (command not empty)
// - Output is going to stdout (not redirected)
// - Stdout is a TTY (terminal)
func (p *Pager) ShouldPage(isStdout bool) bool {
	if !p.enabled || !isStdout {
		return false
	}

	// Check if stdout is a terminal
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// Pipe spawns the pager process and returns a WriteCloser
// The caller should write to this writer and close it when done
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

// pagerWriter wraps the pager's stdin and waits for the command to finish on close
type pagerWriter struct {
	io.WriteCloser
	cmd *exec.Cmd
}

func (pw *pagerWriter) Close() error {
	// Close stdin to signal end of input
	if err := pw.WriteCloser.Close(); err != nil {
		return err
	}

	// Wait for pager to finish
	return pw.cmd.Wait()
}

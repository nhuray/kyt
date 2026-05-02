package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestCLI_DiffIdenticalManifests tests comparing identical manifests
func TestCLI_DiffIdenticalManifests(t *testing.T) {
	// Build the binary first
	buildCmd := exec.Command("go", "build", "-o", "../../bin/kyt-test", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer func() { _ = os.Remove("../../bin/kyt-test") }()

	// Run the comparison
	cmd := exec.Command("../../bin/kyt-test",
		"diff",
		"../../examples/manifests/basic",
		"../../examples/manifests/basic")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Should exit with code 0 (no differences)
	if err != nil {
		t.Errorf("Expected exit code 0, got error: %v\nStderr: %s", err, stderr.String())
	}

	output := stdout.String()
	// With no differences, unified diff format produces no output
	// This is expected behavior (like git diff with no changes)
	if output != "" {
		t.Logf("Output for identical manifests: %q", output)
	}
}

// TestCLI_DiffDifferentManifests tests comparing different manifests
func TestCLI_DiffDifferentManifests(t *testing.T) {
	buildCmd := exec.Command("go", "build", "-o", "../../bin/kyt-test", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer func() { _ = os.Remove("../../bin/kyt-test") }()

	cmd := exec.Command("../../bin/kyt-test",
		"diff",
		"../../examples/manifests/basic",
		"../../examples/manifests/multi-doc")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Should exit with code 1 (differences found)
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() != 1 {
			t.Errorf("Expected exit code 1, got %d", exitErr.ExitCode())
		}
	} else if err == nil {
		t.Error("Expected exit code 1, got 0")
	}

	output := stdout.String()
	if output == "" {
		t.Error("Expected output, got empty string")
	}

	// Check for unified diff format markers
	if !bytes.Contains([]byte(output), []byte("---")) {
		t.Error("Expected '---' marker in unified diff output")
	}
	if !bytes.Contains([]byte(output), []byte("+++")) {
		t.Error("Expected '+++' marker in unified diff output")
	}
}

// TestCLI_DiffFileOutput tests writing output to a file
func TestCLI_DiffFileOutput(t *testing.T) {
	buildCmd := exec.Command("go", "build", "-o", "../../bin/kyt-test", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer func() { _ = os.Remove("../../bin/kyt-test") }()

	// Create temp file for output
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "diff.txt")

	cmd := exec.Command("../../bin/kyt-test",
		"diff",
		"-o", outputFile,
		"../../examples/manifests/basic",
		"../../examples/manifests/multi-doc")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	_ = cmd.Run() // Ignore exit code for this test

	// Check that output file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatalf("Expected output file to be created at %s", outputFile)
	}

	// Read output file
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Check that file contains diff output
	if !bytes.Contains(content, []byte("---")) || !bytes.Contains(content, []byte("+++")) {
		t.Error("Expected unified diff format in output file")
	}
}

// TestCLI_DiffSummary tests --summary flag
func TestCLI_DiffSummary(t *testing.T) {
	buildCmd := exec.Command("go", "build", "-o", "../../bin/kyt-test", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer func() { _ = os.Remove("../../bin/kyt-test") }()

	cmd := exec.Command("../../bin/kyt-test",
		"diff",
		"--summary",
		"../../examples/manifests/basic",
		"../../examples/manifests/multi-doc")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	_ = cmd.Run() // Ignore exit code for this test

	output := stdout.String()

	// Should show summary table header
	if !bytes.Contains([]byte(output), []byte("KIND")) {
		t.Error("Expected 'KIND' header in summary output")
	}
	if !bytes.Contains([]byte(output), []byte("SUMMARY:")) {
		t.Error("Expected 'SUMMARY:' line in summary output")
	}
}

// TestCLI_DiffInvalidYAML tests error handling for invalid YAML
func TestCLI_DiffInvalidYAML(t *testing.T) {
	buildCmd := exec.Command("go", "build", "-o", "../../bin/kyt-test", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer func() { _ = os.Remove("../../bin/kyt-test") }()

	// Create a temp file with invalid YAML
	tmpDir := t.TempDir()
	invalidYAML := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(invalidYAML, []byte("invalid: yaml: content: ["), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	cmd := exec.Command("../../bin/kyt-test",
		"diff",
		"../../examples/manifests/basic",
		invalidYAML)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Should exit with code 2 (error)
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() != 2 {
			t.Errorf("Expected exit code 2 for invalid YAML, got %d", exitErr.ExitCode())
		}
	} else if err == nil {
		t.Error("Expected exit code 2 for invalid YAML, got 0")
	}

	// Should have error message
	stderrStr := stderr.String()
	if stderrStr == "" {
		t.Error("Expected error message on stderr, got empty string")
	}
}

// TestCLI_DiffMissingFile tests error handling for missing files
func TestCLI_DiffMissingFile(t *testing.T) {
	buildCmd := exec.Command("go", "build", "-o", "../../bin/kyt-test", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer func() { _ = os.Remove("../../bin/kyt-test") }()

	cmd := exec.Command("../../bin/kyt-test",
		"diff",
		"../../examples/manifests/basic",
		"../../examples/manifests/nonexistent")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Should exit with code 2 (error)
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() != 2 {
			t.Errorf("Expected exit code 2 for missing file, got %d", exitErr.ExitCode())
		}
	} else if err == nil {
		t.Error("Expected exit code 2 for missing file, got 0")
	}

	// Should have error message
	stderrStr := stderr.String()
	if stderrStr == "" {
		t.Error("Expected error message on stderr, got empty string")
	}
}

// TestCLI_Version tests the version command
func TestCLI_Version(t *testing.T) {
	buildCmd := exec.Command("go", "build", "-o", "../../bin/kyt-test", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer func() { _ = os.Remove("../../bin/kyt-test") }()

	cmd := exec.Command("../../bin/kyt-test", "version")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("Expected exit code 0 for version command, got error: %v", err)
	}

	output := stdout.String()

	// Should contain version info
	if !bytes.Contains([]byte(output), []byte("kyt")) {
		t.Errorf("Expected 'kyt' in version output, got: %s", output)
	}
}

// TestCLI_FmtFile tests fmt command with a file
func TestCLI_FmtFile(t *testing.T) {
	buildCmd := exec.Command("go", "build", "-o", "../../bin/kyt-test", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer func() { _ = os.Remove("../../bin/kyt-test") }()

	cmd := exec.Command("../../bin/kyt-test",
		"fmt",
		"../../examples/manifests/basic/deployment.yaml")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("Expected exit code 0 for fmt command, got error: %v\nStderr: %s", err, stderr.String())
	}

	output := stdout.String()
	if output == "" {
		t.Error("Expected YAML output, got empty string")
	}

	// Should contain YAML content
	if !bytes.Contains([]byte(output), []byte("apiVersion:")) {
		t.Errorf("Expected YAML content in output, got: %s", output)
	}
}

// TestCLI_FmtStdin tests fmt command with stdin
func TestCLI_FmtStdin(t *testing.T) {
	buildCmd := exec.Command("go", "build", "-o", "../../bin/kyt-test", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer func() { _ = os.Remove("../../bin/kyt-test") }()

	// Read sample YAML file
	yamlContent, err := os.ReadFile("../../examples/manifests/basic/deployment.yaml")
	if err != nil {
		t.Fatalf("Failed to read sample YAML: %v", err)
	}

	cmd := exec.Command("../../bin/kyt-test", "fmt")
	cmd.Stdin = bytes.NewReader(yamlContent)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Fatalf("Expected exit code 0 for fmt command, got error: %v\nStderr: %s", err, stderr.String())
	}

	output := stdout.String()
	if output == "" {
		t.Error("Expected YAML output, got empty string")
	}

	// Should contain YAML content
	if !bytes.Contains([]byte(output), []byte("apiVersion:")) {
		t.Errorf("Expected YAML content in output, got: %s", output)
	}
}

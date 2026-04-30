package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestCLI_DiffIdenticalManifests tests comparing identical manifests
func TestCLI_DiffIdenticalManifests(t *testing.T) {
	// Build the binary first
	buildCmd := exec.Command("go", "build", "-o", "../../bin/ky-test", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer func() { _ = os.Remove("../../bin/ky-test") }()

	// Run the comparison
	cmd := exec.Command("../../bin/ky-test",
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
	if output == "" {
		t.Error("Expected output, got empty string")
	}

	// Check for success message
	if !bytes.Contains([]byte(output), []byte("No differences found")) {
		t.Errorf("Expected 'No differences found' in output, got: %s", output)
	}
}

// TestCLI_DiffDifferentManifests tests comparing different manifests
func TestCLI_DiffDifferentManifests(t *testing.T) {
	buildCmd := exec.Command("go", "build", "-o", "../../bin/ky-test", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer func() { _ = os.Remove("../../bin/ky-test") }()

	cmd := exec.Command("../../bin/ky-test",
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

	// Check for differences message
	if !bytes.Contains([]byte(output), []byte("Differences detected")) {
		t.Errorf("Expected 'Differences detected' in output, got: %s", output)
	}

	// Check for added/removed resources
	if !bytes.Contains([]byte(output), []byte("Added Resources")) {
		t.Error("Expected 'Added Resources' section in output")
	}
	if !bytes.Contains([]byte(output), []byte("Removed Resources")) {
		t.Error("Expected 'Removed Resources' section in output")
	}
}

// TestCLI_DiffJSONOutput tests JSON output format
func TestCLI_DiffJSONOutput(t *testing.T) {
	buildCmd := exec.Command("go", "build", "-o", "../../bin/ky-test", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer func() { _ = os.Remove("../../bin/ky-test") }()

	cmd := exec.Command("../../bin/ky-test",
		"diff",
		"-o", "json",
		"../../examples/manifests/basic",
		"../../examples/manifests/multi-doc")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	_ = cmd.Run() // Ignore exit code for this test

	// Parse JSON output
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, stdout.String())
	}

	// Check for expected fields
	if _, ok := result["summary"]; !ok {
		t.Error("Expected 'summary' field in JSON output")
	}
	if _, ok := result["added"]; !ok {
		t.Error("Expected 'added' field in JSON output")
	}
	if _, ok := result["removed"]; !ok {
		t.Error("Expected 'removed' field in JSON output")
	}
	if _, ok := result["modified"]; !ok {
		t.Error("Expected 'modified' field in JSON output")
	}

	// Check summary counts
	summary := result["summary"].(map[string]interface{})
	if summary["added"].(float64) != 3 {
		t.Errorf("Expected 3 added resources, got %v", summary["added"])
	}
	if summary["removed"].(float64) != 2 {
		t.Errorf("Expected 2 removed resources, got %v", summary["removed"])
	}
}

// TestCLI_DiffShowIdentical tests --show-identical flag
func TestCLI_DiffShowIdentical(t *testing.T) {
	buildCmd := exec.Command("go", "build", "-o", "../../bin/ky-test", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer func() { _ = os.Remove("../../bin/ky-test") }()

	cmd := exec.Command("../../bin/ky-test",
		"diff",
		"--show-identical",
		"../../examples/manifests/basic",
		"../../examples/manifests/basic")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("Expected exit code 0, got error: %v", err)
	}

	output := stdout.String()

	// Should show identical resources
	if !bytes.Contains([]byte(output), []byte("Identical Resources")) {
		t.Errorf("Expected 'Identical Resources' section in output with --show-identical, got: %s", output)
	}
}

// TestCLI_DiffInvalidYAML tests error handling for invalid YAML
func TestCLI_DiffInvalidYAML(t *testing.T) {
	buildCmd := exec.Command("go", "build", "-o", "../../bin/ky-test", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer func() { _ = os.Remove("../../bin/ky-test") }()

	// Create a temp file with invalid YAML
	tmpDir := t.TempDir()
	invalidYAML := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(invalidYAML, []byte("invalid: yaml: content: ["), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	cmd := exec.Command("../../bin/ky-test",
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
	buildCmd := exec.Command("go", "build", "-o", "../../bin/ky-test", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer func() { _ = os.Remove("../../bin/ky-test") }()

	cmd := exec.Command("../../bin/ky-test",
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
	buildCmd := exec.Command("go", "build", "-o", "../../bin/ky-test", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer func() { _ = os.Remove("../../bin/ky-test") }()

	cmd := exec.Command("../../bin/ky-test", "version")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("Expected exit code 0 for version command, got error: %v", err)
	}

	output := stdout.String()

	// Should contain version info
	if !bytes.Contains([]byte(output), []byte("ky")) {
		t.Errorf("Expected 'ky' in version output, got: %s", output)
	}
}

// TestCLI_FmtFile tests fmt command with a file
func TestCLI_FmtFile(t *testing.T) {
	buildCmd := exec.Command("go", "build", "-o", "../../bin/ky-test", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer func() { _ = os.Remove("../../bin/ky-test") }()

	cmd := exec.Command("../../bin/ky-test",
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
	buildCmd := exec.Command("go", "build", "-o", "../../bin/ky-test", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer func() { _ = os.Remove("../../bin/ky-test") }()

	// Read sample YAML file
	yamlContent, err := os.ReadFile("../../examples/manifests/basic/deployment.yaml")
	if err != nil {
		t.Fatalf("Failed to read sample YAML: %v", err)
	}

	cmd := exec.Command("../../bin/ky-test", "fmt")
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

package main

import (
	"fmt"
	"io"
	"os"

	"github.com/nhuray/kyt/pkg/manifest"
	"github.com/nhuray/kyt/pkg/normalizer"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var (
	// Lint command flags
	lintWrite bool
)

var lintCmd = &cobra.Command{
	Use:   "lint [path]",
	Short: "Normalize Kubernetes manifests",
	Long: `Normalize Kubernetes manifests by applying ignore rules and transformations.

By default, reads from the specified path and writes to stdout.
Use -w to write changes back to the source files.

Examples:
  # Normalize a file to stdout
  kyt lint deployment.yaml

  # Normalize a directory to stdout
  kyt lint ./manifests

  # Normalize and write back to source files
  kyt lint -w ./manifests

  # Normalize from stdin
  cat deployment.yaml | kyt lint

  # Chain with other tools
  kustomize build . | kyt lint | kubectl apply -f -
`,
	RunE:          runLint,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	lintCmd.Flags().BoolVarP(&lintWrite, "write", "w", false, "write normalized output back to source files")
	rootCmd.AddCommand(lintCmd)
}

func runLint(cmd *cobra.Command, args []string) error {
	var input io.Reader
	var sourcePath string

	if len(args) == 0 {
		// Read from stdin
		input = os.Stdin
		sourcePath = "<stdin>"
		if rootVerbose {
			fmt.Fprintf(os.Stderr, "Reading from stdin...\n")
		}

		// Cannot use -w with stdin
		if lintWrite {
			return fmt.Errorf("cannot use -w/--write with stdin input")
		}
	} else {
		// Read from file/directory
		sourcePath = args[0]
		if rootVerbose {
			fmt.Fprintf(os.Stderr, "Reading from: %s\n", sourcePath)
		}
	}

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Parse manifests
	parser := manifest.NewParser()
	var manifestSet *manifest.ManifestSet

	if input != nil {
		// Parse from reader (stdin)
		manifestSet, err = parser.ParseReader(input)
		if err != nil {
			return fmt.Errorf("failed to parse stdin: %w", err)
		}
	} else {
		// Parse from file or directory
		info, err := os.Stat(sourcePath)
		if err != nil {
			return fmt.Errorf("failed to stat path: %w", err)
		}

		if info.IsDir() {
			manifestSet, err = parser.ParseDirectory(sourcePath)
		} else {
			manifestSet, err = parser.ParseFile(sourcePath)
		}

		if err != nil {
			return fmt.Errorf("failed to parse manifests: %w", err)
		}
	}

	if rootVerbose {
		fmt.Fprintf(os.Stderr, "Found %d resources\n", manifestSet.Len())
	}

	// Normalize all resources
	norm := normalizer.New(cfg)
	normalized := make([]*unstructured.Unstructured, 0, manifestSet.Len())

	for _, obj := range manifestSet.Resources {
		n, err := norm.Normalize(obj)
		if err != nil {
			key := manifest.NewResourceKey(obj)
			return fmt.Errorf("failed to normalize %s: %w", key.String(), err)
		}
		normalized = append(normalized, n)
	}

	if rootVerbose {
		fmt.Fprintf(os.Stderr, "Normalized %d resources\n", len(normalized))
	}

	// Write output
	if lintWrite {
		// Write back to source files
		if rootVerbose {
			fmt.Fprintf(os.Stderr, "Writing normalized manifests back to source...\n")
		}

		// For now, we'll write all normalized resources back to the original path
		// TODO: In a more advanced implementation, we could preserve the original
		// file structure when dealing with directories
		return writeBackToSource(sourcePath, normalized)
	} else {
		// Write to stdout
		if rootVerbose {
			fmt.Fprintf(os.Stderr, "Writing to stdout...\n")
		}
		return manifest.WriteYAML(os.Stdout, normalized)
	}
}

// writeBackToSource writes normalized manifests back to the source file(s)
func writeBackToSource(sourcePath string, resources []*unstructured.Unstructured) error {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to stat path: %w", err)
	}

	if info.IsDir() {
		// For directories, we write all resources to a single file in the directory
		// In a production implementation, you might want to preserve the original file structure
		outputPath := sourcePath + "/normalized.yaml"
		file, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer func() {
			if cerr := file.Close(); cerr != nil && err == nil {
				err = fmt.Errorf("failed to close output file: %w", cerr)
			}
		}()

		if err := manifest.WriteYAML(file, resources); err != nil {
			return fmt.Errorf("failed to write YAML: %w", err)
		}

		if rootVerbose {
			fmt.Fprintf(os.Stderr, "Wrote normalized manifests to: %s\n", outputPath)
		}
	} else {
		// For single files, write back to the same file
		file, err := os.Create(sourcePath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer func() {
			if cerr := file.Close(); cerr != nil && err == nil {
				err = fmt.Errorf("failed to close output file: %w", cerr)
			}
		}()

		if err := manifest.WriteYAML(file, resources); err != nil {
			return fmt.Errorf("failed to write YAML: %w", err)
		}

		if rootVerbose {
			fmt.Fprintf(os.Stderr, "Wrote normalized manifests to: %s\n", sourcePath)
		}
	}

	return nil
}

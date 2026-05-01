package main

import (
	"fmt"
	"io"
	"os"

	"github.com/nhuray/kyt/pkg/formatter"
	"github.com/nhuray/kyt/pkg/manifest"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var (
	// Fmt command flags
	fmtWrite bool
)

var fmtCmd = &cobra.Command{
	Use:   "fmt [path]",
	Short: "Format Kubernetes manifests",
	Long: `Format Kubernetes manifests by applying transformations like sorting keys and arrays.

By default, reads from the specified path and writes to stdout.
Use -w to write changes back to the source files.

Examples:
  # Format a file to stdout
  kyt fmt deployment.yaml

  # Format a directory to stdout
  kyt fmt ./manifests

  # Format and write back to source files
  kyt fmt -w ./manifests

  # Format from stdin
  cat deployment.yaml | kyt fmt

  # Chain with other tools
  kustomize build . | kyt fmt | kubectl apply -f -
`,
	RunE:          runFmt,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	fmtCmd.Flags().BoolVarP(&fmtWrite, "write", "w", false, "write formatted output back to source files")
	rootCmd.AddCommand(fmtCmd)
}

func runFmt(cmd *cobra.Command, args []string) error {
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
		if fmtWrite {
			return fmt.Errorf("cannot use -w/--write with stdin input")
		}
	} else {
		// Read from file/directory
		sourcePath = args[0]
		if rootVerbose {
			fmt.Fprintf(os.Stderr, "Reading from: %s\n", sourcePath)
		}
	}

	// Parse manifests
	parser := manifest.NewParser()
	var manifestSet *manifest.ManifestSet
	var err error

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

	// Format all resources (only key sorting, no normalization)
	fmtr := formatter.New()
	formatted := make([]*unstructured.Unstructured, 0, manifestSet.Len())

	for _, obj := range manifestSet.Resources {
		f, err := fmtr.Format(obj)
		if err != nil {
			key := manifest.NewResourceKey(obj)
			return fmt.Errorf("failed to format %s: %w", key.String(), err)
		}
		formatted = append(formatted, f)
	}

	if rootVerbose {
		fmt.Fprintf(os.Stderr, "Formatted %d resources\n", len(formatted))
	}

	// Write output
	if fmtWrite {
		// Write back to source files
		if rootVerbose {
			fmt.Fprintf(os.Stderr, "Writing formatted manifests back to source...\n")
		}

		// For now, we'll write all formatted resources back to the original path
		// TODO: In a more advanced implementation, we could preserve the original
		// file structure when dealing with directories
		return writeBackToSource(sourcePath, formatted)
	} else {
		// Write to stdout
		if rootVerbose {
			fmt.Fprintf(os.Stderr, "Writing to stdout...\n")
		}
		return manifest.WriteYAML(os.Stdout, formatted)
	}
}

// writeBackToSource writes formatted manifests back to the source file(s)
func writeBackToSource(sourcePath string, resources []*unstructured.Unstructured) error {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to stat path: %w", err)
	}

	if info.IsDir() {
		// For directories, we write all resources to a single file in the directory
		// In a production implementation, you might want to preserve the original file structure
		outputPath := sourcePath + "/formatted.yaml"
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
			fmt.Fprintf(os.Stderr, "Wrote formatted manifests to: %s\n", outputPath)
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
			fmt.Fprintf(os.Stderr, "Wrote formatted manifests to: %s\n", sourcePath)
		}
	}

	return nil
}

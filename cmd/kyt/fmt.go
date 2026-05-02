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
	fmtWrite                  bool
	fmtRemoveServerSideFields bool
)

var fmtCmd = &cobra.Command{
	Use:   "fmt [path]",
	Short: "Format Kubernetes manifests",
	Long: `Format Kubernetes manifests by applying transformations like sorting keys and arrays.

By default, reads from the specified path and writes to stdout.
Use -w to write changes back to the source files.
Use -R to remove server-side fields (useful for formatting applied manifests).

Examples:
  # Format a file to stdout
  kyt fmt deployment.yaml

  # Format a directory to stdout
  kyt fmt ./manifests

  # Format and write back to source files
  kyt fmt -w ./manifests

  # Format from stdin
  cat deployment.yaml | kyt fmt

  # Remove server-side fields from applied manifests
  kubectl get deploy my-app -o yaml | kyt fmt -R

  # Clean and format manifests from the cluster
  kyt fmt -R applied-manifests/

  # Chain with other tools
  kustomize build . | kyt fmt | kubectl apply -f -
`,
	RunE:          runFmt,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	fmtCmd.Flags().BoolVarP(&fmtWrite, "write", "w", false, "write formatted output back to source files")
	fmtCmd.Flags().BoolVarP(&fmtRemoveServerSideFields, "remove-server-side-fields", "R", false, "remove server-side fields (managedFields, resourceVersion, uid, etc.)")
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
	formattedSet := manifest.NewManifestSet()

	for key, obj := range manifestSet.Resources {
		// Remove server-side fields if requested
		if fmtRemoveServerSideFields {
			cleanServerSideFields(obj)
		}

		f, err := fmtr.Format(obj)
		if err != nil {
			return fmt.Errorf("failed to format %s: %w", key.String(), err)
		}
		// Preserve source file information
		if sourcePath, ok := manifestSet.GetSourceFile(key); ok {
			if err := formattedSet.AddWithSource(f, sourcePath); err != nil {
				return fmt.Errorf("failed to add formatted resource %s: %w", key.String(), err)
			}
		} else {
			if err := formattedSet.Add(f); err != nil {
				return fmt.Errorf("failed to add formatted resource %s: %w", key.String(), err)
			}
		}
	}

	if rootVerbose {
		fmt.Fprintf(os.Stderr, "Formatted %d resources\n", formattedSet.Len())
	}

	// Write output
	if fmtWrite {
		// Write back to source files
		if rootVerbose {
			fmt.Fprintf(os.Stderr, "Writing formatted manifests back to source...\n")
		}

		return writeBackToSource(sourcePath, formattedSet)
	} else {
		// Write to stdout
		if rootVerbose {
			fmt.Fprintf(os.Stderr, "Writing to stdout...\n")
		}
		// Convert to slice for WriteYAML
		resources := make([]*unstructured.Unstructured, 0, formattedSet.Len())
		for _, obj := range formattedSet.Resources {
			resources = append(resources, obj)
		}
		return manifest.WriteYAML(os.Stdout, resources)
	}
}

// writeBackToSource writes formatted manifests back to the source file(s)
func writeBackToSource(sourcePath string, manifestSet *manifest.ManifestSet) error {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to stat path: %w", err)
	}

	if info.IsDir() {
		// For directories, group resources by source file and write each group back
		groupedResources := manifestSet.GroupBySourceFile()

		if len(groupedResources) == 0 {
			return fmt.Errorf("no resources to write")
		}

		// Write each group back to its source file
		for filePath, resources := range groupedResources {
			if filePath == "" {
				// Skip resources without source file info
				continue
			}

			file, err := os.Create(filePath)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", filePath, err)
			}

			if err := manifest.WriteYAML(file, resources); err != nil {
				_ = file.Close() // Best effort to close on error
				return fmt.Errorf("failed to write YAML to %s: %w", filePath, err)
			}

			if err := file.Close(); err != nil {
				return fmt.Errorf("failed to close file %s: %w", filePath, err)
			}

			if rootVerbose {
				fmt.Fprintf(os.Stderr, "Wrote %d resources to: %s\n", len(resources), filePath)
			}
		}
	} else {
		// For single files, write all resources back to the same file
		resources := make([]*unstructured.Unstructured, 0, manifestSet.Len())
		for _, obj := range manifestSet.Resources {
			resources = append(resources, obj)
		}

		file, err := os.Create(sourcePath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}

		if err := manifest.WriteYAML(file, resources); err != nil {
			_ = file.Close() // Best effort to close on error
			return fmt.Errorf("failed to write YAML: %w", err)
		}

		if err := file.Close(); err != nil {
			return fmt.Errorf("failed to close output file: %w", err)
		}

		if rootVerbose {
			fmt.Fprintf(os.Stderr, "Wrote %d resources to: %s\n", len(resources), sourcePath)
		}
	}

	return nil
}

// cleanServerSideFields removes server-side fields that are added by Kubernetes
// when resources are applied to the cluster. This is useful for formatting manifests
// that were retrieved from the cluster (e.g., via kubectl get).
func cleanServerSideFields(obj *unstructured.Unstructured) {
	if obj == nil {
		return
	}

	// Get metadata
	metadata, ok := obj.Object["metadata"].(map[string]interface{})
	if !ok {
		return
	}

	// Remove server-side metadata fields
	fieldsToRemove := []string{
		"managedFields",     // Field management information
		"resourceVersion",   // Resource version
		"uid",               // Unique identifier
		"selfLink",          // Deprecated self link
		"generation",        // Generation number
		"creationTimestamp", // Creation timestamp
	}

	for _, field := range fieldsToRemove {
		delete(metadata, field)
	}

	// Also remove status field from the root object
	delete(obj.Object, "status")
}

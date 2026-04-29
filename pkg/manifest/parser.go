package manifest

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	// yamlSeparator is the standard separator for multi-document YAML files
	yamlSeparator = "---"
)

// Parser handles parsing Kubernetes manifests from various sources
type Parser struct {
	// SkipInvalid determines whether to skip invalid resources or return an error
	SkipInvalid bool
}

// NewParser creates a new Parser with default settings
func NewParser() *Parser {
	return &Parser{
		SkipInvalid: false,
	}
}

// ParseFile parses a single YAML file into a ManifestSet
// The file may contain one or more YAML documents separated by "---"
func (p *Parser) ParseFile(path string) (*ManifestSet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	return p.ParseBytes(data)
}

// ParseBytes parses YAML data from a byte slice into a ManifestSet
func (p *Parser) ParseBytes(data []byte) (*ManifestSet, error) {
	manifestSet := NewManifestSet()

	// Split by YAML document separator
	documents := p.splitYAMLDocuments(data)

	for i, doc := range documents {
		// Skip empty documents
		if len(bytes.TrimSpace(doc)) == 0 {
			continue
		}

		obj, err := p.parseYAMLDocument(doc)
		if err != nil {
			if p.SkipInvalid {
				continue
			}
			return nil, fmt.Errorf("failed to parse document %d: %w", i+1, err)
		}

		// Skip if the document didn't parse to a valid K8s resource
		if obj == nil {
			continue
		}

		if err := manifestSet.Add(obj); err != nil {
			return nil, fmt.Errorf("failed to add resource from document %d: %w", i+1, err)
		}
	}

	return manifestSet, nil
}

// ParseDirectory recursively parses all YAML files in a directory
// Files with extensions .yaml, .yml are processed
func (p *Parser) ParseDirectory(path string) (*ManifestSet, error) {
	manifestSet := NewManifestSet()

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process YAML files
		ext := strings.ToLower(filepath.Ext(filePath))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		fileManifests, err := p.ParseFile(filePath)
		if err != nil {
			if p.SkipInvalid {
				return nil // Continue walking even if a file fails
			}
			return fmt.Errorf("failed to parse file %s: %w", filePath, err)
		}

		if err := manifestSet.Merge(fileManifests); err != nil {
			return fmt.Errorf("failed to merge manifests from %s: %w", filePath, err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return manifestSet, nil
}

// ParseReader parses YAML data from an io.Reader into a ManifestSet
func (p *Parser) ParseReader(r io.Reader) (*ManifestSet, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	return p.ParseBytes(data)
}

// splitYAMLDocuments splits multi-document YAML by "---" separator
func (p *Parser) splitYAMLDocuments(data []byte) [][]byte {
	var documents [][]byte

	// Split by newline to process line by line
	lines := bytes.Split(data, []byte("\n"))

	var currentDoc [][]byte
	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)

		// Check if this line is a document separator
		if string(trimmed) == yamlSeparator {
			// Save the current document if it has content
			if len(currentDoc) > 0 {
				documents = append(documents, bytes.Join(currentDoc, []byte("\n")))
				currentDoc = nil
			}
			continue
		}

		currentDoc = append(currentDoc, line)
	}

	// Add the last document
	if len(currentDoc) > 0 {
		documents = append(documents, bytes.Join(currentDoc, []byte("\n")))
	}

	return documents
}

// parseYAMLDocument parses a single YAML document into an unstructured.Unstructured
func (p *Parser) parseYAMLDocument(data []byte) (*unstructured.Unstructured, error) {
	// First unmarshal into a generic map
	var rawObj map[string]interface{}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(false) // Allow unknown fields (Kubernetes is extensible)

	if err := decoder.Decode(&rawObj); err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to decode YAML: %w", err)
	}

	// Skip empty documents
	if len(rawObj) == 0 {
		return nil, nil
	}

	// Create unstructured object
	obj := &unstructured.Unstructured{Object: rawObj}

	// Validate that this is a Kubernetes resource
	if err := p.validateK8sResource(obj); err != nil {
		return nil, err
	}

	return obj, nil
}

// validateK8sResource checks if the object has required Kubernetes fields
func (p *Parser) validateK8sResource(obj *unstructured.Unstructured) error {
	// Check for required fields
	gvk := obj.GroupVersionKind()
	if gvk.Kind == "" {
		return fmt.Errorf("missing required field: kind")
	}

	// apiVersion is required (checked via GVK)
	apiVersion := obj.GetAPIVersion()
	if apiVersion == "" {
		return fmt.Errorf("missing required field: apiVersion")
	}

	// metadata.name is required for all resources
	name := obj.GetName()
	if name == "" {
		return fmt.Errorf("missing required field: metadata.name")
	}

	return nil
}

// ParseFiles parses multiple files into a single ManifestSet
func (p *Parser) ParseFiles(paths []string) (*ManifestSet, error) {
	manifestSet := NewManifestSet()

	for _, path := range paths {
		fileManifests, err := p.ParseFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to parse file %s: %w", path, err)
		}

		if err := manifestSet.Merge(fileManifests); err != nil {
			return nil, fmt.Errorf("failed to merge manifests from %s: %w", path, err)
		}
	}

	return manifestSet, nil
}

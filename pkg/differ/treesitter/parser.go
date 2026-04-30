package treesitter

import (
	"context"
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/yaml"
)

// Parser wraps tree-sitter parser for YAML
type Parser struct {
	parser *sitter.Parser
}

// NewParser creates a new Parser for YAML
func NewParser() *Parser {
	parser := sitter.NewParser()
	parser.SetLanguage(yaml.GetLanguage())
	return &Parser{parser: parser}
}

// ParseYAML parses YAML bytes into a tree-sitter tree
func (p *Parser) ParseYAML(content []byte) (*sitter.Tree, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if tree == nil {
		return nil, fmt.Errorf("parser returned nil tree")
	}

	// Check for syntax errors
	root := tree.RootNode()
	if root.HasError() {
		return nil, fmt.Errorf("YAML contains syntax errors")
	}

	return tree, nil
}

// Close releases parser resources
func (p *Parser) Close() {
	if p.parser != nil {
		p.parser.Close()
	}
}

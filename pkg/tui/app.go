package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nhuray/kyt/pkg/differ"
	"github.com/nhuray/kyt/pkg/tui/diff"
)

// Run starts the TUI application
func Run(result *differ.DiffResult, leftSource, rightSource string) error {
	// Create model
	m := NewModel(result, leftSource, rightSource)

	// Initialize table with default dimensions (will be rebuilt by WindowSizeMsg with actual terminal size)
	m.table = m.buildTable()

	// Initialize viewport
	m.viewport = viewport.New(120, 30)
	m.viewport.KeyMap = viewport.KeyMap{
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "f", " "),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "b"),
		),
		HalfPageUp: key.NewBinding(
			key.WithKeys("u", "ctrl+u"),
		),
		HalfPageDown: key.NewBinding(
			key.WithKeys("d", "ctrl+d"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
		),
	}

	// Initialize renderer
	m.renderer = diff.NewRenderer(120, diff.ModeSideBySide)

	// Create program with alt screen
	p := tea.NewProgram(m, tea.WithAltScreen())

	// Run program
	_, err := p.Run()
	return err
}

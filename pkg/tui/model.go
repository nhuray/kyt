package tui

import (
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nhuray/kyt/pkg/differ"
	"github.com/nhuray/kyt/pkg/manifest"
	"github.com/nhuray/kyt/pkg/tui/diff"
)

// ViewType represents the current view
type ViewType int

const (
	ViewTable ViewType = iota
	ViewDiff
	ViewHelp
)

// TabType represents the current tab filter
type TabType int

const (
	TabAll TabType = iota
	TabAdded
	TabModified
	TabRemoved
)

// Model represents the TUI application state
type Model struct {
	// State
	currentView ViewType
	currentTab  TabType
	diffMode    diff.DiffMode
	filter      string
	filterMode  bool
	commandMode bool
	commandBuf  string

	// Data
	result       *differ.DiffResult
	leftSource   string
	rightSource  string
	allResources []ResourceRow
	filteredRows []ResourceRow

	// Components
	table    table.Model
	viewport viewport.Model
	renderer *diff.Renderer

	// Current diff
	currentDiff *diff.ParsedDiff

	// Dimensions
	width  int
	height int
}

// ResourceRow represents a row in the resource table
type ResourceRow struct {
	Kind            string
	Name            string
	Namespace       string
	LeftName        string // For All/Modified tabs: namespace/name from left
	RightName       string // For All/Modified tabs: namespace/name from right
	ChangeType      differ.ChangeType
	MatchType       string  // "exact" or "similarity" (for Modified)
	SimilarityScore float64 // 0.0-1.0 (for similarity matches)
	Additions       int
	Deletions       int
	DiffText        string
}

// NewModel creates a new TUI model
func NewModel(result *differ.DiffResult, leftSource, rightSource string) *Model {
	m := &Model{
		currentView: ViewTable,
		currentTab:  TabModified, // Start with Modified tab (most common)
		diffMode:    diff.ModeSideBySide,
		result:      result,
		leftSource:  leftSource,
		rightSource: rightSource,
	}

	// Build resource rows from result
	m.allResources = buildResourceRows(result)
	m.applyFilter()

	// Initialize components
	m.renderer = diff.NewRenderer(120, diff.ModeSideBySide)

	return m
}

// buildResourceRows converts DiffResult to ResourceRow slice
func buildResourceRows(result *differ.DiffResult) []ResourceRow {
	rows := []ResourceRow{}

	for _, change := range result.Changes {
		row := ResourceRow{
			ChangeType:      change.ChangeType,
			MatchType:       change.MatchType,
			SimilarityScore: change.SimilarityScore,
			DiffText:        change.DiffText,
			Additions:       change.Insertions,
			Deletions:       change.Deletions,
		}

		// Get resource details from SourceKey or TargetKey
		if change.SourceKey != nil {
			row.Kind = change.SourceKey.Kind
			row.LeftName = formatResourceName(*change.SourceKey)
			// For simple display (Added/Removed tabs)
			row.Name = change.SourceKey.Name
			row.Namespace = change.SourceKey.Namespace
		}

		if change.TargetKey != nil {
			if row.Kind == "" {
				row.Kind = change.TargetKey.Kind
			}
			row.RightName = formatResourceName(*change.TargetKey)
			// Override simple display for added resources
			if change.ChangeType == differ.ChangeTypeAdded {
				row.Name = change.TargetKey.Name
				row.Namespace = change.TargetKey.Namespace
			}
		}

		rows = append(rows, row)
	}

	// Sort by kind, then name
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Kind != rows[j].Kind {
			return rows[i].Kind < rows[j].Kind
		}
		return rows[i].Name < rows[j].Name
	})

	return rows
}

// formatResourceName formats a resource key as namespace/name
func formatResourceName(key manifest.ResourceKey) string {
	if key.Namespace == "" {
		return key.Name
	}
	return key.Namespace + "/" + key.Name
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	return nil
}

// getRowsForTab returns filtered rows based on current tab
func (m *Model) getRowsForTab() []ResourceRow {
	switch m.currentTab {
	case TabAdded:
		return filterByType(m.allResources, differ.ChangeTypeAdded)
	case TabModified:
		return filterByType(m.allResources, differ.ChangeTypeModified)
	case TabRemoved:
		return filterByType(m.allResources, differ.ChangeTypeRemoved)
	default:
		return m.allResources
	}
}

// filterByType filters resources by change type
func filterByType(resources []ResourceRow, changeType differ.ChangeType) []ResourceRow {
	filtered := []ResourceRow{}
	for _, r := range resources {
		if r.ChangeType == changeType {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// applyFilter applies the current filter and tab to resources
func (m *Model) applyFilter() {
	tabRows := m.getRowsForTab()

	if m.filter == "" {
		m.filteredRows = tabRows
		return
	}

	// Filter by name, kind, or namespace
	filtered := []ResourceRow{}
	for _, r := range tabRows {
		if contains(r.Name, m.filter) ||
			contains(r.Kind, m.filter) ||
			contains(r.Namespace, m.filter) {
			filtered = append(filtered, r)
		}
	}
	m.filteredRows = filtered
}

// contains checks if s contains substr (case-insensitive)
func contains(s, substr string) bool {
	// Simple substring check
	return len(substr) > 0 && len(s) >= len(substr) &&
		(s == substr || strings.Contains(strings.ToLower(s), strings.ToLower(substr)))
}

// getTabCounts returns counts for each tab
func (m *Model) getTabCounts() (all, added, modified, removed int) {
	all = len(m.allResources)
	for _, r := range m.allResources {
		switch r.ChangeType {
		case differ.ChangeTypeAdded:
			added++
		case differ.ChangeTypeModified:
			modified++
		case differ.ChangeTypeRemoved:
			removed++
		}
	}
	return
}

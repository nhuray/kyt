package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nhuray/kyt/pkg/differ"
	"github.com/nhuray/kyt/pkg/tui/diff"
)

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Update table size
		m.table.SetWidth(msg.Width)
		m.table.SetHeight(msg.Height - 8) // Leave room for header/footer

		// Update viewport size
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 6

		// Update renderer width
		m.renderer.Width = msg.Width

		return m, nil
	}

	return m, nil
}

// handleKeyPress routes key presses based on current view/mode
func (m *Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle search mode first
	if m.searchMode {
		return m.handleSearchKeys(msg)
	}

	// Route based on view
	switch m.currentView {
	case ViewTable:
		return m.handleTableKeys(msg)
	case ViewDiff:
		return m.handleDiffKeys(msg)
	case ViewHelp:
		return m.handleHelpKeys(msg)
	}

	return m, nil
}

// handleTableKeys handles keyboard input in table view
func (m *Model) handleTableKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {

	case "ctrl+c":
		return m, tea.Quit

	case ":":
		// Start command mode for :q
		m.searchMode = true
		m.search = ":"
		return m, nil

	case "q":
		// Back/escape - clear search if active, otherwise do nothing
		if m.searchMode {
			m.searchMode = false
			m.search = ""
			m.applyFilter()
			m.table = m.buildTable()
		}
		return m, nil

	// Tab navigation
	case "0", "1", "2", "3":
		tabNum := int(msg.String()[0] - '0')
		m.currentTab = TabType(tabNum)
		m.applyFilter()
		m.table = m.buildTable()
		return m, nil

	case "h", "left":
		// Navigate to previous tab
		if m.currentTab > TabAll {
			m.currentTab--
			m.applyFilter()
			m.table = m.buildTable()
		}
		return m, nil

	case "l", "right":
		// Navigate to next tab
		if m.currentTab < TabRemoved {
			m.currentTab++
			m.applyFilter()
			m.table = m.buildTable()
		}
		return m, nil

	// Row navigation
	case "k", "up":
		m.table.MoveUp(1)
		return m, nil

	case "j", "down":
		m.table.MoveDown(1)
		return m, nil

	case "g":
		// Go to top
		m.table.GotoTop()
		return m, nil

	case "G":
		// Go to bottom
		m.table.GotoBottom()
		return m, nil

	case "ctrl+f", "pgdown":
		// Page down
		m.table.MoveDown(m.height - 10)
		return m, nil

	case "ctrl+b", "pgup":
		// Page up
		m.table.MoveUp(m.height - 10)
		return m, nil

	// Sorting
	case "N":
		m.sortField = SortByName
		m.applyFilter()
		m.table = m.buildTable()
		return m, nil

	case "S":
		m.sortField = SortByStatus
		m.applyFilter()
		m.table = m.buildTable()
		return m, nil

	// Search
	case "/":
		m.searchMode = true
		m.search = ""
		return m, nil

	// Actions
	case "enter":
		// View selected resource diff
		if len(m.filteredRows) == 0 {
			return m, nil
		}
		cursor := m.table.Cursor()
		if cursor < len(m.filteredRows) {
			return m.showDiff(m.filteredRows[cursor])
		}
		return m, nil

	case "?":
		// Show help
		m.currentView = ViewHelp
		return m, nil

	default:
		// No default delegation to table for j/k since we handle them explicitly
		return m, nil
	}
}

// handleDiffKeys handles keyboard input in diff view
func (m *Model) handleDiffKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {

	case "ctrl+c":
		return m, tea.Quit

	case "q", "esc":
		// Go back to table view
		m.currentView = ViewTable
		return m, nil

	case "s":
		// Toggle to side-by-side mode
		m.diffMode = diff.ModeSideBySide
		m.renderer.Mode = diff.ModeSideBySide
		m.reloadDiff()
		return m, nil

	case "u":
		// Toggle to unified mode
		m.diffMode = diff.ModeUnified
		m.renderer.Mode = diff.ModeUnified
		m.reloadDiff()
		return m, nil

	case "k", "up":
		m.viewport.LineUp(1)
		return m, nil

	case "j", "down":
		m.viewport.LineDown(1)
		return m, nil

	case "ctrl+f", "pgdown":
		m.viewport.ViewDown()
		return m, nil

	case "ctrl+b", "pgup":
		m.viewport.ViewUp()
		return m, nil

	case "g":
		m.viewport.GotoTop()
		return m, nil

	case "G":
		m.viewport.GotoBottom()
		return m, nil

	default:
		return m, nil
	}
}

// handleHelpKeys handles keyboard input in help view
func (m *Model) handleHelpKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q", "esc", "?":
		m.currentView = ViewTable
		return m, nil
	}
	return m, nil
}

// handleSearchKeys handles keyboard input in search mode
func (m *Model) handleSearchKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {

	case "esc", "ctrl+c", "q":
		// Cancel search
		m.searchMode = false
		m.search = ""
		m.applyFilter()
		m.table = m.buildTable()
		return m, nil

	case "enter":
		// Check if it's :q command to quit
		if m.search == ":q" {
			return m, tea.Quit
		}
		// Apply search
		m.searchMode = false
		m.applyFilter()
		m.table = m.buildTable()
		return m, nil

	case "backspace":
		// Remove last character
		if len(m.search) > 0 {
			m.search = m.search[:len(m.search)-1]
		}
		return m, nil

	case "ctrl+u":
		// Clear entire search
		m.search = ""
		return m, nil

	default:
		// Add character to search
		if len(msg.String()) == 1 {
			m.search += msg.String()
		}
		return m, nil
	}
}

// showDiff displays the diff for a resource
func (m *Model) showDiff(row ResourceRow) (*Model, tea.Cmd) {
	m.currentView = ViewDiff

	// Parse the diff
	parsed, err := diff.ParseUnifiedDiff(row.DiffText)
	if err != nil {
		// Handle error - maybe show error message
		return m, nil
	}
	m.currentDiff = parsed

	// Render with current mode
	content := m.renderer.Render(parsed)

	// Setup viewport
	m.viewport.SetContent(content)
	m.viewport.GotoTop()

	return m, nil
}

// reloadDiff re-renders the current diff with new mode
func (m *Model) reloadDiff() {
	if m.currentDiff != nil {
		content := m.renderer.Render(m.currentDiff)
		m.viewport.SetContent(content)
	}
}

// buildTable builds the table component from filtered rows
func (m *Model) buildTable() table.Model {
	var columns []table.Column
	var rows []table.Row

	// Define columns based on current tab
	switch m.currentTab {
	case TabAll:
		// All tab: CHANGE, KIND, LEFT, RIGHT, MATCH TYPE, SIMILARITY SCORE, DIFF
		columns = []table.Column{
			{Title: "CHANGE", Width: 8},
			{Title: "KIND", Width: 15},
			{Title: "LEFT", Width: 30},
			{Title: "RIGHT", Width: 30},
			{Title: "MATCH TYPE", Width: 12},
			{Title: "SIMILARITY", Width: 11},
			{Title: "DIFF", Width: 12},
		}

		for _, r := range m.filteredRows {
			matchType := ""
			similarity := ""
			if r.ChangeType == differ.ChangeTypeModified {
				matchType = r.MatchType
				if r.MatchType == "similarity" {
					similarity = fmt.Sprintf("%.2f", r.SimilarityScore)
				}
			}

			rows = append(rows, table.Row{
				formatChangeIndicator(r.ChangeType),
				r.Kind,
				r.LeftName,
				r.RightName,
				matchType,
				similarity,
				formatDiff(r),
			})
		}

	case TabAdded:
		// Added tab: KIND, NAME, NAMESPACE, DIFF
		columns = []table.Column{
			{Title: "KIND", Width: 20},
			{Title: "NAME", Width: 40},
			{Title: "NAMESPACE", Width: 25},
			{Title: "DIFF", Width: 15},
		}

		for _, r := range m.filteredRows {
			rows = append(rows, table.Row{
				r.Kind,
				r.Name,
				r.Namespace,
				formatDiff(r),
			})
		}

	case TabModified:
		// Modified tab: KIND, LEFT, RIGHT, MATCH TYPE, SIMILARITY SCORE, DIFF
		columns = []table.Column{
			{Title: "KIND", Width: 15},
			{Title: "LEFT", Width: 30},
			{Title: "RIGHT", Width: 30},
			{Title: "MATCH TYPE", Width: 12},
			{Title: "SIMILARITY", Width: 11},
			{Title: "DIFF", Width: 12},
		}

		for _, r := range m.filteredRows {
			similarity := ""
			if r.MatchType == "similarity" {
				similarity = fmt.Sprintf("%.2f", r.SimilarityScore)
			}

			rows = append(rows, table.Row{
				r.Kind,
				r.LeftName,
				r.RightName,
				r.MatchType,
				similarity,
				formatDiff(r),
			})
		}

	case TabRemoved:
		// Removed tab: KIND, NAME, NAMESPACE, DIFF
		columns = []table.Column{
			{Title: "KIND", Width: 20},
			{Title: "NAME", Width: 40},
			{Title: "NAMESPACE", Width: 25},
			{Title: "DIFF", Width: 15},
		}

		for _, r := range m.filteredRows {
			rows = append(rows, table.Row{
				r.Kind,
				r.Name,
				r.Namespace,
				formatDiff(r),
			})
		}
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(m.height-8),
	)

	// Apply styles
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return t
}

// formatChangeIndicator returns the colored change indicator (A/M/R)
func formatChangeIndicator(changeType differ.ChangeType) string {
	const (
		red    = "\033[31m"
		green  = "\033[32m"
		yellow = "\033[33m"
		reset  = "\033[0m"
	)

	switch changeType {
	case differ.ChangeTypeAdded:
		return green + "A" + reset
	case differ.ChangeTypeModified:
		return yellow + "M" + reset
	case differ.ChangeTypeRemoved:
		return red + "R" + reset
	default:
		return ""
	}
}

// formatDiff formats the diff statistics
func formatDiff(r ResourceRow) string {
	const (
		red   = "\033[31m"
		green = "\033[32m"
		reset = "\033[0m"
	)

	switch r.ChangeType {
	case differ.ChangeTypeAdded:
		return green + fmt.Sprintf("+%d", r.Additions) + reset
	case differ.ChangeTypeRemoved:
		return red + fmt.Sprintf("-%d", r.Deletions) + reset
	case differ.ChangeTypeModified:
		if r.Additions > 0 && r.Deletions > 0 {
			return green + fmt.Sprintf("+%d", r.Additions) + reset + " / " + red + fmt.Sprintf("-%d", r.Deletions) + reset
		} else if r.Additions > 0 {
			return green + fmt.Sprintf("+%d", r.Additions) + reset
		} else if r.Deletions > 0 {
			return red + fmt.Sprintf("-%d", r.Deletions) + reset
		}
	}
	return ""
}

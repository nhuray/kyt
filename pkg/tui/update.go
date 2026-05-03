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
	// Handle filter mode first
	if m.filterMode {
		return m.handleFilterKeys(msg)
	}

	// Handle command mode
	if m.commandMode {
		return m.handleCommandKeys(msg)
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

	case "ctrl+c", "q":
		return m, tea.Quit

	case "0", "1", "2", "3":
		// Switch tabs
		tabNum := int(msg.String()[0] - '0')
		m.currentTab = TabType(tabNum)
		m.applyFilter()
		m.table = m.buildTable()
		return m, nil

	case "left":
		// Navigate to previous tab
		if m.currentTab > TabAll {
			m.currentTab--
			m.applyFilter()
			m.table = m.buildTable()
		}
		return m, nil

	case "right":
		// Navigate to next tab
		if m.currentTab < TabRemoved {
			m.currentTab++
			m.applyFilter()
			m.table = m.buildTable()
		}
		return m, nil

	case "/":
		// Start filter mode
		m.filterMode = true
		m.filter = ""
		return m, nil

	case ":":
		// Start command mode
		m.commandMode = true
		m.commandBuf = ""
		return m, nil

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
		// Delegate to table for navigation (↑/↓, j/k, etc.)
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd
	}
}

// handleDiffKeys handles keyboard input in diff view
func (m *Model) handleDiffKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {

	case "ctrl+c", "q":
		return m, tea.Quit

	case "esc":
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

	default:
		// Delegate to viewport for scrolling
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
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

// handleFilterKeys handles keyboard input in filter mode
func (m *Model) handleFilterKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {

	case "esc", "ctrl+c":
		// Cancel filter
		m.filterMode = false
		m.filter = ""
		m.applyFilter()
		m.table = m.buildTable()
		return m, nil

	case "enter":
		// Apply filter
		m.filterMode = false
		m.applyFilter()
		m.table = m.buildTable()
		return m, nil

	case "backspace":
		// Remove last character
		if len(m.filter) > 0 {
			m.filter = m.filter[:len(m.filter)-1]
		}
		return m, nil

	case "ctrl+u":
		// Clear entire filter
		m.filter = ""
		return m, nil

	default:
		// Add character to filter
		if len(msg.String()) == 1 {
			m.filter += msg.String()
		}
		return m, nil
	}
}

// handleCommandKeys handles keyboard input in command mode
func (m *Model) handleCommandKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {

	case "esc", "ctrl+c":
		// Cancel command
		m.commandMode = false
		m.commandBuf = ""
		return m, nil

	case "enter":
		// Execute command
		m.executeCommand(m.commandBuf)
		m.commandMode = false
		m.commandBuf = ""
		m.applyFilter()
		m.table = m.buildTable()
		return m, nil

	case "backspace":
		// Remove last character
		if len(m.commandBuf) > 0 {
			m.commandBuf = m.commandBuf[:len(m.commandBuf)-1]
		}
		return m, nil

	case "ctrl+u":
		// Clear entire command
		m.commandBuf = ""
		return m, nil

	default:
		// Add character to command
		if len(msg.String()) == 1 {
			m.commandBuf += msg.String()
		}
		return m, nil
	}
}

// executeCommand executes a k9s-style command
func (m *Model) executeCommand(cmd string) {
	// Parse command like "cm" for ConfigMaps, "svc" for Services
	kindMap := map[string]string{
		"cm":     "ConfigMap",
		"svc":    "Service",
		"deploy": "Deployment",
		"sts":    "StatefulSet",
		"secret": "Secret",
		"ns":     "Namespace",
		"sa":     "ServiceAccount",
		"po":     "Pod",
		"ing":    "Ingress",
		"pv":     "PersistentVolume",
		"pvc":    "PersistentVolumeClaim",
		"ds":     "DaemonSet",
		"rs":     "ReplicaSet",
	}

	if kind, ok := kindMap[cmd]; ok {
		m.filter = kind
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

package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Styles
var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("57")).
			Padding(0, 1)

	tabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("57")).
			Padding(0, 1)

	tabInactiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Padding(0, 1)

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	filterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("57")).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Padding(1, 2)
)

// View renders the TUI
func (m *Model) View() string {
	switch m.currentView {
	case ViewTable:
		return m.viewTable()
	case ViewDiff:
		return m.viewDiff()
	case ViewHelp:
		return m.viewHelp()
	default:
		return "Unknown view"
	}
}

// viewTable renders the resource table view
func (m *Model) viewTable() string {
	var b strings.Builder

	// Header
	header := fmt.Sprintf("kyt diff: %s vs %s", m.leftSource, m.rightSource)
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n\n")

	// Tabs
	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")

	// Table
	b.WriteString(m.table.View())
	b.WriteString("\n\n")

	// Search mode input
	if m.searchMode {
		searchLabel := "Search"
		if strings.HasPrefix(m.search, ":") {
			searchLabel = "Command"
		}
		b.WriteString(filterStyle.Render(fmt.Sprintf("%s: %s", searchLabel, m.search)))
		b.WriteString("\n")
	}

	// Footer
	b.WriteString(m.renderFooter())

	return b.String()
}

// viewDiff renders the diff view
func (m *Model) viewDiff() string {
	var b strings.Builder

	// Header with diff mode indicator
	modeStr := "side-by-side"
	if m.diffMode == 1 {
		modeStr = "unified"
	}
	header := fmt.Sprintf("Diff View [%s]", modeStr)
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n\n")

	// Viewport with diff content
	b.WriteString(m.viewport.View())
	b.WriteString("\n\n")

	// Footer
	b.WriteString(footerStyle.Render("[s] Side-by-side | [u] Unified | [j/k] Scroll | [q] Back | [:q] Quit"))

	return b.String()
}

// viewHelp renders the help view
func (m *Model) viewHelp() string {
	help := `
KEYBOARD SHORTCUTS (k9s-style)

Navigation:
  h               Navigate left (previous tab)
  l               Navigate right (next tab)
  k, ↑            Move up
  j, ↓            Move down
  g               Go to top
  G (shift-g)     Go to bottom
  ctrl-f          Page down
  ctrl-b          Page up
  0/1/2/3         Jump to tab (All/Added/Modified/Removed)
  Enter           View selected resource diff

Search:
  /               Search (filter by Kind, Name, or Namespace)
  
Sorting:
  N (shift-n)     Sort by name
  S (shift-s)     Sort by status (Added/Modified/Removed)

Diff View:
  s               Side-by-side diff mode
  u               Unified diff mode  
  j/k, ↑/↓        Scroll diff
  ctrl-f/ctrl-b   Page down/up
  g/G             Go to top/bottom

General:
  ?               Toggle help
  q               Back / Cancel
  :q              Quit application
  ctrl-c          Force quit
`
	return helpStyle.Render(help)
}

// renderTabs renders the tab bar
func (m *Model) renderTabs() string {
	all, added, modified, removed := m.getTabCounts()

	tabs := []string{
		m.renderTab("0", "All", all, m.currentTab == TabAll),
		m.renderTab("1", "Added", added, m.currentTab == TabAdded),
		m.renderTab("2", "Modified", modified, m.currentTab == TabModified),
		m.renderTab("3", "Removed", removed, m.currentTab == TabRemoved),
	}

	return strings.Join(tabs, " │ ")
}

// renderTab renders a single tab
func (m *Model) renderTab(key, label string, count int, active bool) string {
	text := fmt.Sprintf("[%s] %s (%d)", key, label, count)
	if active {
		return tabActiveStyle.Render(text)
	}
	return tabInactiveStyle.Render(text)
}

// renderFooter renders the footer with keyboard hints
func (m *Model) renderFooter() string {
	if m.searchMode {
		return footerStyle.Render("[Enter] Apply | [q] Cancel | [Ctrl+u] Clear")
	}
	return footerStyle.Render("[Enter] View | [h/l] Tabs | [/] Search | [N/S] Sort | [?] Help | [:q] Quit")
}

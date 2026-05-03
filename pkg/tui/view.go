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

	// Filter/Command mode input
	if m.filterMode {
		b.WriteString(filterStyle.Render(fmt.Sprintf("Filter: %s", m.filter)))
		b.WriteString("\n")
	} else if m.commandMode {
		b.WriteString(filterStyle.Render(fmt.Sprintf(":%s", m.commandBuf)))
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
	header := fmt.Sprintf("Diff View [%s] - Press s/u to toggle mode", modeStr)
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n\n")

	// Viewport with diff content
	b.WriteString(m.viewport.View())
	b.WriteString("\n\n")

	// Footer
	b.WriteString(footerStyle.Render("[s] Side-by-side | [u] Unified | [↑/↓] Scroll | [Esc] Back | [q] Quit"))

	return b.String()
}

// viewHelp renders the help view
func (m *Model) viewHelp() string {
	help := `
KEYBOARD SHORTCUTS

Navigation:
  ↑/↓, j/k         Navigate table rows
  ←/→              Navigate between tabs
  0/1/2/3          Jump to tab (All/Added/Modified/Removed)
  Enter            View selected resource diff
  Esc              Go back / Cancel

Filtering:
  /                Start filter mode (type to filter by name/kind/namespace)
  :                Command mode (shortcuts: :cm, :svc, :deploy, etc.)

Diff View:
  s                Side-by-side diff mode
  u                Unified diff mode
  ↑/↓, j/k         Scroll diff

General:
  ?                Toggle help
  q, Ctrl+c        Quit

COMMAND SHORTCUTS
  :cm              ConfigMaps
  :svc             Services
  :deploy          Deployments
  :sts             StatefulSets
  :secret          Secrets
  :ns              Namespaces
  :sa              ServiceAccounts
  :po              Pods
  :ing             Ingresses
  :pv              PersistentVolumes
  :pvc             PersistentVolumeClaims
  :ds              DaemonSets
  :rs              ReplicaSets
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
	if m.filterMode || m.commandMode {
		return footerStyle.Render("[Enter] Apply | [Esc] Cancel | [Ctrl+u] Clear")
	}
	return footerStyle.Render("[Enter] View | [←/→] Tabs | [/] Filter | [:] Command | [?] Help | [q] Quit")
}

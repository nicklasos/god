package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nicklasos/supervisord-tui/internal/supervisor"
)

// ListModel represents the left panel list view
type ListModel struct {
	processes  []*supervisor.Process
	filtered   []*supervisor.Process
	selected   int
	searchTerm string
	width      int
	height     int
}

// NewListModel creates a new list model
func NewListModel(processes []*supervisor.Process) *ListModel {
	return &ListModel{
		processes: processes,
		filtered:  processes,
		selected:  0,
	}
}

// Init initializes the list model
func (m *ListModel) Init() tea.Cmd {
	return nil
}

// Update handles updates to the list model
func (m *ListModel) Update(msg tea.Msg) (*ListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "k", "up":
			if m.selected > 0 {
				m.selected--
			}
		case "j", "down":
			if m.selected < len(m.filtered)-1 {
				m.selected++
			}
		}
	}
	return m, nil
}

// SetProcesses updates the processes list
func (m *ListModel) SetProcesses(processes []*supervisor.Process) {
	m.processes = processes
	m.ApplyFilter()
}

// ApplyFilter applies the current search filter
func (m *ListModel) ApplyFilter() {
	if m.searchTerm == "" {
		m.filtered = m.processes
		if m.selected >= len(m.filtered) {
			m.selected = max(0, len(m.filtered)-1)
		}
		return
	}

	var filtered []*supervisor.Process
	term := strings.ToLower(m.searchTerm)
	for _, proc := range m.processes {
		if strings.Contains(strings.ToLower(proc.Name), term) ||
			strings.Contains(strings.ToLower(proc.Status), term) {
			filtered = append(filtered, proc)
		}
	}

	m.filtered = filtered
	if m.selected >= len(m.filtered) {
		m.selected = max(0, len(m.filtered)-1)
	}
}

// SetSearchTerm sets the search term and applies the filter
func (m *ListModel) SetSearchTerm(term string) {
	m.searchTerm = term
	m.ApplyFilter()
}

// GetSelected returns the currently selected process
func (m *ListModel) GetSelected() *supervisor.Process {
	if len(m.filtered) == 0 || m.selected < 0 || m.selected >= len(m.filtered) {
		return nil
	}
	return m.filtered[m.selected]
}

// SetSelected sets the selected index
func (m *ListModel) SetSelected(index int) {
	if index >= 0 && index < len(m.filtered) {
		m.selected = index
	} else if index < 0 {
		m.selected = 0
	} else if index >= len(m.filtered) && len(m.filtered) > 0 {
		m.selected = len(m.filtered) - 1
	}
}

// GetSelectedIndex returns the currently selected index
func (m *ListModel) GetSelectedIndex() int {
	return m.selected
}

// SetSize sets the size of the list view
func (m *ListModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// View renders the list view
func (m *ListModel) View() string {
	if len(m.filtered) == 0 {
		return listPanelStyle.Width(m.width).Height(m.height).Render(
			titleStyle.Render("Processes") + "\n\n" +
				"No processes found",
		)
	}

	var lines []string
	lines = append(lines, titleStyle.Render("Processes"))

	availableHeight := m.height - 2 - 2 // panel padding
	titleHeight := 2
	availableForEntries := availableHeight - titleHeight
	visibleEntries := max(1, availableForEntries)

	start := max(0, m.selected-visibleEntries/2)
	end := min(len(m.filtered), start+visibleEntries*2)

	entryLinesCount := 0
	actualEnd := start

	for i := start; i < end && entryLinesCount < availableForEntries; i++ {
		proc := m.filtered[i]
		entryLines := m.formatEntry(proc, i == m.selected)
		splitLines := strings.Split(entryLines, "\n")
		if entryLinesCount+len(splitLines) > availableForEntries {
			break
		}
		for _, line := range splitLines {
			lines = append(lines, line)
			entryLinesCount++
		}
		actualEnd = i + 1
	}

	hasMoreAbove := start > 0
	hasMoreBelow := actualEnd < len(m.filtered)

	if hasMoreAbove {
		lines = append([]string{lines[0], "..."}, lines[1:]...)
	}
	if hasMoreBelow {
		lines = append(lines, "...")
	}

	for len(lines) < availableHeight {
		lines = append(lines, "")
	}

	content := strings.Join(lines, "\n")
	return listPanelStyle.Width(m.width).Height(m.height).Render(content)
}

// formatEntry formats a single entry for display
func (m *ListModel) formatEntry(proc *supervisor.Process, selected bool) string {
	statusStyle := GetStatusStyle(proc.Status)
	statusBadge := statusStyle.Render("[" + proc.Status + "]")

	mainLine := proc.Name + " " + statusBadge

	if selected {
		mainLine = "▶ " + mainLine
	} else {
		mainLine = "  " + mainLine
	}

	if selected {
		return listItemSelectedStyle.Render(mainLine)
	}
	return listItemStyle.Render(mainLine)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

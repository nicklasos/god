package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nicklasos/supervisord-tui/internal/supervisor"
)

// Mode represents the current UI mode
type Mode int

const (
	ModeList Mode = iota
	ModeSearch
	ModeEdit
	ModeAdd
	ModeDelete
	ModeViewLogs
)

// refreshMsg is sent periodically to refresh process status
type refreshMsg struct{}

// clearStatusMsg clears the status message
type clearStatusMsg struct{}

// Model represents the main application model
type Model struct {
	listModel   *ListModel
	detailModel *DetailModel
	logsModel   *LogsModel
	editorModel *EditorModel
	client      *supervisor.Client
	config      *supervisor.Config
	configPath  string
	processes   []*supervisor.Process

	mode          Mode
	searchInput   textinput.Model
	deleteConfirm bool

	width     int
	height    int
	err       error
	statusMsg string // Temporary status message (e.g., "Stopping process...")
}

// InitialModel creates the initial model with auto-detected config
func InitialModel() (*Model, error) {
	// Find config file
	configPath, err := supervisor.FindConfigFile()
	if err != nil {
		return nil, fmt.Errorf("failed to find supervisord config: %w", err)
	}
	return InitialModelWithConfig(configPath)
}

// InitialModelWithConfig creates the initial model with a specific config path
func InitialModelWithConfig(configPath string) (*Model, error) {
	// Verify config file exists
	if _, err := os.Stat(configPath); err != nil {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	// Validate config has required sections
	valid, missing := supervisor.ValidateConfig(configPath)
	if !valid {
		// Try to detect socket path
		socketPath := supervisor.DetectSocketPath()
		// Remove unix:// prefix for the config file
		cleanSocketPath := strings.TrimPrefix(socketPath, "unix://")
		minimalConfig := supervisor.GenerateMinimalConfig(cleanSocketPath)
		return nil, fmt.Errorf("supervisord config is missing required sections: %s\n\nYour config file needs these sections. Here's a minimal config to add:\n\n%s\n\nAdd this to the beginning of your config file: %s",
			strings.Join(missing, ", "), minimalConfig, configPath)
	}

	// Load config
	config, err := supervisor.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Create client
	client := supervisor.NewClient()

	// Get initial process status
	// If this fails, we'll start with an empty list and show the error
	processes, err := client.GetStatus()
	if err != nil {
		// Log the error but continue with empty process list
		// The error will be displayed in the UI
		processes = []*supervisor.Process{}
		// We'll set the error later in the model
	}

	// Merge config with processes - try exact match first, then case-insensitive
	for _, proc := range processes {
		cfg := config.GetProcessConfig(proc.Name)
		if cfg == nil {
			// Try case-insensitive match
			for _, prog := range config.Programs {
				if strings.EqualFold(prog.Name, proc.Name) {
					cfg = prog
					break
				}
			}
		}
		if cfg != nil {
			proc.Config = cfg
		}
	}

	// Initialize models
	listModel := NewListModel(processes)
	detailModel := NewDetailModel()
	logsModel := NewLogsModel()
	editorModel := NewEditorModel()

	// Initialize search input
	searchInput := textinput.New()
	searchInput.Placeholder = "Search..."

	model := &Model{
		listModel:     listModel,
		detailModel:   detailModel,
		logsModel:     logsModel,
		editorModel:   editorModel,
		client:        client,
		config:        config,
		configPath:    configPath,
		processes:     processes,
		mode:          ModeList,
		searchInput:   searchInput,
		deleteConfirm: false,
		err:           err, // Store error if status fetch failed
	}

	// Set initial selected process
	if len(processes) > 0 {
		model.updateDetailView()
	}

	return model, nil
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.listModel.Init(),
		m.editorModel.Init(),
		textinput.Blink,
		m.refreshTick(),
	)
}

// refreshTick returns a command that sends a refresh message after a delay
func (m *Model) refreshTick() tea.Cmd {
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return refreshMsg{}
	})
}

// Update handles updates
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateSizes()
		return m, nil

	case refreshMsg:
		// Refresh process status
		processes, err := m.client.GetStatus()
		if err == nil {
			// Reload config to ensure we have the latest
			if newConfig, configErr := supervisor.LoadConfig(m.configPath); configErr == nil {
				m.config = newConfig
			}
			// Merge config with processes - try exact match first, then case-insensitive
			for _, proc := range processes {
				cfg := m.config.GetProcessConfig(proc.Name)
				if cfg == nil {
					// Try case-insensitive match
					for _, prog := range m.config.Programs {
						if strings.EqualFold(prog.Name, proc.Name) {
							cfg = prog
							break
						}
					}
				}
				if cfg != nil {
					proc.Config = cfg
				}
			}
			m.processes = processes
			m.listModel.SetProcesses(processes)
			m.updateDetailView()
			m.err = nil // Clear error on successful refresh
		} else {
			// Keep error for display
			m.err = err
		}
		return m, m.refreshTick()

	case tea.KeyMsg:
		handled, model, keyCmd := m.handleKeyPress(msg)
		if handled {
			return model, keyCmd
		}

		// Handle mode-specific updates
		switch m.mode {
		case ModeSearch:
			var searchCmd tea.Cmd
			m.searchInput, searchCmd = m.searchInput.Update(msg)
			m.listModel.SetSearchTerm(m.searchInput.Value())
			m.updateDetailView()
			return m, searchCmd

		case ModeEdit, ModeAdd:
			var editCmd tea.Cmd
			updatedEditor, editCmd := m.editorModel.Update(msg)
			m.editorModel = updatedEditor
			return m, editCmd
		}

		// List mode updates
		var listCmd tea.Cmd
		updatedList, listUpdateCmd := m.listModel.Update(msg)
		m.listModel = updatedList
		m.updateDetailView()
		return m, tea.Batch(listCmd, listUpdateCmd)
	}

	return m, nil
}

// handleKeyPress handles key presses based on mode
func (m *Model) handleKeyPress(msg tea.KeyMsg) (bool, tea.Model, tea.Cmd) {
	switch m.mode {
	case ModeSearch:
		if msg.String() == "esc" {
			m.mode = ModeList
			m.searchInput.SetValue("")
			m.listModel.SetSearchTerm("")
			m.searchInput.Blur()
			return true, m, nil
		}
		if msg.String() == "enter" {
			m.mode = ModeList
			m.searchInput.Blur()
			return true, m, nil
		}
		return false, m, nil

	case ModeEdit, ModeAdd:
		switch msg.String() {
		case "enter":
			if err := m.editorModel.Validate(); err != nil {
				m.editorModel.SetError(err.Error())
				return true, m, nil
			}
			model, cmd := m.saveProcess()
			return true, model, cmd
		case "esc":
			m.mode = ModeList
			m.editorModel.SetConfig(nil)
			return true, m, nil
		}
		return false, m, nil

	case ModeDelete:
		switch msg.String() {
		case "y", "Y":
			model, cmd := m.confirmDelete()
			return true, model, cmd
		case "n", "N", "esc":
			m.mode = ModeList
			m.deleteConfirm = false
			return true, m, nil
		}
		return false, m, nil

	case ModeList:
		handled, model, cmd := m.handleListKeyPress(msg)
		return handled, model, cmd
	}

	return false, m, nil
}

// handleListKeyPress handles key presses in list mode
func (m *Model) handleListKeyPress(msg tea.KeyMsg) (bool, tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return true, m, tea.Quit

	case "j", "down":
		current := m.listModel.GetSelectedIndex()
		m.listModel.SetSelected(current + 1)
		m.updateDetailView()
		return true, m, nil

	case "k", "up":
		current := m.listModel.GetSelectedIndex()
		if current > 0 {
			m.listModel.SetSelected(current - 1)
		}
		m.updateDetailView()
		return true, m, nil

	case "/":
		m.mode = ModeSearch
		m.searchInput.Focus()
		return true, m, textinput.Blink

	case "s":
		proc := m.listModel.GetSelected()
		if proc != nil {
			statusCmd := m.setStatusMsg(fmt.Sprintf("Starting %s...", proc.Name))
			if err := m.client.Start(proc.Name); err != nil {
				m.err = err
				statusCmd = m.setStatusMsg(fmt.Sprintf("Failed to start %s", proc.Name))
			} else {
				statusCmd = m.setStatusMsg(fmt.Sprintf("Started %s", proc.Name))
				// Refresh immediately
				m.refreshProcesses()
			}
			return true, m, statusCmd
		}
		return true, m, nil

	case "x":
		proc := m.listModel.GetSelected()
		if proc != nil {
			statusCmd := m.setStatusMsg(fmt.Sprintf("Stopping %s...", proc.Name))
			if err := m.client.Stop(proc.Name); err != nil {
				m.err = err
				statusCmd = m.setStatusMsg(fmt.Sprintf("Failed to stop %s", proc.Name))
			} else {
				statusCmd = m.setStatusMsg(fmt.Sprintf("Stopped %s", proc.Name))
				m.refreshProcesses()
			}
			return true, m, statusCmd
		}
		return true, m, nil

	case "r":
		proc := m.listModel.GetSelected()
		if proc != nil {
			statusCmd := m.setStatusMsg(fmt.Sprintf("Restarting %s...", proc.Name))
			if err := m.client.Restart(proc.Name); err != nil {
				m.err = err
				statusCmd = m.setStatusMsg(fmt.Sprintf("Failed to restart %s", proc.Name))
			} else {
				statusCmd = m.setStatusMsg(fmt.Sprintf("Restarted %s", proc.Name))
				m.refreshProcesses()
			}
			return true, m, statusCmd
		}
		return true, m, nil

	case "a":
		m.mode = ModeAdd
		m.editorModel.SetConfig(nil) // nil means new process with template
		return true, m, nil

	case "e":
		proc := m.listModel.GetSelected()
		if proc != nil {
			if proc.Config != nil {
				m.mode = ModeEdit
				m.editorModel.SetConfig(proc.Config)
			} else {
				// If no config, create a new one from template
				// This allows editing processes that don't have configs loaded
				templateConfig := &supervisor.ProcessConfig{
					Name:                  proc.Name,
					Environment:           make(map[string]string),
					Autostart:             true,
					Autorestart:           true,
					StartSecs:             10,
					StartRetries:          3,
					StdoutLogfileMaxBytes: 1024 * 1024, // 1MB
					StdoutLogfileBackups:  10,
					StderrLogfileMaxBytes: 1024 * 1024, // 1MB
					StderrLogfileBackups:  10,
					Priority:              999,
					StopSignal:            "TERM",
					StopWaitSecs:          30,
				}
				m.mode = ModeEdit
				m.editorModel.SetConfig(templateConfig)
			}
		}
		return true, m, nil

	case "d":
		proc := m.listModel.GetSelected()
		if proc != nil {
			m.mode = ModeDelete
			m.deleteConfirm = false
		}
		return true, m, nil

	case "l":
		proc := m.listModel.GetSelected()
		if proc != nil && proc.Config != nil {
			m.viewLogs(proc, "stdout")
		}
		return true, m, nil

	case "L":
		proc := m.listModel.GetSelected()
		if proc != nil && proc.Config != nil {
			m.viewLogs(proc, "stderr")
		}
		return true, m, nil
	}

	return false, m, nil
}

// refreshProcesses refreshes the process list
func (m *Model) refreshProcesses() {
	processes, err := m.client.GetStatus()
	if err == nil {
		// Reload config to ensure we have the latest
		if newConfig, configErr := supervisor.LoadConfig(m.configPath); configErr == nil {
			m.config = newConfig
		}
		// Merge config with processes - try exact match first, then case-insensitive
		for _, proc := range processes {
			cfg := m.config.GetProcessConfig(proc.Name)
			if cfg == nil {
				// Try case-insensitive match
				for _, prog := range m.config.Programs {
					if strings.EqualFold(prog.Name, proc.Name) {
						cfg = prog
						break
					}
				}
			}
			if cfg != nil {
				proc.Config = cfg
			}
		}
		m.processes = processes
		m.listModel.SetProcesses(processes)
		m.updateDetailView()
	}
}

// updateDetailView updates the detail and logs views with the currently selected process
func (m *Model) updateDetailView() {
	proc := m.listModel.GetSelected()
	if proc != nil {
		m.detailModel.SetProcess(proc)
		m.logsModel.SetProcess(proc)
	}
}

// updateSizes updates the sizes of all UI components
func (m *Model) updateSizes() {
	// Account for status bar (1 line) and error message if present
	statusBarHeight := 1
	errorHeight := 0
	if m.err != nil {
		errorHeight = 2 // Error message takes ~2 lines
	}

	// Calculate available height - be VERY conservative
	// Account for: status bar, error message, padding, and safety margin
	availableHeight := m.height - statusBarHeight - errorHeight - 4
	if availableHeight < 6 {
		availableHeight = 6 // Absolute minimum
	}

	// Make left panel take up ~30% of screen width, but ensure it fits
	// Account for borders (2 chars each side = 4 total) and gap (1 char)
	listWidth := (m.width - 5) * 30 / 100 // Subtract 5 for borders and gap
	if listWidth < 25 {
		listWidth = 25 // Minimum width
	}
	if listWidth > (m.width-5)*35/100 {
		listWidth = (m.width - 5) * 35 / 100 // Max 35% of available space
	}

	// Calculate right panel width - ensure it fits
	rightWidth := m.width - listWidth - 5 // Account for borders and gap
	if rightWidth < 20 {
		rightWidth = 20
		listWidth = m.width - rightWidth - 5
	}

	// Split right panel height: info panel gets more space, logs get minimal (3 lines)
	// Account for borders: each panel has 2 lines of border (top+bottom)
	borderOverhead := 6 // 3 panels * 2 borders each

	// Available space for actual content (after borders)
	contentSpace := availableHeight - borderOverhead
	if contentSpace < 7 {
		contentSpace = 7 // Minimum for info + 2 small log panels
	}

	// Info panel: ~55% of content space
	infoHeight := contentSpace * 55 / 100
	if infoHeight < 4 {
		infoHeight = 4
	}

	// Log panels: fixed at 3 lines each (2 lines content + 1 for title)
	// This matches logLines constant (3 lines)
	logContentHeight := 3
	logHeight := logContentHeight

	// Ensure total fits
	totalContent := infoHeight + logHeight*2
	if totalContent > contentSpace {
		// Reduce info height if needed
		infoHeight = contentSpace - logHeight*2
		if infoHeight < 3 {
			infoHeight = 3
			logHeight = (contentSpace - infoHeight) / 2
			if logHeight < 2 {
				logHeight = 2
			}
		}
	}

	// Calculate total right panel height (including borders)
	// Each panel: content + 2 borders
	totalRightHeight := (infoHeight + 2) + (logHeight + 2) + (logHeight + 2)

	// Set panel sizes
	m.listModel.SetSize(listWidth, totalRightHeight)
	m.detailModel.SetSize(rightWidth, infoHeight+2)           // +2 for borders
	m.logsModel.SetSize(rightWidth, logHeight+2, logHeight+2) // +2 for borders
	m.editorModel.SetSize(m.width-4, m.height-4)
}

// saveProcess saves the current process from the editor
func (m *Model) saveProcess() (tea.Model, tea.Cmd) {
	config, err := m.editorModel.GetConfig()
	if err != nil {
		m.editorModel.SetError(err.Error())
		return m, nil
	}

	if m.mode == ModeAdd {
		m.config.AddProgram(config)
	} else {
		oldProc := m.listModel.GetSelected()
		if oldProc != nil {
			m.config.UpdateProgram(oldProc.Name, config)
		}
	}

	// Save config file
	if err := m.config.Save(); err != nil {
		m.editorModel.SetError(err.Error())
		return m, nil
	}

	// Reload config
	newConfig, err := supervisor.LoadConfig(m.configPath)
	if err != nil {
		m.err = err
		m.mode = ModeList
		return m, nil
	}
	m.config = newConfig

	// Reread and update
	if err := m.client.Reread(); err != nil {
		m.err = err
		m.mode = ModeList
		return m, nil
	}

	if err := m.client.Update(config.Name); err != nil {
		m.err = err
		m.mode = ModeList
		return m, nil
	}

	m.mode = ModeList
	m.editorModel.SetConfig(nil)
	m.refreshProcesses()

	// Select the saved process
	for i, proc := range m.processes {
		if proc.Name == config.Name {
			m.listModel.SetSelected(i)
			break
		}
	}

	m.updateDetailView()
	return m, nil
}

// confirmDelete confirms and deletes the selected process
func (m *Model) confirmDelete() (tea.Model, tea.Cmd) {
	proc := m.listModel.GetSelected()
	if proc == nil {
		m.mode = ModeList
		return m, nil
	}

	m.config.DeleteProgram(proc.Name)

	// Save config file
	if err := m.config.Save(); err != nil {
		m.err = err
		m.mode = ModeList
		return m, nil
	}

	// Reload config
	newConfig, err := supervisor.LoadConfig(m.configPath)
	if err != nil {
		m.err = err
		m.mode = ModeList
		return m, nil
	}
	m.config = newConfig

	// Reread and update
	if err := m.client.Reread(); err != nil {
		m.err = err
		m.mode = ModeList
		return m, nil
	}

	if err := m.client.Update(""); err != nil {
		m.err = err
		m.mode = ModeList
		return m, nil
	}

	m.mode = ModeList
	m.deleteConfirm = false
	m.refreshProcesses()

	// Adjust selection
	current := m.listModel.GetSelectedIndex()
	if current >= len(m.processes) && len(m.processes) > 0 {
		m.listModel.SetSelected(len(m.processes) - 1)
	} else if len(m.processes) == 0 {
		m.listModel.SetSelected(0)
	}
	m.updateDetailView()
	return m, nil
}

// viewLogs opens the log file in the default editor
// logType can be "stdout" or "stderr"
func (m *Model) viewLogs(proc *supervisor.Process, logType string) {
	if proc.Config == nil {
		return
	}

	var logFile string
	if logType == "stderr" {
		logFile = proc.Config.StderrLogfile
	} else {
		logFile = proc.Config.StdoutLogfile
	}

	if logFile == "" {
		return
	}

	// Get editor from environment or default to vi
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	cmd := exec.Command(editor, logFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Note: This will block the TUI, but that's expected behavior
	// The user will see the editor and can exit to return to the TUI
	cmd.Run()
}

// View renders the model
func (m *Model) View() string {
	switch m.mode {
	case ModeSearch:
		return m.renderSearch()
	case ModeEdit, ModeAdd:
		return m.renderEditor()
	case ModeDelete:
		return m.renderDeleteConfirm()
	default:
		return m.renderList()
	}
}

// renderList renders the list view
func (m *Model) renderList() string {
	listView := m.listModel.View()
	detailView := m.detailModel.View()
	logsView := m.logsModel.View()

	// Join right panels vertically with minimal gap
	rightView := lipgloss.JoinVertical(lipgloss.Left,
		detailView,
		logsView,
	)

	// Join left and right with minimal gap
	// Constrain width to ensure it fits on screen
	content := lipgloss.JoinHorizontal(lipgloss.Top,
		listView,
		lipgloss.NewStyle().Width(1).Render(""), // Minimal gap
		rightView,
	)
	content = lipgloss.NewStyle().MarginTop(1).Width(m.width).Render(content)

	// Shorten status bar for smaller screens
	statusText := "j/k: nav | /: search | s: start | x: stop | r: restart | a: add | e: edit | d: del | l: stdout | L: stderr | q: quit"
	if m.width < 100 {
		statusText = "j/k: nav | s: start | x: stop | r: restart | a: add | e: edit | d: del | l/L: logs | q: quit"
	}

	// Add status message if present
	if m.statusMsg != "" {
		statusText = m.statusMsg + " | " + statusText
	}

	status := lipgloss.NewStyle().
		Foreground(fgColor).
		Padding(0, 1).
		Render(statusText)

	// Show error at the top if present
	var result string
	if m.err != nil {
		// Format error message with line breaks for better readability
		errText := fmt.Sprintf("⚠ Error: %v", m.err)
		// Split error message into lines if it contains \n
		errLines := strings.Split(errText, "\n")
		var errorMsg strings.Builder
		for i, line := range errLines {
			if i > 0 {
				errorMsg.WriteString("\n")
			}
			errorMsg.WriteString(errorStyle.Render(line))
		}
		result = lipgloss.JoinVertical(lipgloss.Left, errorMsg.String(), content, status)
	} else {
		result = lipgloss.JoinVertical(lipgloss.Left, content, status)
	}

	return result
}

// renderSearch renders the search view
func (m *Model) renderSearch() string {
	listView := m.listModel.View()
	detailView := m.detailModel.View()
	logsView := m.logsModel.View()

	// Join right panels vertically with minimal gap
	rightView := lipgloss.JoinVertical(lipgloss.Left,
		detailView,
		logsView,
	)

	// Join left and right with minimal gap
	// Constrain width to ensure it fits on screen
	content := lipgloss.JoinHorizontal(lipgloss.Top,
		listView,
		lipgloss.NewStyle().Width(1).Render(""), // Minimal gap
		rightView,
	)
	content = lipgloss.NewStyle().MarginTop(1).Width(m.width).Render(content)

	searchQuery := m.searchInput.Value()
	if searchQuery == "" {
		searchQuery = "(empty)"
	}
	statusText := fmt.Sprintf("Search: %s | Enter: select | Esc: cancel", searchQuery)
	if m.width < 80 {
		statusText = fmt.Sprintf("Search: %s | Enter/Esc", searchQuery)
	}
	status := lipgloss.NewStyle().
		Foreground(fgColor).
		Padding(0, 1).
		Render(statusText)

	return lipgloss.JoinVertical(lipgloss.Left, content, status)
}

// renderEditor renders the editor view
func (m *Model) renderEditor() string {
	editorView := m.editorModel.View()
	return "\n" + lipgloss.Place(m.width, m.height-1, lipgloss.Center, lipgloss.Top, editorView)
}

// renderDeleteConfirm renders the delete confirmation view
func (m *Model) renderDeleteConfirm() string {
	proc := m.listModel.GetSelected()
	if proc == nil {
		return ""
	}

	msg := fmt.Sprintf("Delete process '%s'? (y/n)", proc.Name)
	return detailPanelStyle.Width(m.width - 4).Height(10).Render(
		titleStyle.Render("Confirm Delete") + "\n\n" +
			warningStyle.Render(msg) + "\n\n" +
			helpStyle.Render("y: confirm | n/Esc: cancel"),
	)
}

// setStatusMsg sets a temporary status message that will be cleared after 3 seconds
func (m *Model) setStatusMsg(msg string) tea.Cmd {
	m.statusMsg = msg
	// Return a command to clear the message after 3 seconds
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

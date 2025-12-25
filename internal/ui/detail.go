package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/nicklasos/supervisord-tui/internal/supervisor"
)

// DetailModel represents the combined process info, error log, and stdout log section
type DetailModel struct {
	process   *supervisor.Process
	errorLog  []string
	stdoutLog []string
	width     int
	height    int
}

// NewDetailModel creates a new detail model
func NewDetailModel() *DetailModel {
	return &DetailModel{
		errorLog:  []string{},
		stdoutLog: []string{},
	}
}

// SetProcess sets the process to display and loads logs
func (m *DetailModel) SetProcess(process *supervisor.Process) {
	m.process = process
	m.loadLogs()
}

// SetSize sets the size of the detail view
func (m *DetailModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// loadLogs loads the last N lines from error and stdout log files
func (m *DetailModel) loadLogs() {
	m.errorLog = []string{}
	m.stdoutLog = []string{}

	if m.process == nil {
		return
	}

	if m.process.Config == nil {
		return
	}

	// Load error log
	if m.process.Config.StderrLogfile != "" {
		m.errorLog = readLastLines(m.process.Config.StderrLogfile, 6)
	}

	// Load stdout log
	if m.process.Config.StdoutLogfile != "" {
		m.stdoutLog = readLastLines(m.process.Config.StdoutLogfile, 6)
	}
}

// View renders the combined detail view
func (m *DetailModel) View() string {
	if m.process == nil {
		return detailPanelStyle.Width(m.width).Height(m.height).Render(
			titleStyle.Render("Process Details") + "\n\n" +
				"No process selected",
		)
	}

	var lines []string

	// Process Info Section
	lines = append(lines, titleStyle.Render("Process Info"))
	lines = append(lines, "")

	// Name on its own line
	lines = append(lines, labelStyle.Render("Name:")+" "+valueStyle.Render(m.process.Name))

	// Status on its own line
	statusStyle := GetStatusStyle(m.process.Status)
	lines = append(lines, labelStyle.Render("Status:")+" "+statusStyle.Render(m.process.Status))

	// PID on its own line
	if m.process.PID > 0 {
		lines = append(lines, labelStyle.Render("PID:")+" "+valueStyle.Render(fmt.Sprintf("%d", m.process.PID)))
	}

	// Uptime on its own line
	if m.process.Uptime > 0 {
		lines = append(lines, labelStyle.Render("Uptime:")+" "+valueStyle.Render(formatUptime(m.process.Uptime)))
	}

	// Config info if available - each on its own line
	if m.process.Config != nil {
		// Command on its own line
		if m.process.Config.Command != "" {
			cmd := m.process.Config.Command
			maxCmdLen := m.width - 10
			if maxCmdLen > 0 && len(cmd) > maxCmdLen {
				cmd = cmd[:maxCmdLen-3] + "..."
			}
			lines = append(lines, labelStyle.Render("Cmd:")+" "+valueStyle.Render(cmd))
		}

		// User on its own line
		if m.process.Config.User != "" {
			lines = append(lines, labelStyle.Render("User:")+" "+valueStyle.Render(m.process.Config.User))
		}

		// Directory on its own line
		if m.process.Config.Directory != "" {
			dir := m.process.Config.Directory
			maxDirLen := m.width - 10
			if maxDirLen > 0 && len(dir) > maxDirLen {
				dir = dir[:maxDirLen-3] + "..."
			}
			lines = append(lines, labelStyle.Render("Dir:")+" "+valueStyle.Render(dir))
		}
	}

	// Error Log Section
	lines = append(lines, "")
	lines = append(lines, titleStyle.Render("Error Log"))
	lines = append(lines, "")
	if len(m.errorLog) == 0 {
		lines = append(lines, valueStyle.Foreground(subtleColor).Render("No error log available"))
	} else {
		maxLineWidth := m.width - 6
		if maxLineWidth < 10 {
			maxLineWidth = 10
		}
		for _, line := range m.errorLog {
			truncated := truncateLine(line, maxLineWidth)
			lines = append(lines, valueStyle.Foreground(errorColor).Render(truncated))
		}
	}

	// Stdout Log Section
	lines = append(lines, "")
	lines = append(lines, titleStyle.Render("Stdout Log"))
	lines = append(lines, "")
	if len(m.stdoutLog) == 0 {
		lines = append(lines, valueStyle.Foreground(subtleColor).Render("No stdout log available"))
	} else {
		maxLineWidth := m.width - 6
		if maxLineWidth < 10 {
			maxLineWidth = 10
		}
		for _, line := range m.stdoutLog {
			truncated := truncateLine(line, maxLineWidth)
			lines = append(lines, valueStyle.Render(truncated))
		}
	}

	content := strings.Join(lines, "\n")
	return detailPanelStyle.Width(m.width).Height(m.height).Render(content)
}

// formatUptime formats a duration as a human-readable string
func formatUptime(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

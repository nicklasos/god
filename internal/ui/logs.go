package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nicklasos/supervisord-tui/internal/supervisor"
)

const logLines = 5 // Number of lines to show from each log (reduced for smaller screens)

// LogsModel represents the log sections (error and stdout)
type LogsModel struct {
	process      *supervisor.Process
	errorLog     []string
	stdoutLog    []string
	width        int
	errorHeight  int
	stdoutHeight int
}

// NewLogsModel creates a new logs model
func NewLogsModel() *LogsModel {
	return &LogsModel{
		errorLog:  []string{},
		stdoutLog: []string{},
	}
}

// SetProcess sets the process and loads its logs
func (m *LogsModel) SetProcess(process *supervisor.Process) {
	m.process = process
	m.loadLogs()
}

// SetSize sets the size of the logs view
func (m *LogsModel) SetSize(width, errorHeight, stdoutHeight int) {
	m.width = width
	m.errorHeight = errorHeight
	m.stdoutHeight = stdoutHeight
}

// loadLogs loads the last N lines from error and stdout log files
func (m *LogsModel) loadLogs() {
	m.errorLog = []string{}
	m.stdoutLog = []string{}

	if m.process == nil {
		return
	}

	if m.process.Config == nil {
		// Config not loaded - show message
		m.errorLog = []string{"Config not loaded for this process"}
		m.stdoutLog = []string{"Config not loaded for this process"}
		return
	}

	// Load error log
	if m.process.Config.StderrLogfile != "" {
		m.errorLog = readLastLines(m.process.Config.StderrLogfile, logLines)
	} else {
		m.errorLog = []string{"No stderr logfile configured"}
	}

	// Load stdout log
	if m.process.Config.StdoutLogfile != "" {
		m.stdoutLog = readLastLines(m.process.Config.StdoutLogfile, logLines)
	} else {
		m.stdoutLog = []string{"No stdout logfile configured"}
	}
}

// readLastLines reads the last N lines from a file
func readLastLines(filepath string, n int) []string {
	file, err := os.Open(filepath)
	if err != nil {
		return []string{fmt.Sprintf("Error: %v", err)}
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return []string{fmt.Sprintf("Error reading file: %v", err)}
	}

	// Return last N lines
	if len(lines) <= n {
		return lines
	}
	return lines[len(lines)-n:]
}

// View renders the logs view (both error and stdout sections)
func (m *LogsModel) View() string {
	errorView := m.renderErrorLog()
	stdoutView := m.renderStdoutLog()

	return lipgloss.JoinVertical(lipgloss.Left, errorView, stdoutView)
}

// renderErrorLog renders the error log section
func (m *LogsModel) renderErrorLog() string {
	var lines []string
	lines = append(lines, titleStyle.Render("Error Log"))

	if len(m.errorLog) == 0 {
		lines = append(lines, "")
		lines = append(lines, valueStyle.Foreground(subtleColor).Render("No error log available"))
	} else {
		lines = append(lines, "")
		// Calculate max line width (account for padding and borders)
		maxLineWidth := m.width - 6 // Account for borders (2) and padding (4)
		if maxLineWidth < 10 {
			maxLineWidth = 10 // Minimum width
		}
		for _, line := range m.errorLog {
			// Truncate long lines instead of wrapping
			truncated := truncateLine(line, maxLineWidth)
			lines = append(lines, valueStyle.Foreground(errorColor).Render(truncated))
		}
	}

	content := strings.Join(lines, "\n")
	return logPanelStyle.Width(m.width).Height(m.errorHeight).Render(content)
}

// renderStdoutLog renders the stdout log section
func (m *LogsModel) renderStdoutLog() string {
	var lines []string
	lines = append(lines, titleStyle.Render("Stdout Log"))

	if len(m.stdoutLog) == 0 {
		lines = append(lines, "")
		lines = append(lines, valueStyle.Foreground(subtleColor).Render("No stdout log available"))
	} else {
		lines = append(lines, "")
		// Calculate max line width (account for padding and borders)
		maxLineWidth := m.width - 6 // Account for borders (2) and padding (4)
		if maxLineWidth < 10 {
			maxLineWidth = 10 // Minimum width
		}
		for _, line := range m.stdoutLog {
			// Truncate long lines instead of wrapping
			truncated := truncateLine(line, maxLineWidth)
			lines = append(lines, valueStyle.Render(truncated))
		}
	}

	content := strings.Join(lines, "\n")
	return logPanelStyle.Width(m.width).Height(m.stdoutHeight).Render(content)
}

// truncateLine truncates a line to fit within maxWidth, adding "..." if truncated
func truncateLine(line string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	// Count runes (not bytes) for proper display width
	lineRunes := []rune(line)
	if len(lineRunes) <= maxWidth {
		return line
	}

	// Truncate and add ellipsis
	if maxWidth <= 3 {
		return "..."
	}

	truncated := string(lineRunes[:maxWidth-3]) + "..."
	return truncated
}

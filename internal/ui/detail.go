package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/nicklasos/supervisord-tui/internal/supervisor"
)

// DetailModel represents the process info section
type DetailModel struct {
	process *supervisor.Process
	width   int
	height  int
}

// NewDetailModel creates a new detail model
func NewDetailModel() *DetailModel {
	return &DetailModel{}
}

// SetProcess sets the process to display
func (m *DetailModel) SetProcess(process *supervisor.Process) {
	m.process = process
}

// SetSize sets the size of the detail view
func (m *DetailModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// View renders the detail view
func (m *DetailModel) View() string {
	if m.process == nil {
		return detailPanelStyle.Width(m.width).Height(m.height).Render(
			titleStyle.Render("Process Info") + "\n\n" +
				"No process selected",
		)
	}

	var lines []string
	lines = append(lines, titleStyle.Render("Process Info"))
	lines = append(lines, "")

	// Name
	lines = append(lines, labelStyle.Render("Name:"))
	lines = append(lines, valueStyle.Render(m.process.Name))

	lines = append(lines, "")

	// Status
	lines = append(lines, labelStyle.Render("Status:"))
	statusStyle := GetStatusStyle(m.process.Status)
	lines = append(lines, statusStyle.Render(m.process.Status))

	lines = append(lines, "")

	// PID
	if m.process.PID > 0 {
		lines = append(lines, labelStyle.Render("PID:"))
		lines = append(lines, valueStyle.Render(fmt.Sprintf("%d", m.process.PID)))
		lines = append(lines, "")
	}

	// Uptime
	if m.process.Uptime > 0 {
		lines = append(lines, labelStyle.Render("Uptime:"))
		lines = append(lines, valueStyle.Render(formatUptime(m.process.Uptime)))
	}

	// Config info if available
	if m.process.Config != nil {
		lines = append(lines, "")
		lines = append(lines, labelStyle.Render("Command:"))
		if m.process.Config.Command != "" {
			lines = append(lines, valueStyle.Render(m.process.Config.Command))
		} else {
			lines = append(lines, valueStyle.Foreground(subtleColor).Render("(not set)"))
		}

		lines = append(lines, "")
		lines = append(lines, labelStyle.Render("Directory:"))
		if m.process.Config.Directory != "" {
			lines = append(lines, valueStyle.Render(m.process.Config.Directory))
		} else {
			lines = append(lines, valueStyle.Foreground(subtleColor).Render("(not set)"))
		}

		lines = append(lines, "")
		lines = append(lines, labelStyle.Render("User:"))
		if m.process.Config.User != "" {
			lines = append(lines, valueStyle.Render(m.process.Config.User))
		} else {
			lines = append(lines, valueStyle.Foreground(subtleColor).Render("(not set)"))
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

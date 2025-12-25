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

	// Display info in compact rows
	var infoRows []string

	// Row 1: Name and Status
	nameStatus := labelStyle.Render("Name:") + " " + valueStyle.Render(m.process.Name)
	statusStyle := GetStatusStyle(m.process.Status)
	nameStatus += "  |  " + labelStyle.Render("Status:") + " " + statusStyle.Render(m.process.Status)
	infoRows = append(infoRows, nameStatus)

	// Row 2: PID and Uptime
	if m.process.PID > 0 || m.process.Uptime > 0 {
		var row2 string
		if m.process.PID > 0 {
			row2 = labelStyle.Render("PID:") + " " + valueStyle.Render(fmt.Sprintf("%d", m.process.PID))
		}
		if m.process.Uptime > 0 {
			if row2 != "" {
				row2 += "  |  "
			}
			row2 += labelStyle.Render("Uptime:") + " " + valueStyle.Render(formatUptime(m.process.Uptime))
		}
		if row2 != "" {
			infoRows = append(infoRows, row2)
		}
	}

	// Config info if available (compact format)
	if m.process.Config != nil {
		var configRows []string

		// Command and User on one line if space allows
		var cmdUserRow string
		if m.process.Config.Command != "" {
			// Truncate long commands to fit in panel
			cmd := m.process.Config.Command
			maxCmdLen := m.width - 20 // Leave space for label and padding
			if maxCmdLen > 0 && len(cmd) > maxCmdLen {
				cmd = cmd[:maxCmdLen-3] + "..."
			}
			cmdUserRow = labelStyle.Render("Cmd:") + " " + valueStyle.Render(cmd)
		}
		if m.process.Config.User != "" {
			if cmdUserRow != "" {
				cmdUserRow += "  |  "
			}
			cmdUserRow += labelStyle.Render("User:") + " " + valueStyle.Render(m.process.Config.User)
		}
		if cmdUserRow != "" {
			configRows = append(configRows, cmdUserRow)
		}

		// Directory on its own line if present
		if m.process.Config.Directory != "" {
			dir := m.process.Config.Directory
			maxDirLen := m.width - 10 // Leave space for label and padding
			if maxDirLen > 0 && len(dir) > maxDirLen {
				dir = dir[:maxDirLen-3] + "..."
			}
			configRows = append(configRows, labelStyle.Render("Dir:")+" "+valueStyle.Render(dir))
		}

		if len(configRows) > 0 {
			infoRows = append(infoRows, "")
			infoRows = append(infoRows, configRows...)
		}
	}

	lines = append(lines, infoRows...)
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

package ui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors - vim-inspired dark theme
	bgColor      = lipgloss.Color("0")  // Black background
	fgColor      = lipgloss.Color("15") // White foreground
	accentColor  = lipgloss.Color("6")  // Cyan accent
	selectColor  = lipgloss.Color("4")  // Blue for selection
	subtleColor  = lipgloss.Color("8")  // Dark gray for subtle text
	warningColor = lipgloss.Color("3")  // Yellow for warnings
	errorColor   = lipgloss.Color("1")  // Red for errors
	successColor = lipgloss.Color("2")  // Green for success/running

	// Panel styles
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accentColor).
			Padding(1, 2)

	listPanelStyle = panelStyle.Copy().
			Width(40).
			Height(20)

	detailPanelStyle = panelStyle.Copy().
				Width(50).
				Height(20)

	logPanelStyle = panelStyle.Copy().
			Width(50).
			Height(20)

	// Text styles
	titleStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true).
			MarginBottom(1)

	labelStyle = lipgloss.NewStyle().
			Foreground(subtleColor).
			MarginRight(1)

	valueStyle = lipgloss.NewStyle().
			Foreground(fgColor)

	selectedStyle = lipgloss.NewStyle().
			Foreground(selectColor).
			Bold(true).
			Background(lipgloss.Color("8"))

	// List item styles
	listItemStyle = lipgloss.NewStyle().
			Foreground(fgColor).
			PaddingLeft(2)

	listItemSelectedStyle = lipgloss.NewStyle().
				Foreground(accentColor).
				Border(lipgloss.NormalBorder()).
				BorderForeground(accentColor).
				BorderLeft(true).
				BorderRight(false).
				BorderTop(false).
				BorderBottom(false).
				PaddingLeft(1)

	// Status badge styles
	statusRunningStyle = lipgloss.NewStyle().
				Foreground(successColor).
				Bold(true)

	statusStoppedStyle = lipgloss.NewStyle().
				Foreground(errorColor).
				Bold(true)

	statusStartingStyle = lipgloss.NewStyle().
				Foreground(warningColor).
				Bold(true)

	statusStoppingStyle = lipgloss.NewStyle().
				Foreground(warningColor).
				Bold(true)

	statusFatalStyle = lipgloss.NewStyle().
				Foreground(errorColor).
				Bold(true)

	statusExitedStyle = lipgloss.NewStyle().
				Foreground(subtleColor).
				Bold(true)

	statusUnknownStyle = lipgloss.NewStyle().
				Foreground(subtleColor).
				Bold(true)

	// Editor styles
	inputStyle = lipgloss.NewStyle().
			Foreground(fgColor).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accentColor).
			Padding(0, 1)

	inputFocusedStyle = lipgloss.NewStyle().
				Foreground(fgColor).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(selectColor).
				Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(subtleColor).
			MarginTop(1)

	// Error/warning styles
	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(warningColor)
)

// GetStatusStyle returns the appropriate style for a process status
func GetStatusStyle(status string) lipgloss.Style {
	switch status {
	case "RUNNING":
		return statusRunningStyle
	case "STOPPED":
		return statusStoppedStyle
	case "STARTING":
		return statusStartingStyle
	case "STOPPING":
		return statusStoppingStyle
	case "FATAL":
		return statusFatalStyle
	case "EXITED":
		return statusExitedStyle
	default:
		return statusUnknownStyle
	}
}

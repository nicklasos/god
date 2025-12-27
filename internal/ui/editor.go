package ui

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nicklasos/supervisord-tui/internal/supervisor"
)

// EditorModel represents the textarea-based editor for process entries
type EditorModel struct {
	textarea textarea.Model
	config   *supervisor.ProcessConfig
	isNew    bool
	width    int
	height   int
	errorMsg string
}

// NewEditorModel creates a new editor model
func NewEditorModel() *EditorModel {
	ta := textarea.New()
	ta.Placeholder = "Enter supervisord program configuration..."
	ta.CharLimit = 0
	ta.SetWidth(80)
	ta.SetHeight(20)
	ta.Focus()

	return &EditorModel{
		textarea: ta,
	}
}

// Init initializes the editor model
func (m *EditorModel) Init() tea.Cmd {
	return textarea.Blink
}

// SetConfig sets the config to edit (nil for new entry with template)
func (m *EditorModel) SetConfig(config *supervisor.ProcessConfig) {
	m.errorMsg = ""

	if config == nil {
		// New process - use template
		m.config = nil
		m.isNew = true
		m.textarea.SetValue(generateTemplateText())
	} else {
		// Edit existing process
		m.config = config
		m.isNew = false
		m.textarea.SetValue(generateConfigText(config))
	}

	m.textarea.CursorEnd()
}

// SetSize sets the size of the editor
func (m *EditorModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	// Account for borders and padding
	m.textarea.SetWidth(width - 6)
	m.textarea.SetHeight(height - 8)
}

// Update handles updates to the editor model
func (m *EditorModel) Update(msg tea.Msg) (*EditorModel, tea.Cmd) {
	var cmd tea.Cmd

	// Let textarea handle all keys (including Enter for newlines)
	// Shift+Enter will be handled by the parent model
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

// Validate validates the textarea content
func (m *EditorModel) Validate() error {
	content := m.textarea.Value()
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("configuration cannot be empty")
	}

	// Try to parse to ensure it's valid
	_, err := parseConfigText(content)
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	return nil
}

// GetConfig returns the config from the textarea content
func (m *EditorModel) GetConfig() (*supervisor.ProcessConfig, error) {
	content := m.textarea.Value()
	return parseConfigText(content)
}

// SetError sets an error message
func (m *EditorModel) SetError(msg string) {
	m.errorMsg = msg
}

// View renders the editor view
func (m *EditorModel) View() string {
	title := "Edit Process"
	if m.isNew {
		title = "Add New Process"
	}

	var content strings.Builder
	content.WriteString(titleStyle.Render(title))
	content.WriteString("\n\n")

	// Textarea
	content.WriteString(m.textarea.View())
	content.WriteString("\n")

	// Error message
	if m.errorMsg != "" {
		content.WriteString("\n")
		content.WriteString(errorStyle.Render("Error: " + m.errorMsg))
		content.WriteString("\n")
	}

	// Help text
	content.WriteString("\n")
	helpText := "Shift+Enter: save | Esc: cancel"
	content.WriteString(helpStyle.Render(helpText))

	return detailPanelStyle.Width(m.width).Height(m.height).Render(content.String())
}

// generateTemplateText generates the template text for a new process
func generateTemplateText() string {
	return `[program:process-name]
command=/path/to/command
directory=/path/to/directory
user=nicklasos
autostart=true
autorestart=true
startsecs=10
startretries=3
stdout_logfile=/var/log/process.log
stderr_logfile=/var/log/process-error.log
stdout_logfile_maxbytes=1MB
stdout_logfile_backups=10
stderr_logfile_maxbytes=1MB
stderr_logfile_backups=10
environment=KEY1=value1,KEY2=value2
priority=999
stopsignal=TERM
stopwaitsecs=30
`
}

// generateConfigText generates config text from ProcessConfig
func generateConfigText(config *supervisor.ProcessConfig) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("[program:%s]\n", config.Name))

	if config.Command != "" {
		sb.WriteString(fmt.Sprintf("command=%s\n", config.Command))
	}
	if config.Directory != "" {
		sb.WriteString(fmt.Sprintf("directory=%s\n", config.Directory))
	}
	if config.User != "" {
		sb.WriteString(fmt.Sprintf("user=%s\n", config.User))
	}
	sb.WriteString(fmt.Sprintf("autostart=%v\n", config.Autostart))
	sb.WriteString(fmt.Sprintf("autorestart=%v\n", config.Autorestart))
	if config.StartSecs > 0 {
		sb.WriteString(fmt.Sprintf("startsecs=%d\n", config.StartSecs))
	}
	if config.StartRetries > 0 {
		sb.WriteString(fmt.Sprintf("startretries=%d\n", config.StartRetries))
	}
	if config.StdoutLogfile != "" {
		sb.WriteString(fmt.Sprintf("stdout_logfile=%s\n", config.StdoutLogfile))
	}
	if config.StderrLogfile != "" {
		sb.WriteString(fmt.Sprintf("stderr_logfile=%s\n", config.StderrLogfile))
	}
	if config.StdoutLogfileMaxBytes > 0 {
		sb.WriteString(fmt.Sprintf("stdout_logfile_maxbytes=%s\n", formatBytes(config.StdoutLogfileMaxBytes)))
	}
	if config.StdoutLogfileBackups > 0 {
		sb.WriteString(fmt.Sprintf("stdout_logfile_backups=%d\n", config.StdoutLogfileBackups))
	}
	if config.StderrLogfileMaxBytes > 0 {
		sb.WriteString(fmt.Sprintf("stderr_logfile_maxbytes=%s\n", formatBytes(config.StdoutLogfileMaxBytes)))
	}
	if config.StderrLogfileBackups > 0 {
		sb.WriteString(fmt.Sprintf("stderr_logfile_backups=%d\n", config.StderrLogfileBackups))
	}
	if len(config.Environment) > 0 {
		envStr := formatEnvironment(config.Environment)
		sb.WriteString(fmt.Sprintf("environment=%s\n", envStr))
	}
	if config.Priority > 0 {
		sb.WriteString(fmt.Sprintf("priority=%d\n", config.Priority))
	}
	if config.StopSignal != "" {
		sb.WriteString(fmt.Sprintf("stopsignal=%s\n", config.StopSignal))
	}
	if config.StopWaitSecs > 0 {
		sb.WriteString(fmt.Sprintf("stopwaitsecs=%d\n", config.StopWaitSecs))
	}

	return sb.String()
}

// parseConfigText parses config text into ProcessConfig
func parseConfigText(text string) (*supervisor.ProcessConfig, error) {
	config := &supervisor.ProcessConfig{
		Environment: make(map[string]string),
		Autostart:   false,
		Autorestart: false,
	}

	scanner := bufio.NewScanner(strings.NewReader(text))
	var inProgramSection bool

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for [program:name] section
		if strings.HasPrefix(line, "[program:") && strings.HasSuffix(line, "]") {
			name := strings.TrimPrefix(line, "[program:")
			name = strings.TrimSuffix(name, "]")
			config.Name = name
			inProgramSection = true
			continue
		}

		// Check if we're leaving a program section
		if inProgramSection && strings.HasPrefix(line, "[") {
			break
		}

		// Parse program configuration
		if inProgramSection {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			parseConfigLine(key, value, config)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}

	if config.Name == "" {
		return nil, fmt.Errorf("program name is required")
	}

	return config, nil
}

// parseConfigLine parses a single config line
func parseConfigLine(key, value string, config *supervisor.ProcessConfig) {
	switch key {
	case "command":
		config.Command = value
	case "directory":
		config.Directory = value
	case "user":
		config.User = value
	case "autostart":
		config.Autostart = strings.ToLower(value) == "true"
	case "autorestart":
		config.Autorestart = strings.ToLower(value) == "true"
	case "startsecs":
		if i, err := strconv.Atoi(value); err == nil {
			config.StartSecs = i
		}
	case "startretries":
		if i, err := strconv.Atoi(value); err == nil {
			config.StartRetries = i
		}
	case "stdout_logfile":
		config.StdoutLogfile = value
	case "stderr_logfile":
		config.StderrLogfile = value
	case "stdout_logfile_maxbytes":
		config.StdoutLogfileMaxBytes = parseBytes(value)
	case "stdout_logfile_backups":
		if i, err := strconv.Atoi(value); err == nil {
			config.StdoutLogfileBackups = i
		}
	case "stderr_logfile_maxbytes":
		config.StderrLogfileMaxBytes = parseBytes(value)
	case "stderr_logfile_backups":
		if i, err := strconv.Atoi(value); err == nil {
			config.StderrLogfileBackups = i
		}
	case "environment":
		parseEnvironment(value, config.Environment)
	case "priority":
		if i, err := strconv.Atoi(value); err == nil {
			config.Priority = i
		}
	case "stopsignal":
		config.StopSignal = value
	case "stopwaitsecs":
		if i, err := strconv.Atoi(value); err == nil {
			config.StopWaitSecs = i
		}
	}
}

// Helper functions (reused from old editor)
func parseBytes(value string) int64 {
	value = strings.TrimSpace(strings.ToUpper(value))
	if value == "" {
		return 0
	}

	re := regexp.MustCompile(`^(\d+)(KB|MB|GB)?$`)
	matches := re.FindStringSubmatch(value)
	if len(matches) < 2 {
		return 0
	}

	size, _ := strconv.ParseInt(matches[1], 10, 64)
	if len(matches) > 2 {
		switch matches[2] {
		case "KB":
			size *= 1024
		case "MB":
			size *= 1024 * 1024
		case "GB":
			size *= 1024 * 1024 * 1024
		}
	}
	return size
}

func formatBytes(bytes int64) string {
	if bytes == 0 {
		return ""
	}
	if bytes < 1024 {
		return fmt.Sprintf("%dB", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%dKB", bytes/1024)
	} else if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%dMB", bytes/(1024*1024))
	}
	return fmt.Sprintf("%dGB", bytes/(1024*1024*1024))
}

func formatEnvironment(env map[string]string) string {
	var pairs []string
	for k, v := range env {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(pairs, ",")
}

func parseEnvironment(value string, env map[string]string) {
	value = strings.Trim(value, "\"'")
	pairs := strings.Split(value, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			env[key] = val
		}
	}
}

package supervisor

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Config represents a supervisord configuration file
type Config struct {
	Path     string
	Programs []*ProcessConfig
	RawLines []string
}

// FindConfigFile finds the supervisord config file
func FindConfigFile() (string, error) {
	// Check environment variable first
	if configPath := os.Getenv("SUPERVISOR_CONFIG"); configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
	}

	// Check common default paths (Linux)
	paths := []string{
		"/etc/supervisor/supervisord.conf",
		"/etc/supervisord.conf",
	}

	// Check macOS Homebrew paths
	paths = append(paths,
		"/opt/homebrew/etc/supervisord.conf", // Apple Silicon
		"/usr/local/etc/supervisord.conf",    // Intel Mac
	)

	// Check home directory
	if homeDir, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(homeDir, ".supervisord.conf"))
	}

	// Try to get config from supervisorctl
	if configPath := getConfigFromSupervisorctl(); configPath != "" {
		paths = append([]string{configPath}, paths...)
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("supervisord config file not found in common locations. Set SUPERVISOR_CONFIG environment variable or ensure config exists in one of: /etc/supervisor/supervisord.conf, /etc/supervisord.conf, /opt/homebrew/etc/supervisord.conf, /usr/local/etc/supervisord.conf, ~/.supervisord.conf")
}

// getConfigFromSupervisorctl tries to get the config path from supervisorctl
func getConfigFromSupervisorctl() string {
	// Try to get version info which sometimes includes config path
	cmd := exec.Command("supervisorctl", "version")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	// Parse output for config file path (if present)
	// This is a best-effort approach
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "config file") {
			// Try to extract path
			parts := strings.Fields(line)
			for i, part := range parts {
				if strings.Contains(part, "supervisord.conf") {
					return part
				}
				if i > 0 && strings.Contains(parts[i-1], "config") {
					return part
				}
			}
		}
	}

	return ""
}

// LoadConfig loads and parses a supervisord config file
func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	config := &Config{
		Path:     path,
		Programs: []*ProcessConfig{},
		RawLines: []string{},
	}

	scanner := bufio.NewScanner(file)
	var currentProgram *ProcessConfig
	var inProgramSection bool

	lineNum := 0
	for scanner.Scan() {
		line := scanner.Text()
		config.RawLines = append(config.RawLines, line)
		lineNum++

		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Check for [program:name] section
		if strings.HasPrefix(trimmed, "[program:") && strings.HasSuffix(trimmed, "]") {
			// Save previous program if exists
			if currentProgram != nil {
				config.Programs = append(config.Programs, currentProgram)
			}

			// Extract program name
			name := strings.TrimPrefix(trimmed, "[program:")
			name = strings.TrimSuffix(name, "]")

			currentProgram = &ProcessConfig{
				Name:        name,
				Environment: make(map[string]string),
				Autostart:   false,
				Autorestart: false,
			}
			inProgramSection = true
			continue
		}

		// Check if we're leaving a program section (new section or end of file)
		if inProgramSection && strings.HasPrefix(trimmed, "[") {
			if currentProgram != nil {
				config.Programs = append(config.Programs, currentProgram)
			}
			currentProgram = nil
			inProgramSection = false
			continue
		}

		// Parse program configuration
		if inProgramSection && currentProgram != nil {
			parseProgramLine(trimmed, currentProgram)
		}
	}

	// Don't forget the last program
	if currentProgram != nil {
		config.Programs = append(config.Programs, currentProgram)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	return config, nil
}

// parseProgramLine parses a single line of program configuration
func parseProgramLine(line string, config *ProcessConfig) {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

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

// parseBytes parses byte values like "1MB", "500KB", etc.
func parseBytes(value string) int64 {
	value = strings.TrimSpace(strings.ToUpper(value))
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

// parseEnvironment parses environment variables
// Format: KEY1=value1,KEY2=value2
func parseEnvironment(value string, env map[string]string) {
	// Remove quotes if present
	value = strings.Trim(value, "\"'")

	// Split by comma
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

// GetProcessConfig returns the config for a specific process
func (c *Config) GetProcessConfig(name string) *ProcessConfig {
	for _, prog := range c.Programs {
		if prog.Name == name {
			return prog
		}
	}
	return nil
}

// AddProgram adds a new program to the config
func (c *Config) AddProgram(prog *ProcessConfig) {
	c.Programs = append(c.Programs, prog)
}

// UpdateProgram updates an existing program in the config
func (c *Config) UpdateProgram(name string, prog *ProcessConfig) {
	for i, p := range c.Programs {
		if p.Name == name {
			prog.Name = name
			c.Programs[i] = prog
			return
		}
	}
	// If not found, add it
	c.AddProgram(prog)
}

// DeleteProgram removes a program from the config
func (c *Config) DeleteProgram(name string) {
	for i, p := range c.Programs {
		if p.Name == name {
			c.Programs = append(c.Programs[:i], c.Programs[i+1:]...)
			return
		}
	}
}

// Save writes the config file
func (c *Config) Save() error {
	file, err := os.Create(c.Path)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Write all programs
	for i, prog := range c.Programs {
		if i > 0 {
			writer.WriteString("\n")
		}
		writeProgramSection(writer, prog)
	}

	return nil
}

// writeProgramSection writes a [program:name] section
func writeProgramSection(writer *bufio.Writer, prog *ProcessConfig) {
	writer.WriteString(fmt.Sprintf("[program:%s]\n", prog.Name))

	if prog.Command != "" {
		writer.WriteString(fmt.Sprintf("command=%s\n", prog.Command))
	}
	if prog.Directory != "" {
		writer.WriteString(fmt.Sprintf("directory=%s\n", prog.Directory))
	}
	if prog.User != "" {
		writer.WriteString(fmt.Sprintf("user=%s\n", prog.User))
	}
	writer.WriteString(fmt.Sprintf("autostart=%v\n", prog.Autostart))
	writer.WriteString(fmt.Sprintf("autorestart=%v\n", prog.Autorestart))
	if prog.StartSecs > 0 {
		writer.WriteString(fmt.Sprintf("startsecs=%d\n", prog.StartSecs))
	}
	if prog.StartRetries > 0 {
		writer.WriteString(fmt.Sprintf("startretries=%d\n", prog.StartRetries))
	}
	if prog.StdoutLogfile != "" {
		writer.WriteString(fmt.Sprintf("stdout_logfile=%s\n", prog.StdoutLogfile))
	}
	if prog.StderrLogfile != "" {
		writer.WriteString(fmt.Sprintf("stderr_logfile=%s\n", prog.StderrLogfile))
	}
	if prog.StdoutLogfileMaxBytes > 0 {
		writer.WriteString(fmt.Sprintf("stdout_logfile_maxbytes=%s\n", formatBytes(prog.StdoutLogfileMaxBytes)))
	}
	if prog.StdoutLogfileBackups > 0 {
		writer.WriteString(fmt.Sprintf("stdout_logfile_backups=%d\n", prog.StdoutLogfileBackups))
	}
	if prog.StderrLogfileMaxBytes > 0 {
		writer.WriteString(fmt.Sprintf("stderr_logfile_maxbytes=%s\n", formatBytes(prog.StderrLogfileMaxBytes)))
	}
	if prog.StderrLogfileBackups > 0 {
		writer.WriteString(fmt.Sprintf("stderr_logfile_backups=%d\n", prog.StderrLogfileBackups))
	}
	if len(prog.Environment) > 0 {
		envStr := formatEnvironment(prog.Environment)
		writer.WriteString(fmt.Sprintf("environment=%s\n", envStr))
	}
	if prog.Priority > 0 {
		writer.WriteString(fmt.Sprintf("priority=%d\n", prog.Priority))
	}
	if prog.StopSignal != "" {
		writer.WriteString(fmt.Sprintf("stopsignal=%s\n", prog.StopSignal))
	}
	if prog.StopWaitSecs > 0 {
		writer.WriteString(fmt.Sprintf("stopwaitsecs=%d\n", prog.StopWaitSecs))
	}
}

// formatBytes formats bytes to string like "1MB"
func formatBytes(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%dB", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%dKB", bytes/1024)
	} else if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%dMB", bytes/(1024*1024))
	}
	return fmt.Sprintf("%dGB", bytes/(1024*1024*1024))
}

// formatEnvironment formats environment map to string
func formatEnvironment(env map[string]string) string {
	var pairs []string
	for k, v := range env {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(pairs, ",")
}

package supervisor

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Client wraps supervisorctl commands
type Client struct{}

// NewClient creates a new supervisor client
func NewClient() *Client {
	return &Client{}
}

// GetStatus returns the status of all processes
func (c *Client) GetStatus() ([]*Process, error) {
	cmd := exec.Command("supervisorctl", "status")

	// Separate stdout and stderr to handle cases where stderr has warnings
	// but stdout has valid process data
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Always try to parse stdout, even if there's an error
	// supervisorctl may write status to stdout and warnings/errors to stderr
	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	processes, parseErr := c.parseStatus(stdoutStr)

	// If we successfully parsed processes, return them (even if there was an error)
	if len(processes) > 0 {
		// If there's a stderr message, include it as a warning but don't fail
		if stderrStr != "" && !strings.Contains(stderrStr, "does not include supervisorctl section") {
			// Non-fatal warning in stderr, but we have valid processes
			return processes, nil
		}
		// If there's a real error but we have processes, return processes with error
		if err != nil {
			return processes, err
		}
		return processes, nil
	}

	// No processes parsed - check for actual errors
	if err != nil {
		// Check stderr first, then stdout (in case error went to stdout)
		errStr := stderrStr
		if errStr == "" {
			errStr = stdoutStr
		}

		if strings.Contains(errStr, "does not include supervisorctl section") {
			socketPath := DetectSocketPath()
			return nil, fmt.Errorf("supervisord config is missing [supervisorctl] section.\n\nTo fix this, add the following to your supervisord config file:\n\n[supervisorctl]\nserverurl=%s\n\nOr if using TCP:\n[supervisorctl]\nserverurl=http://127.0.0.1:9001", socketPath)
		}
		if strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "No such file") {
			return nil, fmt.Errorf("supervisord is not running or socket not found. Start supervisord first")
		}
		return nil, fmt.Errorf("failed to get status: %s", errStr)
	}

	// Command succeeded but no processes parsed
	if parseErr != nil {
		return processes, parseErr
	}

	// No error, no processes - might be empty
	return processes, nil
}

// parseStatus parses the output of `supervisorctl status`
// Format: process_name                    RUNNING   pid 12345, uptime 0:05:23
// or:     process_name                    RUNNING   pid 12345, uptime 7 days, 10:25:47
// or:     process_name                    STOPPED   Dec 25 08:28 PM
func (c *Client) parseStatus(output string) ([]*Process, error) {
	var processes []*Process
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Split by whitespace to get parts
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		name := parts[0]
		status := parts[1]
		pid := 0
		var uptime time.Duration

		// Check if this is a running process with PID and uptime
		// Look for "pid" keyword in the line
		if strings.Contains(line, "pid") {
			// Extract PID: look for "pid" followed by a number
			pidRe := regexp.MustCompile(`pid\s+(\d+)`)
			pidMatches := pidRe.FindStringSubmatch(line)
			if len(pidMatches) > 1 {
				if p, err := strconv.Atoi(pidMatches[1]); err == nil {
					pid = p
				}
			}

			// Extract uptime: look for "uptime" followed by the time string
			uptimeRe := regexp.MustCompile(`uptime\s+(.+)`)
			uptimeMatches := uptimeRe.FindStringSubmatch(line)
			if len(uptimeMatches) > 1 {
				uptime = c.parseUptime(uptimeMatches[1])
			}
		}
		// If no PID found, it's likely a stopped process - just use status

		process := &Process{
			Name:   name,
			Status: status,
			PID:    pid,
			Uptime: uptime,
		}
		processes = append(processes, process)
	}

	return processes, scanner.Err()
}

// parseUptime parses uptime string like "0:05:23", "1:23:45", or "7 days, 10:25:47"
func (c *Client) parseUptime(uptimeStr string) time.Duration {
	uptimeStr = strings.TrimSpace(uptimeStr)

	// Handle "X days, H:MM:SS" format
	if strings.Contains(uptimeStr, "days") {
		// Extract days and time
		daysRe := regexp.MustCompile(`(\d+)\s+days?,\s+(\d+):(\d+):(\d+)`)
		matches := daysRe.FindStringSubmatch(uptimeStr)
		if len(matches) == 5 {
			days, _ := strconv.Atoi(matches[1])
			hours, _ := strconv.Atoi(matches[2])
			minutes, _ := strconv.Atoi(matches[3])
			seconds, _ := strconv.Atoi(matches[4])

			return time.Duration(days)*24*time.Hour +
				time.Duration(hours)*time.Hour +
				time.Duration(minutes)*time.Minute +
				time.Duration(seconds)*time.Second
		}
	}

	// Handle "H:MM:SS" format
	parts := strings.Split(uptimeStr, ":")
	if len(parts) == 3 {
		hours, _ := strconv.Atoi(parts[0])
		minutes, _ := strconv.Atoi(parts[1])
		seconds, _ := strconv.Atoi(parts[2])

		return time.Duration(hours)*time.Hour +
			time.Duration(minutes)*time.Minute +
			time.Duration(seconds)*time.Second
	}

	return 0
}

// DetectSocketPath tries to detect the socket path from the supervisord config
func DetectSocketPath() string {
	configPath, err := FindConfigFile()
	if err != nil {
		return "unix:///tmp/supervisor.sock" // Default fallback
	}

	file, err := os.Open(configPath)
	if err != nil {
		return "unix:///tmp/supervisor.sock"
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var inUnixSection bool
	var socketPath string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for [unix_http_server] section
		if strings.HasPrefix(line, "[unix_http_server]") {
			inUnixSection = true
			continue
		}

		// Check if we're leaving the section
		if inUnixSection && strings.HasPrefix(line, "[") {
			break
		}

		// Look for file= directive in unix_http_server section
		if inUnixSection && strings.HasPrefix(line, "file=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				socketPath = strings.TrimSpace(parts[1])
				// Expand ~ to home directory
				if strings.HasPrefix(socketPath, "~") {
					if homeDir, err := os.UserHomeDir(); err == nil {
						socketPath = filepath.Join(homeDir, strings.TrimPrefix(socketPath, "~/"))
					}
				}
				return "unix://" + socketPath
			}
		}
	}

	// Default fallback
	return "unix:///tmp/supervisor.sock"
}

// Start starts a process
func (c *Client) Start(name string) error {
	cmd := exec.Command("supervisorctl", "start", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start %s: %s", name, string(output))
	}
	return nil
}

// Stop stops a process
func (c *Client) Stop(name string) error {
	cmd := exec.Command("supervisorctl", "stop", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop %s: %s", name, string(output))
	}
	return nil
}

// Restart restarts a process
func (c *Client) Restart(name string) error {
	cmd := exec.Command("supervisorctl", "restart", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to restart %s: %s", name, string(output))
	}
	return nil
}

// Reread tells supervisord to reread config files
func (c *Client) Reread() error {
	cmd := exec.Command("supervisorctl", "reread")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to reread config: %s", string(output))
	}
	return nil
}

// Update updates process configurations
func (c *Client) Update(name string) error {
	args := []string{"update"}
	if name != "" {
		args = append(args, name)
	}
	cmd := exec.Command("supervisorctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to update %s: %s", name, string(output))
	}
	return nil
}

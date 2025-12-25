package supervisor

import (
	"bufio"
	"fmt"
	"os/exec"
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
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if it's a configuration error
		errStr := string(output)
		if strings.Contains(errStr, "does not include supervisorctl section") {
			return nil, fmt.Errorf("supervisord config is missing [supervisorctl] section. Add this to your config:\n[supervisorctl]\nserverurl=unix:///tmp/supervisor.sock")
		}
		if strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "No such file") {
			return nil, fmt.Errorf("supervisord is not running or socket not found. Start supervisord first")
		}
		return nil, fmt.Errorf("failed to get status: %s", errStr)
	}

	return c.parseStatus(string(output))
}

// parseStatus parses the output of `supervisorctl status`
// Format: process_name                    RUNNING   pid 12345, uptime 0:05:23
func (c *Client) parseStatus(output string) ([]*Process, error) {
	var processes []*Process
	scanner := bufio.NewScanner(strings.NewReader(output))

	// Regex to parse status line
	// Example: "process_name                    RUNNING   pid 12345, uptime 0:05:23"
	// or: "process_name                    STOPPED   Nov 01 10:00 AM"
	statusRe := regexp.MustCompile(`^(\S+)\s+(\w+)\s+(?:pid\s+(\d+),\s+uptime\s+([^\s]+(?:\s+[^\s]+)*)|(.+))$`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		matches := statusRe.FindStringSubmatch(line)
		if len(matches) < 3 {
			// Try simpler parsing
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				name := parts[0]
				status := parts[1]
				process := &Process{
					Name:   name,
					Status: status,
					PID:    0,
					Uptime: 0,
				}
				processes = append(processes, process)
			}
			continue
		}

		name := matches[1]
		status := matches[2]
		pid := 0
		var uptime time.Duration

		if matches[3] != "" {
			// Has PID and uptime
			if p, err := strconv.Atoi(matches[3]); err == nil {
				pid = p
			}
			if matches[4] != "" {
				uptime = c.parseUptime(matches[4])
			}
		}

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

// parseUptime parses uptime string like "0:05:23" or "1:23:45"
func (c *Client) parseUptime(uptimeStr string) time.Duration {
	parts := strings.Split(uptimeStr, ":")
	if len(parts) != 3 {
		return 0
	}

	hours, _ := strconv.Atoi(parts[0])
	minutes, _ := strconv.Atoi(parts[1])
	seconds, _ := strconv.Atoi(parts[2])

	return time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second
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

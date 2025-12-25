package supervisor

import (
	"time"
)

// Process represents a supervisord process
type Process struct {
	Name   string
	Status string // RUNNING, STOPPED, STARTING, STOPPING, FATAL, EXITED, UNKNOWN
	PID    int
	Uptime time.Duration
	Config *ProcessConfig
}

// ProcessConfig represents the configuration for a supervisord process
type ProcessConfig struct {
	Name                  string
	Command               string
	Directory             string
	User                  string
	Autostart             bool
	Autorestart           bool
	StartSecs             int
	StartRetries          int
	StdoutLogfile         string
	StderrLogfile         string
	StdoutLogfileMaxBytes int64
	StdoutLogfileBackups  int
	StderrLogfileMaxBytes int64
	StderrLogfileBackups  int
	Environment           map[string]string
	Priority              int
	StopSignal            string
	StopWaitSecs          int
}

// IsRunning returns true if the process is currently running
func (p *Process) IsRunning() bool {
	return p.Status == "RUNNING"
}

// IsStopped returns true if the process is stopped
func (p *Process) IsStopped() bool {
	return p.Status == "STOPPED"
}

package supervisor

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ValidateConfig checks if the config file has required sections
func ValidateConfig(path string) (bool, []string) {
	file, err := os.Open(path)
	if err != nil {
		return false, []string{fmt.Sprintf("Cannot open config file: %v", err)}
	}
	defer file.Close()

	var missing []string
	sections := make(map[string]bool)
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for section headers
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := strings.Trim(line, "[]")
			sections[section] = true
		}
	}

	// Check for required sections
	if !sections["supervisord"] {
		missing = append(missing, "[supervisord]")
	}
	if !sections["unix_http_server"] && !sections["inet_http_server"] {
		missing = append(missing, "[unix_http_server] or [inet_http_server]")
	}
	if !sections["supervisorctl"] {
		missing = append(missing, "[supervisorctl]")
	}

	return len(missing) == 0, missing
}

// GenerateMinimalConfig generates a minimal valid supervisord config
func GenerateMinimalConfig(socketPath string) string {
	if socketPath == "" {
		socketPath = "/tmp/supervisor.sock"
	}
	
	return fmt.Sprintf(`[unix_http_server]
file=%s
chmod=0700

[supervisord]
logfile=/tmp/supervisord.log
pidfile=/tmp/supervisord.pid

[rpcinterface:supervisor]
supervisor.rpcinterface_factory = supervisor.rpcinterface:make_main_rpcinterface

[supervisorctl]
serverurl=unix://%s

`, socketPath, socketPath)
}


package linux

import (
	"fmt"
	"os/exec"
	"time"
)

// Reboot initiates a system reboot after the specified delay in seconds
func Reboot(delaySeconds int) error {
	// Wait for delay (allows logs to flush)
	if delaySeconds > 0 {
		time.Sleep(time.Duration(delaySeconds) * time.Second)
	}

	// Try multiple reboot methods in order of preference
	// Different Linux systems have different commands available
	methods := [][]string{
		{"systemctl", "reboot"},   // systemd (most modern distros)
		{"shutdown", "-r", "now"}, // traditional
		{"reboot"},                // simple command
		{"/sbin/reboot"},          // explicit path
		{"init", "6"},             // SysV init
	}

	var lastErr error
	for _, method := range methods {
		cmd := exec.Command(method[0], method[1:]...)
		if err := cmd.Start(); err == nil {
			return nil // Success - reboot initiated
		} else {
			lastErr = err
		}
	}

	return fmt.Errorf("all reboot methods failed, last error: %w", lastErr)
}

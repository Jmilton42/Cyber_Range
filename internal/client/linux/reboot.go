package linux

import (
	"fmt"
	"os/exec"
)

// Reboot initiates a system reboot after the specified delay in seconds
func Reboot(delaySeconds int) error {
	// Use shutdown command which works on all Linux distros
	// -r = reboot, +N = delay in minutes (we convert seconds to "+0" for immediate or use "now")
	var cmd *exec.Cmd
	if delaySeconds <= 0 {
		cmd = exec.Command("shutdown", "-r", "now")
	} else {
		// shutdown uses minutes, so for seconds delay we use sleep + shutdown
		// For simplicity, if delay > 0, sleep first then reboot
		cmd = exec.Command("sh", "-c", fmt.Sprintf("sleep %d && shutdown -r now", delaySeconds))
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to initiate reboot: %w", err)
	}

	return nil
}

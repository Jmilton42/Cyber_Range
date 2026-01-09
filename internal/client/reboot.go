package client

import (
	"fmt"
	"os/exec"
)

// Reboot initiates a system reboot with a specified delay in seconds
func Reboot(delaySecs int) error {
	cmd := exec.Command("shutdown", "/r", "/t", fmt.Sprintf("%d", delaySecs), "/c", "Cyber Range configuration complete - rebooting")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to initiate reboot: %s - %w", string(output), err)
	}
	return nil
}


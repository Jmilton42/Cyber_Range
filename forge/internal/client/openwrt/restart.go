package openwrt

import (
	"fmt"
	"os/exec"
)

// RestartNetwork restarts the network service to apply changes
func RestartNetwork() error {
	cmd := exec.Command("/etc/init.d/network", "restart")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to restart network: %s - %w", string(output), err)
	}
	return nil
}

// Reboot initiates a system reboot
func Reboot() error {
	cmd := exec.Command("reboot")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to reboot: %s - %w", string(output), err)
	}
	return nil
}

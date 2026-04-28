package linux

import (
	"fmt"
	"os/exec"
)

// SetHostname sets the Linux hostname using hostnamectl
// This works on all systemd-based distros (Ubuntu 16.04+, RHEL 7+, Debian 8+, Fedora)
func SetHostname(hostname string) error {
	// Use hostnamectl to set the hostname persistently
	cmd := exec.Command("hostnamectl", "set-hostname", hostname)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("hostnamectl failed: %s - %w", string(output), err)
	}

	return nil
}

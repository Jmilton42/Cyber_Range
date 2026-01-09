package windows

import (
	"fmt"
	"net"
	"os/exec"
	"strings"

	"cyber-range-config/internal/config"
)

// ConfigureNetwork applies network configuration using netsh
func ConfigureNetwork(cfg config.NetworkConfig) error {
	adapterName, err := getPrimaryAdapterName()
	if err != nil {
		return fmt.Errorf("failed to find network adapter: %w", err)
	}

	if cfg.DHCP {
		return configureDHCP(adapterName)
	}

	return configureStatic(adapterName, cfg)
}

// configureDHCP sets the adapter to use DHCP
func configureDHCP(adapterName string) error {
	// Set IP to DHCP
	cmd := exec.Command("netsh", "interface", "ip", "set", "address", adapterName, "dhcp")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set DHCP for IP: %s - %w", string(output), err)
	}

	// Set DNS to DHCP
	cmd = exec.Command("netsh", "interface", "ip", "set", "dns", adapterName, "dhcp")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set DHCP for DNS: %s - %w", string(output), err)
	}

	return nil
}

// configureStatic sets a static IP configuration
func configureStatic(adapterName string, cfg config.NetworkConfig) error {
	// Parse CIDR address
	ip, ipNet, err := net.ParseCIDR(cfg.Address)
	if err != nil {
		return fmt.Errorf("invalid address format: %w", err)
	}

	// Convert subnet mask to dotted decimal
	mask := net.IP(ipNet.Mask).String()

	// Set static IP
	cmd := exec.Command("netsh", "interface", "ip", "set", "address",
		adapterName, "static", ip.String(), mask, cfg.Gateway)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set static IP: %s - %w", string(output), err)
	}

	// Set DNS servers
	if len(cfg.DNS) > 0 {
		// Set primary DNS
		cmd = exec.Command("netsh", "interface", "ip", "set", "dns",
			adapterName, "static", cfg.DNS[0])
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to set primary DNS: %s - %w", string(output), err)
		}

		// Add additional DNS servers
		for i := 1; i < len(cfg.DNS); i++ {
			cmd = exec.Command("netsh", "interface", "ip", "add", "dns",
				adapterName, cfg.DNS[i], fmt.Sprintf("index=%d", i+1))
			if output, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to add DNS %s: %s - %w", cfg.DNS[i], string(output), err)
			}
		}
	}

	return nil
}

// getPrimaryAdapterName returns the name of the primary network adapter
func getPrimaryAdapterName() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if len(iface.HardwareAddr) == 0 {
			continue
		}

		name := iface.Name
		if strings.Contains(strings.ToLower(name), "ethernet") ||
			strings.Contains(strings.ToLower(name), "local area") {
			return name, nil
		}
	}

	// Fallback: first non-loopback interface with MAC
	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if len(iface.HardwareAddr) == 0 {
			continue
		}
		return iface.Name, nil
	}

	return "", fmt.Errorf("no suitable network adapter found")
}

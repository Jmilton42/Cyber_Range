package openwrt

import (
	"fmt"
	"net"
	"os/exec"
	"strings"

	"cyber-range-config/internal/config"
)

// InterfaceMapping maps cloud-init interface names to OpenWrt UCI interface names
// You can customize these names - e.g., change "lan2" to "dmz"
var InterfaceMapping = map[string]string{
	"eth0":  "wan",
	"eth-0": "wan",
	"eth1":  "lan",
	"eth-1": "lan",
	"eth2":  "dmz", // Changed from lan2 to dmz
	"eth-2": "dmz",
	"eth3":  "lan3",
	"eth-3": "lan3",
}

// MapInterfaceName maps a cloud-init interface name to OpenWrt UCI interface name
func MapInterfaceName(cloudInitName string) string {
	if uciName, ok := InterfaceMapping[cloudInitName]; ok {
		return uciName
	}
	// Default: strip "eth-" or "eth" prefix and use as suffix
	name := strings.ToLower(cloudInitName)
	name = strings.ReplaceAll(name, "eth-", "")
	name = strings.ReplaceAll(name, "eth", "")
	if name == "0" {
		return "wan"
	}
	if name == "1" {
		return "lan"
	}
	return "lan" + name
}

// ConfigureNetwork applies network configuration to lan interface (legacy)
func ConfigureNetwork(cfg config.NetworkConfig) error {
	return ConfigureInterface("lan", "eth1", cfg)
}

// ConfigureInterface applies network configuration to a specific UCI interface
// physicalDevice is the Linux device name (e.g., eth0, eth1, eth2)
func ConfigureInterface(uciInterface string, physicalDevice string, cfg config.NetworkConfig) error {
	// Ensure the UCI interface exists and is bound to the physical device
	if err := EnsureInterfaceExists(uciInterface, physicalDevice); err != nil {
		return fmt.Errorf("failed to ensure interface exists: %w", err)
	}

	if cfg.DHCP {
		return configureInterfaceDHCP(uciInterface)
	}
	return configureInterfaceStatic(uciInterface, cfg)
}

// configureInterfaceDHCP sets a specific interface to use DHCP
func configureInterfaceDHCP(uciInterface string) error {
	prefix := fmt.Sprintf("network.%s", uciInterface)
	commands := [][]string{
		{"uci", "set", prefix + ".proto=dhcp"},
		{"uci", "delete", prefix + ".ipaddr"},
		{"uci", "delete", prefix + ".netmask"},
		{"uci", "delete", prefix + ".gateway"},
		{"uci", "delete", prefix + ".dns"},
	}

	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		// Ignore errors for delete commands (key might not exist)
		if args[1] != "delete" {
			if output, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to run %v: %s - %w", args, string(output), err)
			}
		} else {
			cmd.Run() // Ignore error for delete
		}
	}

	return nil
}

// EnsureInterfaceExists creates a UCI interface if it doesn't exist and binds it to the physical device
func EnsureInterfaceExists(uciInterface string, physicalDevice string) error {
	// Check if interface already exists
	checkCmd := exec.Command("uci", "get", fmt.Sprintf("network.%s", uciInterface))
	if err := checkCmd.Run(); err != nil {
		// Interface doesn't exist, create it
		createCmd := exec.Command("uci", "set", fmt.Sprintf("network.%s=interface", uciInterface))
		if output, err := createCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to create interface %s: %s - %w", uciInterface, string(output), err)
		}
	}

	// Bind to physical device (using 'device' for DSA, falling back to 'ifname' for older OpenWrt)
	// Try device first (OpenWrt 21.02+)
	deviceCmd := exec.Command("uci", "set", fmt.Sprintf("network.%s.device=%s", uciInterface, physicalDevice))
	if output, err := deviceCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set device for %s: %s - %w", uciInterface, string(output), err)
	}

	return nil
}

// configureInterfaceStatic sets a static IP configuration on a specific interface
func configureInterfaceStatic(uciInterface string, cfg config.NetworkConfig) error {
	// Parse CIDR address
	ip, ipNet, err := net.ParseCIDR(cfg.Address)
	if err != nil {
		return fmt.Errorf("invalid address format: %w", err)
	}

	// Convert subnet mask to dotted decimal
	mask := net.IP(ipNet.Mask).String()

	prefix := fmt.Sprintf("network.%s", uciInterface)

	// Set static IP configuration
	commands := [][]string{
		{"uci", "set", prefix + ".proto=static"},
		{"uci", "set", fmt.Sprintf("%s.ipaddr=%s", prefix, ip.String())},
		{"uci", "set", fmt.Sprintf("%s.netmask=%s", prefix, mask)},
	}

	// Add gateway if specified
	if cfg.Gateway != "" {
		commands = append(commands, []string{"uci", "set", fmt.Sprintf("%s.gateway=%s", prefix, cfg.Gateway)})
	}

	// Add DNS servers if specified
	if len(cfg.DNS) > 0 {
		// Clear existing DNS first
		commands = append(commands, []string{"uci", "delete", prefix + ".dns"})
		// Add each DNS server
		for _, dns := range cfg.DNS {
			commands = append(commands, []string{"uci", "add_list", fmt.Sprintf("%s.dns=%s", prefix, dns)})
		}
	}

	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		// Ignore errors for delete commands (key might not exist)
		if len(args) > 1 && args[1] == "delete" {
			cmd.Run() // Ignore error
			continue
		}
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to run %v: %s - %w", args, string(output), err)
		}
	}

	return nil
}

// CommitNetworkChanges commits all UCI network changes
func CommitNetworkChanges() error {
	cmd := exec.Command("uci", "commit", "network")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to commit network changes: %s - %w", string(output), err)
	}
	return nil
}

// AddInterfaceToFirewallZone adds a UCI interface to a firewall zone (e.g., "lan" zone)
// zoneName is typically "lan" or "wan"
func AddInterfaceToFirewallZone(uciInterface string, zoneName string) error {
	// Find the zone index by name
	zoneIndex, err := findFirewallZoneIndex(zoneName)
	if err != nil {
		return fmt.Errorf("failed to find firewall zone %s: %w", zoneName, err)
	}

	// Add the interface to the zone's network list
	cmd := exec.Command("uci", "add_list", fmt.Sprintf("firewall.@zone[%d].network=%s", zoneIndex, uciInterface))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add %s to firewall zone %s: %s - %w", uciInterface, zoneName, string(output), err)
	}

	return nil
}

// findFirewallZoneIndex finds the index of a firewall zone by name
func findFirewallZoneIndex(zoneName string) (int, error) {
	// Get list of zones and find the one with matching name
	for i := 0; i < 10; i++ { // Check up to 10 zones
		cmd := exec.Command("uci", "get", fmt.Sprintf("firewall.@zone[%d].name", i))
		output, err := cmd.CombinedOutput()
		if err != nil {
			// No more zones
			break
		}
		name := strings.TrimSpace(string(output))
		if name == zoneName {
			return i, nil
		}
	}
	return -1, fmt.Errorf("zone %s not found", zoneName)
}

// AddInterfaceToDNS adds a UCI interface to dnsmasq listen interfaces
func AddInterfaceToDNS(uciInterface string) error {
	cmd := exec.Command("uci", "add_list", fmt.Sprintf("dhcp.@dnsmasq[0].interface=%s", uciInterface))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add %s to DNS listen interfaces: %s - %w", uciInterface, string(output), err)
	}
	return nil
}

// CommitFirewallChanges commits all UCI firewall changes
func CommitFirewallChanges() error {
	cmd := exec.Command("uci", "commit", "firewall")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to commit firewall changes: %s - %w", string(output), err)
	}
	return nil
}

// CommitDHCPChanges commits all UCI DHCP/DNS changes
func CommitDHCPChanges() error {
	cmd := exec.Command("uci", "commit", "dhcp")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to commit DHCP changes: %s - %w", string(output), err)
	}
	return nil
}

// RestartFirewall restarts the firewall service to apply changes
func RestartFirewall() error {
	cmd := exec.Command("/etc/init.d/firewall", "restart")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to restart firewall: %s - %w", string(output), err)
	}
	return nil
}

// RestartDNSMasq restarts the dnsmasq service to apply changes
func RestartDNSMasq() error {
	cmd := exec.Command("/etc/init.d/dnsmasq", "restart")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to restart dnsmasq: %s - %w", string(output), err)
	}
	return nil
}

package openwrt

import (
	"fmt"
	"net"
	"os/exec"
	"strings"

	"cyber-range-config/internal/config"
)

// InterfaceMapping maps cloud-init interface names to OpenWrt UCI interface names
var InterfaceMapping = map[string]string{
	"eth0":   "wan",
	"eth-0":  "wan",
	"eth1":   "lan",
	"eth-1":  "lan",
	"eth2":   "lan2",
	"eth-2":  "lan2",
	"eth3":   "lan3",
	"eth-3":  "lan3",
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
	return ConfigureInterface("lan", cfg)
}

// ConfigureInterface applies network configuration to a specific UCI interface
func ConfigureInterface(uciInterface string, cfg config.NetworkConfig) error {
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

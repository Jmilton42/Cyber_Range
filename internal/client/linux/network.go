package linux

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"cyber-range-config/internal/config"
)

// NetworkMethod represents the detected network configuration method
type NetworkMethod int

const (
	NetworkMethodUnknown NetworkMethod = iota
	NetworkMethodNetworkManager
	NetworkMethodNetplan
	NetworkMethodIfupdown
)

// ConfigureNetwork applies network configuration using the detected method (single interface, backwards compat)
func ConfigureNetwork(cfg config.NetworkConfig) error {
	method := detectNetworkMethod()

	switch method {
	case NetworkMethodNetworkManager:
		return configureNetworkManager("", cfg) // Empty string = auto-detect
	case NetworkMethodNetplan:
		return configureNetplanSingle(cfg)
	case NetworkMethodIfupdown:
		return configureIfupdownSingle(cfg)
	default:
		return fmt.Errorf("no supported network configuration method found")
	}
}

// ConfigureAllNetworks applies network configuration for multiple interfaces
func ConfigureAllNetworks(networks map[string]config.NetworkConfig) error {
	method := detectNetworkMethod()

	switch method {
	case NetworkMethodNetworkManager:
		return configureNetworkManagerAll(networks)
	case NetworkMethodNetplan:
		return configureNetplanAll(networks)
	case NetworkMethodIfupdown:
		return configureIfupdownAll(networks)
	default:
		return fmt.Errorf("no supported network configuration method found")
	}
}

// detectNetworkMethod determines which network configuration system is in use
func detectNetworkMethod() NetworkMethod {
	// Check for NetworkManager first (most common on modern distros)
	if _, err := exec.LookPath("nmcli"); err == nil {
		// Verify NetworkManager is actually running
		cmd := exec.Command("systemctl", "is-active", "--quiet", "NetworkManager")
		if cmd.Run() == nil {
			return NetworkMethodNetworkManager
		}
	}

	// Check for Netplan (Ubuntu 18.04+)
	if _, err := os.Stat("/etc/netplan"); err == nil {
		return NetworkMethodNetplan
	}

	// Check for ifupdown (older Debian/Ubuntu)
	if _, err := os.Stat("/etc/network/interfaces"); err == nil {
		return NetworkMethodIfupdown
	}

	return NetworkMethodUnknown
}

// configureNetworkManager configures a single interface using nmcli
// If ifaceName is empty, auto-detects the primary connection
func configureNetworkManager(ifaceName string, cfg config.NetworkConfig) error {
	var connName string
	var err error

	if ifaceName == "" {
		// Auto-detect primary connection
		connName, err = getNMConnectionName()
		if err != nil {
			return fmt.Errorf("failed to find NetworkManager connection: %w", err)
		}
	} else {
		// Find or create connection for specific interface
		connName, err = getNMConnectionForInterface(ifaceName)
		if err != nil {
			return fmt.Errorf("failed to find connection for %s: %w", ifaceName, err)
		}
	}

	if cfg.DHCP {
		// Set to DHCP
		cmd := exec.Command("nmcli", "connection", "modify", connName,
			"ipv4.method", "auto",
			"ipv4.addresses", "",
			"ipv4.gateway", "",
			"ipv4.dns", "")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("nmcli modify failed: %s - %w", string(output), err)
		}
	} else {
		// Parse CIDR address
		ip, ipNet, err := net.ParseCIDR(cfg.Address)
		if err != nil {
			return fmt.Errorf("invalid address format: %w", err)
		}

		// Get prefix length
		ones, _ := ipNet.Mask.Size()
		address := fmt.Sprintf("%s/%d", ip.String(), ones)

		// Build nmcli command for static IP
		args := []string{"connection", "modify", connName,
			"ipv4.method", "manual",
			"ipv4.addresses", address,
		}

		if cfg.Gateway != "" {
			args = append(args, "ipv4.gateway", cfg.Gateway)
		}

		if len(cfg.DNS) > 0 {
			args = append(args, "ipv4.dns", strings.Join(cfg.DNS, ","))
		}

		cmd := exec.Command("nmcli", args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("nmcli modify failed: %s - %w", string(output), err)
		}
	}

	// Add custom routes (works with both DHCP and static)
	if len(cfg.Routes) > 0 {
		// Clear existing routes first
		cmd := exec.Command("nmcli", "connection", "modify", connName, "ipv4.routes", "")
		cmd.Run() // Ignore error if no routes exist

		// Build routes string: "dest1 nexthop1, dest2 nexthop2"
		var routeStrs []string
		for _, route := range cfg.Routes {
			routeStrs = append(routeStrs, fmt.Sprintf("%s %s", route.To, route.Via))
		}
		routesArg := strings.Join(routeStrs, ", ")

		cmd = exec.Command("nmcli", "connection", "modify", connName, "ipv4.routes", routesArg)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("nmcli routes failed: %s - %w", string(output), err)
		}
	}

	// Apply the changes by reactivating the connection
	cmd := exec.Command("nmcli", "connection", "up", connName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("nmcli connection up failed: %s - %w", string(output), err)
	}

	return nil
}

// configureNetworkManagerAll configures all interfaces using NetworkManager
func configureNetworkManagerAll(networks map[string]config.NetworkConfig) error {
	for ifaceName, cfg := range networks {
		if err := configureNetworkManager(ifaceName, cfg); err != nil {
			return fmt.Errorf("failed to configure %s: %w", ifaceName, err)
		}
	}
	return nil
}

// getNMConnectionForInterface finds or creates a NetworkManager connection for an interface
func getNMConnectionForInterface(ifaceName string) (string, error) {
	// First, try to find existing connection for this interface
	cmd := exec.Command("nmcli", "-t", "-f", "NAME,DEVICE", "connection", "show")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to list connections: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		parts := strings.Split(line, ":")
		if len(parts) >= 2 && parts[1] == ifaceName {
			return parts[0], nil
		}
	}

	// No existing connection, try to find by connection name matching interface
	for _, line := range lines {
		parts := strings.Split(line, ":")
		if len(parts) >= 1 && parts[0] == ifaceName {
			return parts[0], nil
		}
	}

	// Still not found - create a new connection
	connName := fmt.Sprintf("cyber-range-%s", ifaceName)
	cmd = exec.Command("nmcli", "connection", "add", "type", "ethernet",
		"con-name", connName, "ifname", ifaceName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to create connection: %s - %w", string(output), err)
	}

	return connName, nil
}

// getNMConnectionName finds the primary NetworkManager connection name
func getNMConnectionName() (string, error) {
	// List active connections
	cmd := exec.Command("nmcli", "-t", "-f", "NAME,TYPE", "connection", "show", "--active")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to list connections: %w", err)
	}

	// Find first ethernet connection
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		parts := strings.Split(line, ":")
		if len(parts) >= 2 && strings.Contains(parts[1], "ethernet") {
			return parts[0], nil
		}
	}

	// Fallback: return first connection
	if len(lines) > 0 && lines[0] != "" {
		parts := strings.Split(lines[0], ":")
		if len(parts) >= 1 {
			return parts[0], nil
		}
	}

	return "", fmt.Errorf("no active network connection found")
}

// configureNetplanSingle configures a single interface using Netplan (backwards compat)
func configureNetplanSingle(cfg config.NetworkConfig) error {
	// Find primary interface
	ifaceName, err := getPrimaryInterface()
	if err != nil {
		return fmt.Errorf("failed to find primary interface: %w", err)
	}

	networks := map[string]config.NetworkConfig{
		ifaceName: cfg,
	}
	return configureNetplanAll(networks)
}

// configureNetplanAll configures all interfaces using Netplan (Ubuntu 18.04+)
func configureNetplanAll(networks map[string]config.NetworkConfig) error {
	content := `# Generated by Cyber Range Configuration Client
network:
  version: 2
  ethernets:
`

	for ifaceName, cfg := range networks {
		content += fmt.Sprintf("    %s:\n", ifaceName)

		if cfg.DHCP {
			content += "      dhcp4: true\n"
			// Add custom routes for DHCP
			if len(cfg.Routes) > 0 {
				content += "      routes:\n"
				for _, route := range cfg.Routes {
					content += fmt.Sprintf("        - to: %s\n          via: %s\n", route.To, route.Via)
				}
			}
		} else {
			content += "      dhcp4: false\n"

			// Validate and add address
			if cfg.Address != "" {
				_, _, err := net.ParseCIDR(cfg.Address)
				if err != nil {
					return fmt.Errorf("invalid address format for %s: %w", ifaceName, err)
				}
				content += "      addresses:\n"
				content += fmt.Sprintf("        - %s\n", cfg.Address)
			}

			// Add routes section if we have gateway or custom routes
			if cfg.Gateway != "" || len(cfg.Routes) > 0 {
				content += "      routes:\n"
				if cfg.Gateway != "" {
					content += fmt.Sprintf("        - to: default\n          via: %s\n", cfg.Gateway)
				}
				for _, route := range cfg.Routes {
					content += fmt.Sprintf("        - to: %s\n          via: %s\n", route.To, route.Via)
				}
			}

			// Add DNS servers
			if len(cfg.DNS) > 0 {
				content += "      nameservers:\n        addresses:\n"
				for _, dns := range cfg.DNS {
					content += fmt.Sprintf("          - %s\n", dns)
				}
			}
		}
	}

	// Write netplan config
	configPath := "/etc/netplan/99-cyber-range.yaml"
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write netplan config: %w", err)
	}

	// Apply netplan
	cmd := exec.Command("netplan", "apply")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("netplan apply failed: %s - %w", string(output), err)
	}

	return nil
}

// configureIfupdownSingle configures a single interface using ifupdown (backwards compat)
func configureIfupdownSingle(cfg config.NetworkConfig) error {
	// Find primary interface
	ifaceName, err := getPrimaryInterface()
	if err != nil {
		return fmt.Errorf("failed to find primary interface: %w", err)
	}

	networks := map[string]config.NetworkConfig{
		ifaceName: cfg,
	}
	return configureIfupdownAll(networks)
}

// configureIfupdownAll configures all interfaces using /etc/network/interfaces (older Debian)
func configureIfupdownAll(networks map[string]config.NetworkConfig) error {
	content := "# Generated by Cyber Range Configuration Client\n"

	for ifaceName, cfg := range networks {
		if cfg.DHCP {
			content += fmt.Sprintf("\nauto %s\niface %s inet dhcp\n", ifaceName, ifaceName)
		} else {
			// Parse CIDR address
			ip, ipNet, err := net.ParseCIDR(cfg.Address)
			if err != nil {
				return fmt.Errorf("invalid address format for %s: %w", ifaceName, err)
			}

			// Convert netmask to dotted decimal
			mask := net.IP(ipNet.Mask).String()

			content += fmt.Sprintf("\nauto %s\niface %s inet static\n", ifaceName, ifaceName)
			content += fmt.Sprintf("    address %s\n", ip.String())
			content += fmt.Sprintf("    netmask %s\n", mask)

			if cfg.Gateway != "" {
				content += fmt.Sprintf("    gateway %s\n", cfg.Gateway)
			}

			if len(cfg.DNS) > 0 {
				content += fmt.Sprintf("    dns-nameservers %s\n", strings.Join(cfg.DNS, " "))
			}
		}

		// Add custom routes using post-up commands (works with both DHCP and static)
		for _, route := range cfg.Routes {
			content += fmt.Sprintf("    post-up ip route add %s via %s\n", route.To, route.Via)
			content += fmt.Sprintf("    pre-down ip route del %s via %s\n", route.To, route.Via)
		}
	}

	// Ensure interfaces.d directory exists
	interfacesDDir := "/etc/network/interfaces.d"
	if err := os.MkdirAll(interfacesDDir, 0755); err != nil {
		return fmt.Errorf("failed to create interfaces.d: %w", err)
	}

	// Write config file
	configPath := filepath.Join(interfacesDDir, "cyber-range")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write interfaces config: %w", err)
	}

	// Restart networking
	cmd := exec.Command("systemctl", "restart", "networking")
	if err := cmd.Run(); err != nil {
		// Fallback: restart each interface individually
		for ifaceName := range networks {
			exec.Command("ifdown", ifaceName).Run()
			exec.Command("ifup", ifaceName).Run()
		}
	}

	return nil
}

// getPrimaryInterface returns the name of the primary network interface
func getPrimaryInterface() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		// Skip loopback
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		// Skip interfaces without MAC
		if len(iface.HardwareAddr) == 0 {
			continue
		}
		// Skip down interfaces
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		// Prefer eth* or en* interfaces
		name := iface.Name
		if strings.HasPrefix(name, "eth") || strings.HasPrefix(name, "en") {
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

	return "", fmt.Errorf("no suitable network interface found")
}

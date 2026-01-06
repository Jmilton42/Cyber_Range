package client

import (
	"fmt"
	"net"
	"strings"
)

// GetPrimaryMAC returns the MAC address of the primary network adapter
func GetPrimaryMAC() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to get network interfaces: %w", err)
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

		// Skip interfaces that are down
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		// Return first valid MAC
		mac := iface.HardwareAddr.String()
		if mac != "" {
			return strings.ToLower(mac), nil
		}
	}

	return "", fmt.Errorf("no valid network interface found")
}

// GetMACByName returns the MAC address of a specific interface
func GetMACByName(name string) (string, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return "", fmt.Errorf("interface %s not found: %w", name, err)
	}

	if len(iface.HardwareAddr) == 0 {
		return "", fmt.Errorf("interface %s has no MAC address", name)
	}

	return strings.ToLower(iface.HardwareAddr.String()), nil
}

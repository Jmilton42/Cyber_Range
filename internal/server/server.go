package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"cyber-range-config/internal/config"

	"gopkg.in/yaml.v3"
)

// Server handles HTTP requests for configuration
type Server struct {
	instancesFile string
	instances     []config.LXDInstance
	mu            sync.RWMutex

	// Idle timeout tracking
	lastActivity time.Time
	activityMu   sync.RWMutex
}

// NewServer creates a new configuration server
func NewServer(instancesFile string) (*Server, error) {
	s := &Server{
		instancesFile: instancesFile,
		lastActivity:  time.Now(),
	}

	if err := s.loadInstances(); err != nil {
		return nil, err
	}

	return s, nil
}

// loadInstances loads the LXD instances from the JSON file
func (s *Server) loadInstances() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.instancesFile)
	if err != nil {
		return fmt.Errorf("failed to read instances file: %w", err)
	}

	var instances []config.LXDInstance
	if err := json.Unmarshal(data, &instances); err != nil {
		return fmt.Errorf("failed to parse instances JSON: %w", err)
	}

	s.instances = instances
	log.Printf("Loaded %d instances from %s", len(instances), s.instancesFile)

	return nil
}

// Reload reloads the instances file (can be called to refresh)
func (s *Server) Reload() error {
	return s.loadInstances()
}

// updateActivity updates the last activity timestamp
func (s *Server) updateActivity() {
	s.activityMu.Lock()
	s.lastActivity = time.Now()
	s.activityMu.Unlock()
}

// GetLastActivity returns the last activity timestamp
func (s *Server) GetLastActivity() time.Time {
	s.activityMu.RLock()
	defer s.activityMu.RUnlock()
	return s.lastActivity
}

// HandleConfig handles GET /config?mac={mac_address}
func (s *Server) HandleConfig(w http.ResponseWriter, r *http.Request) {
	s.updateActivity()

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get MAC address from query parameter
	mac := r.URL.Query().Get("mac")
	if mac == "" {
		http.Error(w, "Missing 'mac' query parameter", http.StatusBadRequest)
		return
	}

	// Normalize MAC address
	mac = strings.ToLower(strings.ReplaceAll(mac, "-", ":"))
	log.Printf("Config request for MAC: %s", mac)

	// Find instance by MAC address
	s.mu.RLock()
	instance := s.findInstanceByMAC(mac)
	s.mu.RUnlock()

	if instance == nil {
		log.Printf("No instance found for MAC: %s", mac)
		http.Error(w, "Instance not found", http.StatusNotFound)
		return
	}

	log.Printf("Found instance: %s", instance.Name)

	// Parse all network configs
	networks := s.parseAllNetworkConfigs(instance)

	// Get primary network (first one) for backwards compatibility
	var primaryNetwork config.NetworkConfig
	for _, netCfg := range networks {
		primaryNetwork = netCfg
		break
	}
	// If no networks found, default to DHCP
	if len(networks) == 0 {
		primaryNetwork = config.NetworkConfig{DHCP: true}
	}

	// Build response
	response := config.ConfigResponse{
		Hostname: instance.Name,
		Network:  primaryNetwork,
		Networks: networks,
	}

	// Send JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Sent config for %s: %d network(s), primary: dhcp=%v, address=%s", instance.Name, len(networks), primaryNetwork.DHCP, primaryNetwork.Address)
}

// findInstanceByMAC finds an instance by checking volatile.*.hwaddr fields
func (s *Server) findInstanceByMAC(mac string) *config.LXDInstance {
	for i := range s.instances {
		inst := &s.instances[i]

		// Check all config keys for MAC addresses (volatile.eth-X.hwaddr pattern)
		for key, value := range inst.Config {
			if strings.Contains(key, "hwaddr") {
				if strings.EqualFold(value, mac) {
					return inst
				}
			}
		}
	}
	return nil
}

// parseAllNetworkConfigs parses all ethernet configs from cloud-init.network-config
func (s *Server) parseAllNetworkConfigs(instance *config.LXDInstance) map[string]config.NetworkConfig {
	networks := make(map[string]config.NetworkConfig)
	cloudInitConfig := instance.Config["cloud-init.network-config"]

	// Check for simple DHCP string
	if strings.TrimSpace(strings.ToUpper(cloudInitConfig)) == "DHCP" {
		return networks // Empty map, client will use DHCP
	}

	// Try to parse as netplan YAML
	var cloudInit config.CloudInitNetwork
	if err := yaml.Unmarshal([]byte(cloudInitConfig), &cloudInit); err != nil {
		log.Printf("Failed to parse cloud-init config for %s: %v", instance.Name, err)
		return networks
	}

	// Parse ALL ethernet devices
	for ifaceName, eth := range cloudInit.Ethernets {
		netConfig := config.NetworkConfig{DHCP: eth.DHCP4}

		// Get first address (for static IP)
		if len(eth.Addresses) > 0 {
			netConfig.Address = eth.Addresses[0]
		}

		// Get gateway from gateway4
		if eth.Gateway4 != "" {
			netConfig.Gateway = eth.Gateway4
		}

		// Process routes - extract default gateway and collect custom routes
		for _, route := range eth.Routes {
			if route.To == "default" || route.To == "0.0.0.0/0" {
				// Use as gateway if not already set
				if netConfig.Gateway == "" {
					netConfig.Gateway = route.Via
				}
			} else {
				// Custom route - add to routes list
				netConfig.Routes = append(netConfig.Routes, route)
			}
		}

		// Get DNS servers
		netConfig.DNS = eth.Nameservers.Addresses

		networks[ifaceName] = netConfig
	}

	return networks
}

// parseNetworkConfig parses cloud-init.network-config from the instance (legacy, returns first)
func (s *Server) parseNetworkConfig(instance *config.LXDInstance) config.NetworkConfig {
	networks := s.parseAllNetworkConfigs(instance)
	for _, netCfg := range networks {
		return netCfg
	}
	return config.NetworkConfig{DHCP: true}
}

// HandleReload handles POST /reload to refresh the instances file
func (s *Server) HandleReload(w http.ResponseWriter, r *http.Request) {
	s.updateActivity()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := s.Reload(); err != nil {
		log.Printf("Error reloading instances: %v", err)
		http.Error(w, "Failed to reload", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Reloaded %d instances\n", len(s.instances))
}

// HandleStatus handles GET /status to check server status
func (s *Server) HandleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	instanceCount := len(s.instances)
	s.mu.RUnlock()

	lastActivity := s.GetLastActivity()

	status := map[string]interface{}{
		"instances":     instanceCount,
		"last_activity": lastActivity.Format(time.RFC3339),
		"uptime_seconds": time.Since(lastActivity).Seconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// RegisterRoutes registers HTTP routes
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/config", s.HandleConfig)
	mux.HandleFunc("/reload", s.HandleReload)
	mux.HandleFunc("/status", s.HandleStatus)
}

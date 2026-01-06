package config

// ServerConfig holds the server configuration
type ServerConfig struct {
	Listen        string `yaml:"listen"`
	InstancesFile string `yaml:"instances_file"`
}

// ConfigResponse is sent from server to client
type ConfigResponse struct {
	Hostname string        `json:"hostname"`
	Network  NetworkConfig `json:"network"`
}

// NetworkConfig holds network configuration for the client
type NetworkConfig struct {
	DHCP    bool     `json:"dhcp"`
	Address string   `json:"address,omitempty"` // CIDR format: 192.168.1.100/24
	Gateway string   `json:"gateway,omitempty"`
	DNS     []string `json:"dns,omitempty"`
}

// LXDInstance represents an instance from lxc list --format json
type LXDInstance struct {
	Name   string            `json:"name"`
	Config map[string]string `json:"config"`
}

// CloudInitNetwork represents netplan-style network config
type CloudInitNetwork struct {
	Network struct {
		Version   int                          `yaml:"version"`
		Ethernets map[string]CloudInitEthernet `yaml:"ethernets"`
	} `yaml:"network"`
}

// CloudInitEthernet represents an ethernet device in cloud-init
type CloudInitEthernet struct {
	DHCP4       bool     `yaml:"dhcp4"`
	Addresses   []string `yaml:"addresses"`
	Gateway4    string   `yaml:"gateway4"`
	Routes      []Route  `yaml:"routes"`
	Nameservers struct {
		Addresses []string `yaml:"addresses"`
	} `yaml:"nameservers"`
}

// Route represents a network route
type Route struct {
	To  string `yaml:"to"`
	Via string `yaml:"via"`
}

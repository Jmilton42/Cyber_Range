package forge

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// ForgeConfig holds configuration loaded from config.yaml
type ForgeConfig struct {
	Listen      string `yaml:"listen"`       // e.g., "10.8.11.202:8080"
	IdleTimeout string `yaml:"idle_timeout"` // e.g., "5m"
}

// ServerBinary is the path to the server binary
const ServerBinary = "/home/ceroc/InSPIRE/bin/server"

// LoadForgeConfig loads configuration from config.yaml in the server binary's directory
func LoadForgeConfig() (*ForgeConfig, error) {
	// Look for config.yaml in same directory as server binary
	configPath := filepath.Join(filepath.Dir(ServerBinary), "config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg ForgeConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// ParseListenAddress parses a listen address like "10.8.11.202:8080" into IP and port
func ParseListenAddress(listen string) (ip string, port string) {
	// Handle ":8080" format (all interfaces)
	if strings.HasPrefix(listen, ":") {
		return "", strings.TrimPrefix(listen, ":")
	}

	// Handle "ip:port" format
	parts := strings.Split(listen, ":")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}

	// Fallback
	return listen, "8080"
}

// ParseProjectName reads main.tf and extracts the project_name variable default value
func ParseProjectName(dir string) (string, error) {
	mainTfPath := dir + "/main.tf"
	
	content, err := os.ReadFile(mainTfPath)
	if err != nil {
		return "", fmt.Errorf("failed to read main.tf: %w", err)
	}

	// Match: variable "project_name" { ... default = "value" ... }
	// This regex handles multiline variable blocks
	re := regexp.MustCompile(`variable\s+"project_name"\s*\{[^}]*default\s*=\s*"([^"]+)"`)
	matches := re.FindSubmatch(content)
	
	if len(matches) < 2 {
		return "", fmt.Errorf("could not find project_name variable with default value in main.tf")
	}

	return string(matches[1]), nil
}

// GetWorkingDir returns the working directory, applying -chdir if specified
func GetWorkingDir(chdir string) (string, error) {
	if chdir != "" {
		// Verify directory exists
		info, err := os.Stat(chdir)
		if err != nil {
			return "", fmt.Errorf("cannot access directory %s: %w", chdir, err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("%s is not a directory", chdir)
		}
		return chdir, nil
	}

	return os.Getwd()
}

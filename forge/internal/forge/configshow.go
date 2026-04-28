package forge

import (
	"fmt"
	"os"
	"path/filepath"
)

// ResolvedConfig is the merged view of DefaultDeployConfig() plus the source
// path of the config.yaml that contributed to it (if any). It exists so
// `forge config` can show operators exactly which file was loaded.
type ResolvedConfig struct {
	ConfigFile     string `json:"config_file"`
	ServerIP       string `json:"server_ip"`
	ServerPort     string `json:"server_port"`
	IdleTimeout    string `json:"idle_timeout"`
	InstancesFile  string `json:"instances_file"`
	StartWinScript string `json:"start_win_script"`
	ScriptsDir     string `json:"scripts_dir"`
	SubnetsFile    string `json:"subnets_file"`
}

// LoadResolvedConfig populates a ResolvedConfig by combining the defaults
// already produced by DefaultDeployConfig with a best-effort lookup of the
// config.yaml path used by LoadForgeConfig.
func LoadResolvedConfig() ResolvedConfig {
	cfg := DefaultDeployConfig()
	resolved := ResolvedConfig{
		ConfigFile:     findConfigFile(),
		ServerIP:       cfg.ServerIP,
		ServerPort:     cfg.ServerPort,
		IdleTimeout:    cfg.IdleTimeout,
		InstancesFile:  cfg.InstancesFile,
		StartWinScript: cfg.StartWinScript,
		ScriptsDir:     filepath.Dir(cfg.StartWinScript),
		SubnetsFile:    SubnetsFile,
	}
	return resolved
}

// findConfigFile walks the same candidate list as LoadForgeConfig and returns
// the first one that exists. Empty string means we are running on built-in
// defaults.
func findConfigFile() string {
	candidates := []string{}
	if env := os.Getenv("FORGE_CONFIG"); env != "" {
		candidates = append(candidates, env)
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "config.yaml"))
	}
	candidates = append(candidates, "/home/ceroc/InSPIRE/bin/config.yaml")

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

// PrintConfigText renders a ResolvedConfig in a fixed-column text layout.
func PrintConfigText(r ResolvedConfig) {
	if r.ConfigFile == "" {
		fmt.Println("Config file:      (none - using built-in defaults)")
	} else {
		fmt.Printf("Config file:      %s\n", r.ConfigFile)
	}
	fmt.Printf("Server IP:        %s\n", r.ServerIP)
	fmt.Printf("Server port:      %s\n", r.ServerPort)
	fmt.Printf("Idle timeout:     %s\n", r.IdleTimeout)
	fmt.Printf("Instances file:   %s\n", r.InstancesFile)
	fmt.Printf("Start win script: %s\n", r.StartWinScript)
	fmt.Printf("Scripts dir:      %s\n", r.ScriptsDir)
	fmt.Printf("Subnets file:     %s\n", r.SubnetsFile)
}

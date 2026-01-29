package forge

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// DeployConfig holds deployment configuration
type DeployConfig struct {
	ServerBinary    string
	ServerPort      string
	ServerIP        string
	InstancesFile   string
	IdleTimeout     string
	StartWinScript  string
}

// DefaultDeployConfig returns default deployment configuration
func DefaultDeployConfig() DeployConfig {
	return DeployConfig{
		ServerBinary:   "/home/ceroc/InSPIRE/bin/server",
		ServerPort:     "8080",
		ServerIP:       "10.0.14.6",
		InstancesFile:  "instances.json",
		IdleTimeout:    "5m",
		StartWinScript: "/home/ceroc/InSPIRE/bin/scripts/start_win.sh",
	}
}

// WaitForVMs waits for VMs to initialize
func WaitForVMs(seconds int) {
	time.Sleep(time.Duration(seconds) * time.Second)
}

// ExportInstances exports LXD instances to JSON file
func ExportInstances(workDir, projectName, instancesFile string) error {
	// Switch to project
	switchCmd := exec.Command("lxc", "project", "switch", projectName)
	switchCmd.Dir = workDir
	switchCmd.Stdout = os.Stdout
	switchCmd.Stderr = os.Stderr
	if err := switchCmd.Run(); err != nil {
		return fmt.Errorf("failed to switch to project %s: %w", projectName, err)
	}

	// Export instance list
	listCmd := exec.Command("lxc", "list", "--format", "json")
	listCmd.Dir = workDir
	output, err := listCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	// Write to file
	instancesPath := filepath.Join(workDir, instancesFile)
	if err := os.WriteFile(instancesPath, output, 0644); err != nil {
		return fmt.Errorf("failed to write instances file: %w", err)
	}

	// Count instances
	var instances []interface{}
	if err := json.Unmarshal(output, &instances); err == nil {
		fmt.Printf("\033[32m[INFO]\033[0m Exported %d instances to %s\n", len(instances), instancesFile)
	}

	return nil
}

// StartServer starts the config server
func StartServer(workDir string, config DeployConfig) error {
	// Kill any existing server
	StopServer()

	// Check if server binary exists
	if _, err := os.Stat(config.ServerBinary); os.IsNotExist(err) {
		return fmt.Errorf("server binary not found at %s", config.ServerBinary)
	}

	instancesPath := filepath.Join(workDir, config.InstancesFile)
	listenAddr := fmt.Sprintf("%s:%s", config.ServerIP, config.ServerPort)

	// Start server in background
	cmd := exec.Command(config.ServerBinary,
		"-listen", listenAddr,
		"-instances", instancesPath,
		"-idle-timeout", config.IdleTimeout,
	)
	cmd.Dir = workDir

	// Redirect output to log file
	logFile, err := os.Create(filepath.Join(workDir, "server.log"))
	if err != nil {
		return fmt.Errorf("failed to create server log: %w", err)
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("failed to start server: %w", err)
	}

	// Wait a moment and check if server is still running
	time.Sleep(2 * time.Second)

	if cmd.Process != nil {
		// Check if process is still running
		if err := cmd.Process.Signal(os.Signal(nil)); err == nil {
			fmt.Printf("\033[32m[INFO]\033[0m Server started (PID: %d)\n", cmd.Process.Pid)
			fmt.Printf("\033[32m[INFO]\033[0m Server listening on %s\n", listenAddr)
			fmt.Printf("\033[32m[INFO]\033[0m Server will auto-shutdown after %s of inactivity\n", config.IdleTimeout)
		}
	}

	return nil
}

// StopServer stops the config server
func StopServer() {
	// Use pkill to find and kill server processes
	cmd := exec.Command("bash", "-c", "pgrep -x server | xargs -r kill 2>/dev/null || true")
	cmd.Run() // Ignore errors - server might not be running
}

// StartWindowsVMs starts Windows VMs using the start_win.sh script
func StartWindowsVMs(projectName string, scriptPath string) error {
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("start_win.sh script not found at %s", scriptPath)
	}

	cmd := exec.Command("bash", scriptPath, projectName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start Windows VMs: %w", err)
	}

	return nil
}

// RunPostApply runs all post-apply steps
func RunPostApply(workDir, projectName string, config DeployConfig) error {
	fmt.Println()

	// Wait for VMs
	fmt.Printf("\033[32m[INFO]\033[0m Waiting for VMs to initialize...\n")
	WaitForVMs(10)

	// Export instances
	fmt.Printf("\033[32m[INFO]\033[0m Exporting LXD instances...\n")
	if err := ExportInstances(workDir, projectName, config.InstancesFile); err != nil {
		fmt.Printf("\033[33m[WARN]\033[0m %s\n", err.Error())
	}

	// Start server
	fmt.Printf("\033[32m[INFO]\033[0m Starting config server...\n")
	if err := StartServer(workDir, config); err != nil {
		fmt.Printf("\033[33m[WARN]\033[0m %s\n", err.Error())
	}

	// Start Windows VMs
	fmt.Printf("\033[32m[INFO]\033[0m Starting Windows VMs...\n")
	if err := StartWindowsVMs(projectName, config.StartWinScript); err != nil {
		fmt.Printf("\033[33m[WARN]\033[0m %s\n", err.Error())
	}

	return nil
}

// RunPreDestroy runs all pre-destroy steps
func RunPreDestroy() {
	fmt.Printf("\033[32m[INFO]\033[0m Stopping config server...\n")
	StopServer()
}

// PrintDeploymentComplete prints deployment completion info
func PrintDeploymentComplete(config DeployConfig, subnetOctet int) {
	fmt.Println()
	fmt.Println("==========================================")
	fmt.Println("  Deployment Complete!")
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Printf("Server running at: http://%s:%s\n", config.ServerIP, config.ServerPort)
	fmt.Printf("Guac subnet: 10.0.%d.0/24 (gateway: 10.0.%d.1)\n", subnetOctet, subnetOctet)
	fmt.Printf("Idle timeout: %s (server will auto-shutdown after no requests)\n", config.IdleTimeout)
	fmt.Println()
	fmt.Println("Endpoints:")
	fmt.Println("  GET  /config?mac=XX:XX:XX:XX:XX:XX  - Get VM config")
	fmt.Println("  POST /reload                         - Reload instances.json")
	fmt.Println("  GET  /status                         - Check server status")
	fmt.Println()
	fmt.Println("Windows VMs will automatically configure themselves on boot.")
}

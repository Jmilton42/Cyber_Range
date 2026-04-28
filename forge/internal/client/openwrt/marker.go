package openwrt

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// MarkerDir is where marker and log files are stored
	MarkerDir = "/etc/cyber-range"
	// MarkerFile indicates configuration has been applied
	MarkerFile = ".configured"
	// LogFile is the log file name
	LogFile = "config.log"
)

// EnsureMarkerDir creates the marker directory if needed
func EnsureMarkerDir() error {
	return os.MkdirAll(MarkerDir, 0755)
}

// IsConfigured checks if the marker file exists
func IsConfigured() bool {
	markerPath := filepath.Join(MarkerDir, MarkerFile)
	_, err := os.Stat(markerPath)
	return err == nil
}

// CreateMarker creates the marker file
func CreateMarker(instanceName string) error {
	if err := EnsureMarkerDir(); err != nil {
		return fmt.Errorf("failed to create marker directory: %w", err)
	}

	markerPath := filepath.Join(MarkerDir, MarkerFile)
	content := fmt.Sprintf("Configured at: %s\nInstance: %s\n", time.Now().Format(time.RFC3339), instanceName)

	if err := os.WriteFile(markerPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create marker file: %w", err)
	}

	return nil
}

// GetLogPath returns the path to the log file
func GetLogPath() string {
	return filepath.Join(MarkerDir, LogFile)
}

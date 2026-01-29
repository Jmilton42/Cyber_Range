package main

import (
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"time"

	"cyber-range-config/internal/client/common"
	"cyber-range-config/internal/client/linux"
	"cyber-range-config/internal/config"
)

const (
	maxStartupDelay = 30 // Maximum random delay in seconds
)

func main() {
	// Parse flags
	serverURL := flag.String("server", "", "Configuration server URL (e.g., http://server:8080)")
	interfaceName := flag.String("interface", "", "Network interface name (optional)")
	noDelay := flag.Bool("no-delay", false, "Skip random startup delay")
	flag.Parse()

	// Set up logging
	if err := linux.EnsureMarkerDir(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create log directory: %v\n", err)
		os.Exit(1)
	}

	logFile, err := os.OpenFile(linux.GetLogPath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()

	log.SetOutput(io.MultiWriter(os.Stdout, logFile))
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("=== Cyber Range Configuration Client (Linux) Starting ===")

	// Check if already configured
	if linux.IsConfigured() {
		log.Println("System already configured (marker file exists). Exiting.")
		os.Exit(0)
	}

	// Validate server URL
	if *serverURL == "" {
		log.Fatal("Server URL is required. Use -server flag.")
	}

	// Random startup delay to stagger requests
	if !*noDelay {
		delay := randomDelay(maxStartupDelay)
		log.Printf("Waiting %d seconds before requesting config (staggered startup)...", delay)
		time.Sleep(time.Duration(delay) * time.Second)
	}

	// Get MAC address
	var mac string
	if *interfaceName != "" {
		mac, err = common.GetMACByName(*interfaceName)
	} else {
		mac, err = common.GetPrimaryMAC()
	}
	if err != nil {
		log.Fatalf("Failed to get MAC address: %v", err)
	}
	log.Printf("Using MAC address: %s", mac)

	// Request configuration with retries (60 retries × 60s = 60 minutes max)
	cfg, err := requestConfigWithRetry(*serverURL, mac, 60, 60*time.Second)
	if err != nil {
		log.Fatalf("Failed to get configuration: %v", err)
	}
	log.Printf("Received config: hostname=%s, dhcp=%v", cfg.Hostname, cfg.Network.DHCP)

	// Apply hostname
	log.Printf("Setting hostname to: %s", cfg.Hostname)
	if err := linux.SetHostname(cfg.Hostname); err != nil {
		log.Fatalf("Failed to set hostname: %v", err)
	}
	log.Println("Hostname set successfully")

	// Apply network configuration
	log.Println("Configuring network...")

	// Check if we have multiple networks
	if len(cfg.Networks) > 0 {
		log.Printf("Found %d network interface(s) to configure", len(cfg.Networks))
		for ifaceName, netCfg := range cfg.Networks {
			log.Printf("  - %s: dhcp=%v, address=%s, routes=%d", ifaceName, netCfg.DHCP, netCfg.Address, len(netCfg.Routes))
		}
		if err := linux.ConfigureAllNetworks(cfg.Networks); err != nil {
			log.Fatalf("Failed to configure networks: %v", err)
		}
	} else {
		// Fallback to single network (backwards compatibility)
		log.Printf("Using single network config: dhcp=%v, address=%s", cfg.Network.DHCP, cfg.Network.Address)
		if err := linux.ConfigureNetwork(cfg.Network); err != nil {
			log.Fatalf("Failed to configure network: %v", err)
		}
	}
	log.Println("Network configured successfully")

	// Create marker file
	if err := linux.CreateMarker(cfg.Hostname); err != nil {
		log.Fatalf("Failed to create marker file: %v", err)
	}
	log.Println("Marker file created.")

	log.Println("=== Configuration Complete ===")
	log.Println("Initiating system reboot in 5 seconds...")

	// Reboot with 5 second delay to allow logs to flush
	if err := linux.Reboot(5); err != nil {
		log.Fatalf("Failed to initiate reboot: %v", err)
	}
}

// randomDelay returns a random number between 0 and max
func randomDelay(max int) int {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max+1)))
	if err != nil {
		return max / 2 // Fallback to middle value
	}
	return int(n.Int64())
}

// requestConfigWithRetry requests config with retries
func requestConfigWithRetry(serverURL, mac string, maxRetries int, retryDelay time.Duration) (*config.ConfigResponse, error) {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			log.Printf("Retry %d/%d after %v...", i, maxRetries-1, retryDelay)
			time.Sleep(retryDelay)
		}

		cfg, err := requestConfig(serverURL, mac)
		if err == nil {
			return cfg, nil
		}

		lastErr = err
		log.Printf("Request failed: %v", err)
	}

	return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

// requestConfig requests configuration from the server
func requestConfig(serverURL, mac string) (*config.ConfigResponse, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, fmt.Errorf("invalid server URL: %w", err)
	}
	u.Path = "/config"
	q := u.Query()
	q.Set("mac", mac)
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	var cfg config.ConfigResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &cfg, nil
}

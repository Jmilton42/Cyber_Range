package main

import (
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"cyber-range-config/internal/client/common"
	"cyber-range-config/internal/client/windows"
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
	if err := windows.EnsureMarkerDir(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create log directory: %v\n", err)
		os.Exit(1)
	}

	logFile, err := os.OpenFile(windows.GetLogPath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()

	log.SetOutput(io.MultiWriter(os.Stdout, logFile))
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("=== Cyber Range Configuration Client (Windows) Starting ===")

	// Check if already configured
	if windows.IsConfigured() {
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

	// Request configuration with retries
	cfg, err := requestConfigWithRetry(*serverURL, mac, 10, 15*time.Second)
	if err != nil {
		log.Fatalf("Failed to get configuration: %v", err)
	}
	log.Printf("Received config: hostname=%s, dhcp=%v", cfg.Hostname, cfg.Network.DHCP)

	// Apply hostname
	log.Printf("Setting hostname to: %s", cfg.Hostname)
	if err := windows.SetHostname(cfg.Hostname); err != nil {
		log.Fatalf("Failed to set hostname: %v", err)
	}
	log.Println("Hostname set successfully (requires reboot)")

	// Apply network configuration to each local adapter, matched by MAC.
	// For single-NIC VMs cfg.Networks may be empty (legacy servers) - fall
	// back to the old single-shot ConfigureNetwork path in that case.
	log.Println("Configuring network(s)...")
	if len(cfg.Networks) > 0 {
		if err := configureAllAdapters(cfg.Networks); err != nil {
			log.Fatalf("Failed to configure network: %v", err)
		}
	} else {
		if err := windows.ConfigureNetwork(cfg.Network); err != nil {
			log.Fatalf("Failed to configure network: %v", err)
		}
	}
	log.Println("Network configured successfully")

	// Create marker file
	if err := windows.CreateMarker(cfg.Hostname); err != nil {
		log.Fatalf("Failed to create marker file: %v", err)
	}
	log.Println("Marker file created.")

	log.Println("=== Configuration Complete ===")
	log.Println("Initiating system reboot in 5 seconds...")

	// Reboot with 5 second delay to allow logs to flush
	if err := windows.Reboot(5); err != nil {
		log.Fatalf("Failed to initiate reboot: %v", err)
	}
}

// configureAllAdapters walks every non-loopback local adapter with a MAC and
// applies the config whose key matches that adapter's MAC (lowercase, colon
// form). Adapters with no matching config are logged and left alone.
//
// This replaces the previous single-adapter "guess by name" flow, which would
// configure whichever adapter happened to sort first as "Ethernet ..." in
// Windows' enumeration - often not the adapter the config was meant for.
func configureAllAdapters(networks map[string]config.NetworkConfig) error {
	ifaces, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("failed to enumerate local interfaces: %w", err)
	}

	var (
		applied  int
		firstErr error
	)
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if len(iface.HardwareAddr) == 0 {
			continue
		}

		mac := strings.ToLower(iface.HardwareAddr.String())
		netCfg, ok := networks[mac]
		if !ok {
			log.Printf("No config for adapter %q (MAC %s) - skipping", iface.Name, mac)
			continue
		}

		log.Printf("Configuring adapter %q (MAC %s): dhcp=%v address=%s gateway=%s",
			iface.Name, mac, netCfg.DHCP, netCfg.Address, netCfg.Gateway)
		if err := windows.ConfigureAdapter(iface.Name, netCfg); err != nil {
			log.Printf("Failed to configure adapter %q: %v", iface.Name, err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		applied++
	}

	if applied == 0 {
		if firstErr != nil {
			return fmt.Errorf("no adapters configured: %w", firstErr)
		}
		return fmt.Errorf("no adapters matched any MAC in server response (%d configs available)", len(networks))
	}

	log.Printf("Configured %d adapter(s)", applied)
	return firstErr // nil on full success, first error if any adapter failed after at least one succeeded
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

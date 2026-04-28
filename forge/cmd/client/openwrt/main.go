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
	"cyber-range-config/internal/client/openwrt"
	"cyber-range-config/internal/config"
)

const (
	maxStartupDelay  = 30    // Maximum random delay in seconds
	defaultInterface = "eth1" // LAN interface on OpenWrt firewalls
)

func main() {
	// Parse flags
	serverURL := flag.String("server", "", "Configuration server URL (e.g., http://server:8080)")
	interfaceName := flag.String("interface", defaultInterface, "Network interface name for MAC lookup")
	noDelay := flag.Bool("no-delay", false, "Skip random startup delay")
	flag.Parse()

	// Set up logging
	if err := openwrt.EnsureMarkerDir(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create log directory: %v\n", err)
		os.Exit(1)
	}

	logFile, err := os.OpenFile(openwrt.GetLogPath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()

	log.SetOutput(io.MultiWriter(os.Stdout, logFile))
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("=== Cyber Range Configuration Client (OpenWrt) Starting ===")

	// Check if already configured
	if openwrt.IsConfigured() {
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

	// Get MAC address for the specified interface (default eth1)
	mac, err := common.GetMACByName(*interfaceName)
	if err != nil {
		log.Fatalf("Failed to get MAC address for %s: %v", *interfaceName, err)
	}
	log.Printf("Using MAC address from %s: %s", *interfaceName, mac)

	// Request configuration with retries (60 retries × 60s = 60 minutes max)
	cfg, err := requestConfigWithRetry(*serverURL, mac, 60, 60*time.Second)
	if err != nil {
		log.Fatalf("Failed to get configuration: %v", err)
	}
	log.Printf("Received config for instance: %s", cfg.Hostname)

	// Apply network configuration via UCI
	log.Println("Configuring network via UCI...")

	// Check if we have multiple networks
	if len(cfg.Networks) > 0 {
		log.Printf("Found %d network interface(s) to configure", len(cfg.Networks))
		for cloudInitName, netCfg := range cfg.Networks {
			// Use custom UCI name if specified, otherwise use mapping
			uciName := netCfg.UCIName
			if uciName == "" {
				uciName = openwrt.MapInterfaceName(cloudInitName)
			}
			log.Printf("Configuring %s -> UCI %s: dhcp=%v, address=%s", cloudInitName, uciName, netCfg.DHCP, netCfg.Address)
			if err := openwrt.ConfigureInterface(uciName, cloudInitName, netCfg); err != nil {
				log.Printf("Warning: Failed to configure %s: %v", uciName, err)
				// Continue with other interfaces
			}

			// Add interface to firewall zone (all non-wan interfaces go to lan zone)
			firewallZone := "lan"
			if uciName == "wan" {
				firewallZone = "wan"
			}
			log.Printf("Adding %s to firewall zone: %s", uciName, firewallZone)
			if err := openwrt.AddInterfaceToFirewallZone(uciName, firewallZone); err != nil {
				log.Printf("Warning: Failed to add %s to firewall zone: %v", uciName, err)
			}

			// Add interface to DNS listen interfaces (except wan)
			if uciName != "wan" {
				log.Printf("Adding %s to DNS listen interfaces", uciName)
				if err := openwrt.AddInterfaceToDNS(uciName); err != nil {
					log.Printf("Warning: Failed to add %s to DNS: %v", uciName, err)
				}
			}
		}
		// Commit all changes at once
		if err := openwrt.CommitNetworkChanges(); err != nil {
			log.Fatalf("Failed to commit network changes: %v", err)
		}
		if err := openwrt.CommitFirewallChanges(); err != nil {
			log.Printf("Warning: Failed to commit firewall changes: %v", err)
		}
		if err := openwrt.CommitDHCPChanges(); err != nil {
			log.Printf("Warning: Failed to commit DHCP changes: %v", err)
		}
	} else {
		// Fallback to single network (backwards compatibility)
		log.Printf("Using single network config: dhcp=%v, address=%s", cfg.Network.DHCP, cfg.Network.Address)
		if err := openwrt.ConfigureInterface("lan", "eth1", cfg.Network); err != nil {
			log.Fatalf("Failed to configure network: %v", err)
		}

		// Add lan to firewall zone (should already be there, but ensure it)
		log.Println("Adding lan to firewall zone: lan")
		if err := openwrt.AddInterfaceToFirewallZone("lan", "lan"); err != nil {
			log.Printf("Warning: Failed to add lan to firewall zone: %v", err)
		}

		// Add lan to DNS listen interfaces
		log.Println("Adding lan to DNS listen interfaces")
		if err := openwrt.AddInterfaceToDNS("lan"); err != nil {
			log.Printf("Warning: Failed to add lan to DNS: %v", err)
		}

		if err := openwrt.CommitNetworkChanges(); err != nil {
			log.Fatalf("Failed to commit network changes: %v", err)
		}
		if err := openwrt.CommitFirewallChanges(); err != nil {
			log.Printf("Warning: Failed to commit firewall changes: %v", err)
		}
		if err := openwrt.CommitDHCPChanges(); err != nil {
			log.Printf("Warning: Failed to commit DHCP changes: %v", err)
		}
	}
	log.Println("Network configured successfully")

	// Create marker file
	if err := openwrt.CreateMarker(cfg.Hostname); err != nil {
		log.Fatalf("Failed to create marker file: %v", err)
	}
	log.Println("Marker file created.")

	log.Println("=== Configuration Complete ===")
	log.Println("Restarting services...")

	// Restart network to apply changes
	if err := openwrt.RestartNetwork(); err != nil {
		log.Printf("Warning: Failed to restart network: %v", err)
		log.Println("You may need to restart the network manually or reboot.")
	} else {
		log.Println("Network service restarted successfully")
	}

	// Restart firewall to apply changes
	if err := openwrt.RestartFirewall(); err != nil {
		log.Printf("Warning: Failed to restart firewall: %v", err)
	} else {
		log.Println("Firewall service restarted successfully")
	}

	// Restart dnsmasq to apply DNS changes
	if err := openwrt.RestartDNSMasq(); err != nil {
		log.Printf("Warning: Failed to restart dnsmasq: %v", err)
	} else {
		log.Println("DNSMasq service restarted successfully")
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

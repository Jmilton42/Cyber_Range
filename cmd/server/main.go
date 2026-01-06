package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cyber-range-config/internal/config"
	"cyber-range-config/internal/server"

	"gopkg.in/yaml.v3"
)

const (
	defaultIdleTimeout = 15 * time.Minute
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	instancesFile := flag.String("instances", "", "Path to instances JSON file (overrides config)")
	listenAddr := flag.String("listen", "", "Listen address (overrides config)")
	idleTimeout := flag.Duration("idle-timeout", defaultIdleTimeout, "Shutdown after this duration of inactivity (0 to disable)")
	flag.Parse()

	// Load configuration
	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Printf("Warning: Could not load config file: %v", err)
		cfg = &config.ServerConfig{
			Listen:        ":8080",
			InstancesFile: "instances.json",
		}
	}

	// Override with command line flags
	if *instancesFile != "" {
		cfg.InstancesFile = *instancesFile
	}
	if *listenAddr != "" {
		cfg.Listen = *listenAddr
	}

	// Validate
	if cfg.InstancesFile == "" {
		log.Fatal("No instances file specified")
	}

	// Create server
	srv, err := server.NewServer(cfg.InstancesFile)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Set up routes
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:    cfg.Listen,
		Handler: mux,
	}

	// Channel to signal shutdown
	shutdown := make(chan struct{})

	// Start idle timeout monitor
	if *idleTimeout > 0 {
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()

			log.Printf("Idle timeout enabled: %v", *idleTimeout)

			for {
				select {
				case <-ticker.C:
					lastActivity := srv.GetLastActivity()
					idleTime := time.Since(lastActivity)

					if idleTime >= *idleTimeout {
						log.Printf("Server idle for %v, initiating shutdown...", idleTime.Round(time.Second))
						close(shutdown)
						return
					}

					remaining := *idleTimeout - idleTime
					log.Printf("Idle for %v, shutdown in %v if no activity", idleTime.Round(time.Second), remaining.Round(time.Second))

				case <-shutdown:
					return
				}
			}
		}()
	}

	// Handle OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case sig := <-sigChan:
			log.Printf("Received signal %v, shutting down...", sig)
			close(shutdown)
		case <-shutdown:
		}
	}()

	// Start server in goroutine
	go func() {
		log.Printf("Starting server on %s", cfg.Listen)
		log.Printf("Instances file: %s", cfg.InstancesFile)
		log.Printf("Endpoints: GET /config?mac=XX:XX:XX:XX:XX:XX, POST /reload, GET /status")

		if *idleTimeout > 0 {
			log.Printf("Will shutdown after %v of inactivity", *idleTimeout)
		}

		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-shutdown

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Println("Shutting down server...")
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("Server stopped")
}

func loadConfig(path string) (*config.ServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg config.ServerConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Set defaults
	if cfg.Listen == "" {
		cfg.Listen = ":8080"
	}
	if cfg.InstancesFile == "" {
		cfg.InstancesFile = "instances.json"
	}

	return &cfg, nil
}

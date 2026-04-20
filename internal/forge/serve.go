package forge

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cyber-range-config/internal/server"
)

// ServeOptions holds runtime options for the in-process HTTP config server.
type ServeOptions struct {
	Listen        string        // e.g. "10.0.14.6:8080"
	InstancesFile string        // path to instances.json
	IdleTimeout   time.Duration // 0 disables idle shutdown
}

// RunServer starts the HTTP config server in the current process and blocks
// until it shuts down (via idle timeout, SIGINT, or SIGTERM). This is the code
// path invoked by `forge serve`, and is what `forge apply` re-execs into as a
// detached child so the HTTP server outlives the apply invocation.
func RunServer(opts ServeOptions) error {
	if opts.InstancesFile == "" {
		return fmt.Errorf("no instances file specified")
	}
	if opts.Listen == "" {
		opts.Listen = ":8080"
	}

	srv, err := server.NewServer(opts.InstancesFile)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	httpServer := &http.Server{
		Addr:    opts.Listen,
		Handler: mux,
	}

	shutdown := make(chan struct{})
	var shutdownOnce = make(chan struct{}, 1) // used to guard closes

	closeShutdown := func() {
		select {
		case shutdownOnce <- struct{}{}:
			close(shutdown)
		default:
		}
	}

	// Idle timeout monitor.
	if opts.IdleTimeout > 0 {
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()

			log.Printf("Idle timeout enabled: %v", opts.IdleTimeout)

			for {
				select {
				case <-ticker.C:
					lastActivity := srv.GetLastActivity()
					idleTime := time.Since(lastActivity)

					if idleTime >= opts.IdleTimeout {
						log.Printf("Server idle for %v, initiating shutdown...", idleTime.Round(time.Second))
						closeShutdown()
						return
					}

					remaining := opts.IdleTimeout - idleTime
					log.Printf("Idle for %v, shutdown in %v if no activity", idleTime.Round(time.Second), remaining.Round(time.Second))

				case <-shutdown:
					return
				}
			}
		}()
	}

	// Signal handling.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case sig := <-sigChan:
			log.Printf("Received signal %v, shutting down...", sig)
			closeShutdown()
		case <-shutdown:
		}
	}()

	serveErr := make(chan error, 1)
	go func() {
		log.Printf("Starting server on %s", opts.Listen)
		log.Printf("Instances file: %s", opts.InstancesFile)
		log.Printf("Endpoints: GET /config?mac=XX:XX:XX:XX:XX:XX, POST /reload, GET /status")
		if opts.IdleTimeout > 0 {
			log.Printf("Will shutdown after %v of inactivity", opts.IdleTimeout)
		}

		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			serveErr <- err
			closeShutdown()
			return
		}
		serveErr <- nil
	}()

	<-shutdown

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Println("Shutting down server...")
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	// Surface any fatal ListenAndServe error.
	select {
	case err := <-serveErr:
		if err != nil {
			return fmt.Errorf("server error: %w", err)
		}
	default:
	}

	log.Println("Server stopped")
	return nil
}

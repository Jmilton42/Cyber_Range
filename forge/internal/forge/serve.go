package forge

import (
	"context"
	"encoding/json"
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
	LogFormat     string        // "text" (default) or "json"
}

// logEvent emits a single log line in either text or JSON form depending on
// opts.LogFormat. Keeping this in one place lets the rest of serve.go stay
// readable while still producing structured log output when asked.
func logEvent(format, event, msg string, fields map[string]interface{}) {
	if format == "json" {
		entry := map[string]interface{}{
			"ts":    time.Now().UTC().Format(time.RFC3339Nano),
			"event": event,
			"msg":   msg,
		}
		for k, v := range fields {
			entry[k] = v
		}
		if data, err := json.Marshal(entry); err == nil {
			fmt.Println(string(data))
			return
		}
	}
	if len(fields) == 0 {
		log.Printf("%s", msg)
		return
	}
	log.Printf("%s %v", msg, fields)
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
	if opts.LogFormat == "" {
		opts.LogFormat = "text"
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

			logEvent(opts.LogFormat, "idle_timer_start", "Idle timeout enabled",
				map[string]interface{}{"timeout": opts.IdleTimeout.String()})

			for {
				select {
				case <-ticker.C:
					lastActivity := srv.GetLastActivity()
					idleTime := time.Since(lastActivity)

					if idleTime >= opts.IdleTimeout {
						logEvent(opts.LogFormat, "idle_shutdown", "Server idle, initiating shutdown",
							map[string]interface{}{"idle": idleTime.Round(time.Second).String()})
						closeShutdown()
						return
					}

					remaining := opts.IdleTimeout - idleTime
					logEvent(opts.LogFormat, "idle_tick", "Idle, shutdown pending if no activity",
						map[string]interface{}{
							"idle":      idleTime.Round(time.Second).String(),
							"remaining": remaining.Round(time.Second).String(),
						})

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
			logEvent(opts.LogFormat, "signal", "Received signal, shutting down",
				map[string]interface{}{"signal": sig.String()})
			closeShutdown()
		case <-shutdown:
		}
	}()

	serveErr := make(chan error, 1)
	go func() {
		logEvent(opts.LogFormat, "server_start", "Starting server",
			map[string]interface{}{
				"listen":         opts.Listen,
				"instances_file": opts.InstancesFile,
				"idle_timeout":   opts.IdleTimeout.String(),
			})

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

	logEvent(opts.LogFormat, "server_shutdown_begin", "Shutting down server", nil)
	if err := httpServer.Shutdown(ctx); err != nil {
		logEvent(opts.LogFormat, "server_shutdown_error", "Error during shutdown",
			map[string]interface{}{"err": err.Error()})
	}

	// Surface any fatal ListenAndServe error.
	select {
	case err := <-serveErr:
		if err != nil {
			return fmt.Errorf("server error: %w", err)
		}
	default:
	}

	logEvent(opts.LogFormat, "server_stopped", "Server stopped", nil)
	return nil
}

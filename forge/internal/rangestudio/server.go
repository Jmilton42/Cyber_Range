package rangestudio

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// ServeConfig configures the Range Studio HTTP server.
type ServeConfig struct {
	Listen       string
	Mock         bool
	SubnetsPath  string
	InspireRoots []string
	FrontendDir  string // path to the Website/ directory with static files
}

// Run starts the Range Studio HTTP server.
func Run(cfg ServeConfig) error {
	// Build the appropriate backend
	var b Backend
	if cfg.Mock {
		testdataDir := findTestdataDir()
		projectRoots := findProjectRoots()
		b = NewMockBackend(testdataDir, projectRoots)
		log.Println("[RANGE STUDIO] Starting in MOCK mode")
		log.Printf("[RANGE STUDIO] Testdata dir: %s", testdataDir)
		log.Printf("[RANGE STUDIO] Project roots: %v", projectRoots)
	} else {
		b = NewLiveBackend(cfg.SubnetsPath, cfg.InspireRoots)
		log.Println("[RANGE STUDIO] Starting in LIVE mode")
	}

	mux := http.NewServeMux()

	// Register API routes
	RegisterAPI(mux, b, cfg)

	// Serve static frontend from the Website/ directory
	frontendDir := cfg.FrontendDir
	if frontendDir == "" {
		frontendDir = findFrontendDir()
	}
	if frontendDir != "" {
		log.Printf("[RANGE STUDIO] Serving frontend from: %s", frontendDir)
		fs := http.FileServer(http.Dir(frontendDir))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
			fs.ServeHTTP(w, r)
		})
	} else {
		log.Println("[WARN] No frontend directory found; API-only mode")
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprintln(w, "Range Studio API is running. No frontend directory configured.")
			fmt.Fprintln(w, "Try: /api/info, /api/projects, /api/lxd/images")
		})
	}

	log.Printf("[RANGE STUDIO] Listening on %s", cfg.Listen)
	return http.ListenAndServe(cfg.Listen, mux)
}

// findTestdataDir locates the testdata directory relative to the
// current working directory or the module root.
func findTestdataDir() string {
	// Try relative to cwd first
	candidates := []string{
		filepath.Join("internal", "rangestudio", "testdata"),
		filepath.Join("forge", "internal", "rangestudio", "testdata"),
	}

	cwd, _ := os.Getwd()
	for _, c := range candidates {
		abs := filepath.Join(cwd, c)
		if info, err := os.Stat(abs); err == nil && info.IsDir() {
			return abs
		}
	}

	// Walk up from cwd looking for forge/internal/rangestudio/testdata
	dir := cwd
	for {
		candidate := filepath.Join(dir, "forge", "internal", "rangestudio", "testdata")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return filepath.Join(cwd, "testdata")
}

// findProjectRoots locates the range/ directory for mock mode.
func findProjectRoots() []string {
	cwd, _ := os.Getwd()

	candidates := []string{
		filepath.Join(cwd, "range"),
		filepath.Join(cwd, "..", "range"),
	}

	// Walk up looking for range/
	dir := cwd
	for {
		candidate := filepath.Join(dir, "range")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return []string{candidate}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Check the simple candidates
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return []string{c}
		}
	}

	return nil
}

// findFrontendDir locates the Website/ directory for serving static files.
func findFrontendDir() string {
	cwd, _ := os.Getwd()

	// Walk up from cwd looking for Website/
	dir := cwd
	for {
		candidate := filepath.Join(dir, "Website")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return ""
}

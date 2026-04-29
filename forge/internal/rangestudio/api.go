package rangestudio

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
)

// RegisterAPI wires up all /api/* routes on the given mux.
func RegisterAPI(mux *http.ServeMux, b Backend, cfg ServeConfig) {
	mux.HandleFunc("/api/info", handleInfo(b))
	mux.HandleFunc("/api/projects/create", handleCreateProject(b, cfg))
	mux.HandleFunc("/api/projects", handleProjects(b))
	mux.HandleFunc("/api/projects/", handleProjectDetail(b))
	mux.HandleFunc("/api/lxd/images", handleImages(b))
	mux.HandleFunc("/api/lxd/profiles", handleProfiles(b))
	mux.HandleFunc("/api/lxd/cluster", handleCluster(b))
	mux.HandleFunc("/api/templates", handleTemplates(b))
}

func handleInfo(b Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, ServiceInfo{
			Mock:    b.IsMock(),
			Version: "1.0.0",
		})
	}
}

func handleProjects(b Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		projects, err := b.ListProjects()
		if err != nil {
			log.Printf("[ERROR] ListProjects: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, projects)
	}
}

func handleProjectDetail(b Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// Extract project name from /api/projects/{name}
		name := strings.TrimPrefix(r.URL.Path, "/api/projects/")
		if name == "" {
			http.Error(w, "project name required", http.StatusBadRequest)
			return
		}
		detail, err := b.GetProject(name)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				http.Error(w, err.Error(), http.StatusNotFound)
			} else {
				log.Printf("[ERROR] GetProject(%s): %v", name, err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		writeJSON(w, detail)
	}
}

func handleImages(b Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		images, err := b.ListImages()
		if err != nil {
			log.Printf("[ERROR] ListImages: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, images)
	}
}

func handleProfiles(b Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		profiles, err := b.ListProfiles()
		if err != nil {
			log.Printf("[ERROR] ListProfiles: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, profiles)
	}
}

func handleCluster(b Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		members, err := b.ListClusterMembers()
		if err != nil {
			log.Printf("[ERROR] ListClusterMembers: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, members)
	}
}

func handleTemplates(b Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		templates, err := b.ListTemplates()
		if err != nil {
			log.Printf("[ERROR] ListTemplates: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, templates)
	}
}

// handleCreateProject handles POST /api/projects/create — the wizard endpoint.
func handleCreateProject(b Backend, cfg ServeConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// CORS preflight
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var req CreateProjectRequest
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		log.Printf("[WIZARD] Creating project: name=%s template=%s group=%s", req.Name, req.Template, req.Group)

		testdataDir := findTestdataDir()
		result, err := CreateProject(req, b.IsMock(), testdataDir, cfg.InspireRoots)
		if err != nil {
			log.Printf("[ERROR] CreateProject: %v", err)
			writeJSON(w, map[string]interface{}{
				"error": err.Error(),
			})
			return
		}

		log.Printf("[WIZARD] Project created: %s at %s (%d files)", result.Name, result.OutputDir, len(result.Files))
		writeJSON(w, result)
	}
}

// writeJSON marshals v and writes it as a JSON response.
func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("[ERROR] encode JSON: %v", err)
	}
}

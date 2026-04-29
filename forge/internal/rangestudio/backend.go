package rangestudio

import (
	"cyber-range-config/internal/forge"
)

// Backend is the interface that abstracts mock vs live data sources.
// The API handlers call through this interface so the frontend receives
// identical JSON shapes regardless of which backend is active.
type Backend interface {
	// ListProjects returns every known project allocation with resolved paths.
	ListProjects() ([]Project, error)
	// GetProject returns detail for a single project by name.
	GetProject(name string) (*ProjectDetail, error)

	// LXD resource catalogs.
	ListImages() ([]LXDImage, error)
	ListProfiles() ([]LXDProfile, error)
	ListClusterMembers() ([]ClusterMember, error)

	// Templates reuses forge.LoadTemplates but routed through the interface.
	ListTemplates() ([]forge.Template, error)

	// IsMock reports whether this backend is using fixture data.
	IsMock() bool
}

// ---------- DTO structs ----------

// Project is the list-level representation of an allocated range project.
type Project struct {
	Name        string `json:"name"`
	SubnetOctet int    `json:"subnet_octet"`
	Subnet      string `json:"subnet"`
	Gateway     string `json:"gateway"`
	AllocatedAt string `json:"allocated_at"`
	WorkDir     string `json:"work_dir,omitempty"`
	HasMainTF   bool   `json:"has_main_tf"`
	Status      string `json:"status"` // "active", "unknown", "missing_dir"
}

// ProjectDetail extends Project with file-level information.
type ProjectDetail struct {
	Project
	Files    []FileEntry `json:"files,omitempty"`
	HasState bool        `json:"has_state"`
}

// FileEntry describes one file inside a project directory.
type FileEntry struct {
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size"`
}

// LXDImage represents an image entry for the UI.
type LXDImage struct {
	Aliases      []string `json:"aliases"`
	Fingerprint  string   `json:"fingerprint"`
	Description  string   `json:"description"`
	Size         int64    `json:"size"`
	Type         string   `json:"type"` // "container" | "virtual-machine"
	UploadedAt   string   `json:"uploaded_at"`
	Architecture string   `json:"architecture"`
}

// LXDProfile represents a profile entry for the UI.
type LXDProfile struct {
	Name        string                       `json:"name"`
	Description string                       `json:"description"`
	Devices     map[string]map[string]string `json:"devices"`
	Config      map[string]string            `json:"config"`
}

// ClusterMember represents a node in the LXD cluster.
type ClusterMember struct {
	ServerName   string   `json:"server_name"`
	URL          string   `json:"url"`
	Status       string   `json:"status"`
	Roles        []string `json:"roles"`
	Architecture string   `json:"architecture"`
	GPU          bool     `json:"gpu"`
}

// ServiceInfo is returned by /api/info.
type ServiceInfo struct {
	Mock    bool   `json:"mock"`
	Version string `json:"version"`
}

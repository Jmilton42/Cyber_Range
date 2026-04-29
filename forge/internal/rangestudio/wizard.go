package rangestudio

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"cyber-range-config/internal/forge"
)

// CreateProjectRequest is the JSON body sent by the wizard frontend.
type CreateProjectRequest struct {
	Name      string `json:"name"`
	Group     string `json:"group"`     // Education, Research, Outreach, CIG
	SubGroup  string `json:"sub_group"` // DCIG, OCIG, COMP, CTF (when Group=CIG)
	Template  string `json:"template"`  // template id (e.g. "CSC-3410-single", "blank")
	TeamCount int    `json:"team_count"`
	NeedsGPU  bool   `json:"needs_gpu"` // Research only
}

// CreateProjectResponse is returned after a successful project scaffold.
type CreateProjectResponse struct {
	Name      string `json:"name"`
	OutputDir string `json:"output_dir"`
	Template  string `json:"template"`
	Files     []string `json:"files"` // files created
	Message   string `json:"message"`
}

// projectNameRe mirrors forge's validation.
var projectNameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{1,40}$`)

// CreateProject scaffolds a new project using forge.RunNew logic.
// In mock mode output goes to testdata/generated/<name>/.
// In live mode output goes to the InSPIRE path based on group.
func CreateProject(req CreateProjectRequest, mock bool, testdataDir string, inspireRoots []string) (*CreateProjectResponse, error) {
	// Validate name
	if !projectNameRe.MatchString(req.Name) {
		return nil, fmt.Errorf("invalid project name %q: must start with alphanumeric, 2-41 chars, only letters/digits/dash/underscore", req.Name)
	}

	if req.Template == "" {
		req.Template = "blank"
	}

	// Determine output directory
	var targetDir string
	if mock {
		genDir := filepath.Join(testdataDir, "generated")
		targetDir = filepath.Join(genDir, req.Name)
	} else {
		targetDir = resolveInspirePath(req, inspireRoots)
	}

	// Check if directory already exists
	if _, err := os.Stat(targetDir); err == nil {
		return nil, fmt.Errorf("directory %s already exists — choose a different name or delete it first", targetDir)
	}

	// Use forge.RunNew to scaffold
	opts := forge.NewOptions{
		Template:    req.Template,
		Name:        req.Name,
		TargetDir:   targetDir,
		Reserve:     false, // Don't allocate subnet in mock mode
		AutoYes:     true,
		Interactive: false,
	}

	if err := forge.RunNew(opts); err != nil {
		return nil, fmt.Errorf("scaffold failed: %w", err)
	}

	// If team_count was specified and it's not the blank template,
	// try to write a terraform.tfvars with team_count override
	if req.TeamCount > 0 {
		tfvarsPath := filepath.Join(targetDir, "terraform.tfvars")
		tfvarsContent := fmt.Sprintf("team_count = %d\n", req.TeamCount)
		if req.NeedsGPU {
			tfvarsContent += "# GPU requested — place GPU VMs on micro-07-gpu\n"
		}
		os.WriteFile(tfvarsPath, []byte(tfvarsContent), 0644)
	}

	// Write a studio metadata file so we know this project's group
	meta := map[string]interface{}{
		"name":       req.Name,
		"group":      req.Group,
		"sub_group":  req.SubGroup,
		"template":   req.Template,
		"team_count": req.TeamCount,
		"needs_gpu":  req.NeedsGPU,
		"created_at": time.Now().Format(time.RFC3339),
		"created_by": "range-studio",
	}
	metaJSON, _ := json.MarshalIndent(meta, "", "  ")
	os.WriteFile(filepath.Join(targetDir, ".rangestudio.json"), metaJSON, 0644)

	// List what was created
	var files []string
	filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(targetDir, path)
		if rel != "." {
			files = append(files, rel)
		}
		return nil
	})

	return &CreateProjectResponse{
		Name:      req.Name,
		OutputDir: targetDir,
		Template:  req.Template,
		Files:     files,
		Message:   fmt.Sprintf("Project %q created successfully from template %q", req.Name, req.Template),
	}, nil
}

// resolveInspirePath picks the right InSPIRE subdirectory based on group.
func resolveInspirePath(req CreateProjectRequest, roots []string) string {
	if len(roots) == 0 {
		return req.Name // fallback: current dir
	}
	base := roots[0]
	switch req.Group {
	case "Education":
		return filepath.Join(base, "Classes", req.Name)
	case "CIG":
		switch req.SubGroup {
		case "DCIG":
			return filepath.Join(base, "CIG", "DCIG", req.Name)
		case "OCIG":
			return filepath.Join(base, "CIG", "OCIG", req.Name)
		case "COMP":
			return filepath.Join(base, "CIG", "COMP", req.Name)
		case "CTF":
			return filepath.Join(base, "CIG", "CTF", req.Name)
		default:
			return filepath.Join(base, "CIG", req.Name)
		}
	case "Research":
		return filepath.Join(base, "Research", req.Name)
	case "Outreach":
		return filepath.Join(base, "Outreach", req.Name)
	default:
		return filepath.Join(base, req.Name)
	}
}

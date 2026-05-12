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
// All optional fields default to sensible values when omitted so the
// frontend can send only what the user actually filled in.
type CreateProjectRequest struct {
	Name        string          `json:"name"`
	Group       string          `json:"group"`       // Education, Research, Outreach, CIG
	SubGroup    string          `json:"sub_group"`   // DCIG, OCIG, COMP, CTF
	Template    string          `json:"template"`    // template id; "custom" for HCL generator
	TeamCount   int             `json:"team_count"`
	Description string          `json:"description"` // user-provided project description
	NeedsGPU    bool            `json:"needs_gpu"`   // Research only: project-wide GPU requirement
	Topology    *CustomTopology `json:"topology,omitempty"`
}

// CreateProjectResponse is returned after a successful project scaffold.
type CreateProjectResponse struct {
	Name      string   `json:"name"`
	OutputDir string   `json:"output_dir"`
	Template  string   `json:"template"`
	Files     []string `json:"files"`
	Message   string   `json:"message"`
}

// projectNameRe mirrors forge's validation.
var projectNameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{1,40}$`)

// CreateProject scaffolds a new project. In mock mode output is written
// under <testdata>/generated/<name>/. In live mode it goes to the
// InSPIRE path determined by group / sub_group.
func CreateProject(req CreateProjectRequest, mock bool, testdataDir string, inspireRoots []string) (*CreateProjectResponse, error) {
	if !projectNameRe.MatchString(req.Name) {
		return nil, fmt.Errorf("invalid project name %q: must start with alphanumeric, 2-41 chars, only letters/digits/dash/underscore", req.Name)
	}

	if req.Template == "" {
		req.Template = "blank"
	}

	var targetDir string
	if mock {
		genDir := filepath.Join(testdataDir, "generated")
		targetDir = filepath.Join(genDir, req.Name)
	} else {
		targetDir = resolveInspirePath(req, inspireRoots)
	}

	if _, err := os.Stat(targetDir); err == nil {
		return nil, fmt.Errorf("directory %s already exists - choose a different name or delete it first", targetDir)
	}

	if req.Template == "custom" {
		if req.Topology == nil {
			return nil, fmt.Errorf("custom template requires a topology")
		}
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return nil, fmt.Errorf("create target dir: %w", err)
		}

		// Apply project-wide GPU flag to every VM if the user
		// requested it from step 1; per-VM overrides still win.
		topo := *req.Topology
		if req.NeedsGPU {
			for i := range topo.VMs {
				if !topo.VMs[i].NeedsGPU && topo.VMs[i].Target == "" {
					topo.VMs[i].NeedsGPU = true
				}
			}
		}

		hcl, err := GenerateMainTF(req.Name, req.TeamCount, topo)
		if err != nil {
			return nil, fmt.Errorf("generate custom HCL: %w", err)
		}

		if err := os.WriteFile(filepath.Join(targetDir, "main.tf"), []byte(hcl), 0644); err != nil {
			return nil, fmt.Errorf("write main.tf: %w", err)
		}
	} else {
		opts := forge.NewOptions{
			Template:    req.Template,
			Name:        req.Name,
			TargetDir:   targetDir,
			Reserve:     false,
			AutoYes:     true,
			Interactive: false,
		}
		if err := forge.RunNew(opts); err != nil {
			return nil, fmt.Errorf("scaffold failed: %w", err)
		}
	}

	if req.TeamCount > 0 {
		tfvarsPath := filepath.Join(targetDir, "terraform.tfvars")
		tfvars := fmt.Sprintf("team_count = %d\n", req.TeamCount)
		if req.NeedsGPU {
			tfvars += "# GPU requested for this project. Custom templates route\n"
			tfvars += "# all VMs to micro-07-gpu via target = \"@micro-07-gpu\".\n"
		}
		_ = os.WriteFile(tfvarsPath, []byte(tfvars), 0644)
	}

	meta := map[string]interface{}{
		"name":        req.Name,
		"group":       req.Group,
		"sub_group":   req.SubGroup,
		"template":    req.Template,
		"team_count":  req.TeamCount,
		"description": req.Description,
		"needs_gpu":   req.NeedsGPU,
		"created_at":  time.Now().Format(time.RFC3339),
		"created_by":  "range-studio",
	}
	metaJSON, _ := json.MarshalIndent(meta, "", "  ")
	_ = os.WriteFile(filepath.Join(targetDir, ".rangestudio.json"), metaJSON, 0644)

	var files []string
	_ = filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
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
		Message:   fmt.Sprintf("Project %q created from template %q", req.Name, req.Template),
	}, nil
}

// resolveInspirePath picks the right InSPIRE subdirectory based on
// group, mirroring the convention in docs/README.md.
func resolveInspirePath(req CreateProjectRequest, roots []string) string {
	if len(roots) == 0 {
		return req.Name
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

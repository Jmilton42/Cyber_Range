package rangestudio

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"cyber-range-config/internal/forge"
)

// MockBackend implements Backend using committed JSON fixture files.
// It is used for local development where no LXD socket or InSPIRE
// filesystem is available.
type MockBackend struct {
	// testdataDir is the absolute path to the testdata directory
	// containing fixture JSON files.
	testdataDir string
	// projectRoots are directories scanned when resolving project
	// work_dir (typically the repo's range/ directory in mock mode).
	projectRoots []string
}

// NewMockBackend creates a MockBackend. testdataDir should point at
// the directory containing subnets.json, images.json, etc. projectRoots
// lists directories whose immediate children are treated as project
// directories (e.g. the repo's range/ folder).
func NewMockBackend(testdataDir string, projectRoots []string) *MockBackend {
	return &MockBackend{
		testdataDir:  testdataDir,
		projectRoots: projectRoots,
	}
}

func (m *MockBackend) IsMock() bool { return true }

// ListProjects reads the fixture subnets.json and resolves work dirs
// by scanning projectRoots for matching directory names.
func (m *MockBackend) ListProjects() ([]Project, error) {
	data, err := os.ReadFile(filepath.Join(m.testdataDir, "subnets.json"))
	if err != nil {
		return nil, fmt.Errorf("read mock subnets: %w", err)
	}

	var sd forge.SubnetsData
	if err := json.Unmarshal(data, &sd); err != nil {
		return nil, fmt.Errorf("parse mock subnets: %w", err)
	}

	projects := make([]Project, 0, len(sd.Allocations))
	for _, alloc := range sd.Allocations {
		p := Project{
			Name:        alloc.Project,
			SubnetOctet: alloc.SubnetOctet,
			Subnet:      forge.FormatSubnet(alloc.SubnetOctet),
			Gateway:     forge.FormatGateway(alloc.SubnetOctet),
			AllocatedAt: alloc.AllocatedAt,
			Status:      "unknown",
		}
		// Try to resolve a work directory
		if dir := m.resolveProjectDir(alloc.Project); dir != "" {
			p.WorkDir = dir
			p.Status = "active"
			if _, err := os.Stat(filepath.Join(dir, "main.tf")); err == nil {
				p.HasMainTF = true
			}
		} else {
			p.Status = "missing_dir"
		}
		projects = append(projects, p)
	}
	return projects, nil
}

// GetProject returns detail for a single project.
func (m *MockBackend) GetProject(name string) (*ProjectDetail, error) {
	projects, err := m.ListProjects()
	if err != nil {
		return nil, err
	}
	for _, p := range projects {
		if p.Name == name {
			detail := &ProjectDetail{Project: p}
			if p.WorkDir != "" {
				detail.Files = listDir(p.WorkDir)
				if _, err := os.Stat(filepath.Join(p.WorkDir, "terraform.tfstate")); err == nil {
					detail.HasState = true
				}
				// Also check .terraform directory
				if _, err := os.Stat(filepath.Join(p.WorkDir, ".terraform")); err == nil {
					detail.HasState = true
				}
			}
			return detail, nil
		}
	}
	return nil, fmt.Errorf("project %q not found", name)
}

func (m *MockBackend) ListImages() ([]LXDImage, error) {
	return loadFixture[LXDImage](filepath.Join(m.testdataDir, "images.json"))
}

func (m *MockBackend) ListProfiles() ([]LXDProfile, error) {
	return loadFixture[LXDProfile](filepath.Join(m.testdataDir, "profiles.json"))
}

func (m *MockBackend) ListClusterMembers() ([]ClusterMember, error) {
	return loadFixture[ClusterMember](filepath.Join(m.testdataDir, "cluster_members.json"))
}

func (m *MockBackend) ListTemplates() ([]forge.Template, error) {
	return forge.LoadTemplates()
}

// mockDirMap maps fixture project names to their corresponding
// directory names under range/. This is necessary because the repo's
// range/ directory names don't follow a consistent convention relative
// to the project_name inside each main.tf.
var mockDirMap = map[string]string{
	"CSC-3410":     "CSC-3410-single",
	"CIG-Lab":      "win-lin-Muli",
	"CPTC-Mock":    "CPTC - Muli",
	"DCIGS":        "LIN-LAB-Muli",
	// Research-GPU has no matching dir in range/; left unmapped.
}

// resolveProjectDir scans projectRoots for a directory whose name
// matches the project name. Uses mockDirMap first for known mappings,
// then falls back to prefix matching.
func (m *MockBackend) resolveProjectDir(projectName string) string {
	for _, root := range m.projectRoots {
		// Try the explicit mapping first
		if mapped, ok := mockDirMap[projectName]; ok {
			candidate := filepath.Join(root, mapped)
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				return candidate
			}
		}

		// Fall back to scanning for prefix match
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			if matchesProject(e.Name(), projectName) {
				return filepath.Join(root, e.Name())
			}
		}
	}
	return ""
}

// matchesProject does a flexible match between a directory name and
// project name. In mock mode, range/ dirs have names like
// "CSC-3410-single" for a project named "CSC-3410".
func matchesProject(dirName, projectName string) bool {
	// Exact match
	if dirName == projectName {
		return true
	}
	// Dir starts with project name (e.g. "CSC-3410-single" matches "CSC-3410")
	if len(dirName) > len(projectName) && dirName[:len(projectName)] == projectName {
		return true
	}
	return false
}

// listDir returns a flat listing of the given directory.
func listDir(dir string) []FileEntry {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var files []FileEntry
	for _, e := range entries {
		fe := FileEntry{
			Name:  e.Name(),
			IsDir: e.IsDir(),
		}
		if info, err := e.Info(); err == nil {
			fe.Size = info.Size()
		}
		files = append(files, fe)
	}
	return files
}

// loadFixture reads a JSON fixture file into a slice of T.
func loadFixture[T any](path string) ([]T, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read fixture %s: %w", path, err)
	}
	var items []T
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("parse fixture %s: %w", path, err)
	}
	return items, nil
}

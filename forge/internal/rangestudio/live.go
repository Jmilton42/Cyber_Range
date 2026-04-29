package rangestudio

import (
	"fmt"

	"cyber-range-config/internal/forge"
)

// LiveBackend implements Backend using the real LXD unix socket and
// filesystem paths on the deployed range host.
//
// Phase 2: currently stubbed — returns errors directing the user to
// use --mock mode until the live implementation is complete.
type LiveBackend struct {
	subnetsPath  string
	inspireRoots []string
}

// NewLiveBackend creates a LiveBackend.
func NewLiveBackend(subnetsPath string, inspireRoots []string) *LiveBackend {
	return &LiveBackend{
		subnetsPath:  subnetsPath,
		inspireRoots: inspireRoots,
	}
}

func (l *LiveBackend) IsMock() bool { return false }

func (l *LiveBackend) ListProjects() ([]Project, error) {
	allocs, err := forge.GetAllAllocations()
	if err != nil {
		return nil, fmt.Errorf("read subnets: %w", err)
	}

	projects := make([]Project, 0, len(allocs))
	for _, alloc := range allocs {
		p := Project{
			Name:        alloc.Project,
			SubnetOctet: alloc.SubnetOctet,
			Subnet:      forge.FormatSubnet(alloc.SubnetOctet),
			Gateway:     forge.FormatGateway(alloc.SubnetOctet),
			AllocatedAt: alloc.AllocatedAt,
			Status:      "active",
		}
		projects = append(projects, p)
	}
	return projects, nil
}

func (l *LiveBackend) GetProject(name string) (*ProjectDetail, error) {
	projects, err := l.ListProjects()
	if err != nil {
		return nil, err
	}
	for _, p := range projects {
		if p.Name == name {
			return &ProjectDetail{Project: p}, nil
		}
	}
	return nil, fmt.Errorf("project %q not found", name)
}

func (l *LiveBackend) ListImages() ([]LXDImage, error) {
	return nil, fmt.Errorf("live LXD image listing not implemented (Phase 2)")
}

func (l *LiveBackend) ListProfiles() ([]LXDProfile, error) {
	return nil, fmt.Errorf("live LXD profile listing not implemented (Phase 2)")
}

func (l *LiveBackend) ListClusterMembers() ([]ClusterMember, error) {
	return nil, fmt.Errorf("live LXD cluster listing not implemented (Phase 2)")
}

func (l *LiveBackend) ListTemplates() ([]forge.Template, error) {
	return forge.LoadTemplates()
}

package forge

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

// InstanceUsage is one VM's resource snapshot. Memory and disk are bytes.
type InstanceUsage struct {
	Name        string `json:"name"`
	Project     string `json:"project"`
	Node        string `json:"node"`
	CPU         int    `json:"cpu"`
	MemoryBytes int64  `json:"memory_bytes"`
	DiskBytes   int64  `json:"disk_bytes"`
	Status      string `json:"status"`
}

// NodeCapacity is one cluster member's headline numbers. MemoryUsed is the
// kernel's view ("really used" - buffers/cache excluded by lxd).
type NodeCapacity struct {
	Name        string `json:"name"`
	CPUTotal    int    `json:"cpu_total"`
	MemoryTotal int64  `json:"memory_total"`
	MemoryUsed  int64  `json:"memory_used"`
}

// UsageReport is the aggregate returned by GatherUsage. Projects maps a
// project name to its instances. NodeCapacity is unsorted; rendering code
// is responsible for ordering.
type UsageReport struct {
	Projects     map[string][]InstanceUsage `json:"projects"`
	NodeCapacity []NodeCapacity             `json:"node_capacity"`
}

// GatherUsage queries LXD for every running instance (across every
// project, not just the ones listed in subnets.json) plus per-node
// capacity, and groups the result by project name.
//
// If filterProject is non-empty, only that project's instances are kept
// (but we still issue the cluster-wide query so we can also compare
// against the LXD project list and warn when the supplied name is a
// case mismatch).
//
// Errors from individual nodes / projects are tolerated so a single
// transient failure doesn't blank the whole report.
func GatherUsage(filterProject string) (*UsageReport, error) {
	report := &UsageReport{Projects: map[string][]InstanceUsage{}}

	all, err := lxcAllInstances()
	if err != nil {
		// If the cluster-wide query fails, fall back to per-project
		// queries against subnets.json so we degrade gracefully on
		// older LXD versions that don't support all-projects.
		fallback := []string{}
		if filterProject != "" {
			fallback = []string{filterProject}
		} else if allocs, aerr := GetAllAllocations(); aerr == nil {
			for _, a := range allocs {
				fallback = append(fallback, a.Project)
			}
		}
		for _, p := range fallback {
			if list, perr := lxcInstancesForUsage(p); perr == nil {
				report.Projects[p] = list
			}
		}
	} else {
		for _, ins := range all {
			if filterProject != "" && !strings.EqualFold(ins.Project, filterProject) {
				continue
			}
			// When the user explicitly asked for a project, key the
			// bucket by their spelling so render code can look it up
			// directly even if LXD has the project under a different
			// casing.
			key := ins.Project
			if filterProject != "" {
				key = filterProject
			}
			report.Projects[key] = append(report.Projects[key], ins)
		}
		// If a filter was given but matched nothing, still create an
		// empty bucket so render code shows "no instances" instead of
		// "no projects".
		if filterProject != "" {
			if _, ok := report.Projects[filterProject]; !ok {
				report.Projects[filterProject] = nil
			}
		}
	}

	for proj := range report.Projects {
		list := report.Projects[proj]
		sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
		report.Projects[proj] = list
	}

	if nodes, err := lxcClusterCapacity(); err == nil {
		report.NodeCapacity = nodes
	}

	return report, nil
}

// lxdRawInstance only declares the fields we read out of `lxc query`. The
// rest of the LXD response is silently ignored.
type lxdRawInstance struct {
	Name     string            `json:"name"`
	Project  string            `json:"project"`
	Status   string            `json:"status"`
	Location string            `json:"location"`
	Config   map[string]string `json:"config"`
	State    *struct {
		Disk map[string]struct {
			Usage int64 `json:"usage"`
		} `json:"disk"`
	} `json:"state"`
}

// lxcAllInstances pulls every instance from every LXD project in a
// single call. LXD ≥ 4.18 supports `all-projects=true`; older versions
// will return an error and the caller falls back to per-project queries.
func lxcAllInstances() ([]InstanceUsage, error) {
	out, err := exec.Command("lxc", "query", "/1.0/instances?recursion=2&all-projects=true").Output()
	if err != nil {
		return nil, fmt.Errorf("lxc query all-projects: %w", err)
	}
	var raw []lxdRawInstance
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("parse lxc instance list: %w", err)
	}
	return convertInstances(raw, ""), nil
}

func lxcInstancesForUsage(project string) ([]InstanceUsage, error) {
	cmd := exec.Command("lxc", "query", "--project", project, "/1.0/instances?recursion=2")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("lxc query for project %q: %w", project, err)
	}
	var raw []lxdRawInstance
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("parse lxc instance list for %q: %w", project, err)
	}
	return convertInstances(raw, project), nil
}

// convertInstances translates the LXD response into our InstanceUsage
// shape. defaultProject is used when the response item has no project
// field (older LXD or single-project queries).
func convertInstances(raw []lxdRawInstance, defaultProject string) []InstanceUsage {
	out := make([]InstanceUsage, 0, len(raw))
	for _, r := range raw {
		cpu, _ := strconv.Atoi(r.Config["limits.cpu"])
		mem, _ := ParseMemory(r.Config["limits.memory"])
		var disk int64
		if r.State != nil {
			if root, ok := r.State.Disk["root"]; ok {
				disk = root.Usage
			} else {
				for _, d := range r.State.Disk {
					disk += d.Usage
				}
			}
		}
		project := r.Project
		if project == "" {
			project = defaultProject
		}
		out = append(out, InstanceUsage{
			Name:        r.Name,
			Project:     project,
			Node:        r.Location,
			CPU:         cpu,
			MemoryBytes: mem,
			DiskBytes:   disk,
			Status:      r.Status,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// lxdClusterMember mirrors the slice of /1.0/cluster/members we care about.
type lxdClusterMember struct {
	ServerName string `json:"server_name"`
	Status     string `json:"status"`
}

// lxdResources mirrors the slice of /1.0/resources we care about.
type lxdResources struct {
	CPU struct {
		Total int `json:"total"`
	} `json:"cpu"`
	Memory struct {
		Total int64 `json:"total"`
		Used  int64 `json:"used"`
	} `json:"memory"`
}

func lxcClusterCapacity() ([]NodeCapacity, error) {
	out, err := exec.Command("lxc", "query", "/1.0/cluster/members?recursion=1").Output()
	if err != nil {
		return nil, fmt.Errorf("lxc query cluster members: %w", err)
	}
	var members []lxdClusterMember
	if err := json.Unmarshal(out, &members); err != nil {
		return nil, fmt.Errorf("parse cluster members: %w", err)
	}
	out2 := make([]NodeCapacity, 0, len(members))
	for _, m := range members {
		nc := NodeCapacity{Name: m.ServerName}
		if m.ServerName != "" {
			if r, err := lxcNodeResources(m.ServerName); err == nil {
				nc.CPUTotal = r.CPU.Total
				nc.MemoryTotal = r.Memory.Total
				nc.MemoryUsed = r.Memory.Used
			}
		}
		out2 = append(out2, nc)
	}
	sort.Slice(out2, func(i, j int) bool { return out2[i].Name < out2[j].Name })
	return out2, nil
}

// lxcNodeResources fetches CPU/RAM totals for one cluster member.
// LXD routes cluster-aware endpoints via the `?target=<member>` URL
// parameter; `lxc query` does not have a --target flag, so passing one
// makes the query fail silently and every node looks like it has zero
// capacity. Always use the URL parameter form.
func lxcNodeResources(node string) (*lxdResources, error) {
	url := fmt.Sprintf("/1.0/resources?target=%s", node)
	out, err := exec.Command("lxc", "query", url).Output()
	if err != nil {
		return nil, err
	}
	var r lxdResources
	if err := json.Unmarshal(out, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// ParseMemory parses an LXD limits.memory value into bytes. Accepts plain
// integers (assumed bytes) and the standard binary / decimal suffixes:
// B, KB, MB, GB, TB and KiB, MiB, GiB, TiB. Empty input returns 0, nil.
func ParseMemory(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	i := 0
	for i < len(s) && (s[i] == '.' || (s[i] >= '0' && s[i] <= '9')) {
		i++
	}
	numStr := s[:i]
	unit := strings.ToLower(strings.TrimSpace(s[i:]))
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid memory value %q: %w", s, err)
	}
	var mult float64
	switch unit {
	case "", "b":
		mult = 1
	case "kb":
		mult = 1000
	case "mb":
		mult = 1000 * 1000
	case "gb":
		mult = 1000 * 1000 * 1000
	case "tb":
		mult = 1000 * 1000 * 1000 * 1000
	case "kib":
		mult = 1024
	case "mib":
		mult = 1024 * 1024
	case "gib":
		mult = 1024 * 1024 * 1024
	case "tib":
		mult = 1024 * 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("unknown memory unit %q in %q", unit, s)
	}
	return int64(num * mult), nil
}

// FormatBytes renders a byte count as the largest unit that keeps the
// number reasonably small. Uses GB/MB/KB (decimal) since that's what
// operators expect on a usage dashboard.
func FormatBytes(b int64) string {
	const (
		kb = 1000
		mb = 1000 * 1000
		gb = 1000 * 1000 * 1000
		tb = 1000 * 1000 * 1000 * 1000
	)
	switch {
	case b >= tb:
		return fmt.Sprintf("%.1f TB", float64(b)/float64(tb))
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// SummarizeUsage rolls a list of InstanceUsage records into the headline
// per-project numbers (instance count, total vCPU, memory, disk, node
// list). Returned nodes are sorted and deduplicated.
type ProjectSummary struct {
	Project       string
	InstanceCount int
	CPUTotal      int
	MemoryTotal   int64
	DiskTotal     int64
	Nodes         []string
}

func SummarizeUsage(project string, instances []InstanceUsage) ProjectSummary {
	s := ProjectSummary{Project: project, InstanceCount: len(instances)}
	nodeSet := map[string]bool{}
	for _, ins := range instances {
		s.CPUTotal += ins.CPU
		s.MemoryTotal += ins.MemoryBytes
		s.DiskTotal += ins.DiskBytes
		if ins.Node != "" {
			nodeSet[ins.Node] = true
		}
	}
	for n := range nodeSet {
		s.Nodes = append(s.Nodes, n)
	}
	sort.Strings(s.Nodes)
	return s
}

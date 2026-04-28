package forge

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

// lxdNetwork is the minimal shape we need from `lxc network list --format json`.
type lxdNetwork struct {
	Name   string            `json:"name"`
	Type   string            `json:"type"`
	Config map[string]string `json:"config"`
}

// RunImport registers an existing LXD project into subnets.json by inspecting
// its OVN network and deriving the third octet from ipv4.address (the same
// shape produced by every range main.tf: 10.0.<octet>.1/24). If the project
// is already registered it is a no-op; if no OVN network is present we
// surface a clear error rather than guessing.
func RunImport(project string, yes bool) error {
	if project == "" {
		return fmt.Errorf("project name is required")
	}

	if existing, err := GetProjectSubnet(project); err == nil && existing > 0 {
		fmt.Printf("Project %q already registered with %s\n", project, FormatSubnet(existing))
		return nil
	}

	out, err := exec.Command("lxc", "network", "list", "--project", project, "--format", "json").Output()
	if err != nil {
		return fmt.Errorf("lxc network list failed for project %q: %w", project, err)
	}

	var networks []lxdNetwork
	if err := json.Unmarshal(out, &networks); err != nil {
		return fmt.Errorf("parse lxc network list output: %w", err)
	}

	octet, network, err := pickOctetFromNetworks(networks)
	if err != nil {
		return err
	}

	fmt.Printf("Detected OVN network %q with subnet %s (octet %d) in project %q.\n",
		network, FormatSubnet(octet), octet, project)
	if !ConfirmPrompt("Register this allocation in subnets.json?", yes) {
		return fmt.Errorf("aborted by user")
	}

	data, err := readSubnetsFile()
	if err != nil {
		return err
	}
	for _, a := range data.Allocations {
		if a.SubnetOctet == octet && a.Project != project {
			return fmt.Errorf("octet %d is already held by project %q - resolve the conflict before importing", octet, a.Project)
		}
	}
	data.Allocations = append(data.Allocations, Allocation{
		Project:     project,
		SubnetOctet: octet,
		AllocatedAt: time.Now().Format(time.RFC3339),
	})
	sort.Slice(data.Allocations, func(i, j int) bool {
		return data.Allocations[i].SubnetOctet < data.Allocations[j].SubnetOctet
	})
	if err := writeSubnetsFile(data); err != nil {
		return err
	}
	fmt.Printf("Imported project %q -> %s\n", project, FormatSubnet(octet))
	return nil
}

// pickOctetFromNetworks scans the given networks and returns the third octet
// of the first OVN network with an ipv4.address of the form 10.0.X.1/24.
func pickOctetFromNetworks(networks []lxdNetwork) (int, string, error) {
	for _, n := range networks {
		if !strings.EqualFold(n.Type, "ovn") {
			continue
		}
		addr, ok := n.Config["ipv4.address"]
		if !ok || addr == "" || addr == "none" {
			continue
		}
		octet, err := parseGuacOctet(addr)
		if err != nil {
			continue
		}
		return octet, n.Name, nil
	}
	return 0, "", fmt.Errorf("no OVN network with a 10.0.X.1/24 ipv4.address found - cannot infer subnet octet automatically; use 'forge subnets reserve' instead")
}

// parseGuacOctet extracts X from "10.0.X.1/24". Returns an error for any
// address that does not match the guac /24 pattern.
func parseGuacOctet(addr string) (int, error) {
	if i := strings.Index(addr, "/"); i >= 0 {
		addr = addr[:i]
	}
	parts := strings.Split(addr, ".")
	if len(parts) != 4 {
		return 0, fmt.Errorf("not a dotted quad: %s", addr)
	}
	if parts[0] != "10" || parts[1] != "0" {
		return 0, fmt.Errorf("not in 10.0.0.0/16: %s", addr)
	}
	return strconv.Atoi(parts[2])
}

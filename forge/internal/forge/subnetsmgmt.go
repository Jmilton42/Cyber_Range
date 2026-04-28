package forge

import (
	"fmt"
	"sort"
	"time"
)

// SubnetsList prints every allocation in subnets.json. With jsonOut true the
// list is emitted as JSON for piping into jq. Returns an error only if
// reading subnets.json fails.
func SubnetsList(jsonOut bool) error {
	allocations, err := GetAllAllocations()
	if err != nil {
		return err
	}

	if jsonOut {
		return PrintJSON(map[string]interface{}{
			"subnets_file": SubnetsFile,
			"count":        len(allocations),
			"allocations":  allocations,
		})
	}

	fmt.Printf("Subnets file: %s\n", SubnetsFile)
	if len(allocations) == 0 {
		fmt.Println("No subnets currently allocated.")
		return nil
	}

	sort.Slice(allocations, func(i, j int) bool {
		return allocations[i].SubnetOctet < allocations[j].SubnetOctet
	})

	fmt.Printf("\n%d allocation(s):\n", len(allocations))
	fmt.Printf("  %-30s  %-15s  %s\n", "PROJECT", "SUBNET", "ALLOCATED AT")
	for _, a := range allocations {
		fmt.Printf("  %-30s  %-15s  %s\n", a.Project, FormatSubnet(a.SubnetOctet), a.AllocatedAt)
	}
	return nil
}

// SubnetsFree releases a project's allocation. Always prompts on stdin
// unless yes is true; this matches the design choice that subnet writes
// require explicit confirmation. Returns the released octet on success.
func SubnetsFree(project string, yes bool) error {
	if project == "" {
		return fmt.Errorf("project name is required")
	}

	octet, err := GetProjectSubnet(project)
	if err != nil {
		return err
	}
	if octet == 0 {
		return fmt.Errorf("no allocation found for project %q", project)
	}

	fmt.Printf("Project %q currently holds %s.\n", project, FormatSubnet(octet))
	if !ConfirmPrompt("Release this allocation?", yes) {
		return fmt.Errorf("aborted by user")
	}

	released, err := ReleaseSubnet(project)
	if err != nil {
		return err
	}
	fmt.Printf("Released %s (project %q removed from %s)\n", FormatSubnet(released), project, SubnetsFile)
	return nil
}

// SubnetsReserve hand-allocates a specific octet to a project. Used when an
// existing LXD project (built outside forge) needs to be registered with the
// allocation table. Always prompts unless yes is true.
func SubnetsReserve(project string, octet int, yes bool) error {
	if project == "" {
		return fmt.Errorf("project name is required")
	}
	if octet < 1 || octet > 254 {
		return fmt.Errorf("octet must be between 1 and 254 (got %d)", octet)
	}

	data, err := readSubnetsFile()
	if err != nil {
		return err
	}

	for _, a := range data.Allocations {
		if a.Project == project {
			return fmt.Errorf("project %q already holds %s", project, FormatSubnet(a.SubnetOctet))
		}
		if a.SubnetOctet == octet {
			return fmt.Errorf("octet %d is already taken by project %q", octet, a.Project)
		}
	}

	fmt.Printf("Reserve %s for project %q?\n", FormatSubnet(octet), project)
	if !ConfirmPrompt("Proceed?", yes) {
		return fmt.Errorf("aborted by user")
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
	fmt.Printf("Reserved %s for %q\n", FormatSubnet(octet), project)
	return nil
}

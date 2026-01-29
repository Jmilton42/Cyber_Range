package forge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// SubnetsFile is the constant path to the central subnets.json
const SubnetsFile = "/home/ceroc/InSPIRE/bin/guac_subnet/subnets.json"

// Allocation represents a subnet allocation for a project
type Allocation struct {
	Project     string `json:"project"`
	SubnetOctet int    `json:"subnet_octet"`
	AllocatedAt string `json:"allocated_at"`
}

// SubnetsData represents the subnets.json file structure
type SubnetsData struct {
	Allocations []Allocation `json:"allocations"`
}

// InitSubnetsFile creates the subnets.json file and parent directories if they don't exist
func InitSubnetsFile() error {
	// Check if file already exists
	if _, err := os.Stat(SubnetsFile); err == nil {
		return nil // File exists
	}

	// Create parent directories
	dir := filepath.Dir(SubnetsFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Create empty subnets file
	data := SubnetsData{Allocations: []Allocation{}}
	return writeSubnetsFile(data)
}

// readSubnetsFile reads and parses the subnets.json file
func readSubnetsFile() (SubnetsData, error) {
	var data SubnetsData

	content, err := os.ReadFile(SubnetsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return SubnetsData{Allocations: []Allocation{}}, nil
		}
		return data, fmt.Errorf("failed to read subnets file: %w", err)
	}

	if err := json.Unmarshal(content, &data); err != nil {
		return data, fmt.Errorf("failed to parse subnets file: %w", err)
	}

	return data, nil
}

// writeSubnetsFile writes the subnets data to file
func writeSubnetsFile(data SubnetsData) error {
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal subnets data: %w", err)
	}

	if err := os.WriteFile(SubnetsFile, content, 0644); err != nil {
		return fmt.Errorf("failed to write subnets file: %w", err)
	}

	return nil
}

// GetProjectSubnet returns the subnet octet for a project, or 0 if not found
func GetProjectSubnet(projectName string) (int, error) {
	data, err := readSubnetsFile()
	if err != nil {
		return 0, err
	}

	for _, alloc := range data.Allocations {
		if alloc.Project == projectName {
			return alloc.SubnetOctet, nil
		}
	}

	return 0, nil // Not found
}

// AllocateSubnet allocates a subnet for a project, returns the octet
// If the project already has an allocation, returns the existing octet
func AllocateSubnet(projectName string) (int, error) {
	data, err := readSubnetsFile()
	if err != nil {
		return 0, err
	}

	// Check if project already has allocation
	for _, alloc := range data.Allocations {
		if alloc.Project == projectName {
			return alloc.SubnetOctet, nil
		}
	}

	// Find next available octet
	usedOctets := make(map[int]bool)
	for _, alloc := range data.Allocations {
		usedOctets[alloc.SubnetOctet] = true
	}

	nextOctet := 1
	for nextOctet <= 254 {
		if !usedOctets[nextOctet] {
			break
		}
		nextOctet++
	}

	if nextOctet > 254 {
		return 0, fmt.Errorf("no available subnet octets (all 1-254 are allocated)")
	}

	// Add allocation
	data.Allocations = append(data.Allocations, Allocation{
		Project:     projectName,
		SubnetOctet: nextOctet,
		AllocatedAt: time.Now().Format(time.RFC3339),
	})

	// Sort by octet for cleaner file
	sort.Slice(data.Allocations, func(i, j int) bool {
		return data.Allocations[i].SubnetOctet < data.Allocations[j].SubnetOctet
	})

	if err := writeSubnetsFile(data); err != nil {
		return 0, err
	}

	return nextOctet, nil
}

// ReleaseSubnet removes a project's subnet allocation
func ReleaseSubnet(projectName string) (int, error) {
	data, err := readSubnetsFile()
	if err != nil {
		return 0, err
	}

	var releasedOctet int
	newAllocations := make([]Allocation, 0, len(data.Allocations))

	for _, alloc := range data.Allocations {
		if alloc.Project == projectName {
			releasedOctet = alloc.SubnetOctet
		} else {
			newAllocations = append(newAllocations, alloc)
		}
	}

	if releasedOctet == 0 {
		return 0, fmt.Errorf("no subnet allocation found for project %s", projectName)
	}

	data.Allocations = newAllocations

	if err := writeSubnetsFile(data); err != nil {
		return 0, err
	}

	return releasedOctet, nil
}

// GetAllAllocations returns all current allocations
func GetAllAllocations() ([]Allocation, error) {
	data, err := readSubnetsFile()
	if err != nil {
		return nil, err
	}
	return data.Allocations, nil
}

package main

import (
	"fmt"
	"os"
	"strings"

	"cyber-range-config/internal/forge"
)

const version = "1.0.0"

func main() {
	args := os.Args[1:]

	// Parse global options
	var chdir string
	var showHelp, showVersion bool
	var commandArgs []string
	var command string

	i := 0
	for i < len(args) {
		arg := args[i]

		if strings.HasPrefix(arg, "-chdir=") {
			chdir = strings.TrimPrefix(arg, "-chdir=")
			i++
			continue
		}

		if arg == "-help" || arg == "--help" || arg == "-h" {
			showHelp = true
			i++
			continue
		}

		if arg == "-version" || arg == "--version" || arg == "-v" {
			showVersion = true
			i++
			continue
		}

		// First non-flag argument is the command
		if command == "" && !strings.HasPrefix(arg, "-") {
			command = arg
			i++
			continue
		}

		// Everything after command is passed through
		commandArgs = append(commandArgs, arg)
		i++
	}

	// Handle global flags
	if showVersion {
		fmt.Printf("Forge v%s\n", version)
		return
	}

	if showHelp || command == "" || command == "help" {
		printHelp()
		return
	}

	// Get working directory
	workDir, err := forge.GetWorkingDir(chdir)
	if err != nil {
		printError(err.Error())
		os.Exit(1)
	}

	// Execute command
	var exitCode int
	switch command {
	case "init":
		exitCode = runInit(workDir, commandArgs)
	case "validate":
		exitCode = runValidate(workDir, commandArgs)
	case "plan":
		exitCode = runPlan(workDir, commandArgs)
	case "apply":
		exitCode = runApply(workDir, commandArgs)
	case "destroy":
		exitCode = runDestroy(workDir, commandArgs)
	case "status":
		exitCode = runStatus(workDir)
	case "version":
		fmt.Printf("Forge v%s\n", version)
		exitCode = 0
	default:
		printError(fmt.Sprintf("Unknown command: %s", command))
		printHelp()
		exitCode = 1
	}

	os.Exit(exitCode)
}

func printHelp() {
	help := `Usage: forge [global options] <subcommand> [args]

Forge is a wrapper around OpenTofu that automatically manages guac subnet
allocations for Cyber Range projects.

Main commands:
  init          Prepare your working directory for other commands
  validate      Check whether the configuration is valid
  plan          Show changes required by the current configuration
  apply         Create or update infrastructure (full deployment)
  destroy       Destroy previously-created infrastructure (full teardown)

Other commands:
  status        Show current project's subnet allocation
  help          Show this help output
  version       Show the current Forge version

Global options:
  -chdir=DIR    Switch to a different working directory before executing
  -help         Show this help output
  -version      Show version

Forge automatically:
  - Parses project_name from main.tf in the current directory
  - Manages subnet allocations in /home/ceroc/InSPIRE/bin/guac_subnet/subnets.json
  - Injects -var project_name=X -var guac_subnet_octet=Y to tofu commands

On 'forge apply':
  1. Allocates subnet from subnets.json
  2. Runs tofu apply
  3. Waits for VMs to initialize
  4. Exports LXD instances to instances.json
  5. Starts config server
  6. Starts Windows VMs

On 'forge destroy':
  1. Stops config server
  2. Runs tofu destroy
  3. Releases subnet allocation

Examples:
  forge init                    Initialize and create subnets.json
  forge plan                    Plan with auto-allocated subnet
  forge apply                   Full deployment
  forge apply -auto-approve     Full deployment without confirmation
  forge destroy                 Full teardown and release subnet
  forge status                  Show current allocation
`
	fmt.Print(help)
}

func printError(msg string) {
	fmt.Fprintf(os.Stderr, "\033[31m[ERROR]\033[0m %s\n", msg)
}

func printInfo(msg string) {
	fmt.Printf("\033[32m[INFO]\033[0m %s\n", msg)
}

func printWarn(msg string) {
	fmt.Printf("\033[33m[WARN]\033[0m %s\n", msg)
}

// runInit initializes subnets.json and runs tofu init
func runInit(workDir string, args []string) int {
	// Check for -help
	if forge.CheckHelp(args) {
		return runPassthrough(workDir, "init", args)
	}

	// Initialize subnets file
	printInfo("Initializing subnets file...")
	if err := forge.InitSubnetsFile(); err != nil {
		printError(err.Error())
		return 1
	}
	printInfo(fmt.Sprintf("Subnets file ready: %s", forge.SubnetsFile))

	// Run tofu init
	printInfo("Running tofu init...")
	if err := forge.RunTofuPassthrough(workDir, "init", args); err != nil {
		return 1
	}

	return 0
}

// runValidate runs tofu validate (passthrough)
func runValidate(workDir string, args []string) int {
	return runPassthrough(workDir, "validate", args)
}

// runPlan allocates subnet and runs tofu plan
func runPlan(workDir string, args []string) int {
	// Check for -help
	if forge.CheckHelp(args) {
		return runPassthrough(workDir, "plan", args)
	}

	projectName, subnetOctet, err := getProjectAndSubnet(workDir, false)
	if err != nil {
		printError(err.Error())
		return 1
	}

	printInfo(fmt.Sprintf("Project: %s", projectName))
	printInfo(fmt.Sprintf("Subnet: %s (gateway: %s)", forge.FormatSubnet(subnetOctet), forge.FormatGateway(subnetOctet)))
	fmt.Println()

	if err := forge.RunTofu(workDir, "plan", args, projectName, subnetOctet); err != nil {
		return 1
	}

	return 0
}

// runApply allocates subnet and runs tofu apply, then post-apply steps
func runApply(workDir string, args []string) int {
	// Check for -help
	if forge.CheckHelp(args) {
		return runPassthrough(workDir, "apply", args)
	}

	projectName, subnetOctet, err := getProjectAndSubnet(workDir, true)
	if err != nil {
		printError(err.Error())
		return 1
	}

	fmt.Println("==========================================")
	fmt.Println("  Cyber Range Deployment")
	fmt.Printf("  Project: %s\n", projectName)
	fmt.Println("==========================================")
	fmt.Println()

	printInfo(fmt.Sprintf("Project: %s", projectName))
	printInfo(fmt.Sprintf("Subnet: %s (gateway: %s)", forge.FormatSubnet(subnetOctet), forge.FormatGateway(subnetOctet)))
	fmt.Println()

	if err := forge.RunTofu(workDir, "apply", args, projectName, subnetOctet); err != nil {
		return 1
	}

	// Run post-apply steps (wait, export instances, start server, start windows)
	config := forge.DefaultDeployConfig()
	if err := forge.RunPostApply(workDir, projectName, config); err != nil {
		printWarn(err.Error())
	}

	forge.PrintDeploymentComplete(config, subnetOctet)

	return 0
}

// runDestroy stops server, runs tofu destroy, and releases subnet
func runDestroy(workDir string, args []string) int {
	// Check for -help
	if forge.CheckHelp(args) {
		return runPassthrough(workDir, "destroy", args)
	}

	// Get project name
	projectName, err := forge.ParseProjectName(workDir)
	if err != nil {
		printError(err.Error())
		return 1
	}

	// Get existing subnet (don't allocate new one)
	subnetOctet, err := forge.GetProjectSubnet(projectName)
	if err != nil {
		printError(err.Error())
		return 1
	}

	if subnetOctet == 0 {
		printError(fmt.Sprintf("No subnet allocation found for project '%s'", projectName))
		return 1
	}

	fmt.Println("==========================================")
	fmt.Println("  Cyber Range Destroy")
	fmt.Printf("  Project: %s\n", projectName)
	fmt.Println("==========================================")
	fmt.Println()

	printInfo(fmt.Sprintf("Project: %s", projectName))
	printInfo(fmt.Sprintf("Subnet: %s (will be released after destroy)", forge.FormatSubnet(subnetOctet)))
	fmt.Println()

	// Stop server before destroy
	forge.RunPreDestroy()

	// Run tofu destroy
	if err := forge.RunTofu(workDir, "destroy", args, projectName, subnetOctet); err != nil {
		return 1
	}

	// Release subnet after successful destroy
	printInfo("Releasing subnet allocation...")
	releasedOctet, err := forge.ReleaseSubnet(projectName)
	if err != nil {
		printWarn(fmt.Sprintf("Failed to release subnet: %s", err.Error()))
	} else {
		printInfo(fmt.Sprintf("Released subnet %s", forge.FormatSubnet(releasedOctet)))
	}

	fmt.Println()
	printInfo("Destroy complete!")
	printInfo(fmt.Sprintf("Subnet 10.0.%d.0/24 has been released and is available for reuse.", releasedOctet))

	return 0
}

// runStatus shows current project's subnet allocation
func runStatus(workDir string) int {
	// Get project name
	projectName, err := forge.ParseProjectName(workDir)
	if err != nil {
		printError(err.Error())
		return 1
	}

	// Get subnet
	subnetOctet, err := forge.GetProjectSubnet(projectName)
	if err != nil {
		printError(err.Error())
		return 1
	}

	fmt.Printf("Project:  %s\n", projectName)
	fmt.Printf("Work Dir: %s\n", workDir)
	fmt.Println()

	if subnetOctet > 0 {
		fmt.Printf("Subnet:   %s\n", forge.FormatSubnet(subnetOctet))
		fmt.Printf("Gateway:  %s\n", forge.FormatGateway(subnetOctet))
		fmt.Printf("Octet:    %d\n", subnetOctet)
	} else {
		fmt.Println("Status:   No subnet allocated (run 'forge apply' to allocate)")
	}

	fmt.Println()
	fmt.Printf("Subnets file: %s\n", forge.SubnetsFile)

	// Show all allocations
	allocations, err := forge.GetAllAllocations()
	if err == nil && len(allocations) > 0 {
		fmt.Println()
		fmt.Println("All allocations:")
		for _, a := range allocations {
			marker := "  "
			if a.Project == projectName {
				marker = "* "
			}
			fmt.Printf("%s%-30s  10.0.%d.0/24\n", marker, a.Project, a.SubnetOctet)
		}
	}

	return 0
}

// runPassthrough runs tofu command directly without variable injection
func runPassthrough(workDir string, command string, args []string) int {
	if err := forge.RunTofuPassthrough(workDir, command, args); err != nil {
		return 1
	}
	return 0
}

// getProjectAndSubnet gets project name and allocates/retrieves subnet
func getProjectAndSubnet(workDir string, allocate bool) (string, int, error) {
	projectName, err := forge.ParseProjectName(workDir)
	if err != nil {
		return "", 0, err
	}

	var subnetOctet int
	if allocate {
		subnetOctet, err = forge.AllocateSubnet(projectName)
	} else {
		// For plan, allocate if not exists (so we can show what will be used)
		subnetOctet, err = forge.AllocateSubnet(projectName)
	}

	if err != nil {
		return "", 0, err
	}

	return projectName, subnetOctet, nil
}

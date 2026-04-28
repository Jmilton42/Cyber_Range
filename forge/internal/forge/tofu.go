package forge

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

// RunTofu executes tofu with the given command and arguments
// For plan/apply/destroy, it injects -var flags for project_name and guac_subnet_octet
func RunTofu(workDir string, command string, args []string, projectName string, subnetOctet int) error {
	tofuArgs := []string{command}

	// For commands that need variables, inject them
	needsVars := command == "plan" || command == "apply" || command == "destroy"
	
	if needsVars && projectName != "" && subnetOctet > 0 {
		tofuArgs = append(tofuArgs,
			"-var", fmt.Sprintf("project_name=%s", projectName),
			"-var", fmt.Sprintf("guac_subnet_octet=%d", subnetOctet),
		)
	}

	// Append any additional arguments passed by user
	tofuArgs = append(tofuArgs, args...)

	cmd := exec.Command("tofu", tofuArgs...)
	cmd.Dir = workDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// RunTofuPassthrough runs tofu with arguments passed through directly (no var injection)
func RunTofuPassthrough(workDir string, command string, args []string) error {
	tofuArgs := []string{command}
	tofuArgs = append(tofuArgs, args...)

	cmd := exec.Command("tofu", tofuArgs...)
	cmd.Dir = workDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// Commands that need subnet/project variables
var commandsNeedingVars = map[string]bool{
	"plan":    true,
	"apply":   true,
	"destroy": true,
}

// Commands that are pass-through only
var passthroughCommands = map[string]bool{
	"init":     true,
	"validate": true,
}

// NeedsVars returns true if the command needs project/subnet variables injected
func NeedsVars(command string) bool {
	return commandsNeedingVars[command]
}

// IsPassthrough returns true if the command should be passed through without modification
func IsPassthrough(command string) bool {
	return passthroughCommands[command]
}

// CheckHelp returns true if -help is in the args
func CheckHelp(args []string) bool {
	for _, arg := range args {
		if arg == "-help" || arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}

// FormatSubnet returns a human-readable subnet string
func FormatSubnet(octet int) string {
	return fmt.Sprintf("10.0.%d.0/24", octet)
}

// FormatGateway returns the gateway IP for a subnet
func FormatGateway(octet int) string {
	return fmt.Sprintf("10.0.%d.1", octet)
}

// StringToInt converts string to int with error handling
func StringToInt(s string) (int, error) {
	return strconv.Atoi(s)
}

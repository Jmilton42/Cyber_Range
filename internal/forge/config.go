package forge

import (
	"fmt"
	"os"
	"regexp"
)

// ParseProjectName reads main.tf and extracts the project_name variable default value
func ParseProjectName(dir string) (string, error) {
	mainTfPath := dir + "/main.tf"
	
	content, err := os.ReadFile(mainTfPath)
	if err != nil {
		return "", fmt.Errorf("failed to read main.tf: %w", err)
	}

	// Match: variable "project_name" { ... default = "value" ... }
	// This regex handles multiline variable blocks
	re := regexp.MustCompile(`variable\s+"project_name"\s*\{[^}]*default\s*=\s*"([^"]+)"`)
	matches := re.FindSubmatch(content)
	
	if len(matches) < 2 {
		return "", fmt.Errorf("could not find project_name variable with default value in main.tf")
	}

	return string(matches[1]), nil
}

// GetWorkingDir returns the working directory, applying -chdir if specified
func GetWorkingDir(chdir string) (string, error) {
	if chdir != "" {
		// Verify directory exists
		info, err := os.Stat(chdir)
		if err != nil {
			return "", fmt.Errorf("cannot access directory %s: %w", chdir, err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("%s is not a directory", chdir)
		}
		return chdir, nil
	}

	return os.Getwd()
}

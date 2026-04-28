package forge

import (
	"encoding/json"
	"fmt"
	"os"
)

// PrintJSON marshals v with two-space indentation and writes it to stdout
// followed by a newline. It is the single entry point used by every command
// that supports --json so output formatting stays consistent.
func PrintJSON(v interface{}) error {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	if _, err := os.Stdout.Write(out); err != nil {
		return err
	}
	_, err = os.Stdout.Write([]byte("\n"))
	return err
}

// ConfirmPrompt asserts the user wants to proceed. If autoYes is true, no
// prompt is shown and the function returns true immediately. Otherwise it
// reads a single line from stdin and returns true only on "y" or "yes"
// (case-insensitive). Any other input (including empty) returns false so
// callers can treat the default as "no".
func ConfirmPrompt(message string, autoYes bool) bool {
	if autoYes {
		return true
	}
	fmt.Printf("%s [y/N]: ", message)
	var response string
	if _, err := fmt.Scanln(&response); err != nil {
		return false
	}
	switch response {
	case "y", "Y", "yes", "Yes", "YES":
		return true
	}
	return false
}

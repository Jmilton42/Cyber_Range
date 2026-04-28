// forge-cost is the bundled LXD implementation of the `forge cost`
// command. It produces a focused per-project resource breakdown
// (vCPU / RAM / disk per instance) by querying LXD.
//
// It's shipped as a plugin so future hypervisor backends can replace
// it with a forge-cost binary of their own without touching forge core.
package main

import (
	"os"

	"cyber-range-config/internal/forge"
)

func main() {
	jsonOut := os.Getenv("FORGE_JSON") != ""

	// FORGE_WORK_DIR is set when forge dispatches us. When invoked
	// directly (developer use), fall back to the current process cwd
	// so `forge-cost` still works as a standalone command.
	workDir := os.Getenv("FORGE_WORK_DIR")
	if workDir == "" {
		if cwd, err := os.Getwd(); err == nil {
			workDir = cwd
		}
	}

	args := make([]string, 0, len(os.Args)-1)
	for _, a := range os.Args[1:] {
		switch a {
		case "-json", "--json":
			jsonOut = true
		default:
			args = append(args, a)
		}
	}

	os.Exit(forge.RunCostCommand(workDir, args, jsonOut))
}

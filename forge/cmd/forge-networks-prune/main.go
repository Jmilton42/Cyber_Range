package main

import (
	"flag"
	"fmt"
	"os"

	"cyber-range-config/internal/forge"
)

func main() {
	fs := flag.NewFlagSet("forge-networks-prune", flag.ContinueOnError)
	project := fs.String("project", "", "LXD project to operate in (defaults to current)")
	dryRun := fs.Bool("dry-run", false, "Show what would be deleted without deleting anything")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	pos := fs.Args()
	if len(pos) < 1 {
		fmt.Fprintln(os.Stderr, "usage: forge-networks-prune <prefix> [--project P] [--dry-run]")
		os.Exit(2)
	}
	autoYes := os.Getenv("FORGE_AUTO_YES") != ""
	if err := forge.RunNetworksPrune(forge.NetworksPruneOptions{
		Prefix:  pos[0],
		Project: *project,
		DryRun:  *dryRun,
		Yes:     autoYes,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "\033[31m[ERROR]\033[0m %s\n", err)
		os.Exit(1)
	}
}

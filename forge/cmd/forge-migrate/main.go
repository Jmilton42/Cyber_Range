package main

import (
	"flag"
	"fmt"
	"os"

	"cyber-range-config/internal/forge"
)

func main() {
	fs := flag.NewFlagSet("forge-migrate", flag.ContinueOnError)
	source := fs.String("source", "", "Only migrate instances currently on this node (drain mode)")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	pos := fs.Args()
	if len(pos) < 2 {
		fmt.Fprintln(os.Stderr, "usage: forge-migrate <project> <target-node> [--source <node>]")
		os.Exit(2)
	}
	autoYes := os.Getenv("FORGE_AUTO_YES") != ""
	if err := forge.RunMigrate(pos[0], pos[1], *source, autoYes); err != nil {
		fmt.Fprintf(os.Stderr, "\033[31m[ERROR]\033[0m %s\n", err)
		os.Exit(1)
	}
}

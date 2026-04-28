package main

import (
	"fmt"
	"os"

	"cyber-range-config/internal/forge"
)

func main() {
	project := ""
	if len(os.Args) >= 2 {
		project = os.Args[1]
	}
	if project == "" {
		project = os.Getenv("FORGE_PROJECT")
	}
	if project == "" {
		fmt.Fprintln(os.Stderr, "usage: forge-snapshot <project>")
		os.Exit(2)
	}
	if err := forge.RunSnapshot(project); err != nil {
		fmt.Fprintf(os.Stderr, "\033[31m[ERROR]\033[0m %s\n", err)
		os.Exit(1)
	}
}

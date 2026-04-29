package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"cyber-range-config/internal/rangestudio"
)

func main() {
	listen := flag.String("listen", ":8800", "Listen address (e.g. :8800 or 127.0.0.1:8800)")
	mock := flag.Bool("mock", false, "Enable mock mode (use fixture data instead of live LXD/subnets)")
	subnets := flag.String("subnets", "", "Path to subnets.json (live mode; defaults to forge.SubnetsFile)")
	roots := flag.String("inspire-roots", "", "Comma-separated InSPIRE root directories (live mode)")
	frontend := flag.String("frontend", "", "Path to the Website/ frontend directory (auto-detected if empty)")
	flag.Parse()

	// Also check environment variable for mock mode
	if !*mock {
		if env := os.Getenv("RANGE_STUDIO_MOCK"); env == "1" || env == "true" {
			*mock = true
		}
	}

	var inspireRoots []string
	if *roots != "" {
		inspireRoots = strings.Split(*roots, ",")
	}

	fmt.Println("========================================")
	fmt.Println("  Range Studio")
	if *mock {
		fmt.Println("  Mode: MOCK (fixture data)")
	} else {
		fmt.Println("  Mode: LIVE")
	}
	fmt.Printf("  Listen: %s\n", *listen)
	fmt.Println("========================================")
	fmt.Println()

	cfg := rangestudio.ServeConfig{
		Listen:       *listen,
		Mock:         *mock,
		SubnetsPath:  *subnets,
		InspireRoots: inspireRoots,
		FrontendDir:  *frontend,
	}

	if err := rangestudio.Run(cfg); err != nil {
		log.Fatalf("[FATAL] %v", err)
	}
}

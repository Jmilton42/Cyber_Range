package forge

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
)

// RunCostCommand is a focused per-project resource breakdown: vCPU,
// RAM, and disk for every instance in the project, plus totals. It
// always operates on exactly one project (positional arg, FORGE_PROJECT,
// or the project resolved from main.tf in workDir) and filters the
// cluster-wide /1.0/instances?all-projects=true response down to that
// project, so it stays fast even on large LXD deployments.
//
// Exported so the shipped `forge-cost` plugin shim can drive it.
func RunCostCommand(workDir string, args []string, jsonOut bool) int {
	fs := flag.NewFlagSet("cost", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: forge cost [<project>]")
		fmt.Fprintln(os.Stderr, "  Show per-instance vCPU / RAM / disk for a project.")
		fmt.Fprintln(os.Stderr, "  If <project> is omitted, project_name is read from main.tf in cwd.")
	}
	if err := fs.Parse(args); err != nil {
		return 1
	}

	project := ""
	if fs.NArg() > 0 {
		project = fs.Arg(0)
	} else {
		// Prefer FORGE_PROJECT (set by forge when dispatching to a plugin)
		// before falling back to parsing main.tf, so the cost plugin works
		// the same whether invoked through `forge cost` or directly.
		if envProj := os.Getenv("FORGE_PROJECT"); envProj != "" {
			project = envProj
		} else {
			name, err := ParseProjectName(workDir)
			if err != nil {
				usageRenderError("no project given and could not read project_name from main.tf: " + err.Error())
				return 1
			}
			project = name
		}
	}

	report, err := GatherUsage(project)
	if err != nil {
		if jsonOut {
			_ = PrintJSON(map[string]string{"error": err.Error()})
			return 1
		}
		usageRenderError(err.Error())
		return 1
	}

	instances, ok := report.Projects[project]
	if !ok {
		instances = nil
	}

	if jsonOut {
		summary := SummarizeUsage(project, instances)
		out := map[string]any{
			"project":   project,
			"instances": instances,
			"totals": map[string]any{
				"instance_count": summary.InstanceCount,
				"cpu_total":      summary.CPUTotal,
				"memory_bytes":   summary.MemoryTotal,
				"disk_bytes":     summary.DiskTotal,
				"nodes":          summary.Nodes,
			},
		}
		if err := PrintJSON(out); err != nil {
			usageRenderError(err.Error())
			return 1
		}
		return 0
	}

	printCost(project, instances)
	return 0
}

func printCost(project string, instances []InstanceUsage) {
	fmt.Printf("Project: %s\n\n", project)

	if len(instances) == 0 {
		fmt.Println("No instances found.")
		fmt.Println("If this project was just created, run `forge apply` to spin up its instances.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Instance\tStatus\tNode\tvCPU\tRAM\tDisk")
	fmt.Fprintln(w,
		strings.Repeat("-", 8)+"\t"+
			strings.Repeat("-", 8)+"\t"+
			strings.Repeat("-", 8)+"\t"+
			strings.Repeat("-", 4)+"\t"+
			strings.Repeat("-", 8)+"\t"+
			strings.Repeat("-", 8))

	sorted := make([]InstanceUsage, len(instances))
	copy(sorted, instances)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	running := 0
	stopped := 0
	for _, ins := range sorted {
		if ins.Status == "Running" {
			running++
		} else {
			stopped++
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n",
			ins.Name, ins.Status, ins.Node, ins.CPU,
			FormatBytes(ins.MemoryBytes),
			FormatBytes(ins.DiskBytes))
	}

	summary := SummarizeUsage(project, instances)
	fmt.Fprintln(w,
		strings.Repeat("-", 8)+"\t"+
			strings.Repeat("-", 8)+"\t"+
			strings.Repeat("-", 8)+"\t"+
			strings.Repeat("-", 4)+"\t"+
			strings.Repeat("-", 8)+"\t"+
			strings.Repeat("-", 8))
	fmt.Fprintf(w, "TOTAL (%d)\t%d running, %d stopped\t%s\t%d\t%s\t%s\n",
		summary.InstanceCount,
		running, stopped,
		strings.Join(summary.Nodes, ","),
		summary.CPUTotal,
		FormatBytes(summary.MemoryTotal),
		FormatBytes(summary.DiskTotal))
	_ = w.Flush()
}

// usageRenderError mirrors cmd/forge's printError so plugin output looks
// identical to the legacy built-in (red [ERROR] prefix on stderr).
func usageRenderError(msg string) {
	fmt.Fprintf(os.Stderr, "\033[31m[ERROR]\033[0m %s\n", msg)
}

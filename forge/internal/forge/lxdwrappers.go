package forge

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// scriptPath returns the absolute path to a script in the configured
// scripts directory (the same place start_win.sh lives).
func scriptPath(name string) string {
	return filepath.Join(filepath.Dir(DefaultDeployConfig().StartWinScript), name)
}

// runScript invokes a shell script with stdout/stderr/stdin wired to the
// caller's terminal. Provides a single chokepoint for "missing script"
// diagnostics so each wrapper doesn't repeat the same boilerplate.
func runScript(name string, args ...string) error {
	path := scriptPath(name)
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("%s not found at %s", name, path)
	}
	cmd := exec.Command("bash", append([]string{path}, args...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunSnapshot creates a timestamped snapshot of every instance in the
// project (delegates to snapshot.sh).
func RunSnapshot(project string) error {
	if project == "" {
		return fmt.Errorf("project name is required")
	}
	return runScript("snapshot.sh", project)
}

// RunStart starts every instance in the project (delegates to start_vms.sh).
func RunStart(project string) error {
	if project == "" {
		return fmt.Errorf("project name is required")
	}
	return runScript("start_vms.sh", project)
}

// RunStop force-stops every instance in the project (delegates to
// stop_vms.sh).
func RunStop(project string) error {
	if project == "" {
		return fmt.Errorf("project name is required")
	}
	return runScript("stop_vms.sh", project)
}

// lxdInstance is the minimal shape of an entry from `lxc list --format
// json`. We only need name and location (cluster member).
type lxdInstance struct {
	Name     string `json:"name"`
	Location string `json:"location"`
	Status   string `json:"status"`
}

// RunMigrate moves instances between cluster members. The two operating
// modes mirror the underlying scripts:
//
//   - source != "":  move only instances currently on `source` to `target`.
//     Delegates to move_vms_nodes.sh.
//   - source == "":  move every instance in the project to `target`.
//     Delegates to move_vms.sh.
//
// Before invoking the script, this wrapper queries `lxc list` and prints a
// preview of the affected instances so the operator can spot mistakes.
// Skips the prompt when yes is true (for CI / scripted use).
func RunMigrate(project, target, source string, yes bool) error {
	if project == "" || target == "" {
		return fmt.Errorf("project and target node are required")
	}

	out, err := exec.Command("lxc", "list", "--project", project, "--format", "json").Output()
	if err != nil {
		return fmt.Errorf("lxc list failed for project %q: %w", project, err)
	}
	var instances []lxdInstance
	if err := json.Unmarshal(out, &instances); err != nil {
		return fmt.Errorf("parse lxc list output: %w", err)
	}

	affected := []lxdInstance{}
	for _, inst := range instances {
		if source == "" || inst.Location == source {
			affected = append(affected, inst)
		}
	}
	if len(affected) == 0 {
		if source == "" {
			fmt.Printf("Project %q has no instances - nothing to migrate.\n", project)
		} else {
			fmt.Printf("No instances in project %q currently live on %q - nothing to migrate.\n", project, source)
		}
		return nil
	}

	if source == "" {
		fmt.Printf("Migrating %d instance(s) in project %q to node %q:\n", len(affected), project, target)
	} else {
		fmt.Printf("Draining %d instance(s) from %q -> %q (project %q):\n", len(affected), source, target, project)
	}
	for _, inst := range affected {
		fmt.Printf("  - %-30s  current=%-12s  status=%s\n", inst.Name, inst.Location, inst.Status)
	}

	if !ConfirmPrompt("Proceed?", yes) {
		return fmt.Errorf("aborted by user")
	}

	if source != "" {
		return runScript("move_vms_nodes.sh", project, target, source)
	}
	return runScript("move_vms.sh", project, target)
}

// NetworksPruneOptions configures a `forge networks prune` invocation.
// They mirror the flags accepted by remove_networks.sh so the wrapper
// stays a thin facade.
type NetworksPruneOptions struct {
	Prefix  string
	Project string
	DryRun  bool
	Yes     bool
}

// RunNetworksPrune deletes orphan OVN networks whose names start with the
// given prefix, by delegating to remove_networks.sh. The wrapper exists so
// operators get the same UX as other forge commands (typed confirmation
// is handled inside the script - we just pass --yes through).
func RunNetworksPrune(opts NetworksPruneOptions) error {
	if opts.Prefix == "" {
		return fmt.Errorf("prefix is required (e.g. 'forge networks prune cptc-')")
	}
	args := []string{opts.Prefix}
	if opts.Project != "" {
		args = append(args, "--project", opts.Project)
	}
	if opts.DryRun {
		args = append(args, "--dry-run")
	}
	if opts.Yes {
		args = append(args, "--yes")
	}
	return runScript("remove_networks.sh", args...)
}

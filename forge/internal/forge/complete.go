package forge

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// builtinCommands lists every command compiled directly into the forge
// binary. The moved-to-plugins commands (snapshot/start/stop/migrate/
// networks-prune/usage/cost) are intentionally excluded - they show up
// via ListPlugins below when their forge-* binaries are installed, and
// only then.
var builtinCommands = []string{
	"init", "validate", "plan", "apply", "destroy",
	"status", "doctor", "config",
	"serve", "logs", "reload",
	"subnets", "import",
	"new", "plugins",
	"help", "version",
}

// RunComplete handles the hidden `forge __complete <kind> [args...]`
// command. Output is one match per line on stdout. Errors are swallowed
// (and a clean exit 0 is returned) so a broken cluster never wedges the
// user's shell completion.
func RunComplete(args []string) int {
	if len(args) == 0 {
		return 0
	}
	kind := args[0]
	rest := args[1:]

	switch kind {
	case "projects":
		printProjects()
	case "nodes":
		printNodes()
	case "instances":
		project := ""
		if len(rest) > 0 {
			project = rest[0]
		}
		printInstances(project)
	case "templates":
		printTemplates()
	case "commands":
		printCommands()
	}
	return 0
}

// printCommands enumerates every name reachable as `forge <name>`. Built-ins
// come from the static list above; plugins discovered via $PATH are
// appended so any `forge-foo` binary becomes tab-completable without
// recompiling.
func printCommands() {
	seen := map[string]bool{}
	for _, c := range builtinCommands {
		if seen[c] {
			continue
		}
		seen[c] = true
		fmt.Println(c)
	}
	plugins, err := ListPlugins()
	if err != nil {
		return
	}
	for _, p := range plugins {
		if seen[p.Name] {
			continue
		}
		seen[p.Name] = true
		fmt.Println(p.Name)
	}
}

// printTemplates lists template ids discovered as subdirectories of the
// templates dir, plus the synthetic "blank" template. Errors are
// silently swallowed so completion stays robust if the dir is missing
// or unreadable.
func printTemplates() {
	tpls, err := LoadTemplates()
	if err != nil {
		return
	}
	for _, t := range tpls {
		fmt.Println(t.ID)
	}
}

func printProjects() {
	allocs, err := GetAllAllocations()
	if err != nil {
		return
	}
	for _, a := range allocs {
		fmt.Println(a.Project)
	}
}

func printNodes() {
	out, err := exec.Command("lxc", "cluster", "list", "--format", "json").Output()
	if err != nil {
		return
	}
	var members []struct {
		ServerName string `json:"server_name"`
	}
	if jerr := json.Unmarshal(out, &members); jerr != nil {
		return
	}
	for _, m := range members {
		if m.ServerName != "" {
			fmt.Println(m.ServerName)
		}
	}
}

func printInstances(project string) {
	args := []string{"list", "--format", "json"}
	if project != "" {
		args = append([]string{"--project", project}, args...)
	}
	out, err := exec.Command("lxc", args...).Output()
	if err != nil {
		return
	}
	var instances []struct {
		Name string `json:"name"`
	}
	if jerr := json.Unmarshal(out, &instances); jerr != nil {
		return
	}
	for _, inst := range instances {
		if inst.Name != "" {
			fmt.Println(inst.Name)
		}
	}
}

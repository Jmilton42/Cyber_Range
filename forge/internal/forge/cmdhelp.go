package forge

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// SubCommandHelp is a lightweight man-page entry for one forge command.
// All fields are optional except Synopsis + Description.
type SubCommandHelp struct {
	Synopsis    string   // first-line `Usage: ...` text
	Description string   // 1-3 paragraph what/why
	Options     []string // pre-formatted `  -flag       desc` lines
	Examples    []string // pre-formatted `  forge ...` lines
	SeeAlso     []string // related command names ("status", "doctor", ...)
	Notes       string   // free-form gotchas
}

// ArgsHaveHelp returns true when args contain -h / -help / --help in any
// position. Used by every runXxx so a command can short-circuit straight
// into its help blurb without parsing flags.
func ArgsHaveHelp(args []string) bool {
	for _, a := range args {
		switch a {
		case "-h", "-help", "--help":
			return true
		}
	}
	return false
}

// PrintSubCommandHelp emits the man-page entry for one command to w.
func PrintSubCommandHelp(w io.Writer, command string, h SubCommandHelp) {
	fmt.Fprintf(w, "Usage: %s\n\n", h.Synopsis)
	if h.Description != "" {
		fmt.Fprintln(w, strings.TrimRight(h.Description, "\n"))
		fmt.Fprintln(w)
	}
	if len(h.Options) > 0 {
		fmt.Fprintln(w, "Options:")
		for _, line := range h.Options {
			fmt.Fprintln(w, line)
		}
		fmt.Fprintln(w)
	}
	if len(h.Examples) > 0 {
		fmt.Fprintln(w, "Examples:")
		for _, line := range h.Examples {
			fmt.Fprintln(w, line)
		}
		fmt.Fprintln(w)
	}
	if h.Notes != "" {
		fmt.Fprintln(w, "Notes:")
		fmt.Fprintln(w, strings.TrimRight(h.Notes, "\n"))
		fmt.Fprintln(w)
	}
	if len(h.SeeAlso) > 0 {
		fmt.Fprintf(w, "See also: %s\n", strings.Join(h.SeeAlso, ", "))
	}
	fmt.Fprintf(w, "Full docs:  docs/forge/%s.md\n", helpDocFile(command))
}

// helpDocFile maps a command name to its docs/forge/<file>.md basename.
// Most commands map 1:1; subnets shares one page, networks-prune is
// hyphenated.
func helpDocFile(command string) string {
	switch command {
	case "subnets":
		return "subnets"
	case "networks-prune":
		return "networks-prune"
	default:
		return command
	}
}

// LookupSubCommandHelp returns the help entry for command (case-sensitive
// on the canonical name; aliases like `networks prune` should be passed
// already normalised to `networks-prune`).
func LookupSubCommandHelp(command string) (SubCommandHelp, bool) {
	h, ok := subCommandHelp[command]
	return h, ok
}

// SubCommandNames returns every command we have a help entry for, sorted.
// Used by `forge help <topic>` to show what's documented.
func SubCommandNames() []string {
	names := make([]string, 0, len(subCommandHelp))
	for k := range subCommandHelp {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// PrintSubCommandHelpTo writes help for command to stdout, returning
// whether an entry was found. The caller is expected to exit 0 when this
// returns true and fall through to "unknown command" otherwise.
func PrintSubCommandHelpTo(command string) bool {
	h, ok := LookupSubCommandHelp(command)
	if !ok {
		return false
	}
	PrintSubCommandHelp(os.Stdout, command, h)
	return true
}

// subCommandHelp is the source of truth for every `forge <cmd> -help`
// page. Keep these short and useful: synopsis + 1-3 paragraphs + a few
// flags + examples is the target.
var subCommandHelp = map[string]SubCommandHelp{
	"init": {
		Synopsis: "forge init",
		Description: `Prepare the working directory for OpenTofu by running ` + "`tofu init`" + ` and
ensuring subnets.json exists. Run this once per project (or after
changing providers / modules) before plan/apply.`,
		Options: []string{
			"  -help        Show this help",
			"  Anything else is forwarded to `tofu init`.",
		},
		Examples: []string{
			"  forge init",
		},
		SeeAlso: []string{"plan", "apply", "doctor"},
	},

	"validate": {
		Synopsis: "forge validate",
		Description: `Run ` + "`tofu validate`" + ` against the current configuration. Catches HCL
syntax errors and obvious provider misuse without contacting a backend.`,
		Examples: []string{"  forge validate"},
		SeeAlso:  []string{"plan", "doctor"},
	},

	"plan": {
		Synopsis: "forge plan",
		Description: `Show the changes ` + "`forge apply`" + ` would make. Allocates a /24 guac
subnet for the project (if one isn't allocated yet) and forwards
project_name + guac_subnet_octet into ` + "`tofu plan`" + `.`,
		Options: []string{
			"  -help        Show this help",
			"  Other flags are passed through to `tofu plan`.",
		},
		Examples: []string{
			"  forge plan",
			"  forge plan -target=lxd_instance.dc01",
		},
		SeeAlso: []string{"apply", "destroy", "subnets"},
	},

	"apply": {
		Synopsis: "forge apply",
		Description: `Create or update the project's infrastructure (full deployment).
Allocates a guac subnet on first run, persists it in subnets.json, and
runs ` + "`tofu apply`" + ` with project_name + guac_subnet_octet wired in.`,
		Options: []string{
			"  -yes         Skip the confirmation prompt (also -y or --yes)",
			"  -help        Show this help",
			"  Other flags are passed through to `tofu apply`.",
		},
		Examples: []string{
			"  forge apply",
			"  forge -yes apply",
		},
		SeeAlso: []string{"plan", "destroy", "subnets"},
	},

	"destroy": {
		Synopsis: "forge destroy",
		Description: `Tear down every resource in the project (full destroy). The project's
guac subnet is released back into the pool unless you pass --keep-subnet.

Always asks for confirmation unless -yes is set.`,
		Options: []string{
			"  -yes              Skip the confirmation prompt",
			"  --keep-subnet     Don't release the project's subnet from subnets.json",
			"  -help             Show this help",
		},
		Examples: []string{
			"  forge destroy",
			"  forge -yes destroy",
			"  forge destroy --keep-subnet",
		},
		SeeAlso: []string{"apply", "subnets", "import"},
	},

	"status": {
		Synopsis: "forge status [<project>] [-json]",
		Description: `Show subnet allocation context. With no argument, prints the project
in the cwd plus its allocated /24. Cluster-wide listing comes via
` + "`forge subnets list`" + `; status is the in-project view.`,
		Options: []string{
			"  -json    Emit a JSON object instead of a table",
			"  -help    Show this help",
		},
		Examples: []string{
			"  forge status",
			"  forge status CPTC-Mock",
			"  forge -json status",
		},
		SeeAlso: []string{"subnets", "doctor", "config"},
	},

	"doctor": {
		Synopsis: "forge doctor [-json]",
		Description: `Run preflight checks: tofu/lxc/jq binaries, subnets.json, config.yaml,
LXD cluster reachability, template directory, plugin presence, and the
optional config server's /status endpoint.

Exit code is non-zero when any FAIL check fires; warnings don't fail.`,
		Options: []string{
			"  -json    Machine-readable output (one record per check)",
			"  -help    Show this help",
		},
		Examples: []string{
			"  forge doctor",
			"  forge -json doctor | jq '.healthy'",
		},
		SeeAlso: []string{"config", "status", "plugins"},
	},

	"config": {
		Synopsis: "forge config [-json]",
		Description: `Print the resolved deploy config: which config.yaml was loaded, every
override from FORGE_* env vars, and the final values that will be sent
into tofu/lxd.`,
		Options: []string{
			"  -json    Emit JSON instead of the indented summary",
			"  -help    Show this help",
		},
		Examples: []string{"  forge config"},
		SeeAlso:  []string{"doctor"},
	},

	"serve": {
		Synopsis: "forge serve [--addr ADDR] [--instances FILE]",
		Description: `Run the HTTP configuration server. Auto-discovers ./instances.json in
the current project unless --instances is given. Logs to ./server.log.`,
		Options: []string{
			"  --addr ADDR        Bind address (default :8080, env FORGE_SERVE_ADDR)",
			"  --instances FILE   Path to instances.json (default ./instances.json)",
			"  -help              Show this help",
		},
		Examples: []string{
			"  forge serve",
			"  forge serve --addr 127.0.0.1:9000",
		},
		SeeAlso: []string{"reload", "logs"},
	},

	"logs": {
		Synopsis: "forge logs [-f]",
		Description: `Show or tail the running server's server.log in the current project
directory. -f tails new lines until Ctrl-C.`,
		Options: []string{
			"  -f       Follow / tail mode",
			"  -help    Show this help",
		},
		Examples: []string{
			"  forge logs",
			"  forge logs -f",
		},
		SeeAlso: []string{"serve", "reload"},
	},

	"reload": {
		Synopsis: "forge reload",
		Description: `POST /reload to the running config server so it re-reads
instances.json without dropping connections.`,
		Examples: []string{"  forge reload"},
		SeeAlso:  []string{"serve", "logs"},
	},

	"subnets": {
		Synopsis: "forge subnets <list|free|reserve> [args...]",
		Description: `Manage allocations in subnets.json.

  list                       Show every (project, /24) pair, plus free
                             octets (-json for a machine-readable view).
  free <project>             Release a project's subnet (asks for
                             confirmation unless -yes is set).
  reserve <project> <octet>  Pre-allocate a specific /24 octet (1-254).`,
		Options: []string{
			"  -json    JSON output (subnets list only)",
			"  -yes     Skip free/reserve confirmation prompts",
			"  -help    Show this help",
		},
		Examples: []string{
			"  forge subnets list",
			"  forge subnets list -json",
			"  forge subnets reserve CPTC-Mock 42",
			"  forge -yes subnets free OldProject",
		},
		SeeAlso: []string{"status", "import"},
	},

	"import": {
		Synopsis: "forge import <project>",
		Description: `Register an existing LXD project in subnets.json. Useful when the
project was created outside forge (or before subnets.json existed) but
is now living on the cluster.`,
		Options: []string{
			"  -yes     Skip the confirmation prompt",
			"  -help    Show this help",
		},
		Examples: []string{"  forge import LegacyLab"},
		SeeAlso:  []string{"subnets", "status"},
	},

	"new": {
		Synopsis: "forge new [--template ID] [--name NAME] [--dir DIR] [--reserve|--no-reserve]",
		Description: `Scaffold a new project from a template directory. With no flags
forge prompts interactively. Template IDs are subdirectories of
$FORGE_TEMPLATES (default /home/ceroc/InSPIRE/templates) that contain
a main.tf.`,
		Options: []string{
			"  --template ID   Template id (directory name); blank uses the empty template",
			"  --name NAME     Project name (also rewrites project_name in main.tf)",
			"  --dir DIR       Output directory (default ./<name>)",
			"  --reserve       Allocate a guac subnet up front",
			"  --no-reserve    Skip subnet allocation; first apply will reserve one",
			"  -help           Show this help",
		},
		Examples: []string{
			"  forge new",
			"  forge new --template csc-3410 --name myCSC",
			"  forge new --template blank --name throwaway --no-reserve",
		},
		SeeAlso: []string{"plan", "apply"},
	},

	"plugins": {
		Synopsis: "forge plugins list [-json]",
		Description: `Enumerate every forge-* binary on $PATH. Bundled plugins are listed
even when shadowed by a user-installed copy higher up the PATH.`,
		Options: []string{
			"  -json    Machine-readable output",
			"  -help    Show this help",
		},
		Examples: []string{
			"  forge plugins list",
			"  forge plugins list -json | jq '.plugins[].name'",
		},
		SeeAlso: []string{"doctor"},
	},

	"snapshot": {
		Synopsis: "forge snapshot <project>",
		Description: `Snapshot every instance in the project (delegates to the bundled
forge-snapshot plugin, which uses LXD snapshots).`,
		Examples: []string{"  forge snapshot CPTC-Mock"},
		SeeAlso:  []string{"start", "stop", "migrate"},
		Notes:    "Implemented by the forge-snapshot plugin. Run `forge plugins list` to confirm it's installed.",
	},

	"start": {
		Synopsis:    "forge start <project>",
		Description: `Start every instance in the project (forge-start plugin).`,
		Examples:    []string{"  forge start CPTC-Mock"},
		SeeAlso:     []string{"stop", "snapshot"},
		Notes:       "Implemented by the forge-start plugin.",
	},

	"stop": {
		Synopsis:    "forge stop <project>",
		Description: `Force-stop every instance in the project (forge-stop plugin).`,
		Examples:    []string{"  forge stop CPTC-Mock"},
		SeeAlso:     []string{"start", "snapshot"},
		Notes:       "Implemented by the forge-stop plugin.",
	},

	"migrate": {
		Synopsis: "forge migrate <project> <target-node> [--source NODE] [--dry-run]",
		Description: `Move every instance in the project to a different cluster member.
With --source, only instances currently on that node are migrated; the
default is to move every instance regardless of its current home.`,
		Options: []string{
			"  --source NODE   Only move instances currently on this node",
			"  --dry-run       Print what would move and exit 0 without moving",
			"  -yes            Skip the confirmation prompt",
			"  -help           Show this help",
		},
		Examples: []string{
			"  forge migrate CPTC-Mock micro-01",
			"  forge migrate CPTC-Mock micro-01 --source micro-05",
			"  forge -yes migrate CPTC-Mock micro-02 --dry-run",
		},
		SeeAlso: []string{"cost", "snapshot"},
		Notes:   "Implemented by the forge-migrate plugin.",
	},

	"networks-prune": {
		Synopsis: "forge networks prune <prefix> [--project P] [--dry-run]",
		Description: `Delete OVN networks whose names start with <prefix>. Useful for
cleaning up after a lab where networks were created outside the
project's tofu state and never destroyed.`,
		Options: []string{
			"  --project P   Only consider networks in this LXD project",
			"  --dry-run     Print what would be deleted, exit 0 without deleting",
			"  -yes          Skip the confirmation prompt",
			"  -help         Show this help",
		},
		Examples: []string{
			"  forge networks prune teamA-",
			"  forge networks prune teamA- --dry-run",
			"  forge -yes networks prune teamA- --project CPTC-Mock",
		},
		SeeAlso: []string{"plugins"},
		Notes:   "Implemented by the forge-networks-prune plugin.",
	},

	"cost": {
		Synopsis: "forge cost [<project>] [-json]",
		Description: `Per-instance vCPU / RAM / disk for one project, plus totals. If
<project> is omitted, project_name is read from main.tf in the cwd.
Project lookup is case-insensitive against LXD.

CPU and RAM are declared limits (limits.cpu / limits.memory). Disk is
the live root-device usage.

Data source: lxc query "/1.0/instances?recursion=2&all-projects=true",
filtered by project. Run ` + "`lxc project list`" + ` to see the
canonical project names LXD knows about.`,
		Options: []string{
			"  -json     JSON output instead of a table",
			"  -help     Show this help",
		},
		Examples: []string{
			"  forge cost",
			"  forge cost CPTC-Mock",
			"  forge -json cost CPTC-Mock | jq '.totals'",
		},
		SeeAlso: []string{"status", "migrate"},
		Notes: `Implemented by the forge-cost plugin.

If "No instances found" appears unexpectedly, the project_name in
main.tf doesn't match any LXD project. Run ` + "`lxc project list`" +
			` and update either main.tf or the LXD project to align them.`,
	},

	"version": {
		Synopsis:    "forge version",
		Description: `Print the current forge version and exit.`,
		Examples:    []string{"  forge version"},
	},

	"help": {
		Synopsis: "forge help [<topic>]",
		Description: `With no argument, prints the global forge help summary. With a
topic, prints the man-page entry for that command (equivalent to
` + "`forge <topic> -help`" + `).`,
		Examples: []string{
			"  forge help",
			"  forge help destroy",
			"  forge help cost",
		},
	},
}

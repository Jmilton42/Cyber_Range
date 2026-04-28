package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"cyber-range-config/internal/forge"
)

const version = "2.0.0"

// globalFlags holds CLI-wide flags that any subcommand may read. They are
// parsed by parseGlobals before subcommand dispatch so individual commands
// don't have to hand-roll the same boilerplate.
type globalFlags struct {
	chdir       string
	jsonOut     bool
	autoYes     bool
	completion  string
	showHelp    bool
	showVersion bool
}

func main() {
	args := os.Args[1:]

	gf, command, commandArgs := parseGlobals(args)

	if gf.completion != "" {
		if err := forge.RunCompletion(gf.completion); err != nil {
			printError(err.Error())
			os.Exit(1)
		}
		return
	}

	if gf.showVersion {
		fmt.Printf("Forge v%s\n", version)
		return
	}

	if gf.showHelp || command == "" {
		printHelp()
		return
	}

	// `forge help [<topic>]` prints the global summary or a per-command
	// man-page entry (`forge help destroy`).
	if command == "help" {
		if len(commandArgs) > 0 {
			topic := normalizeHelpTopic(commandArgs)
			if forge.PrintSubCommandHelpTo(topic) {
				return
			}
			printError(fmt.Sprintf("no help topic for %q (try `forge help`)", topic))
			os.Exit(1)
		}
		printHelp()
		return
	}

	// `forge <cmd> -help` prints the man-page entry for that command.
	// We intercept here so every command gets consistent help without
	// each runXxx having to re-implement the check.
	if forge.ArgsHaveHelp(commandArgs) {
		topic := command
		if command == "networks" && len(commandArgs) > 0 && commandArgs[0] == "prune" {
			topic = "networks-prune"
		}
		if forge.PrintSubCommandHelpTo(topic) {
			return
		}
		// Fall through: command may still be a plugin which can render
		// its own help page.
	}

	workDir, err := forge.GetWorkingDir(gf.chdir)
	if err != nil {
		printError(err.Error())
		os.Exit(1)
	}

	var exitCode int
	switch command {
	case "init":
		exitCode = runInit(workDir, commandArgs)
	case "validate":
		exitCode = runValidate(workDir, commandArgs)
	case "plan":
		exitCode = runPlan(workDir, commandArgs)
	case "apply":
		exitCode = runApply(workDir, commandArgs)
	case "destroy":
		exitCode = runDestroy(workDir, commandArgs)
	case "status":
		exitCode = runStatus(workDir, gf.jsonOut)
	case "serve":
		exitCode = runServe(commandArgs)
	case "doctor":
		exitCode = runDoctor(gf.jsonOut)
	case "config":
		exitCode = runConfig(gf.jsonOut)
	case "logs":
		exitCode = runLogs(workDir, commandArgs)
	case "reload":
		exitCode = runReload()
	case "subnets":
		exitCode = runSubnets(commandArgs, gf)
	case "import":
		exitCode = runImport(commandArgs, gf.autoYes)
	case "plugins":
		exitCode = runPlugins(commandArgs, gf.jsonOut)
	case "new":
		exitCode = runNew(commandArgs, gf.autoYes)
	case "version":
		fmt.Printf("Forge v%s\n", version)
		exitCode = 0
	case "__complete":
		exitCode = forge.RunComplete(commandArgs)
	default:
		exitCode = dispatchPluginOrUnknown(workDir, command, commandArgs, gf)
	}

	os.Exit(exitCode)
}

// parseGlobals strips global flags from args and returns the remaining
// command name and command-specific arguments. Unknown flags before the
// command are passed through untouched so subcommand-specific flags (e.g.
// --source for migrate) reach their handlers intact.
func parseGlobals(args []string) (globalFlags, string, []string) {
	var gf globalFlags
	var command string
	var rest []string

	i := 0
	for i < len(args) {
		arg := args[i]

		switch {
		case strings.HasPrefix(arg, "-chdir="):
			gf.chdir = strings.TrimPrefix(arg, "-chdir=")
		case arg == "-help", arg == "--help", arg == "-h":
			// Once we've identified the subcommand, -help belongs to
			// it (e.g. `forge cost -help` should show the cost
			// man-page, not the global one).
			if command == "" {
				gf.showHelp = true
			} else {
				rest = append(rest, arg)
			}
		case arg == "-version", arg == "--version", arg == "-v":
			if command == "" {
				gf.showVersion = true
			} else {
				rest = append(rest, arg)
			}
		case arg == "-json", arg == "--json":
			gf.jsonOut = true
		case arg == "-yes", arg == "--yes", arg == "-y":
			gf.autoYes = true
		case strings.HasPrefix(arg, "-completion="):
			gf.completion = strings.TrimPrefix(arg, "-completion=")
		case strings.HasPrefix(arg, "--completion="):
			gf.completion = strings.TrimPrefix(arg, "--completion=")
		case command == "" && !strings.HasPrefix(arg, "-"):
			command = arg
		default:
			rest = append(rest, arg)
		}
		i++
	}
	return gf, command, rest
}

func printHelp() {
	help := `Usage: forge [global options] <subcommand> [args]

Forge is a wrapper around OpenTofu that automatically manages guac subnet
allocations for Cyber Range projects, plus day-2 LXD operations.

Infrastructure:
  init              Prepare your working directory for other commands
  new               Scaffold a new project from a template directory
  validate          Check whether the configuration is valid
  plan              Show changes required by the current configuration
  apply             Create or update infrastructure (full deployment)
  destroy           Destroy previously-created infrastructure (full teardown)

Diagnostics:
  status            Show subnet allocation (in-project or cluster-wide)
  doctor            Run preflight checks (tofu, lxc, subnets.json, server, ...)
  config            Show resolved deploy config + loaded config.yaml path

Server control:
  serve             Run the HTTP config server (auto-discovers ./instances.json)
  logs [-f]         Show / tail server.log in the current project
  reload            POST /reload to the running config server

Subnets:
  subnets list                   List all allocations in subnets.json
  subnets free <project>         Release a project's subnet (confirm by default)
  subnets reserve <p> <octet>    Pre-allocate a specific octet (confirm by default)
  import <project>               Register an existing LXD project in subnets.json

Plugins:
  plugins list                   List discovered forge-* binaries on $PATH
  <name> ...                     Any unknown command resolves to forge-<name>
                                 if it's on $PATH (kubectl-style plugins).
                                 Bundled: forge-snapshot, forge-start,
                                          forge-stop, forge-migrate,
                                          forge-networks-prune,
                                          forge-cost
  cost [<project>]               Per-project vCPU / RAM / disk
                                 breakdown (forge-cost plugin)

Other:
  version           Show the current Forge version
  help              Show this help output
  help <topic>      Show the man-page entry for one command
                    (also: forge <topic> -help)

Global options:
  -chdir=DIR        Switch to a different working directory before executing
  -help             Show this help output
  -version          Show version
  -json             Emit JSON output where supported (status, subnets list,
                    config, doctor)
  -yes              Skip interactive confirmation prompts
  -completion=SH    Print shell completion (bash|zsh) and exit

Examples:
  forge init                             Initialize and create subnets.json
  forge plan                             Plan with auto-allocated subnet
  forge apply                            Full deployment
  forge destroy                          Full teardown and release subnet
  forge status -json                     Cluster-wide JSON view
  forge doctor                           Preflight environment checks
  forge subnets list                     Show every allocation
  forge migrate CIG-Lab micro-01 --source micro-05
                                         Drain micro-05 onto micro-01
  forge networks prune teamA- --dry-run  Show what would be deleted
`
	fmt.Print(help)
	if plugins, err := forge.ListPlugins(); err == nil && len(plugins) > 0 {
		fmt.Println("\nInstalled plugins:")
		for _, p := range plugins {
			fmt.Printf("  %-22s %s\n", p.Name, p.Path)
		}
	}
}

func printError(msg string) {
	fmt.Fprintf(os.Stderr, "\033[31m[ERROR]\033[0m %s\n", msg)
}

// normalizeHelpTopic turns the args after `forge help` into a single
// canonical topic name. Joins multi-word topics like `networks prune`
// into `networks-prune` so they match the help table key.
func normalizeHelpTopic(args []string) string {
	if len(args) == 0 {
		return ""
	}
	if len(args) == 1 {
		return args[0]
	}
	// Multi-word topics live in subCommandHelp under hyphenated keys.
	// Today only `networks prune` qualifies.
	if args[0] == "networks" && args[1] == "prune" {
		return "networks-prune"
	}
	return args[0]
}

func printInfo(msg string) {
	fmt.Printf("\033[32m[INFO]\033[0m %s\n", msg)
}

func printWarn(msg string) {
	fmt.Printf("\033[33m[WARN]\033[0m %s\n", msg)
}

func runInit(workDir string, args []string) int {
	if forge.CheckHelp(args) {
		return runPassthrough(workDir, "init", args)
	}

	printInfo("Initializing subnets file...")
	if err := forge.InitSubnetsFile(); err != nil {
		printError(err.Error())
		return 1
	}
	printInfo(fmt.Sprintf("Subnets file ready: %s", forge.SubnetsFile))

	printInfo("Running tofu init...")
	if err := forge.RunTofuPassthrough(workDir, "init", args); err != nil {
		return 1
	}

	return 0
}

func runValidate(workDir string, args []string) int {
	return runPassthrough(workDir, "validate", args)
}

func runPlan(workDir string, args []string) int {
	if forge.CheckHelp(args) {
		return runPassthrough(workDir, "plan", args)
	}

	projectName, subnetOctet, err := getProjectAndSubnet(workDir, false)
	if err != nil {
		printError(err.Error())
		return 1
	}

	printInfo(fmt.Sprintf("Project: %s", projectName))
	printInfo(fmt.Sprintf("Subnet: %s (gateway: %s)", forge.FormatSubnet(subnetOctet), forge.FormatGateway(subnetOctet)))
	fmt.Println()

	if err := forge.RunTofu(workDir, "plan", args, projectName, subnetOctet); err != nil {
		return 1
	}

	return 0
}

func runApply(workDir string, args []string) int {
	if forge.CheckHelp(args) {
		return runPassthrough(workDir, "apply", args)
	}

	projectName, subnetOctet, err := getProjectAndSubnet(workDir, true)
	if err != nil {
		printError(err.Error())
		return 1
	}

	fmt.Println("==========================================")
	fmt.Println("  Cyber Range Deployment")
	fmt.Printf("  Project: %s\n", projectName)
	fmt.Println("==========================================")
	fmt.Println()

	printInfo(fmt.Sprintf("Project: %s", projectName))
	printInfo(fmt.Sprintf("Subnet: %s (gateway: %s)", forge.FormatSubnet(subnetOctet), forge.FormatGateway(subnetOctet)))
	fmt.Println()

	if err := forge.RunTofu(workDir, "apply", args, projectName, subnetOctet); err != nil {
		return 1
	}

	config := forge.DefaultDeployConfig()
	if err := forge.RunPostApply(workDir, projectName, config); err != nil {
		printWarn(err.Error())
	}

	forge.PrintDeploymentComplete(config, subnetOctet)

	return 0
}

func runDestroy(workDir string, args []string) int {
	if forge.CheckHelp(args) {
		return runPassthrough(workDir, "destroy", args)
	}

	projectName, err := forge.ParseProjectName(workDir)
	if err != nil {
		printError(err.Error())
		return 1
	}

	subnetOctet, err := forge.GetProjectSubnet(projectName)
	if err != nil {
		printError(err.Error())
		return 1
	}

	if subnetOctet == 0 {
		printError(fmt.Sprintf("No subnet allocation found for project '%s'", projectName))
		return 1
	}

	fmt.Println("==========================================")
	fmt.Println("  Cyber Range Destroy")
	fmt.Printf("  Project: %s\n", projectName)
	fmt.Println("==========================================")
	fmt.Println()

	printInfo(fmt.Sprintf("Project: %s", projectName))
	printInfo(fmt.Sprintf("Subnet: %s (will be released after destroy)", forge.FormatSubnet(subnetOctet)))
	fmt.Println()

	forge.RunPreDestroy()

	if err := forge.RunTofu(workDir, "destroy", args, projectName, subnetOctet); err != nil {
		return 1
	}

	printInfo("Releasing subnet allocation...")
	releasedOctet, err := forge.ReleaseSubnet(projectName)
	if err != nil {
		printWarn(fmt.Sprintf("Failed to release subnet: %s", err.Error()))
	} else {
		printInfo(fmt.Sprintf("Released subnet %s", forge.FormatSubnet(releasedOctet)))
	}

	fmt.Println()
	printInfo("Destroy complete!")
	printInfo(fmt.Sprintf("Subnet 10.0.%d.0/24 has been released and is available for reuse.", releasedOctet))

	return 0
}

// runServe starts the in-process HTTP configuration server. Defaults are
// inherited from DefaultDeployConfig (the same source `forge apply` uses),
// and the instances file defaults to ./instances.json in the current working
// directory. New: --log-format=text|json toggles structured server logs.
func runServe(args []string) int {
	deployCfg := forge.DefaultDeployConfig()

	defaultListen := fmt.Sprintf("%s:%s", deployCfg.ServerIP, deployCfg.ServerPort)

	defaultInstances := deployCfg.InstancesFile
	if cwd, err := os.Getwd(); err == nil {
		defaultInstances = filepath.Join(cwd, deployCfg.InstancesFile)
	}

	defaultIdle := 15 * time.Minute
	if deployCfg.IdleTimeout != "" {
		if d, err := time.ParseDuration(deployCfg.IdleTimeout); err == nil {
			defaultIdle = d
		}
	}

	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	listenAddr := fs.String("listen", defaultListen, "Listen address, e.g. 10.0.14.6:8080")
	instancesFile := fs.String("instances", defaultInstances, "Path to instances JSON file")
	idleTimeout := fs.Duration("idle-timeout", defaultIdle, "Shutdown after this duration of inactivity (0 disables)")
	logFormat := fs.String("log-format", "text", "Log format: text or json")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if err := forge.RunServer(forge.ServeOptions{
		Listen:        *listenAddr,
		InstancesFile: *instancesFile,
		IdleTimeout:   *idleTimeout,
		LogFormat:     *logFormat,
	}); err != nil {
		printError(err.Error())
		return 1
	}
	return 0
}

// runStatus shows subnet allocation status. If invoked from a project
// directory it highlights that project; otherwise it falls back to a
// cluster-wide view of every allocation in subnets.json.
func runStatus(workDir string, jsonOut bool) int {
	projectName, projectErr := forge.ParseProjectName(workDir)

	switch {
	case projectErr == nil:
		return runStatusInProject(workDir, projectName, jsonOut)
	default:
		return runStatusGlobal(workDir, projectErr, jsonOut)
	}
}

func runStatusInProject(workDir, projectName string, jsonOut bool) int {
	subnetOctet, err := forge.GetProjectSubnet(projectName)
	if err != nil {
		printError(err.Error())
		return 1
	}

	allocations, _ := forge.GetAllAllocations()

	if jsonOut {
		_ = forge.PrintJSON(map[string]interface{}{
			"project":      projectName,
			"work_dir":     workDir,
			"subnet_octet": subnetOctet,
			"subnet":       forge.FormatSubnet(subnetOctet),
			"gateway":      forge.FormatGateway(subnetOctet),
			"subnets_file": forge.SubnetsFile,
			"allocations":  allocations,
		})
		return 0
	}

	fmt.Printf("Project:  %s\n", projectName)
	fmt.Printf("Work Dir: %s\n", workDir)
	fmt.Println()

	if subnetOctet > 0 {
		fmt.Printf("Subnet:   %s\n", forge.FormatSubnet(subnetOctet))
		fmt.Printf("Gateway:  %s\n", forge.FormatGateway(subnetOctet))
		fmt.Printf("Octet:    %d\n", subnetOctet)
	} else {
		fmt.Println("Status:   No subnet allocated (run 'forge apply' to allocate)")
	}

	fmt.Println()
	fmt.Printf("Subnets file: %s\n", forge.SubnetsFile)

	if len(allocations) > 0 {
		fmt.Println()
		fmt.Println("All allocations:")
		for _, a := range allocations {
			marker := "  "
			if a.Project == projectName {
				marker = "* "
			}
			fmt.Printf("%s%-30s  10.0.%d.0/24\n", marker, a.Project, a.SubnetOctet)
		}
	}

	return 0
}

func runStatusGlobal(workDir string, projectErr error, jsonOut bool) int {
	allocations, err := forge.GetAllAllocations()

	if jsonOut {
		out := map[string]interface{}{
			"work_dir":     workDir,
			"project":      nil,
			"reason":       projectErr.Error(),
			"subnets_file": forge.SubnetsFile,
		}
		if err != nil {
			out["error"] = err.Error()
		} else {
			out["allocations"] = allocations
		}
		_ = forge.PrintJSON(out)
		return 0
	}

	fmt.Printf("Work Dir: %s\n", workDir)
	fmt.Println("Project:  (not a forge project directory)")
	fmt.Printf("Reason:   %s\n", projectErr.Error())
	fmt.Println()
	fmt.Printf("Subnets file: %s\n", forge.SubnetsFile)

	if err != nil {
		fmt.Println()
		printWarn(fmt.Sprintf("Could not read allocations: %s", err.Error()))
		return 0
	}

	if len(allocations) == 0 {
		fmt.Println()
		fmt.Println("No subnets currently allocated.")
		return 0
	}

	fmt.Println()
	fmt.Printf("Cluster-wide allocations (%d):\n", len(allocations))
	for _, a := range allocations {
		fmt.Printf("  %-30s  10.0.%d.0/24\n", a.Project, a.SubnetOctet)
	}

	return 0
}

func runDoctor(jsonOut bool) int {
	report := forge.RunDoctor()
	if jsonOut {
		if err := forge.PrintJSON(report); err != nil {
			printError(err.Error())
			return 1
		}
	} else {
		forge.PrintDoctorText(report)
	}
	if !report.Healthy {
		return 1
	}
	return 0
}

func runConfig(jsonOut bool) int {
	resolved := forge.LoadResolvedConfig()
	if jsonOut {
		if err := forge.PrintJSON(resolved); err != nil {
			printError(err.Error())
			return 1
		}
		return 0
	}
	forge.PrintConfigText(resolved)
	return 0
}

func runLogs(workDir string, args []string) int {
	fs := flag.NewFlagSet("logs", flag.ContinueOnError)
	follow := fs.Bool("f", false, "Follow log output (poll every 500ms)")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if err := forge.RunLogs(workDir, *follow); err != nil {
		printError(err.Error())
		return 1
	}
	return 0
}

func runReload() int {
	if err := forge.RunReload(); err != nil {
		printError(err.Error())
		return 1
	}
	return 0
}

// runSubnets dispatches the subnets sub-tree (list/free/reserve).
func runSubnets(args []string, gf globalFlags) int {
	if len(args) == 0 {
		printError("subnets requires a subcommand: list | free | reserve")
		return 1
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		if err := forge.SubnetsList(gf.jsonOut); err != nil {
			printError(err.Error())
			return 1
		}
		return 0
	case "free":
		if len(rest) < 1 {
			printError("usage: forge subnets free <project>")
			return 1
		}
		if err := forge.SubnetsFree(rest[0], gf.autoYes); err != nil {
			printError(err.Error())
			return 1
		}
		return 0
	case "reserve":
		if len(rest) < 2 {
			printError("usage: forge subnets reserve <project> <octet>")
			return 1
		}
		octet, err := forge.StringToInt(rest[1])
		if err != nil {
			printError(fmt.Sprintf("invalid octet %q: %s", rest[1], err.Error()))
			return 1
		}
		if err := forge.SubnetsReserve(rest[0], octet, gf.autoYes); err != nil {
			printError(err.Error())
			return 1
		}
		return 0
	default:
		printError(fmt.Sprintf("unknown subnets subcommand: %s", sub))
		return 1
	}
}

func runImport(args []string, autoYes bool) int {
	if len(args) < 1 {
		printError("usage: forge import <project>")
		return 1
	}
	if err := forge.RunImport(args[0], autoYes); err != nil {
		printError(err.Error())
		return 1
	}
	return 0
}

// dispatchPluginOrUnknown handles the `default` arm of the command
// switch. It looks for a plugin matching the typed command (with one
// level of two-word lookahead so `forge networks prune` resolves to
// forge-networks-prune), prints a helpful error if a known-moved built-in
// is missing, and falls back to the standard unknown-command help.
func dispatchPluginOrUnknown(workDir, command string, args []string, gf globalFlags) int {
	pluginEnv := newPluginEnv(workDir, gf)

	if len(args) >= 1 {
		twoWord := command + "-" + args[0]
		if path, ok := forge.FindPlugin(twoWord); ok {
			code, err := forge.RunPlugin(path, args[1:], pluginEnv)
			if err != nil {
				printError(err.Error())
			}
			return code
		}
	}
	if path, ok := forge.FindPlugin(command); ok {
		code, err := forge.RunPlugin(path, args, pluginEnv)
		if err != nil {
			printError(err.Error())
		}
		return code
	}

	if msg, ok := legacyMovedNotice(command, args); ok {
		printError(msg)
		return 1
	}

	printError(fmt.Sprintf("Unknown command: %s", command))
	printHelp()
	return 1
}

// newPluginEnv assembles the env passed into every plugin invocation.
// FORGE_PROJECT is filled in opportunistically: a missing main.tf is not
// an error here because plenty of plugins won't care about it.
func newPluginEnv(workDir string, gf globalFlags) forge.PluginEnv {
	env := forge.PluginEnv{
		WorkDir:     workDir,
		SubnetsFile: forge.SubnetsFile,
		AutoYes:     gf.autoYes,
		JSON:        gf.jsonOut,
		Version:     version,
	}
	if name, err := forge.ParseProjectName(workDir); err == nil {
		env.Project = name
	}
	if path := os.Getenv("FORGE_CONFIG"); path != "" {
		env.ConfigPath = path
	}
	return env
}

// legacyMovedNotice returns a friendly message when the user types one of
// the now-pluginized command names but the corresponding forge-* binary
// isn't installed yet. Returns ("", false) otherwise.
func legacyMovedNotice(command string, args []string) (string, bool) {
	movedSingles := map[string]string{
		"snapshot": "forge-snapshot",
		"start":    "forge-start",
		"stop":     "forge-stop",
		"migrate":  "forge-migrate",
		"cost":     "forge-cost",
	}
	if bin, ok := movedSingles[command]; ok {
		return fmt.Sprintf("'%s' is now a plugin. Install %s on $PATH (run `forge/scripts/build_all.sh` or copy from your forge_bin dir).", command, bin), true
	}
	if command == "networks" {
		sub := ""
		if len(args) > 0 {
			sub = args[0]
		}
		if sub == "prune" {
			return "'networks prune' is now a plugin. Install forge-networks-prune on $PATH (run `forge/scripts/build_all.sh`).", true
		}
		return "'networks' moved to plugins. Currently the only subcommand is `prune` -> forge-networks-prune.", true
	}
	return "", false
}

// runPlugins implements `forge plugins list`. JSON output is supported so
// other tools (CI, doctors) can enumerate what's installed.
func runPlugins(args []string, jsonOut bool) int {
	if len(args) == 0 || args[0] == "list" {
		plugins, err := forge.ListPlugins()
		if err != nil {
			printError(err.Error())
			return 1
		}
		if jsonOut {
			if err := forge.PrintJSON(plugins); err != nil {
				printError(err.Error())
				return 1
			}
			return 0
		}
		if len(plugins) == 0 {
			fmt.Println("No forge-* plugins found on $PATH.")
			fmt.Println("Build the bundled ones with `forge/scripts/build_all.sh`.")
			return 0
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tPATH")
		for _, p := range plugins {
			fmt.Fprintf(w, "%s\t%s\n", p.Name, p.Path)
		}
		_ = w.Flush()
		return 0
	}
	printError(fmt.Sprintf("unknown plugins subcommand: %s", args[0]))
	return 1
}

// runNew scaffolds a new project from a template. Flags are conservative:
// when --template or --name are missing AND stdin is a TTY, an interactive
// picker prompts for them; otherwise we error out so scripted use never
// silently hangs waiting for input.
func runNew(args []string, autoYes bool) int {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	template := fs.String("template", "", "Template id (directory name under templates/, or `blank`)")
	name := fs.String("name", "", "Project name (letters, digits, '-' or '_'; e.g. CSC-4100-Test)")
	dir := fs.String("dir", "", "Target directory (default: ./<name>)")
	reserve := fs.Bool("reserve", true, "Allocate a subnet octet up front")
	noReserve := fs.Bool("no-reserve", false, "Skip subnet allocation")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *noReserve {
		*reserve = false
	}

	interactive := !autoYes && isTTY(os.Stdin)

	if *template == "" {
		if !interactive {
			printError("--template is required (use `forge new` interactively to pick from a list)")
			return 1
		}
		tpls, err := forge.LoadTemplates()
		if err != nil {
			printError(err.Error())
			return 1
		}
		picked, err := forge.PromptForTemplate(tpls)
		if err != nil {
			printError(err.Error())
			return 1
		}
		*template = picked
	}

	if *name == "" {
		if !interactive {
			printError("--name is required")
			return 1
		}
		picked, err := forge.PromptForProjectName()
		if err != nil {
			printError(err.Error())
			return 1
		}
		*name = picked
	}

	if err := forge.RunNew(forge.NewOptions{
		Template:    *template,
		Name:        *name,
		TargetDir:   *dir,
		Reserve:     *reserve,
		AutoYes:     autoYes,
		Interactive: interactive,
	}); err != nil {
		printError(err.Error())
		return 1
	}
	return 0
}

// isTTY returns true when the file descriptor is attached to an
// interactive terminal. Used to decide whether `forge new` (and any
// future picker) can safely prompt for missing fields.
func isTTY(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// `forge cost` is no longer a built-in. It ships as the forge-cost
// plugin; the rendering logic lives in
// internal/forge/usagerender.go so each shim can drive it. Routing
// happens via dispatchPluginOrUnknown.

func runPassthrough(workDir string, command string, args []string) int {
	if err := forge.RunTofuPassthrough(workDir, command, args); err != nil {
		return 1
	}
	return 0
}

func getProjectAndSubnet(workDir string, allocate bool) (string, int, error) {
	projectName, err := forge.ParseProjectName(workDir)
	if err != nil {
		return "", 0, err
	}

	var subnetOctet int
	if allocate {
		subnetOctet, err = forge.AllocateSubnet(projectName)
	} else {
		subnetOctet, err = forge.AllocateSubnet(projectName)
	}

	if err != nil {
		return "", 0, err
	}

	return projectName, subnetOctet, nil
}

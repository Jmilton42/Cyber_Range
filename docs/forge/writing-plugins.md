# Writing Forge Plugins

Forge uses a **kubectl-style plugin system**: any executable on `$PATH`
named `forge-<name>` becomes available as `forge <name>`. There is no
SDK, no central registry, and no plugin manifest — drop a binary in a
`$PATH` directory and it works.

This page is the contract. If your plugin obeys it, `forge <name>`,
`forge plugins list`, and tab-completion all just work.

---

## Table of contents

- [Quick start](#quick-start)
- [Naming and discovery](#naming-and-discovery)
- [Environment variables](#environment-variables)
- [Argument and flag conventions](#argument-and-flag-conventions)
- [Exit codes](#exit-codes)
- [Output: stdout vs stderr, JSON mode](#output-stdout-vs-stderr-json-mode)
- [Confirmation prompts](#confirmation-prompts)
- [How completion works](#how-completion-works)
- [Versioning](#versioning)
- [Distribution](#distribution)
- [Reference: a complete Go plugin](#reference-a-complete-go-plugin)
- [Reference: a minimal bash plugin](#reference-a-minimal-bash-plugin)
- [Common mistakes](#common-mistakes)

---

## Quick start

A plugin in three lines of bash:

```bash
$ cat > ~/bin/forge-hello <<'EOF'
#!/usr/bin/env bash
echo "Hello from $(basename "$0")! FORGE_PROJECT=${FORGE_PROJECT:-<none>}"
EOF
$ chmod +x ~/bin/forge-hello
```

Make sure `~/bin` is on your `$PATH`, then:

```bash
$ forge hello
Hello from forge-hello! FORGE_PROJECT=<none>

$ forge plugins list | grep hello
forge-hello   /home/joeym/bin/forge-hello
```

That's the entire contract: forge found `forge-hello`, ran it, and
returned its exit code.

---

## Naming and discovery

| Rule | Detail |
|------|--------|
| Filename prefix | Must start with `forge-`. The bit after the prefix is the command name. |
| Location | Must be on `$PATH`. Forge does not search any other directories. |
| Permissions | Must be executable (`chmod +x`). Windows uses `PATHEXT` (`forge-foo.exe`, `forge-foo.bat`, etc.). |
| Shadowing | First match in `$PATH` wins; later matches are silently ignored. |
| Sub-namespacing | `forge-foo-bar` is matched **only** when the user types `forge foo bar`. Plain `forge foo` does **not** auto-route to `forge-foo-bar`. Pick one. |
| Reserved names | The built-in commands listed in the [main docs](../forge.md#command-index) cannot be overridden — forge dispatches to them before consulting `$PATH`. |

Any name that's not in the built-in list is fair game.

---

## Environment variables

When forge launches your plugin, it sets these env vars (and removes any
pre-existing values for the same keys):

| Variable | When set | Purpose |
|----------|----------|---------|
| `FORGE_VERSION` | always | The Forge version string. Useful for compat checks. |
| `FORGE_WORK_DIR` | always | Resolved working directory after `-chdir=...`. Use this instead of `cwd`. |
| `FORGE_PROJECT` | when forge could read `project_name` from `main.tf` in `FORGE_WORK_DIR` | The current project's name. Empty otherwise. |
| `FORGE_SUBNETS_FILE` | always | Absolute path to the cluster `subnets.json`. |
| `FORGE_CONFIG_PATH` | when a `config.yaml` was loaded | Path to the loaded config file. Empty if defaults are in use. |
| `FORGE_AUTO_YES` | when global `--yes` was passed | Set to `"1"` if the user opted into non-interactive mode. |
| `FORGE_JSON` | when global `--json` was passed | Set to `"1"` if the user wants JSON output. |

Plugins MUST treat all of these as advisory: be defensive. Don't crash
if `FORGE_PROJECT` is empty — interactively prompt or print a usage
hint.

The plugin inherits the rest of the user's environment as-is, including
`PATH`, `HOME`, `LXC_*`, etc.

---

## Argument and flag conventions

- Anything after the subcommand on the forge CLI is passed verbatim as
  `os.Args[1:]` to the plugin. So `forge mything --foo bar baz`
  invokes `forge-mything --foo bar baz`.
- Use the standard `--help` / `-h` flag to print usage. `forge mything -h`
  passes through.
- Don't try to re-parse forge's global flags (`-chdir`, `-yes`, `-json`,
  `-version`) — forge has already stripped them and given you the
  resolved value via env vars. If your plugin needs `--yes`, check
  `FORGE_AUTO_YES`.
- Be lenient: if the user types `forge mything help`, treat it like
  `--help`.

---

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success. |
| `1` | Generic failure. Print a single human-readable message to stderr. |
| `2` | Usage error (bad flag, missing required arg). Print usage to stderr. |
| `>2` | Plugin-specific. Document them. |

Forge propagates your exit code unchanged.

---

## Output: stdout vs stderr, JSON mode

| Stream | What goes there |
|--------|-----------------|
| `stdout` | Primary, scriptable output (table data, JSON, etc.). |
| `stderr` | Status / progress / errors. `[INFO]`, `[WARN]`, `[ERROR]` prefixes are a convention, not required. |

If `FORGE_JSON=1`, your plugin **should** emit machine-readable JSON
on stdout (one document, no banners, no progress). Don't crash if your
plugin doesn't have a sensible JSON form — just print a clear error to
stderr and exit 2.

```go
if os.Getenv("FORGE_JSON") == "1" {
    json.NewEncoder(os.Stdout).Encode(result)
    return 0
}
fmt.Fprintf(os.Stderr, "[INFO] Done: %d items\n", n)
```

---

## Confirmation prompts

When a plugin is destructive (deletes things, restarts services), it
should:

1. Print a one-line summary of what it's about to do to stderr.
2. Skip the prompt entirely if `FORGE_AUTO_YES=1`.
3. Otherwise, prompt on `/dev/tty` (not stdin — stdin may be piped).
4. Default to "no" on empty input.

The bundled `forge-stop` plugin is a fine reference here.

---

## How completion works

Forge ships dynamic bash and zsh completion. Top-level command
completion (`forge <TAB>`) is built from:

- The list of built-in subcommands.
- Every `forge-*` binary on `$PATH`, via `forge __complete commands`.

So **dropping a plugin onto `$PATH` is enough** to make it appear in
the top-level completion. No manifest update is needed.

If your plugin has its own argument structure (project names, node
names, snapshot ids, ...) and you want tab-completion for **its** args,
you have two options:

1. **Reuse forge's helpers.** Forge exposes a hidden completion helper
   command:

   ```bash
   forge __complete projects     # list project names from subnets.json
   forge __complete nodes        # list LXD cluster member names
   forge __complete instances <project>
   forge __complete templates    # list scaffolding templates
   ```

   Wire your plugin's completion into the bash/zsh script that loads
   `forge`'s completion. Easiest path: maintain your own
   `_forge_<name>_completion` function in the same `forge -completion=bash`
   output by patching it with your plugin's installer.

2. **Ship your own completion script.** Anything works; forge does not
   constrain you. For most plugins the first approach is enough because
   project/node/instance is what people want to autocomplete.

---

## Versioning

If your plugin needs to cooperate with a specific Forge release:

```go
forgeVer := os.Getenv("FORGE_VERSION")
if !semver.IsCompatible(forgeVer, "1.2") {
    fmt.Fprintf(os.Stderr, "forge-myplugin requires forge >= 1.2 (got %s)\n", forgeVer)
    os.Exit(2)
}
```

Forge itself does not enforce a version handshake. The plugin is in
charge of bailing out if something looks wrong.

---

## Distribution

Plugins are independent binaries. The CEROC platform team usually
ships a plugin via one of:

- **In-repo, like the bundled five.** Add a directory under
  `forge/cmd/forge-<name>/` with a `main.go`, then add a line to
  `forge/scripts/build_all.sh`. Operators get it for free on the next
  build.
- **External Go module.** `go install github.com/you/forge-foo@latest`
  drops a `forge-foo` binary into `$GOBIN`. As long as `$GOBIN` is on
  `$PATH`, forge picks it up.
- **Anything else.** Bash, Python, a stand-alone Rust binary — forge
  doesn't care. As long as the file is named `forge-<name>` and it's
  executable, it works.

---

## Reference: a complete Go plugin

A small Go plugin that lists running instances in the current project,
respects `FORGE_JSON` and `FORGE_AUTO_YES`, exits with the right codes,
and uses standard libs only:

```go
package main

import (
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
)

func main() { os.Exit(run()) }

func run() int {
    project := os.Getenv("FORGE_PROJECT")
    if project == "" {
        fmt.Fprintln(os.Stderr, "[ERROR] no FORGE_PROJECT set; cd into a project directory")
        return 2
    }

    out, err := exec.Command("lxc", "list", "--project", project, "-f", "json").Output()
    if err != nil {
        fmt.Fprintf(os.Stderr, "[ERROR] lxc list: %v\n", err)
        return 1
    }

    var instances []struct {
        Name   string `json:"name"`
        Status string `json:"status"`
    }
    if err := json.Unmarshal(out, &instances); err != nil {
        fmt.Fprintf(os.Stderr, "[ERROR] parse: %v\n", err)
        return 1
    }

    if os.Getenv("FORGE_JSON") == "1" {
        return jsonOutput(instances)
    }

    for _, i := range instances {
        if i.Status == "Running" {
            fmt.Println(i.Name)
        }
    }
    return 0
}

func jsonOutput(instances any) int {
    enc := json.NewEncoder(os.Stdout)
    enc.SetIndent("", "  ")
    if err := enc.Encode(map[string]any{"instances": instances}); err != nil {
        fmt.Fprintf(os.Stderr, "[ERROR] encode: %v\n", err)
        return 1
    }
    return 0
}
```

Build and install:

```bash
go build -o ~/bin/forge-running .
forge plugins list | grep running
forge running
```

---

## Reference: a minimal bash plugin

```bash
#!/usr/bin/env bash
# forge-cluster-summary: print one-line summary of every LXD cluster member.

set -euo pipefail

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    cat <<'EOF'
Usage: forge cluster-summary

Prints one line per cluster member with role and online status.
Honors FORGE_JSON=1.
EOF
    exit 0
fi

if [[ "${FORGE_JSON:-}" == "1" ]]; then
    lxc cluster list -f json
else
    lxc cluster list -f csv | awk -F, '{ printf "%-12s %-10s %s\n", $1, $3, $4 }'
fi
```

```bash
chmod +x forge-cluster-summary
mv forge-cluster-summary ~/bin/
forge cluster-summary
```

---

## Common mistakes

- **Forgetting to `chmod +x`.** `forge plugins list` will not show it.
- **Putting it in a directory that isn't on `$PATH`.** `forge doctor`
  doesn't check arbitrary plugins, only the bundled ones, so a missing
  custom plugin is silent. Verify with `forge plugins list`.
- **Re-parsing global flags.** Don't try to handle `-chdir` or `--yes`
  yourself. Forge already did, and it gave you the result via env vars.
- **Crashing on empty `FORGE_PROJECT`.** Plugins are runnable from
  anywhere. If your plugin needs a project, it should error cleanly,
  not panic.
- **Writing to `subnets.json` directly.** That file is forge's. Use
  `forge subnets reserve` / `forge subnets free` from inside your
  plugin if you must.
- **Naming clashes with future built-ins.** If you publish a plugin
  publicly, prefix it with your org name (`forge-acme-foo`) so we can
  add `forge foo` later without breaking you.

---

## Related

- [`forge plugins list`](./plugins.md) — see what's installed.
- [`forge doctor`](./doctor.md) — verifies the bundled plugins.
- [Main Forge reference](../forge.md) — the index of all commands.

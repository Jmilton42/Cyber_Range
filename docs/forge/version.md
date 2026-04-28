# forge version

Print the Forge build version.

## Synopsis

```
forge version
forge -version
forge --version
forge -v
```

All four spellings work and produce the same output.

## Why use it

You'd run this when:

- Filing a bug report (please include the version).
- Confirming `build_all.sh` actually rebuilt the binary.
- Checking that `INSTALL=1` did the right thing on the OpenTofu host
  before deploying.

## What it does

Prints the version string baked in at build time and exits 0. No
network calls, no file reads.

## Examples

```bash
$ forge version
forge 1.2.0

$ forge -v
forge 1.2.0
```

## Related

- [`forge doctor`](./doctor.md) — broader environment diagnostic.

## Notes

- The version is set at build time via `-ldflags "-X main.version=..."`.
  In a `go build` without that flag (e.g. a developer build), the
  string is `dev`.

---

# forge help

The bundled help text. Synopsis, command index, and global flag list.

## Synopsis

```
forge help
forge -help
forge --help
forge -h
```

## Why use it

`forge help` is the offline answer to "what commands does forge have?"
and is what shows up when you run `forge` with no arguments. The list of
plugin commands at the bottom is generated dynamically from `$PATH`, so
it stays correct as you install or uninstall plugins.

## Output

Looks roughly like:

```
Usage: forge [global flags] <command> [args]

Infrastructure:
  init        Initialize subnets file and run tofu init
  new         Scaffold a new project from a template
  validate    Pass through to tofu validate
  plan        Preview changes
  apply       Deploy
  destroy     Tear down

Diagnostics:
  status      Show subnet allocations
  doctor      Preflight checks
  config      Show effective config
  usage       Resource usage

Server:
  serve       Run the config server
  logs        View server.log
  reload      POST /reload

Subnets:
  subnets     list / free / reserve allocations
  import      Adopt an existing LXD project

Plugins:
  plugins     list installed forge-* binaries
  snapshot    (plugin) snapshot every instance in a project
  start       (plugin) start every instance
  stop        (plugin) stop every instance
  migrate     (plugin) move instances between cluster members
  networks    (plugin: networks-prune) delete orphan OVN nets

Run `forge <command> -h` for command-specific flags.
```

## Notes

- Per-command help (`forge plan -h`, `forge new -h`, etc.) shows the
  flags for that subcommand only.
- For full prose docs, see [the command index](../forge.md#command-index).

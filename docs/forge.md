# Forge CLI

Forge is a wrapper around OpenTofu that automatically manages guac subnet
allocations for Cyber Range projects, runs the HTTP config server,
scaffolds new projects from templates, and provides a kubectl-style
plugin system for day-2 LXD operations.

This page is the entry point. Each command has its own dedicated page in
[`docs/forge/`](./forge/) — see the index below.

---

## Install

Build everything (core binary + the six bundled plugins) in one shot:

```bash
cd /path/to/Cyber_Range/forge
./scripts/build_all.sh
# binaries land in ./bin/: forge, forge-snapshot, forge-start,
# forge-stop, forge-migrate, forge-networks-prune,
# forge-cost
```

Install onto the OpenTofu host:

```bash
INSTALL=1 ./scripts/build_all.sh
# copies into /home/ceroc/InSPIRE/bin/forge_bin/
```

Or system-wide:

```bash
sudo install -m 0755 ./bin/forge ./bin/forge-* /usr/local/bin/
```

Every `forge-*` plugin must be on `$PATH` for `forge snapshot`,
`forge start`, `forge stop`, `forge migrate`, and `forge networks prune`
to work. `forge doctor` warns if any are missing.

---

## Command index

### Infrastructure — author and lifecycle a project

| Command | Purpose |
|---------|---------|
| [`forge init`](./forge/init.md) | Create `subnets.json` (if missing) and run `tofu init` |
| [`forge new`](./forge/new.md) | Scaffold a new project from a template in `/home/ceroc/InSPIRE/templates/` |
| [`forge validate`](./forge/validate.md) | Run `tofu validate` (passthrough) |
| [`forge plan`](./forge/plan.md) | Preview the changes `apply` would make |
| [`forge apply`](./forge/apply.md) | Full deployment: tofu apply + start config server + start Windows VMs |
| [`forge destroy`](./forge/destroy.md) | Full teardown: stop server + tofu destroy + release the subnet |

### Diagnostics — find out what's going on

| Command | Purpose |
|---------|---------|
| [`forge status`](./forge/status.md) | This project's subnet, or a cluster-wide allocation table |
| [`forge doctor`](./forge/doctor.md) | Preflight checks (tofu/lxc/jq, subnets.json, plugins, server) |
| [`forge config`](./forge/config.md) | Print the resolved deploy config and which `config.yaml` was loaded |

### Server — control the HTTP configuration server

| Command | Purpose |
|---------|---------|
| [`forge serve`](./forge/serve.md) | Run the HTTP config server (auto-discovers `instances.json`) |
| [`forge logs`](./forge/logs.md) | Show or `-f` tail `server.log` in the current project |
| [`forge reload`](./forge/reload.md) | POST `/reload` so the running server re-reads `instances.json` |

### Subnets — manage allocations

| Command | Purpose |
|---------|---------|
| [`forge subnets`](./forge/subnets.md) | `list` / `free` / `reserve` allocations in `subnets.json` |
| [`forge import`](./forge/import.md) | Register an existing LXD project (created outside forge) |

### Plugins — day-2 LXD operations

| Command | Purpose |
|---------|---------|
| [`forge plugins`](./forge/plugins.md) | List discovered `forge-*` binaries |
| [`forge snapshot`](./forge/snapshot.md) | Snapshot every instance in a project |
| [`forge start`](./forge/start.md) | Start every instance in a project |
| [`forge stop`](./forge/stop.md) | Force-stop every instance in a project |
| [`forge migrate`](./forge/migrate.md) | Move instances between cluster members |
| [`forge networks prune`](./forge/networks-prune.md) | Delete orphan OVN networks by name prefix |
| [`forge cost`](./forge/cost.md) | Per-instance vCPU / RAM / disk breakdown for one project |

### Meta

| Command | Purpose |
|---------|---------|
| [`forge version`](./forge/version.md) | Show the current Forge version |
| `forge help` / `forge -help` | Print the in-CLI help summary |

### Authoring plugins

If you want to extend forge with your own commands (Discord notifier,
grade exporter, custom integrations), see
[Writing Forge Plugins](./forge/writing-plugins.md).

---

## Global options

These flags work with any subcommand and are stripped before dispatch.

| Flag | Effect |
|------|--------|
| `-chdir=DIR` | Change to `DIR` before executing |
| `-help` / `--help` / `-h` | Show top-level or per-command help |
| `-version` / `--version` / `-v` | Show version |
| `-json` / `--json` | Emit JSON output (status, subnets list, config, doctor, plugins; forwarded to plugins via `FORGE_JSON`) |
| `-yes` / `--yes` / `-y` | Skip interactive confirmation prompts |
| `-completion=SH` | Print bash or zsh completion script and exit |

---

## Subnet scheme

Every forge project gets a unique `/24` inside `10.0.0.0/16`:

| Octet | Subnet | Gateway | Guac VM IPs |
|-------|--------|---------|-------------|
| 1 | 10.0.1.0/24 | 10.0.1.1 | 10.0.1.2+ |
| 2 | 10.0.2.0/24 | 10.0.2.1 | 10.0.2.2+ |
| ... | ... | ... | ... |
| 254 | 10.0.254.0/24 | 10.0.254.1 | 10.0.254.2+ |

Allocations are recorded in `/home/ceroc/InSPIRE/bin/guac_subnet/subnets.json`.
See [`forge subnets`](./forge/subnets.md) for management commands.

---

## Shell completion

```bash
# Bash (system-wide):
forge -completion=bash | sudo tee /etc/bash_completion.d/forge >/dev/null

# Zsh (per-user):
mkdir -p ~/.zsh/completions
forge -completion=zsh > ~/.zsh/completions/_forge
# then in ~/.zshrc:
fpath=(~/.zsh/completions $fpath); autoload -Uz compinit && compinit
```

The completion is dynamic: `forge destroy <TAB>` lists current
allocations, `forge migrate <project> <TAB>` lists cluster nodes,
`forge new --template=<TAB>` lists templates, and any `forge-*` plugin
on `$PATH` is auto-discovered. See
[Writing Forge Plugins](./forge/writing-plugins.md#how-completion-works)
for how to make your own plugin tab-completable.

---

## Configuration

Default deploy settings (overridable via `config.yaml`):

| Setting | Default | Description |
|---------|---------|-------------|
| Server IP | `10.0.14.6` | Config server listen address |
| Server Port | `8080` | Config server port |
| Idle Timeout | `5m` | Server auto-shutdown after inactivity |
| Instances File | `instances.json` | LXD instance export (created in project dir) |
| Start Win Script | `/home/ceroc/InSPIRE/bin/scripts/start_win.sh` | Windows VM start script |
| Subnets File | `/home/ceroc/InSPIRE/bin/guac_subnet/subnets.json` | Cluster-wide subnet allocations |
| Templates Dir | `/home/ceroc/InSPIRE/templates` | Where `forge new` discovers templates (every subdirectory with a `main.tf` becomes a template) |

Run [`forge config`](./forge/config.md) to see what's actually in effect
on your machine, including which `config.yaml` was loaded.

---

## Troubleshooting

### "could not find project_name variable"
Add a `project_name` variable to your `main.tf`:

```hcl
variable "project_name" {
  type    = string
  default = "your-project-name"
}
```

### "no subnet allocation found"
Run [`forge apply`](./forge/apply.md) first to allocate a subnet before
running [`forge destroy`](./forge/destroy.md).

### "no available subnet octets"
All 254 subnets are allocated. Run `forge destroy` on unused projects, or
use [`forge subnets free <project>`](./forge/subnets.md) to release one
manually.

### Permission denied on subnets.json

```bash
mkdir -p /home/ceroc/InSPIRE/bin/guac_subnet
sudo chown $USER /home/ceroc/InSPIRE/bin/guac_subnet
```

### "'snapshot' is now a plugin"
The day-2 LXD commands (snapshot/start/stop/migrate/networks prune)
ship as separate `forge-*` binaries since v1.2. Build and install them:

```bash
cd /path/to/Cyber_Range/forge
INSTALL=1 ./scripts/build_all.sh
```

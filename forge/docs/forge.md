# Forge CLI

Forge is a wrapper around OpenTofu that automatically manages guac subnet allocations for Cyber Range projects.

## Features

- **Automatic subnet allocation** - Each project gets a unique `/24` subnet (10.0.1.0/16, 10.0.2.0/16, etc.)
- **Transparent tofu wrapper** - All tofu flags work with forge (`-auto-approve`, `-parallelism`, etc.)
- **Project detection** - Automatically reads `project_name` from `main.tf`
- **Central allocation tracking** - All projects share `/home/ceroc/InSPIRE/bin/guac_subnet/subnets.json`

## Installation

### Build from source

```bash
cd /path/to/Cyber_Range
go build -o ./bin/forge ./cmd/forge
```

### Install to system path

```bash
# Build forge (a single binary that also runs the HTTP config server
# via `forge serve`, invoked automatically by `forge apply`)
go build -o ./bin/forge ./cmd/forge

# Move binary to InSPIRE bin
mv ./bin/forge /home/ceroc/InSPIRE/bin/forge_bin/

# Or install to system path
sudo mv ./bin/forge /usr/local/bin/
```

## Setup

1. **Build the binary:**
   ```bash
   go build -o ./bin/forge ./cmd/forge
   ```

2. **Initialize in your first project:**
   ```bash
   cd /home/ceroc/InSPIRE/CIG/OCIG/Win-lin
   forge init
   ```
   This creates:
   - `/home/ceroc/InSPIRE/bin/guac_subnet/subnets.json` (if it doesn't exist)
   - Runs `tofu init`

3. **Ensure your `main.tf` has a `project_name` variable:**
   ```hcl
   variable "project_name" {
     type    = string
     default = "my-project-name"
   }
   
   variable "guac_subnet_octet" {
     type        = number
     default     = 1
     description = "Third octet for guac subnet"
   }
   ```

## Usage

### Basic Commands

```bash
# Initialize (creates subnets.json + tofu init)
forge init

# Plan infrastructure
forge plan

# Apply infrastructure (allocates subnet automatically)
forge apply

# Apply without confirmation
forge apply -auto-approve

# Destroy infrastructure (releases subnet)
forge destroy

# Check current status
forge status
```

### Example (from this repo)

This repository includes an example OpenTofu configuration at `infra/tofu/main.tf`. You can run Forge against it like this:

```bash
./bin/forge -chdir=infra/tofu init
./bin/forge -chdir=infra/tofu plan
```

### Help

```bash
# Main help
forge -help
forge help

# Command-specific help (passes through to tofu)
forge apply -help
forge plan -help
forge destroy -help
```

### Global Options

```bash
# Change directory before executing
forge -chdir=/path/to/project apply

# Show version
forge -version
```

## How It Works

### Full Deployment (`forge apply`)

When you run `forge apply`:

1. Reads `project_name` from `main.tf` in the current directory
2. Allocates next available subnet octet (1, 2, 3, ..., 254)
3. Saves allocation to `subnets.json`
4. Runs `tofu apply -var project_name=X -var guac_subnet_octet=Y`
5. Waits for VMs to initialize (10 seconds)
6. Exports LXD instances to `instances.json`
7. Starts the config server
8. Starts Windows VMs (via `/home/ceroc/InSPIRE/bin/scripts/start_win.sh`)

### Full Teardown (`forge destroy`)

When you run `forge destroy`:

1. Stops the config server
2. Runs `tofu destroy -var project_name=X -var guac_subnet_octet=Y`
3. Removes allocation from `subnets.json`
4. The octet becomes available for future projects

### Example `subnets.json`

```json
{
  "allocations": [
    {
      "project": "ocig-win-lin",
      "subnet_octet": 1,
      "allocated_at": "2026-01-12T10:30:00-05:00"
    },
    {
      "project": "csc-3410-lab",
      "subnet_octet": 2,
      "allocated_at": "2026-01-12T11:00:00-05:00"
    },
    {
      "project": "security-workshop",
      "subnet_octet": 3,
      "allocated_at": "2026-01-13T09:15:00-05:00"
    }
  ]
}
```

## Subnet Scheme

Each project gets a `/24` subnet within the `10.0.0.0/16` network:

| Octet | Subnet       | Gateway    | Guac VM IPs |
|-------|--------------|------------|-------------|
| 1     | 10.0.1.0/24  | 10.0.1.1   | 10.0.1.2+   |
| 2     | 10.0.2.0/24  | 10.0.2.1   | 10.0.2.2+   |
| 3     | 10.0.3.0/24  | 10.0.3.1   | 10.0.3.2+   |
| ...   | ...          | ...        | ...         |
| 254   | 10.0.254.0/24| 10.0.254.1 | 10.0.254.2+ |

## Commands Reference

### Infrastructure

| Command | Description |
|---------|-------------|
| `forge init` | Create subnets.json (if missing) and run `tofu init` |
| `forge validate` | Run `tofu validate` (passthrough) |
| `forge plan` | Allocate subnet and run `tofu plan -var ...` |
| `forge apply` | Full deployment: tofu apply + export instances + start server + start Windows |
| `forge destroy` | Full teardown: stop server + tofu destroy + release subnet |

### Diagnostics

| Command | Description |
|---------|-------------|
| `forge status` | In a project: show that project's subnet. Outside a project: cluster-wide allocation table. Supports `-json`. |
| `forge doctor` | Preflight checks: tofu / lxc / jq on `$PATH`, `subnets.json` valid JSON, `config.yaml` loadable, `start_win.sh` and lxd_scripts present, `lxc cluster list` parseable, config server `/status` reachable. Exits non-zero on any FAIL. Supports `-json`. |
| `forge config` | Print resolved deploy config plus the `config.yaml` path actually loaded. Supports `-json`. |

### Server control

| Command | Description |
|---------|-------------|
| `forge serve` | Restart the HTTP config server. Auto-detects `./instances.json` and the same listen address `forge apply` uses. New: `--log-format=json` emits one-line JSON log records. |
| `forge logs [-f]` | Show or tail `server.log` in the current project directory. `-f` polls every 500ms. |
| `forge reload` | POST `/reload` to the running config server so it re-reads `instances.json` without restarting. |

### Subnets

| Command | Description |
|---------|-------------|
| `forge subnets list` | Print every allocation in `subnets.json`. Supports `-json`. |
| `forge subnets free <project>` | Release a project's subnet. Confirms before writing; `-yes` to skip. |
| `forge subnets reserve <project> <octet>` | Hand-allocate a specific octet. Confirms before writing; `-yes` to skip. |
| `forge import <project>` | Register an existing LXD project by inferring its octet from the OVN network's `ipv4.address`. Confirms before writing. |

### LXD operations (script wrappers)

| Command | Description |
|---------|-------------|
| `forge snapshot <project>` | Wraps `snapshot.sh`. |
| `forge start <project>` | Wraps `start_vms.sh`. |
| `forge stop <project>` | Wraps `stop_vms.sh`. |
| `forge migrate <project> <target> [--source <node>]` | Wraps `move_vms.sh` (whole project) or `move_vms_nodes.sh` (drain a single source node). Prints affected instances and prompts before running. |
| `forge networks prune <prefix> [--project P] [--dry-run]` | Wraps `remove_networks.sh`. Deletes orphan OVN networks by prefix. |

### Other

| Command | Description |
|---------|-------------|
| `forge version` | Show version |
| `forge help` | Show help |

## Global options

| Flag | Effect |
|------|--------|
| `-chdir=DIR` | Change to `DIR` before executing |
| `-json` | Emit JSON output for `status`, `subnets list`, `config`, `doctor` |
| `-yes` / `-y` | Skip interactive confirmation prompts |
| `-completion=bash\|zsh` | Print a shell completion script to stdout and exit |
| `-help` | Show help |
| `-version` | Show version |

## Shell completion

```bash
# Bash (system-wide):
forge -completion=bash | sudo tee /etc/bash_completion.d/forge >/dev/null

# Zsh (per-user):
mkdir -p ~/.zsh/completions
forge -completion=zsh > ~/.zsh/completions/_forge
```

## Configuration

The following defaults are used during deployment:

| Setting | Default Value | Description |
|---------|---------------|-------------|
| Server IP | `10.0.14.6` | Config server listen address |
| Server Port | `8080` | Config server port |
| Idle Timeout | `5m` | Server auto-shutdown after inactivity |
| Instances File | `instances.json` | LXD instance export file (created in project dir) |
| Start Win Script | `/home/ceroc/InSPIRE/bin/scripts/start_win.sh` | Windows VM start script |

**Note:** The HTTP configuration server is part of the `forge` binary itself - `forge apply` launches it as a detached `forge serve` child process. There is no separate `server` binary. The Windows start script is expected to be at `/home/ceroc/InSPIRE/bin/scripts/start_win.sh`. Only the `instances.json` file is created in the project directory.

## Troubleshooting

### "could not find project_name variable"

Make sure your `main.tf` has:
```hcl
variable "project_name" {
  type    = string
  default = "your-project-name"
}
```

### "no subnet allocation found"

Run `forge apply` first to allocate a subnet before running `forge destroy`.

### "no available subnet octets"

All 254 subnets are allocated. Run `forge destroy` on unused projects to free up octets.

### Permission denied on subnets.json

Ensure the directory exists and you have write permissions:
```bash
mkdir -p /home/ceroc/InSPIRE/bin/guac_subnet
```

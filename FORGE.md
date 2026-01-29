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
go build -o forge ./cmd/forge
```

### Install to system path

```bash
# Build forge
go build -o forge ./cmd/forge

# Build server (if not already built)
go build -o server ./cmd/server

# Move binaries to InSPIRE bin
mv forge /home/ceroc/InSPIRE/bin/forge_bin/
mv server /home/ceroc/InSPIRE/bin/

# Or install to system path
sudo mv forge /usr/local/bin/
sudo mv server /usr/local/bin/
```

## Setup

1. **Build the binary:**
   ```bash
   go build -o forge ./cmd/forge
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

| Command | Description |
|---------|-------------|
| `forge init` | Create subnets.json (if missing) and run `tofu init` |
| `forge validate` | Run `tofu validate` (passthrough) |
| `forge plan` | Allocate subnet and run `tofu plan -var ...` |
| `forge apply` | Full deployment: tofu apply + export instances + start server + start Windows |
| `forge destroy` | Full teardown: stop server + tofu destroy + release subnet |
| `forge status` | Show current project's subnet allocation |
| `forge help` | Show help |
| `forge version` | Show version |

## Configuration

The following defaults are used during deployment:

| Setting | Default Value | Description |
|---------|---------------|-------------|
| Server Binary | `/home/ceroc/InSPIRE/bin/server` | Config server binary path |
| Server IP | `10.0.14.6` | Config server listen address |
| Server Port | `8080` | Config server port |
| Idle Timeout | `5m` | Server auto-shutdown after inactivity |
| Instances File | `instances.json` | LXD instance export file (created in project dir) |
| Start Win Script | `/home/ceroc/InSPIRE/bin/scripts/start_win.sh` | Windows VM start script |

**Note:** The server binary and scripts are expected to be in `/home/ceroc/InSPIRE/bin/`. Only the `instances.json` file is created in the project directory.

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

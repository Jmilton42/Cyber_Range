# Building Infrastructure

How to author, deploy, and tear down a range project. This page is the
overview — it points to the two project patterns and lists the decision
points you have to settle before you write any HCL.

> Prerequisite: read [Concepts & Terminology](./concepts.md) first if any
> of "OVN", "LXD project", or "guac subnet octet" are unfamiliar.

---

## At a glance

```text
   ┌──────────────────────┐    ┌──────────────────────┐    ┌──────────────────────┐
   │ 1. Decide pattern    │───▶│ 2. Author main.tf    │───▶│ 3. forge init        │
   │    (single / multi)  │    │  in project's dir    │    │    in that dir       │
   └──────────────────────┘    └──────────────────────┘    └──────────────────────┘
                                                                       │
   ┌──────────────────────┐    ┌──────────────────────┐    ┌──────────▼───────────┐
   │ 6. forge destroy     │◀───│ 5. Run the exercise  │◀───│ 4. forge plan        │
   │    when finished     │    │    (Guac, Salt, etc.)│    │    forge apply       │
   └──────────────────────┘    └──────────────────────┘    └──────────────────────┘
```

## Step 1 — Decide which pattern fits

Almost every request we deploy is one of two patterns. Pick before you start
writing HCL — the network topology and the per-team variable layout are
substantially different.

| Pattern | Topology | Use when… | Reference project | Detailed page |
|---------|----------|-----------|-------------------|---------------|
| **Single-network** | One shared `team-lan` for the whole project, plus the project firewall and salt LAN. | Classroom labs, intro courses, anything where teams legitimately share an L2 segment. Lowest network resource footprint. | `/home/ceroc/InSPIRE/Classes/CSC-3410-CS/` | [Single-Network Projects](./single-network.md) |
| **Multi-network**  | One `team${n}-lan` per team, isolated from each other. Optional extra segments (DMZ / business / sensitive) per team. | Any exercise where teams must be invisible to each other (CPTC, defensive labs, security competitions). Required for any "blue vs blue" scenario. | `/home/ceroc/InSPIRE/CIG/OCIG/Win-lin/`, `/home/ceroc/InSPIRE/CIG/DCIG/Lin-Lab/`, `/home/ceroc/InSPIRE/CIG/COMP/CPTC/` | [Multi-Network Projects](./multi-network.md) |

If you are unsure, default to **multi-network**. The cost is a few extra
OVN switches; the benefit is hard isolation between teams.

## Step 2 — Settle the project decision points

Write these down before you open `main.tf`. Every project starts by
choosing answers to the same four questions.

### 2a. `project_name`

Short, kebab-cased, unique on the cluster. This becomes:

- the LXD project name,
- the prefix on every network and instance name,
- the key in `subnets.json`.

```hcl
variable "project_name" {
  type    = string
  default = "CSC-3410"   # change me
}
```

> **Do not** rename a project after `forge apply`. Renaming forces
> recreation of every dependent resource. If you must rename, destroy
> first.

### 2b. `team_count`

How many parallel teams the project supports. Drives every `count = …`
in the file. Common values: `6` (CPTC mock), `35`–`65` (classroom).

```hcl
variable "team_count" {
  type    = number
  default = 45
}
```

You can grow `team_count` after the fact — `forge apply` will create the
new teams in place. Shrinking destroys the trailing teams; you generally
want to `forge destroy` instead.

### 2c. Cluster target

Which LXD cluster member do team instances live on? Set on every
instance via `target = "@Cluster-C"` (or `@default`, `@micro-06`, …).

The reference projects use:

| Project | Target |
|---------|--------|
| `Classes/CSC-3410-CS` | `@Cluster-C` |
| `CIG/OCIG/Win-lin`    | `@Cluster-C` |
| `CIG/DCIG/Lin-Lab`    | `@default`  |
| `CIG/COMP/CPTC`       | `@default`  |

Pick one and use it consistently per team.

### 2d. Per-team workload mix

What runs on each team's LAN? List the images and friendly names; the
reference patterns walk these as parallel lists indexed by `count.index`.

```hcl
variable "lan_image" {
  type    = list(string)
  default = ["guac-xfce4-v01", "windows-10-base"]
}

variable "lan_name" {
  type    = list(string)
  default = ["ubuntu", "win"]
}
```

Available pre-baked images on the cluster (current as of the latest
projects in this repo):

| Image | Type | Used as |
|-------|------|---------|
| `openwrt-project`     | container | Project-level firewall (north of all teams) |
| `openwrt-team-new`    | container | Per-team firewall |
| `salt-master-new`     | VM        | Salt master |
| `guac-xfce4-v01` / `v02` | VM     | Guacamole gateway / Linux LAN host |
| `windows-10-base`     | VM        | Windows 10 LAN host |
| `windows-2019-base`   | VM        | Windows Server 2019 LAN host |

## Step 3 — Lay out the project directory

Every project lives in its own directory under `/home/ceroc/InSPIRE/`,
filed under the track that owns it:

```text
/home/ceroc/InSPIRE/
├── Classes/                # course-based ranges
│   └── <class-name>/
│       └── main.tf
└── CIG/                    # CIG program ranges
    ├── COMP/               # competition track (e.g. CPTC)
    │   └── <project>/
    │       └── main.tf
    ├── DCIG/               # defensive track
    │   └── <project>/
    │       └── main.tf
    └── OCIG/               # offensive track
        └── <project>/
            └── main.tf
```

We deliberately keep one file per project. It is verbose, but every request
is reviewed, audited, and snapshotted as a single unit — splitting into
modules makes that harder. If you find yourself copy-pasting blocks
between projects, extract a module *only after* the second project ships.

## Step 4 — Standard `main.tf` skeleton

Every request follows the same skeleton. Match the order; reviewers expect
it.

```hcl
# 1. Variables
variable "project_name"      { ... }
variable "team_count"        { ... }
variable "guac_subnet_octet" { ... }   # filled in by Forge
# (plus any project-specific lan_image / lan_name lists)

# 2. Project
resource "lxd_project" "project" { ... }
data "lxd_project" "proj"        { ... }

# 3. Networks
resource "lxd_network" "team_wan"  { ... }
resource "lxd_network" "salt_lan"  { ... }
resource "lxd_network" "team_lan"  { ... }   # count = team_count for multi-network

# 4. Locals (network name lists, image lookups)
locals {
  windows_images = ["windows-10-base", "windows-2019-base"]
  project_fw_net = [...]
  guac_net       = [...]
}

# 5. Project-wide instances
resource "lxd_instance" "project_fw"        { ... }   # OpenWrt at the top
resource "lxd_instance" "project_salt"      { ... }   # Salt master
resource "lxd_instance" "project_guac_salt" { ... }   # Guac for staff

# 6. Per-team instances
resource "lxd_instance" "team_fw"   { count = var.team_count, ... }
resource "lxd_instance" "lan_linux" { count = var.team_count, ... }
resource "lxd_instance" "guac"      { count = var.team_count, ... }
```

Pick up the [single-network](./single-network.md) or
[multi-network](./multi-network.md) walkthrough for fully-worked
examples.

## Step 5 — `forge init`

In the project directory:

```bash
cd /home/ceroc/InSPIRE/<track>/<project-name>     # e.g. CIG/OCIG/Win-lin
forge init
```

This is a one-time step per working copy of the project. It runs `tofu
init` (downloads providers) and creates the shared `subnets.json` if it
does not exist yet. Re-run it any time you upgrade providers or move to a
new machine.

## Step 6 — `forge plan` (always)

```bash
forge plan
```

`plan` is read-only. It allocates a guac subnet octet for the project (so
the plan output is realistic) and prints the resource diff. **Always run
plan before apply** for any non-trivial change. Look for:

- `# resource "lxd_instance" "..." must be replaced` — anything being
  *replaced* will lose its data. Make sure that is intentional.
- Network `force-replace` triggers — usually mean a rename. Destroy
  first if you can.
- Counts changing (e.g. `+/- 12 to add, 12 to destroy`) — usually means
  a list variable was reordered. Reorder back before applying or every
  team gets renamed.

## Step 7 — `forge apply`

```bash
# Interactive, with confirmation
forge apply

# CI / scripted use
forge apply -auto-approve
```

`forge apply` does the following, in order:

1. Reads `project_name` from `main.tf`.
2. Allocates the next free `/24` and writes it to `subnets.json`.
3. Runs `tofu apply -var project_name=… -var guac_subnet_octet=…`.
4. Waits ~10 seconds for VMs to boot.
5. Exports the LXD instance list to `instances.json` in the project dir.
6. Starts the Forge config server (HTTP, idle-timeout 5 min).
7. Triggers the Windows VM start script.

If any step fails, the cluster state already created stays in place —
re-running `forge apply` is safe and will pick up where it left off.

### Common apply options

```bash
# Build a different working dir without cd'ing into it
forge -chdir=/home/ceroc/InSPIRE/<track>/<project-name> apply

# Limit parallelism (good for big projects on slow disks)
forge apply -parallelism=4

# Target a single resource (rare; use with care)
forge apply -target=lxd_instance.project_fw
```

All standard `tofu` flags pass through unchanged. See `forge apply -help`
for the full list.

## Step 8 — `forge destroy`

```bash
forge destroy            # asks for confirmation
forge destroy -auto-approve
```

`forge destroy`:

1. Stops the config server.
2. Runs `tofu destroy` with the same variables it applied with.
3. Removes the project from `subnets.json`, freeing the octet for reuse.

> **Always** prefer `forge destroy` over `lxc project delete` followed by
> a manual cleanup. `forge destroy` keeps the subnet registry honest;
> manual deletes leak octets that are eventually impossible to debug.

## Recovery & escape hatches

### Diagnostics first

Before you start poking at state, run the two read-only diagnostic
commands. They surface most "why is forge unhappy" problems without
touching anything:

```bash
forge doctor      # tofu, lxc, jq, subnets.json, config.yaml,
                  # start_win.sh, lxd_scripts, lxc cluster, /status
forge config      # show resolved deploy config + which config.yaml loaded
forge status      # this project's allocation, or cluster-wide if
                  # not in a project directory
```

`forge doctor` exits non-zero on any FAIL, so it is safe to use as a
gate in pre-deployment scripts (`forge doctor && forge apply`).

### Dropping down to raw `tofu`

You can always drop down to raw `tofu` in a project directory — Forge
just shells out to it. Useful when:

- Forge can't find `project_name` (you renamed the variable).
- You need to import an existing resource.
- A previous apply was interrupted in a state Forge doesn't recognize.

```bash
cd /home/ceroc/InSPIRE/<track>/<project-name>

tofu state list
tofu state show 'lxd_instance.team_fw[3]'
tofu import 'lxd_instance.team_fw[3]' /1.0/instances/...
```

If you take a manual action, **re-run `forge plan`** before the next
apply to confirm Forge and tofu agree on state.

### Importing a project that was built by hand

If a project was created outside forge (raw `tofu apply` or hand-rolled
`lxc` commands) and never registered in `subnets.json`, register it
without re-deploying:

```bash
forge import <project-name>
# Detects the OVN network's ipv4.address (10.0.X.1/24), confirms,
# writes X to subnets.json. Use 'forge subnets reserve' if the project
# has no detectable OVN network and you need to hand-pick an octet.
```

## Where to go next

- Single shared LAN → [Single-Network Projects](./single-network.md)
- Per-team isolation → [Multi-Network Projects](./multi-network.md)
- Operational tasks → [LXD Operations Guide](./lxd.md)
- Forge command reference → [Forge CLI](./forge.md)

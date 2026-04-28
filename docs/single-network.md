# Single-Network Projects

A **single-network** project gives every team one shared LAN. There is one
team firewall, one team LAN, and `team_count` Guac VMs all hanging off it.
This is the simplest pattern we ship; reach for it whenever teams are
*allowed* to see each other on the wire (classroom labs, intro courses,
shared CTFs).

> **Reference implementation:** `/home/ceroc/InSPIRE/Classes/CSC-3410-CS/main.tf` (65 teams)

If you need teams to be invisible to each other, use the
[multi-network pattern](./multi-network.md) instead.

---

## Topology

```text
                              ┌──────────────────────┐
                              │   GUAC_WAN (cluster) │
                              └──────────┬───────────┘
                                         │
                                  ┌──────▼──────┐
                                  │  Guac salt  │   project-wide
                                  │  (project)  │
                                  └──────┬──────┘
                                         │ salt-lan (172.31.31.0/24)
                                         │
                              ┌──────────▼──────────┐
                              │  project_fw (OWRT)  │  CLASS_WAN ◀──── upstream
                              │  ─ team-wan         │
                              │  ─ salt-lan         │
                              └──────────┬──────────┘
                                         │ team-wan (no IP)
                                         │
                              ┌──────────▼──────────┐
                              │   team_fw (OWRT)    │   ONE firewall for the project
                              │   ─ team-wan        │
                              │   ─ team-lan        │
                              └──────────┬──────────┘
                                         │ team-lan (192.168.x.0/24, shared)
              ┌──────────────────────────┼──────────────────────────┐
              │                          │                          │
       ┌──────▼──────┐            ┌──────▼──────┐            ┌──────▼──────┐
       │  team1 Guac │            │  team2 Guac │   …        │ teamN Guac  │
       │ 192.168.1.2 │            │ 192.168.1.3 │            │192.168.1.N+1│
       │ 10.0.X.3    │            │ 10.0.X.4    │            │ 10.0.X.N+2  │
       └─────────────┘            └─────────────┘            └─────────────┘
```

Note the shared `team_lan` and the *single* `team_fw` — that is what
makes this pattern "single network". Compare to the multi-network
diagram in [multi-network.md](./multi-network.md), which has one
`team_fw[i]` and one `team_lan[i]` per team.

## Address plan

| Network        | CIDR                                | Notes |
|----------------|-------------------------------------|-------|
| Class WAN      | `CLASS_WAN` (cluster uplink)        | Provides north-bound transit |
| Project salt-lan | `172.31.31.0/24`                  | Salt master `.2`, Guac salt `.3` |
| Team WAN       | layer-2 only, no IP                 | Between project FW and team FW |
| Team LAN       | `192.168.${i+1}.0/24` (per team)    | Each team's Guac picks `.${i+2}` |
| Guac WAN (`/16`) | `10.0.${guac_subnet_octet}.0/16`  | Guac VMs at `.3`, `.4`, …, `.N+2` |

`guac_subnet_octet` is allocated by Forge — never set by hand.

## File walkthrough

The `CSC-3410-CS` project is the canonical example. The annotations
below explain *why* each block exists; line numbers refer to
`/home/ceroc/InSPIRE/Classes/CSC-3410-CS/main.tf`.

### 1. Variables (lines 1–9)

```hcl
variable "project_name" {
  type    = string
  default = "CSC-3410"
}

variable "team_count" {
  type    = number
  default = 65
}
```

`project_name` is read by Forge. `team_count` drives the per-team
`count = …` later in the file.

### 2. LXD project (lines 14–29)

```hcl
resource "lxd_project" "proj" {
  name        = var.project_name
  description = "${var.project_name}"
  config = {
    "features.storage.volumes" = true
    "features.images"          = false
    "features.profiles"        = false
    "features.storage.buckets" = true
    "features.networks"        = false
  }
}

data "lxd_project" "proj" {
  name       = lxd_project.proj.name
  depends_on = [lxd_project.proj]
}
```

The `features.*` flags say "this project has its own storage volumes and
buckets, but uses cluster-wide images, profiles, and networks". This is
the standard config for every range — copy it.

The `data` lookup right after the resource is a deliberate trick: it
gives downstream resources a stable `data.lxd_project.proj.name`
reference that does not change if you rename the resource block in HCL.

### 3. Networks (lines 34–72)

Three networks, all OVN, all `ipv4.address = "none"`:

```hcl
resource "lxd_network" "salt_lan" { ... }
resource "lxd_network" "team_lan" { ... }   # SINGLE LAN — no count
resource "lxd_network" "team_wan" { ... }
```

The key thing that makes this a single-network project: `team_lan` has
no `count`. There is exactly one `team_lan` for the whole project, and
every team's Guac attaches to it.

### 4. Locals (lines 74–78)

```hcl
locals {
  project_fw_net = concat(["CLASS_WAN", lxd_network.team_wan.name, lxd_network.salt_lan.name])
  guac_net       = ["GUAC_WAN", lxd_network.salt_lan.name]
  guac_wan       = "GUAC_WAN"
}
```

These name-lists keep `dynamic "device"` blocks readable downstream.
The order matters — `eth0` gets the first, `eth1` the second, etc.

### 5. Project firewall (lines 83–103)

OpenWrt container with three NICs: cluster WAN, team-wan to the team
firewall, salt-lan to the salt master.

### 6. Salt + project Guac (lines 105–197)

A salt master VM and a project-wide Guac VM. The Guac picks up
`10.0.${var.guac_subnet_octet}.2/16` — that `.2` is reserved for the
project Guac.

### 7. Single team firewall (lines 204–236)

```hcl
resource "lxd_instance" "team_fw" {
  project = data.lxd_project.proj.name
  name    = "${var.project_name}-team-fw"
  image   = "openwrt-team-new"
  type    = "container"
  ...
  target  = "@Cluster-C"
}
```

**No `count`** — this is what makes it single-network. One firewall
fronts the shared `team_lan` for *all* teams.

### 8. Per-team Guac (lines 249–314)

```hcl
resource "lxd_instance" "guac" {
  count = var.team_count
  ...
  config = {
    "cloud-init.network-config" = <<-EOF
      version: 2
      ethernets:
        enp5s0:
          dhcp4: false
          addresses:
            - 10.0.${var.guac_subnet_octet}.${3 + count.index}/16
          ...
        enp6s0:
          dhcp4: false
          addresses:
            - 192.168.1.${2 + count.index}/24
          ...
    EOF
  }
}
```

Two interfaces per Guac VM:

- `enp5s0` is on `GUAC_WAN`, takes `10.0.X.${3 + count.index}` so each
  team's Guac is reachable from the staff Guac.
- `enp6s0` is on the shared `team_lan`, takes `192.168.1.${2 +
  count.index}` so each Guac is unique on the team LAN.

That `192.168.1.*` is shared by every team in the project — that is the
point of the single-network pattern.

## Build it

```bash
cd /home/ceroc/InSPIRE/Classes/CSC-3410-CS

forge init
forge plan          # always plan first
forge apply
```

Expected output (abridged):

```text
Allocated subnet octet 7 for project CSC-3410
...
Apply complete! Resources: 134 added, 0 changed, 0 destroyed.
Started config server on :8080 (idle-timeout 5m)
```

When the exercise is done:

```bash
forge destroy
```

## Common modifications

### Add more teams mid-project

```hcl
variable "team_count" {
  type    = number
  default = 80   # was 65
}
```

```bash
forge plan        # confirm only `+ lxd_instance.guac[65..79]` etc.
forge apply
```

The shared `team_fw` and `team_lan` are unchanged. Only the new Guac
instances are created.

### Add a Linux VM to the shared LAN

Inside the existing `lan_linux` block (or add one), keep `count =
var.team_count`, point its NIC at `lxd_network.team_lan.name` (no
`[count.index]`), and pick a non-conflicting `192.168.1.*` IP:

```hcl
device {
  name = "eth0"
  type = "nic"
  properties = { network = lxd_network.team_lan.name }
}

config = {
  "cloud-init.network-config" = <<-EOF
    version: 2
    ethernets:
      enp5s0:
        dhcp4: false
        addresses:
          - 192.168.1.${100 + count.index}/24
        routes:
          - to: default
            via: 192.168.1.1
    EOF
}
```

Pick a non-overlapping range — `.${100 + count.index}` works for up to
~150 teams without colliding with the existing `.${2 + count.index}`
Guac range.

## When the single-network pattern is the wrong choice

- Teams must run scans against each other → use multi-network.
- The exercise involves SMB, Kerberos, or any L2-broadcast-sensitive
  service that would leak between teams → use multi-network.
- You need different image sets per team → use multi-network with a
  per-team `lan_image[count.index]` indexing scheme.

For all of the above, see [Multi-Network Projects](./multi-network.md).

---

[← Back to docs index](./README.md)

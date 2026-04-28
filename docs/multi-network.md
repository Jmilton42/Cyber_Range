# Multi-Network Projects

A **multi-network** project gives every team its own isolated LAN (and
sometimes several). Teams cannot see each other on the wire — they can
only reach each other through the project firewall, if it is configured
to allow that. This is the pattern to use whenever isolation matters:
defensive labs, attack/defense competitions, anything where one team's
broadcast domain bleeding into another would invalidate the exercise.

> **Reference implementations:**
>
> - `/home/ceroc/InSPIRE/CIG/OCIG/Win-lin/` — Windows + Linux, one LAN per team (45 teams)
> - `/home/ceroc/InSPIRE/CIG/DCIG/Lin-Lab/` — Linux-only, three Ubuntu hosts per team (35 teams)
> - `/home/ceroc/InSPIRE/CIG/COMP/CPTC/`    — Three networks per team: DMZ, business, sensitive (6 teams, mock CPTC)

If teams are *allowed* to share a wire, use the simpler
[single-network pattern](./single-network.md) instead.

---

## Variants of the pattern

Multi-network is a family, not a single template. Pick the variant that
fits the exercise:

| Variant | Networks per team | Reference | Use when… |
|---------|-------------------|-----------|-----------|
| **Flat per-team** | 1 (`team-lan`)                              | `CIG/OCIG/Win-lin`, `CIG/DCIG/Lin-Lab` | Standard isolated team — Windows/Linux mix. |
| **Segmented per-team** | 3+ (`team-dmz` + `team-business` + `team-sensitive`) | `CIG/COMP/CPTC`         | Realistic enterprise simulations: pivoting between segments is part of the exercise. |

Everything below applies to both variants — the segmented variant just
declares more `lxd_network` resources and gives the team firewall extra
NICs.

## Topology — flat per-team (most common)

```text
                              ┌──────────────────────┐
                              │   GUAC_WAN (cluster) │
                              └──────────┬───────────┘
                                         │
                                  ┌──────▼──────┐
                                  │  Guac salt  │   project-wide
                                  └──────┬──────┘
                                         │ salt-lan (172.31.31.0/24)
                                         │
                              ┌──────────▼──────────┐
                              │  project_fw (OWRT)  │  DCIG_WAN ◀──── upstream
                              │  ─ team-wan         │
                              │  ─ salt-lan         │
                              └──────────┬──────────┘
                                         │ team-wan (shared, no IP)
        ┌────────────────────────────────┼────────────────────────────────┐
        │                                │                                │
 ┌──────▼────────┐                ┌──────▼────────┐                ┌──────▼────────┐
 │ team1_fw      │                │ team2_fw      │   …            │ teamN_fw      │
 │ ─ team-wan    │                │ ─ team-wan    │                │ ─ team-wan    │
 │ ─ team1-lan   │                │ ─ team2-lan   │                │ ─ teamN-lan   │
 └──────┬────────┘                └──────┬────────┘                └──────┬────────┘
        │ 192.168.1.0/24                 │ 192.168.2.0/24                 │ 192.168.N.0/24
        │ (team1 only)                   │ (team2 only)                   │ (teamN only)
   ┌────▼─────┐ ┌─────┐             ┌────▼─────┐ ┌─────┐             ┌────▼─────┐ ┌─────┐
   │team1 Guac│ │ ubu │             │team2 Guac│ │ ubu │             │teamN Guac│ │ ubu │
   │team1 win │ │ … │               │team2 win │ │ … │               │teamN win │ │ … │
   └──────────┘ └─────┘             └──────────┘ └─────┘             └──────────┘ └─────┘
```

Each team's `team-lan` is a separate OVN switch — instances on
`team1-lan` *cannot* see ARP/L2 from `team2-lan`. Inter-team traffic only
exists if `project_fw` is explicitly configured to forward it.

## Topology — segmented per-team (CPTC-style)

```text
        ┌───── teamN_fw ─────┐
        │                    │
     team-wan ───  eth0      │  (shared upstream)
        │                    │
        ├── team-dmz       eth1 ──── jump host, public-facing services
        ├── team-business  eth2 ──── workstations, internal apps
        └── team-sensitive eth3 ──── crown-jewel servers, isolated
```

Same idea, just more interfaces on `team_fw` and more `lxd_network`
resources with `count = var.team_count`.

## Address plan (flat variant)

| Network                | CIDR                                       | Notes |
|------------------------|--------------------------------------------|-------|
| Project upstream       | `DCIG_WAN` / `CLASS_WAN` (cluster uplink)  | Provides north-bound transit |
| Project salt-lan       | `172.31.31.0/24`                           | Salt master `.2`, project Guac `.3` |
| Team WAN               | layer-2 only, no IP                        | Trunk between project FW and team FWs |
| Team `i` LAN           | `192.168.${i+1}.0/24`                      | Team firewall `.1`, hosts `.2`, `.3`, … |
| Guac WAN (`/16`)       | `10.0.${guac_subnet_octet}.0/16`           | Per-team Guac at `.${3 + i}` (or `.${4 + i}`) |

`guac_subnet_octet` is allocated by Forge — never set by hand. Inside a
project, every team's Guac shares one `/24` octet inside the global
`/16`.

## File walkthrough — flat variant (`CIG/OCIG/Win-lin`)

The annotations below explain the parts that *change* compared to the
single-network walkthrough. The rest (project block, salt master,
project Guac) is identical.

### 1. Per-team networks (lines 35–61)

```hcl
resource "lxd_network" "team_lan" {
  count    = var.team_count                                      # ◀── one per team
  project  = data.lxd_project.proj.name
  name     = "${var.project_name}-team${count.index + 1}-lan"     # team1-lan, team2-lan, …
  type     = "ovn"
  config = {
    "bridge.mtu"   = "1500"
    "ipv4.address" = "none"
    "network"      = "internal_link5"
  }
}

resource "lxd_network" "team_wan" {                              # one shared trunk
  project = data.lxd_project.proj.name
  name    = "${var.project_name}-team-wan"
  type    = "ovn"
  config  = { ... }
}
```

`team_lan` has `count = var.team_count` — that is what creates the
isolation. `team_wan` does not — it is the shared trunk that the project
firewall and every team firewall hang off of.

### 2. Locals (lines 75–84)

```hcl
locals {
  windows_images = ["windows-10-base", "windows-2019-base"]

  shared_wan      = lxd_network.team_wan.name
  team_lan_names  = [for n in lxd_network.team_lan : n.name]   # ◀── ordered list

  guac_net = ["GUAC_WAN", lxd_network.salt_lan.name]
  guac_wan = "GUAC_WAN"
}
```

`team_lan_names` is the trick that makes downstream blocks readable:
`local.team_lan_names[count.index]` always gives you the right team's
LAN name without re-deriving it.

### 3. Per-team firewalls (lines 224–275)

```hcl
resource "lxd_instance" "team_fw" {
  count    = var.team_count                                          # ◀── one per team
  project  = data.lxd_project.proj.name
  name     = "${var.project_name}-team${count.index + 1}-fw"
  image    = "openwrt-team-new"
  type     = "container"
  profiles = ["pfsense"]

  dynamic "device" {
    for_each = {
      for i, net in tolist([
        local.shared_wan,                          # eth0 = trunk
        local.team_lan_names[count.index]          # eth1 = THIS team's LAN
      ]) : i => net
    }
    content {
      name       = "eth${device.key}"
      type       = "nic"
      properties = { network = device.value }
    }
  }

  config = {
    "cloud-init.network-config" = <<-EOF
      version: 2
      ethernets:
        eth0:
          dhcp4: true
        eth1:
          dhcp4: false
          addresses:
            - 192.168.${1 + count.index}.1/24
          routes:
            - to: default
              via: 192.168.${1 + count.index}.1
          nameservers:
            addresses: [192.168.${1 + count.index}.1]
        EOF
  }

  target = "@Cluster-C"
}
```

The team firewall's WAN side gets DHCP from the project firewall (over
the shared `team-wan`); its LAN side is the gateway `192.168.${i+1}.1`
for that team. Each team firewall sees only its own team's hosts.

### 4. Per-team LAN hosts (lines 281–329)

The `lan_linux` block in `Win-lin` is the standard "alternating
images by index" pattern. With `lan_image = ["guac-xfce4-v01",
"windows-10-base"]` and `lan_name = ["ubuntu", "win"]`:

```text
count.index = 0  →  team1-ubuntu  (linux VM, on team1-lan, .2)
count.index = 1  →  team1-win     (windows VM, on team1-lan, .2 ⚠)
count.index = 2  →  team2-ubuntu  (linux VM, on team2-lan, .2)
count.index = 3  →  team2-win     (windows VM, on team2-lan, .2 ⚠)
...
```

The HCL that does the indexing:

```hcl
name  = "${var.project_name}-team${floor(count.index / length(var.lan_name)) + 1}-${var.lan_name[count.index % length(var.lan_name)]}"
type  = var.lan_name[count.index % length(var.lan_name)] == "mint" ? "container" : "virtual-machine"
image = var.lan_image[count.index % length(var.lan_image)]
profiles = contains(local.windows_images, var.lan_image[count.index % length(var.lan_image)]) ? ["default-windows"] : ["guac-linux"]

device {
  name = "eth0"
  type = "nic"
  properties = {
    network = lxd_network.team_lan[floor(count.index / length(var.lan_name))].name
  }
}
```

⚠ **The `Win-lin` reference has a known IP collision** — every host
on a team-lan ends up with `192.168.${1 + count.index}.2/24`, where
`count.index` is the *global* index. That works in practice because the
team_lan VMs only have one host each that wins the address race, but if
you want a deterministic plan use `floor(count.index / length(var.lan_name))`
in the address too:

```hcl
addresses:
  - 192.168.${1 + floor(count.index / length(var.lan_name))}.${2 + (count.index % length(var.lan_name))}/24
```

That is exactly what `Lin-Lab` (lines 309–323) does, and it is the
recommended formulation for any new project.

### 5. Per-team Guac (lines 342–393)

```hcl
resource "lxd_instance" "guac" {
  count = var.team_count
  ...
  dynamic "device" {
    for_each = {
      for i, net in tolist([
        local.guac_wan,                              # eth0 = guac uplink
        local.team_lan_names[count.index]            # eth1 = THIS team's LAN
      ]) : i => net
    }
    content {
      name       = "eth${device.key}"
      type       = "nic"
      properties = { network = device.value }
    }
  }

  config = {
    "cloud-init.network-config" = <<-EOF
      version: 2
      ethernets:
        enp5s0:
          dhcp4: false
          addresses:
            - 10.0.${var.guac_subnet_octet}.${3 + count.index}/16
          routes:
            - to: default
              via: 10.0.0.1
        enp6s0:
          dhcp4: false
          addresses:
            - 192.168.${1 + count.index}.3/24
          routes:
            - to: 172.31.31.2
              via: 192.168.${1 + count.index}.1
      EOF
  }
}
```

Each Guac sits on **two** networks: the shared `GUAC_WAN` so staff can
reach it via the project Guac, and that *one* team's LAN so the team
sees their own Guacamole. Cross-team access is impossible — `team1`'s
Guac literally has no NIC on `team2-lan`.

## Segmented variant (`CIG/COMP/CPTC`)

The segmented variant changes only two things:

1. **Multiple per-team network resources.** Three networks per team:

   ```hcl
   resource "lxd_network" "team_dmz"       { count = var.team_count, ... }
   resource "lxd_network" "team_business"  { count = var.team_count, ... }
   resource "lxd_network" "team_sensitive" { count = var.team_count, ... }
   ```

2. **Team firewall has more NICs.** Four in total — WAN trunk + DMZ +
   business + sensitive:

   ```hcl
   for_each = {
     for i, net in tolist([
       local.shared_wan,
       local.team_dmz_names[count.index],
       local.team_business_names[count.index],
       local.team_sensitive_names[count.index]
     ]) : i => net
   }
   ```

LAN hosts then attach to whichever segment they belong on by indexing
`local.team_<segment>_names[count.index]` instead of
`local.team_lan_names[count.index]`. Everything else (project, salt,
guac, address plan offsets) is unchanged.

## Build it

```bash
cd /home/ceroc/InSPIRE/CIG/OCIG/Win-lin     # or CIG/DCIG/Lin-Lab, or CIG/COMP/CPTC

forge init
forge plan          # always plan first
forge apply
```

Expected output (abridged):

```text
Allocated subnet octet 12 for project CIG-Lab
Allocated 45 team networks
Allocated 45 team firewalls
Allocated 90 LAN VMs (45 ubuntu + 45 windows)
Allocated 45 Guac VMs
...
Apply complete! Resources: 230 added, 0 changed, 0 destroyed.
```

A 45-team multi-network project on the current cluster takes roughly
20–35 minutes to fully apply. Run `forge apply -auto-approve` in `tmux`
or `screen` and walk away.

When the exercise is done:

```bash
forge destroy
```

## Common modifications

### Add a third image to the rotation

Extend the parallel lists. The `Lin-Lab` project uses three:

```hcl
variable "lan_image" {
  type    = list(string)
  default = ["guac-xfce4-v02", "guac-xfce4-v02", "guac-xfce4-v02"]
}

variable "lan_name" {
  type    = list(string)
  default = ["ubuntu1", "ubuntu2", "ubuntu3"]
}

resource "lxd_instance" "lan_linux" {
  count = var.team_count * length(var.lan_name)   # ◀── multiplied
  ...
}
```

The `count = var.team_count * length(var.lan_name)` is the key change
when going from "one host per team" to "N hosts per team".

### Convert a flat project into a segmented one

This is a destroy-and-recreate operation, not an in-place migration. The
LAN resource names change, every NIC's `properties.network` changes, and
every host's IP changes. Expect to:

1. `forge destroy` the current project.
2. Add the extra `lxd_network` resources and re-wire `team_fw`.
3. Update each LAN host to attach to its target segment.
4. `forge apply` from scratch.

### Increase or decrease `team_count`

```hcl
variable "team_count" {
  type    = number
  default = 50      # was 45
}
```

`forge plan` will show `team1`–`team45` unchanged and `team46`–`team50`
to be created. Apply normally. Going *down* destroys the trailing teams
— do this only between exercises.

## When the multi-network pattern is the wrong choice

- All teams genuinely share resources (same shared CTF box, same Active
  Directory) → use [single-network](./single-network.md) and save
  yourself ~`team_count - 1` OVN switches and team firewalls.
- One-off labs with two or three teams where you would manage the IP
  plan by hand anyway → still use multi-network; the boilerplate is
  worth it for the consistency.

---

[← Back to docs index](./README.md)

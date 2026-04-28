# Concepts & Terminology

A short primer on the technologies that make up a range project. Read this
once before you author your first `main.tf`; everything else in these docs
assumes you are comfortable with the vocabulary below.

> **TL;DR** &nbsp; You declare LXD instances and OVN networks in HCL, run
> `forge apply`, and Forge wraps OpenTofu so each project lands in its own
> LXD project with its own `/24` guac subnet, fully isolated from the rest
> of the range.

---

## OpenTofu

[OpenTofu](https://opentofu.org/) is an open-source, MPL-licensed fork of
Terraform. It uses the same HCL configuration language, the same provider
ecosystem, and the same plan/apply lifecycle. For our purposes, anything
written for Terraform 1.5 works in OpenTofu unchanged.

We use the [`terraform-lxd/lxd`](https://registry.terraform.io/providers/terraform-lxd/lxd)
provider to declare LXD projects, networks, and instances.

### Lifecycle commands

| Command  | Purpose |
|----------|---------|
| `tofu init`     | Download providers, initialize the working dir |
| `tofu validate` | Static check of HCL syntax and references |
| `tofu plan`     | Compute the diff between desired and actual state |
| `tofu apply`    | Make the cluster match the desired state |
| `tofu destroy`  | Remove every resource declared in this configuration |

In day-to-day work you will run `forge` instead of `tofu` directly. Forge
forwards every flag through to `tofu`, so anything you would pass to
`tofu apply` works the same with `forge apply`.

## Forge

`forge` is our thin wrapper around `tofu`. It exists to solve three
problems the bare provider does not handle for us:

1. **Subnet allocation.** Each project gets a unique `/24` inside
   `10.0.0.0/16` for its Guacamole gateway VMs. Forge picks the next free
   octet and writes the allocation to a shared registry at
   `/home/ceroc/InSPIRE/bin/guac_subnet/subnets.json`.
2. **Variable injection.** Forge reads `project_name` from `main.tf` and
   passes it, plus the allocated `guac_subnet_octet`, on every `tofu`
   invocation so you do not have to edit `tfvars` per deployment.
3. **Lifecycle orchestration.** `forge apply` does more than `tofu apply`
   — it also exports the LXD instance list, starts the config server,
   and triggers the Windows VMs after VMs have settled. `forge destroy`
   reverses all of it and frees the subnet.

For the full command list, allocation model, and troubleshooting, see the
[Forge CLI reference](./forge.md).

## LXD

[LXD](https://canonical.com/lxd) is a container and VM manager. The range
runs as an LXD **cluster** of physical hosts (cluster members like
`micro-06`, `Cluster-C`, etc.). Inside the cluster we use four constructs:

| Construct  | Description |
|------------|-------------|
| **Project**  | Namespace boundary. Instances, networks, and storage volumes inside a project are invisible to other projects. One range exercise = one LXD project. |
| **Profile**  | Reusable bundle of instance config (devices, limits, security). Profiles like `pfsense`, `Salt-master`, `guac-linux`, `default-windows` already exist on the cluster. |
| **Network**  | An OVN logical switch that lives inside one LXD project. Multiple networks per project are normal. |
| **Instance** | Either a `container` (LXC) or a `virtual-machine` (QEMU/KVM). Instances attach to networks via NIC `device` blocks. |

### Cluster targeting

Every range we run today pins instances to a cluster member with the
`target` attribute, e.g. `target = "@Cluster-C"`. This is intentional —
co-locating each team on a single member keeps OVN traffic local and makes
post-event cleanup deterministic. If you need to spread load, change the
target before applying; do **not** mix targets inside one team.

## OVN Networking

All in-project networking is OVN. From the OpenTofu side it looks like a
normal `lxd_network` resource with `type = "ovn"`:

```hcl
resource "lxd_network" "team_lan" {
  project = data.lxd_project.proj.name
  name    = "${var.project_name}-team-lan"
  type    = "ovn"
  config = {
    "bridge.mtu"    = "1500"
    "ipv4.address"  = "none"     # we manage IPs in cloud-init, not LXD
    "network"       = "internal_link5"
  }
}
```

Three things to know:

- **`ipv4.address = "none"`** — we *intentionally* let LXD/OVN provide
  layer-2 only. IP addressing is configured per-instance through
  `cloud-init.network-config`. This keeps the IP plan in HCL where it is
  reviewable, and avoids LXD's DHCP server stepping on the project
  firewall.
- **`network = "internal_link5"`** — this is the upstream OVN parent
  network on our cluster. It provides the shared underlay; do not change
  it unless you are intentionally re-homing a project.
- **Special uplinks `CLASS_WAN` / `DCIG_WAN` / `GUAC_WAN`** — these are
  pre-provisioned cluster-level networks the project firewall and the
  guac VMs hang off of. They live outside any single LXD project and are
  referenced by name as strings.

## The Guac Subnet Plan

The `10.0.0.0/16` space is reserved for Guacamole gateway VMs across the
entire range. Forge carves it into `/24`s, one per project:

| Octet | Subnet           | Used by |
|-------|------------------|---------|
| 0     | `10.0.0.0/24`    | Reserved (gateway, DNS) |
| 1     | `10.0.1.0/24`    | First project to apply |
| 2     | `10.0.2.0/24`    | Second project |
| …     | …                | … |
| 254   | `10.0.254.0/24`  | Last available |

Inside a project, the octet is referenced as `var.guac_subnet_octet`. You
write your cloud-init like this and Forge fills in the value:

```hcl
addresses:
  - 10.0.${var.guac_subnet_octet}.2/16
```

You never edit the octet by hand. `forge apply` allocates one and
`forge destroy` releases it.

## Project / Naming Standards

| Element | Convention | Example |
|---------|-----------|---------|
| Project name (`var.project_name`)       | Short, kebab-cased, descriptive | `CSC-3410`, `CIG-Lab`, `CPTC-Mock` |
| Network name                            | `${project_name}-<role>` or `${project_name}-team${n}-<role>` | `CIG-Lab-team3-lan` |
| Instance name                           | `${project_name}-team${n}-<role>` | `CIG-Lab-team3-Guac` |
| Per-team LAN block                      | `192.168.${team_index}.0/24` | team 5 → `192.168.5.0/24` |
| Salt LAN (project-wide)                 | `172.31.31.0/24` | Salt master `.2`, Guac salt `.3` |
| Guac WAN                                | `10.0.${guac_subnet_octet}.0/24` inside `10.0.0.0/16` | octet 7 → `10.0.7.2`, `10.0.7.3`, … |

Stick to these; the configuration agent, Guacamole connection import
scripts, and the salt minion bootstraps all assume them.

## Where to go next

- Author a new project → [Building Infrastructure](./infrastructure.md)
- Single shared LAN → [Single-Network Projects](./single-network.md)
- Per-team isolation → [Multi-Network Projects](./multi-network.md)
- Day-2 LXD ops → [LXD Operations Guide](./lxd.md)

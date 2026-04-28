# Cyber Range Documentation

Engineering documentation for the InSPIRE Cyber Range platform — declarative
infrastructure built on **OpenTofu** and **LXD**, deployed and managed through
the **Forge** CLI.

This site is the entry point for everything operators, range engineers, and
contributors need to plan, build, run, and tear down range projects.

---

## Start Here

| Page | When to read it |
|------|-----------------|
| [Concepts & Terminology](./concepts.md) | First time on the platform. Explains OpenTofu, Forge, LXD projects, OVN networks, and how the pieces fit. |
| [Building Infrastructure](./infrastructure.md) | You are about to author or modify a `main.tf`. Covers the standard project layout, decision points, and deployment workflow. |

## Project Patterns

Two patterns cover essentially every range we deploy. Pick the one that
matches the exercise you are building.

| Pattern | Use when… | Reference page |
|---------|-----------|----------------|
| **Single-network project** | All teams share one LAN (e.g. classroom labs, CTF-style shared range). | [Single-Network Projects](./single-network.md) |
| **Multi-network project**  | Each team gets its own isolated network(s). Required for any exercise where teams must not see each other. | [Multi-Network Projects](./multi-network.md) |

Concrete reference implementations live on the OpenTofu host under
`/home/ceroc/InSPIRE/`:

- `/home/ceroc/InSPIRE/Classes/CSC-3410-CS/` — Single-network classroom lab (65 teams)
- `/home/ceroc/InSPIRE/CIG/OCIG/Win-lin/`    — Multi-network Windows + Linux lab (45 teams)
- `/home/ceroc/InSPIRE/CIG/DCIG/Lin-Lab/`    — Multi-network Linux-only lab (35 teams)
- `/home/ceroc/InSPIRE/CIG/COMP/CPTC/`       — Multi-network multi-segment CPTC mock (DMZ / business / sensitive)

## LXD Operations

Day-2 operations on the LXD cluster: switching projects, migrating VMs
between cluster members, snapshotting, and bulk start/stop/cleanup.

- [LXD Operations Guide](./lxd.md)

## Forge Tooling

The Forge CLI and its companion config server are the supported way to
drive OpenTofu against the range.

| Page | Contents |
|------|----------|
| [Forge CLI Reference](./forge.md) | Command reference, subnet allocation model, `subnets.json`, troubleshooting. |
| [Setup Guide](./setup.md) | End-to-end setup: server, Windows/Linux/OpenWrt clients, Terraform config, scheduled tasks. |
| [Architecture / Plan](./plan.md) | Original architecture diagram and component breakdown for the configuration agent. |

---

## Quick Reference

### Typical lifecycle of a range project

```text
  Author main.tf  ─►  forge init  ─►  forge plan  ─►  forge apply  ─►  Run exercise
                                                          │
                                                          ▼
                                                    forge destroy
```

### Where things live (OpenTofu host)

| Path | Purpose |
|------|---------|
| `/home/ceroc/InSPIRE/Classes/<class>/main.tf`     | OpenTofu config for a course-based range (e.g. `CSC-3410-CS`) |
| `/home/ceroc/InSPIRE/CIG/COMP/<project>/main.tf`  | Competition-track range projects (e.g. `CPTC`) |
| `/home/ceroc/InSPIRE/CIG/DCIG/<project>/main.tf`  | Defensive-track range projects (e.g. `Lin-Lab`) |
| `/home/ceroc/InSPIRE/CIG/OCIG/<project>/main.tf`  | Offensive-track range projects (e.g. `Win-lin`) |
| `/home/ceroc/InSPIRE/bin/forge_bin/forge`         | Installed `forge` binary |
| `/home/ceroc/InSPIRE/bin/guac_subnet/subnets.json`| Cluster-wide guac `/24` subnet allocation table |
| `/home/ceroc/InSPIRE/bin/scripts/`                | Bulk LXD ops scripts (start, stop, snapshot, move, network cleanup) |

### Conventions used in these docs

- Code blocks tagged `bash` are run on the OpenTofu host (Linux).
- Code blocks tagged `hcl` are OpenTofu/Terraform HCL.
- `forge` is always preferred over raw `tofu`. Raw `tofu` is only shown for
  diagnostics or recovery.
- `<placeholders>` in angle brackets must be replaced before running.

---

If something in these docs is wrong, missing, or ambiguous, open an issue
or a PR against the `Cyber_Range` repository. These pages are the source
of truth — keep them current.

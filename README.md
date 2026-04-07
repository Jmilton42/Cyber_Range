# Cyber Range

Go tooling for Cyber Range automation:

- **Config server**: serves per-instance hostname/network config based on MAC address.
- **Clients**: Windows, Linux, and OpenWrt agents that fetch config and apply it once.
- **Forge**: a wrapper around OpenTofu that manages guac subnet allocation and post-apply steps.

## Quick links

- Setup guide: `docs/setup.md`
- Forge CLI: `docs/forge.md`

## Repository layout

- `cmd/`: entrypoints (`server`, `forge`, and OS-specific `client`s)
- `internal/`: implementation packages
- `scripts/`: helper scripts (`deploy.sh`, `destroy.sh`, `setup-task.ps1`)
- `docs/`: documentation
- `configs/`: example configs
- `examples/`: sample data files (e.g. `subnets.json`, `instances.json`)
- `infra/`: example OpenTofu/Terraform configuration (see `infra/tofu/`)
- `dist/`: optional prebuilt binaries (if committed)
- `bin/`: local build outputs (ignored)


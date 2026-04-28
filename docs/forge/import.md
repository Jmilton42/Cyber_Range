# forge import

Bring an LXD project that was created outside forge under forge
management. Allocates a subnet for it and writes (or updates)
`instances.json` so the config server starts answering for the project's
VMs.

## Synopsis

```
forge import <lxd-project> [-octet <n>] [-yes]
```

## Why use it

Useful when:

- Somebody created a project with raw `lxc project create` and now wants
  the config server to host its VMs.
- You're migrating an existing manual setup to a forge-managed workflow.
- A `forge apply` was interrupted before `instances.json` was written
  and you want to recover state instead of redeploying.

## What it does

1. Validates that `<lxd-project>` exists in LXD (`lxc project list`).
2. If the project is not in `subnets.json`, allocates the next free
   octet (or the one passed via `-octet`).
3. Switches the LXD client to the project, runs `lxc list -f json`, and
   writes `instances.json` to the cwd.
4. Prints the resolved subnet, instance count, and the recommended
   `forge reload` follow-up.

## Examples

```bash
# Import an existing project, auto-pick an octet
forge import legacy-2025-fall

# Reserve a specific octet
forge import legacy-2025-fall -octet 64 -yes

# From inside the project's working directory
cd /home/ceroc/InSPIRE/legacy-2025-fall
forge import legacy-2025-fall
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-octet <n>` | next free | Specific octet to reserve. Errors if taken. |
| `-yes` / `-y` | off | Skip the confirmation prompt. |

## Output

```
[INFO] Found LXD project legacy-2025-fall (5 instances)
[INFO] Reserving subnet octet 64 (10.0.64.0/24)
[INFO] Wrote instances.json (5 instances)

Project imported. Next steps:
  forge reload          # if a config server is already running
  forge serve           # otherwise
```

## Related

- [`forge subnets reserve`](./subnets.md) — reserve without writing
  `instances.json`.
- [`forge reload`](./reload.md) — tell the running server to pick up the
  newly written `instances.json`.
- [`forge apply`](./apply.md) — the normal flow when you do want
  OpenTofu to manage the project going forward.

## Notes

- Import does **not** generate a `main.tf`. The project still has no
  OpenTofu state; future changes still need to happen via raw `lxc`
  commands or by writing a `main.tf` and running `forge apply`.
- `instances.json` is overwritten if it already exists.

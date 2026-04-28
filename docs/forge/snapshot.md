# forge snapshot

Take an LXD snapshot of every instance in a project. Bundled plugin
(`forge-snapshot`).

## Synopsis

```
forge snapshot <project> [-name <snapshot-name>] [-y]
```

## Why use it

Snapshots are the cheapest "save point" you have during a competition or
exercise. Take one:

- Right before students start a graded section.
- Before a risky `forge apply` or in-place reconfiguration.
- At the end of an exercise so you can roll back to a clean state next
  time.

Doing this with raw `lxc` for 65 VMs is tedious and error-prone. The
plugin loops in parallel and reports per-instance success/failure.

## What it does

1. Switches the LXD client to `<project>`.
2. Lists every instance in the project.
3. Issues `lxc snapshot <instance> <snapshot-name>` for each, with a
   small worker pool (currently 8 in parallel).
4. Prints a per-instance status line and a final summary.

## Examples

```bash
# Default snapshot name (timestamp)
forge snapshot ocig-win-lin

# Named snapshot, no prompt
forge snapshot ocig-win-lin -name pre-graded-section -y

# All projects (loop in shell)
for p in $(forge subnets list -json | jq -r '.allocations[].project'); do
  forge snapshot "$p" -name nightly -y
done
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-name <name>` | `forge-YYYYMMDD-HHMMSS` | Snapshot name to apply to every instance. |
| `-y` / `-yes` | off | Skip confirmation. Honors `FORGE_AUTO_YES=1`. |

## Output

```
[INFO] Project: ocig-win-lin
[INFO] Snapshot name: forge-20260428-103000
[INFO] 12 instances will be snapshotted

About to snapshot 12 instances. Continue? [y/N] y

  [OK]  kali-1     -> forge-20260428-103000
  [OK]  dc01       -> forge-20260428-103000
  [OK]  win10-1    -> forge-20260428-103000
  ...

Snapshotted: 12 / 12
```

## Related

- [`forge start`](./start.md) / [`forge stop`](./stop.md) — usual companions.
- [Writing Forge Plugins](./writing-plugins.md) — `forge-snapshot` is a
  good reference implementation.

## Notes

- Snapshots count against your LXD storage pool. They are deduplicated
  via copy-on-write, but heavy churn still grows the pool. Run
  `lxc snapshot delete` periodically.
- The plugin reads `FORGE_PROJECT` (set by forge automatically), so
  `forge snapshot` (no positional arg) implicitly uses the project from
  the cwd's `main.tf`.

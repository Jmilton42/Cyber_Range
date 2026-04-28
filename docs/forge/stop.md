# forge stop

Force-stop every instance in a project. Bundled plugin (`forge-stop`).

## Synopsis

```
forge stop <project> [-y]
```

## Why use it

When an exercise ends and you want to free CPU/RAM without destroying
the project, you need every VM in the project shut down. Asking 65
students to `shutdown` cleanly never happens; this plugin issues a
forced stop in parallel and reports the outcome.

This is the right command **before**:

- A long maintenance window.
- A migration off this cluster.
- A `forge snapshot` of cold disks (faster + smaller).

## What it does

1. Switches the LXD client to `<project>`.
2. Lists every instance in the project.
3. Issues `lxc stop <instance> --force` for each running instance, in
   parallel.
4. Prints a per-instance status line and a final summary.

## Examples

```bash
forge stop ocig-win-lin
forge stop ocig-win-lin -y    # no prompt
```

## Output

```
[INFO] Project: ocig-win-lin
[INFO] 12 instances (10 running, 2 already stopped)

About to force-stop 10 instances. Continue? [y/N] y

  [OK]    kali-1      stopping
  [OK]    dc01        stopping
  [SKIP]  win10-1     already stopped
  ...

Stopped: 10 / 10 (2 already stopped)
```

## Related

- [`forge start`](./start.md) — symmetric counterpart.
- [`forge snapshot`](./snapshot.md) — typically run before/after stop.
- [`forge destroy`](./destroy.md) — when you also want to release the
  resources, not just power them down.

## Notes

- Uses `--force`. There is **no** clean shutdown attempt — guests do
  not get a chance to flush. Take a snapshot first if you need a
  consistent state.
- Honors `FORGE_PROJECT` and `FORGE_AUTO_YES=1`.

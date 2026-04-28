# forge start

Start every instance in a project. Bundled plugin (`forge-start`).

## Synopsis

```
forge start <project> [-y]
```

## Why use it

After a power event, a maintenance window, or a `forge stop`, you need
the project's VMs running again. `lxc start --all` is project-scoped in
a way that's easy to mis-remember; this plugin handles the project
switch for you and reports per-instance status.

## What it does

1. Switches the LXD client to `<project>`.
2. Lists every instance in the project.
3. Issues `lxc start <instance>` for each instance not already running,
   with a small worker pool.
4. Prints a per-instance status line and a final summary.

## Examples

```bash
# Start everything in the project
forge start ocig-win-lin

# Skip the prompt
forge start ocig-win-lin -y
```

## Output

```
[INFO] Project: ocig-win-lin
[INFO] 12 instances (10 stopped, 2 already running)

About to start 10 instances. Continue? [y/N] y

  [OK]    kali-1      starting
  [OK]    dc01        starting
  [SKIP]  win10-1     already running
  ...

Started: 10 / 10 (2 already running)
```

## Related

- [`forge stop`](./stop.md) — symmetric counterpart.
- [`forge cost`](./cost.md) — confirm what a project will consume before starting a big
  project.
- [`forge apply`](./apply.md) — already starts everything as part of
  deployment.

## Notes

- VMs that are mid-boot or in an `ERROR` state are reported but not
  forced — if you need to recover, escalate to `lxc start <inst>
  --force`.
- Honors `FORGE_PROJECT` and `FORGE_AUTO_YES=1` (set by forge when you
  pass `--yes`).

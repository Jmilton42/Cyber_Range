# forge migrate

Move every instance in a project to a different cluster member. Bundled
plugin (`forge-migrate`).

## Synopsis

```
forge migrate <project> <target-node> [-y]
```

## Why use it

Use migrate when:

- A cluster member is overloaded (compare [`forge cost`](./cost.md) across projects on that node) and
  you want to evacuate one project off it.
- You're putting a node into maintenance.
- You're balancing storage pressure between members.

Doing this with raw `lxc move` requires the right `--target` flag for
every instance and is easy to get wrong. The plugin handles the loop and
reports any instance that won't move (e.g. a storage pool that doesn't
exist on the target).

## What it does

1. Confirms `<target-node>` is a member of the LXD cluster.
2. Switches to `<project>` in LXD.
3. Lists every instance and its current home node.
4. Skips instances that are already on `<target-node>`.
5. Issues `lxc move <instance> <instance> --target <target-node>` for
   each remaining instance, sequentially (live migration is
   bandwidth-heavy; parallelism is intentionally low).
6. Prints a per-instance status line and a final summary.

## Examples

```bash
# Move project off member2 onto member4
forge migrate ocig-win-lin member4

# No prompt
forge migrate ocig-win-lin member4 -y
```

## Output

```
[INFO] Project:  ocig-win-lin
[INFO] Target:   member4
[INFO] 12 instances (10 to migrate, 2 already on member4)

About to migrate 10 instances to member4. Continue? [y/N] y

  [OK]    kali-1     member2 -> member4
  [OK]    dc01       member1 -> member4
  [SKIP]  win10-1    already on member4
  [FAIL]  bigwin     storage pool "default" not present on member4
  ...

Migrated: 9 / 10  (1 failed, 2 already on target)
```

Exit non-zero if any instance fails to migrate.

## Related

- [`forge cost`](./cost.md) — see what a project is consuming before relocating it.
- [`forge stop`](./stop.md) — for a faster cold-migration: stop, move,
  start again. Only useful if you don't need live migration.
- [`forge networks prune`](./networks-prune.md) — cleanup orphaned OVN
  networks left behind after large moves.

## Notes

- Live migration only works for VMs (not containers) and only between
  members that share a storage pool name and the same OVN network.
  Otherwise, stop the project first and re-`apply` against the target —
  that's faster anyway for big moves.
- `<target-node>` is tab-completable; forge calls
  `lxc cluster list -f json` to populate the candidate list.
- Honors `FORGE_PROJECT` and `FORGE_AUTO_YES=1`.

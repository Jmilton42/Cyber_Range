# forge networks prune

Delete OVN networks whose name starts with a given prefix. Bundled
plugin (`forge-networks-prune`).

## Synopsis

```
forge networks prune <prefix> [-y] [-dry-run]
```

## Why use it

A failed `forge destroy` (or a manual `lxc project delete`) can leave
behind orphan OVN networks named after the project — they show up in
`lxc network list` forever and clutter the OVN integration. Cleaning
them up by hand is annoying because you have to be careful not to nuke
networks that are still in use.

`forge networks prune` does it safely:

- Refuses to touch a network with attached instances.
- Has a `-dry-run` mode that lists what *would* be deleted.
- Forces you to type a prefix, so a typo doesn't kill the production
  shared network.

## What it does

1. Calls `lxc network list -f json`.
2. Filters to networks of type `ovn` whose name starts with
   `<prefix>`.
3. For each candidate, checks the `used_by` list:
   - If empty → eligible for deletion.
   - Otherwise → skip and report.
4. With `-dry-run`, prints the eligible list and exits.
5. Otherwise, prompts for confirmation (skip with `-y`), then deletes
   each eligible network.

## Examples

```bash
# What would I delete? (no changes)
forge networks prune ocig-win-lin -dry-run

# Actually delete them
forge networks prune ocig-win-lin

# Scripted cleanup
forge networks prune ocig-win-lin -y
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-dry-run` | off | List what would be deleted; do nothing. |
| `-y` / `-yes` | off | Skip confirmation. Honors `FORGE_AUTO_YES=1`. |

## Output

```
[INFO] Prefix: ocig-win-lin
[INFO] Found 4 OVN networks matching prefix
[INFO] 3 eligible (no attached instances)
[INFO] 1 skipped (still in use)

Eligible for deletion:
  ocig-win-lin-team1
  ocig-win-lin-team2
  ocig-win-lin-uplink

Skipped (in use):
  ocig-win-lin-mgmt        used_by: 1 instance

Delete 3 networks? [y/N] y

  [OK]  ocig-win-lin-team1
  [OK]  ocig-win-lin-team2
  [OK]  ocig-win-lin-uplink

Deleted: 3 / 3
```

## Related

- [`forge destroy`](./destroy.md) — usually cleans these up
  automatically. Run prune only when destroy didn't finish.
- [`forge migrate`](./migrate.md) — large migrations can leave per-team
  uplinks behind on the source nodes.

## Notes

- The empty-prefix case is **rejected**: `forge networks prune ""`
  would delete every OVN network on the cluster. The plugin refuses.
- Only OVN networks are touched. Bridge networks (`lxdbr0`) and
  physical uplinks are ignored even if their names match.
- Honors `FORGE_AUTO_YES=1`.

# forge plugins

List the `forge-*` plugin binaries currently discoverable on `$PATH`.
Useful both as a sanity check ("did `build_all.sh` actually install
everything?") and as the primary way to see what custom plugins are
installed locally.

## Synopsis

```
forge plugins list [-json]
forge plugins ls         # alias
```

## Why use it

Forge's day-2 LXD operations and any operator-authored extensions all
live as separate `forge-<name>` binaries. They get discovered at runtime
by walking `$PATH`. This command shows you exactly what was found, in
the same order forge itself would resolve them.

Use it when:

- A plugin command (`forge snapshot`, `forge start`, etc.) returns
  "moved to plugin" and you want to confirm whether the plugin is
  installed.
- You wrote a custom `forge-foo` and want to verify the directory it
  lives in is on `$PATH`.
- You need to share with a teammate exactly which extensions you have.

## What it does

1. Splits `$PATH` and walks each directory.
2. Collects every executable file whose name starts with `forge-`.
3. De-duplicates by command name (the first one on `$PATH` wins, just
   like shell resolution).
4. Prints them in resolution order.

## Examples

```bash
# Standard human-friendly view
forge plugins list

# JSON for scripts / dashboards
forge plugins list -json | jq '.plugins[].name'
```

## Output

```
NAME                  PATH
forge-cost            /home/ceroc/InSPIRE/bin/forge_bin/forge-cost
forge-migrate         /home/ceroc/InSPIRE/bin/forge_bin/forge-migrate
forge-networks-prune  /home/ceroc/InSPIRE/bin/forge_bin/forge-networks-prune
forge-snapshot        /home/ceroc/InSPIRE/bin/forge_bin/forge-snapshot
forge-start           /home/ceroc/InSPIRE/bin/forge_bin/forge-start
forge-stop            /home/ceroc/InSPIRE/bin/forge_bin/forge-stop
forge-grade-export    /home/joeym/.local/bin/forge-grade-export

7 plugin(s)
```

## JSON shape

```json
{
  "plugins": [
    {"name":"forge-migrate","command":"migrate","path":"/home/ceroc/InSPIRE/bin/forge_bin/forge-migrate"},
    {"name":"forge-snapshot","command":"snapshot","path":"/home/ceroc/InSPIRE/bin/forge_bin/forge-snapshot"}
  ],
  "count": 2
}
```

## Related

- [Writing Forge Plugins](./writing-plugins.md) — how to author your
  own.
- [`forge doctor`](./doctor.md) — flags missing bundled plugins.
- [`forge snapshot`](./snapshot.md), [`forge start`](./start.md),
  [`forge stop`](./stop.md), [`forge migrate`](./migrate.md),
  [`forge networks prune`](./networks-prune.md) — the plugins that
  ship in the box.

## Notes

- "Shadowing" works the same as for shell commands: if you have two
  `forge-snapshot` binaries on `$PATH`, the first one wins and the
  second is silently ignored (forge does **not** warn).
- Tab-completion `forge <TAB>` includes plugin commands automatically;
  no extra registration is needed.

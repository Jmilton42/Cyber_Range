# forge subnets

Manage entries in the cluster-wide `subnets.json` allocation table
directly. Three subcommands: `list`, `free`, `reserve`.

## Synopsis

```
forge subnets list [-json]
forge subnets free <project> [-y]
forge subnets reserve <project> [<octet>] [-y]
```

## Why use it

Most of the time the lifecycle commands handle subnets for you:

- `forge plan` / `forge apply` — allocate.
- `forge destroy` — release.

You reach for `forge subnets` when something has drifted:

- A project's LXD resources got nuked outside of forge, but the entry is
  still in `subnets.json`.
- You want to reserve a specific octet for a planned project before
  anybody else snags it.
- You're auditing the cluster ("which projects exist? which octets are
  free?").

## Subcommands

### `forge subnets list`

Prints every allocation in the table, sorted by octet.

```bash
$ forge subnets list
Project              Octet   Subnet            Allocated At
csc-3410-lab         2       10.0.2.0/24       2026-01-12T11:00:00-05:00
ocig-win-lin         4       10.0.4.0/24       2026-04-08T09:00:00-05:00
cptc-muli            7       10.0.7.0/24       2026-04-22T13:30:00-05:00

$ forge subnets list -json
{ "allocations": [ {"project":"csc-3410-lab","subnet_octet":2,...} ] }
```

### `forge subnets free <project>`

Removes a row from `subnets.json`. Confirms by default; pass `-y` to
skip the prompt.

```bash
$ forge subnets free old-class
Free subnet 10.0.5.0/24 from project "old-class"? [y/N] y
[INFO] Released 10.0.5.0/24 (octet 5)
```

Use this **only** when the LXD resources have already been destroyed.
Forge does not touch LXD here — it's a metadata edit.

### `forge subnets reserve <project> [<octet>]`

Reserve a row without running `tofu apply`. Either pick a specific
octet or let forge pick the next free one.

```bash
# Pick the next free octet
$ forge subnets reserve fall26-lab
[INFO] Reserved octet 8 (10.0.8.0/24) for project "fall26-lab"

# Demand a specific octet
$ forge subnets reserve fall26-lab 42 -y
[INFO] Reserved octet 42 (10.0.42.0/24) for project "fall26-lab"
```

Errors if:
- The project is already in the table.
- The octet you asked for is in use.
- The octet is outside `1..254`.

## Examples

```bash
# Snapshot the current state of the cluster
forge subnets list -json > /tmp/subnets-before.json

# Reserve octet 100 for a planned semester project
forge subnets reserve sp26-csc4410 100 -y

# Clean up after a project that was destroyed by hand
forge subnets free old-experiment -y
```

## Related

- [`forge status`](./status.md) — same data with project context.
- [`forge new --reserve`](./new.md) — reserves automatically as part of
  scaffolding.
- [`forge destroy`](./destroy.md) — releases automatically as part of
  teardown.

## Notes

- `subnets.json` is a flat-file source of truth at
  `/home/ceroc/InSPIRE/bin/guac_subnet/subnets.json`. Hand-edit at your
  own risk; forge will refuse to load malformed JSON.
- Tab-completion of `<project>` for `free` is dynamic and reads the
  current allocation list.

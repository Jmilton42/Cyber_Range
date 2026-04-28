# forge status

Show the subnet allocation. Inside a project directory, you get this
project's allocation plus the cluster-wide list. Outside one, you get
just the cluster-wide list.

## Synopsis

```
forge status [-json]
```

## Why use it

You'd run this when:

- A teammate asks "what subnet is `ocig-win-lin` on?"
- You want to know what's currently deployed across the cluster.
- You need to debug a network conflict.
- A script needs structured allocation data (`-json`).

## What it does

1. Tries to read `project_name` from `main.tf` in the cwd.
2. If found, looks up that project's octet in `subnets.json` and prints
   it in context with the rest of the cluster's allocations.
3. If not found, just prints the cluster-wide list.

## Examples

```bash
# Inside a project: highlights "this" project with a *
forge status

# From your home dir: cluster-wide table
cd ~ && forge status

# Machine-readable
forge status -json
```

## Output (in-project)

```
Project:  ocig-win-lin
Work Dir: /home/ceroc/InSPIRE/CIG/OCIG/Win-lin

Subnet:   10.0.4.0/24
Gateway:  10.0.4.1
Octet:    4

Subnets file: /home/ceroc/InSPIRE/bin/guac_subnet/subnets.json

All allocations:
* ocig-win-lin                    10.0.4.0/24
  csc-3410-lab                    10.0.2.0/24
  cptc-muli                       10.0.7.0/24
```

## Output (cluster-wide)

```
Work Dir: /home/joeym
Project:  (not a forge project directory)
Reason:   could not find project_name variable with default value in main.tf

Subnets file: /home/ceroc/InSPIRE/bin/guac_subnet/subnets.json

Cluster-wide allocations (3):
  ocig-win-lin                    10.0.4.0/24
  csc-3410-lab                    10.0.2.0/24
  cptc-muli                       10.0.7.0/24
```

## JSON shape

```json
{
  "project": "ocig-win-lin",
  "work_dir": "/home/ceroc/InSPIRE/CIG/OCIG/Win-lin",
  "subnet_octet": 4,
  "subnet": "10.0.4.0/24",
  "gateway": "10.0.4.1",
  "subnets_file": "/home/ceroc/InSPIRE/bin/guac_subnet/subnets.json",
  "allocations": [
    {"project":"csc-3410-lab","subnet_octet":2,"allocated_at":"2026-01-12T11:00:00-05:00"},
    {"project":"ocig-win-lin","subnet_octet":4,"allocated_at":"2026-04-08T09:00:00-05:00"}
  ]
}
```

## Related

- [`forge subnets list`](./subnets.md) — same allocation list, no project context.
- [`forge cost`](./cost.md) — adds per-instance CPU/RAM/disk for the project.
- [`forge doctor`](./doctor.md) — sanity-check `subnets.json` before reading it.

## Notes

- Read-only. Never modifies `subnets.json`.
- Tab-completing `forge status <TAB>` does nothing — there are no
  positional args.

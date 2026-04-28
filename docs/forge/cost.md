# forge cost

Focused per-project resource breakdown: vCPU, RAM, and disk for every
instance in one project, plus totals — rendered as a per-instance
table.

> **Plugin command.** `forge cost` is shipped as the bundled
> `forge-cost` plugin, not as a built-in. The forge core stays
> hypervisor-agnostic so a different backend (Proxmox, vSphere,
> libvirt, …) can ship its own `forge-cost` binary that drops in over
> the bundled one. See [Writing forge plugins](./writing-plugins.md).
>
> Build it with `forge/scripts/build_all.sh` (or copy `forge-cost`
> from your `forge_bin` directory onto `$PATH`). `forge doctor` will
> warn if it's missing.

## Synopsis

```
forge cost [<project>] [-json]
```

If `<project>` is omitted, forge reads `project_name` from `main.tf`
in the current directory.

## Why use it

You want the answer to "how much is **this** project actually
consuming?" without the noise of every other project on the cluster.

Use `forge cost` when you're:

- Sizing a project before scaling it up.
- Confirming you released what you thought you released after a stop.
- Sharing a per-project resource snapshot with someone on a ticket.

## What it does

1. Resolves the project name:
   - positional arg `<project>` if given, else
   - `FORGE_PROJECT` env var if forge core dispatched us, else
   - `project_name` parsed out of `main.tf` in the cwd.
2. Runs `lxc query "/1.0/instances?recursion=2&all-projects=true"`
   (one call, every project, every instance).
3. Filters instances by `.project` (case-insensitive) so a `main.tf`
   value of `CPTC-Mock` matches an LXD project named `cptc-mock`.
4. For every surviving instance, reads `limits.cpu`, `limits.memory`,
   the running state, the cluster member it lives on, and the
   live root-disk usage out of `state.disk.root.usage`.
5. Renders a table sorted by instance name, with totals at the bottom.

CPU and RAM are **declared limits** (so a VM configured with
`limits.memory=8GiB` shows 8 GiB even when idle). Disk is **actual
current usage** at query time.

## Examples

```bash
# In a project directory (project name auto-detected)
forge cost

# Explicit project name
forge cost ocig-win-lin

# Pipe into jq
forge cost ocig-win-lin -json | jq '.totals'
```

## Output

```
Project: ocig-win-lin

Instance      Status    Node       vCPU  RAM        Disk
--------      --------  --------   ----  --------   --------
dc01          Running   member1    4     8.0 GB     50.2 GB
kali-1        Running   member2    2     4.0 GB     18.4 GB
ubuntu-srv    Running   member1    2     4.0 GB     12.1 GB
win10-1       Stopped   member3    4     8.0 GB     45.0 GB
win10-2       Stopped   member3    4     8.0 GB     44.7 GB
--------      --------  --------   ----  --------   --------
TOTAL (5)     3 running, 2 stopped  member1,member2,member3  16  32.0 GB  170.4 GB
```

If the project has no instances yet:

```
Project: ocig-win-lin

No instances found.
If this project was just created, run `forge apply` to spin up its instances.
```

## JSON shape

```json
{
  "project": "ocig-win-lin",
  "instances": [
    {
      "name": "dc01",
      "project": "ocig-win-lin",
      "node": "member1",
      "cpu": 4,
      "memory_bytes": 8589934592,
      "disk_bytes": 53929754624,
      "status": "Running"
    }
  ],
  "totals": {
    "instance_count": 5,
    "cpu_total": 16,
    "memory_bytes": 34359738368,
    "disk_bytes": 182935896064,
    "nodes": ["member1", "member2", "member3"]
  }
}
```

## Related

- [`forge status`](./status.md) — adds subnet allocation context.
- [`forge migrate`](./migrate.md) — once you've identified an
  oversized project, evacuate it off a hot node.

## Notes

- Disk numbers reflect **actual usage** at query time, not the
  configured root device size. A VM with a 100 GB root disk that's only
  written 12 GB will show 12 GB.
- Instances with no `limits.cpu` / `limits.memory` declared show `0`
  for those columns — LXD lets them grow opportunistically, but `cost`
  can't show what wasn't declared.
- Zero numbers across the board for every instance? See
  [`forge doctor`](./doctor.md) — usually means `lxc query` isn't
  reachable or the project doesn't exist on the LXD side.

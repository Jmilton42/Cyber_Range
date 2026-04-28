# LXD Operations Guide

Day-2 operations on the LXD cluster. Everything here assumes the project
already exists (it was created by `forge apply`); these are the tasks
that come up *while* a range is running or *after* an exercise wraps
up — switching projects, snapshotting before a session, draining a
cluster member, or cleaning up orphaned networks.

> The scripts referenced below live in `/home/ceroc/InSPIRE/bin/scripts/`. They are
> simple bash wrappers around `lxc` and `jq`; read them before running
> them.

---

## Quick reference

| Task | Forge command | Underlying script |
|------|---------------|-------------------|
| List your projects                | —                                 | `lxc project list` |
| Switch your active project        | —                                 | `lxc project switch <name>` |
| List instances in a project       | —                                 | `lxc list --project <name>` |
| Bulk start every VM in a project  | `forge start <p>`                 | [`start_vms.sh`](#start--stop-all-vms-in-a-project) |
| Bulk stop every VM in a project   | `forge stop <p>`                  | [`stop_vms.sh`](#start--stop-all-vms-in-a-project) |
| Daily snapshot every VM           | `forge snapshot <p>`              | [`snapshot.sh`](#snapshot-every-vm-in-a-project) |
| Migrate every VM to another node  | `forge migrate <p> <target>`      | [`move_vms.sh`](#migrate-every-vm-to-another-cluster-member) |
| Drain VMs from one specific node  | `forge migrate <p> <target> --source <src>` | [`move_vms_nodes.sh`](#drain-a-single-node-move-only-the-vms-on-one-host) |
| Delete networks by prefix         | `forge networks prune <prefix>`   | [`remove_networks.sh`](#delete-networks-by-prefix) |
| Register an existing project      | `forge import <p>`                | — |
| Delete an entire project          | [forge destroy](#deleting-an-entire-project), then `lxc project delete` | — |

> Prefer the `forge` wrappers over the raw scripts. They add a per-instance
> preview, an interactive confirmation, and uniform exit codes — the
> scripts themselves are exactly what gets executed.

---

## LXD projects

### What an LXD project is

An LXD project is a namespace. Every range we deploy lives in its own
project — instances, OVN networks, and storage volumes are scoped to
that project and are invisible from any other project. Authorization is
also per-project on the cluster.

You will work with three or four projects regularly:

| Project | Source | Use |
|---------|--------|-----|
| `default`            | Built-in           | Cluster-wide images, profiles, base networks. Do not deploy ranges here. |
| `<your range>`       | `forge apply`      | One per range. e.g. `CSC-3410`, `CIG-Lab`, `DCIGS`, `CPTC-Mock`. |

### Listing projects

```bash
lxc project list
```

Shows every project, which features are enabled per-project, and which
project your current shell is using.

### Switching the active project

`lxc` commands act on whichever project your shell is currently
"switched" to. Get into the habit of running:

```bash
lxc project switch <project-name>
```

…before any of the bulk operations below. Most of the scripts in
`/home/ceroc/InSPIRE/bin/scripts/` will switch for you, but `lxc list` / `lxc info`
without `--project` will quietly show the wrong project's data.

If you want to act on a single command without changing your active
project, use `--project`:

```bash
lxc list --project CSC-3410
lxc info CSC-3410-team1-Guac --project CSC-3410
```

### Inspecting a project

```bash
# What instances exist?
lxc list --project <name>

# What networks exist?
lxc network list --project <name>

# What's the project config?
lxc project show <name>
```

For range projects, the `features.*` flags should match what the
`lxd_project` resource declares in `main.tf`:

```yaml
config:
  features.images: "false"      # uses cluster-wide images
  features.networks: "false"    # OVN networks live cluster-wide
  features.profiles: "false"    # uses cluster-wide profiles
  features.storage.volumes: "true"
  features.storage.buckets: "true"
```

If a project drifts from this, the next `forge apply` will try to fix
it. Investigate before forcing.

---

## Bulk VM operations

The scripts in `/home/ceroc/InSPIRE/bin/scripts/` are small, deliberately readable
bash. They all take the **project name** as their first argument and
walk every instance in that project.

### Start / Stop all VMs in a project

`start_vms.sh` and `stop_vms.sh` are mirror images of each other. Use
them at the start and end of an exercise to turn a whole project on or
off without `forge apply` / `forge destroy`.

**Preferred (Forge wrappers):**

```bash
forge start CSC-3410
forge stop  CSC-3410
```

**Direct script invocation (equivalent):**

```bash
# Start every VM in the CSC-3410 LXD project
/home/ceroc/InSPIRE/bin/scripts/start_vms.sh CSC-3410

# Stop every VM in CSC-3410 (force-stop, won't wait for clean shutdown)
/home/ceroc/InSPIRE/bin/scripts/stop_vms.sh CSC-3410
```

Behavior:

- Both switch your active project to the named project (`lxc project
  switch`).
- Both pause 15 seconds between instances to keep the cluster from
  thundering. For a 65-team project that is roughly 30 minutes — plan
  accordingly.
- `stop_vms.sh` uses `lxc stop --force`, which kills the VM hard. Run
  `lxc stop` manually if you need clean shutdowns (it returns
  immediately if a VM is already down).
- `start_vms.sh` reports "Failed to start" if a VM is already running —
  that is expected, not an error.

When **not** to use these:

- You actually want to *destroy* the VMs. Use `forge destroy` instead;
  these scripts only stop the runtime.
- You want to start a single team. Use `lxc start <instance> --project
  <name>` directly.

### Snapshot every VM in a project

`snapshot.sh` takes a timestamped snapshot of every instance in a
project. Run it before risky exercises (live attack/defense, image
upgrades, kernel changes) so you can roll back to a known state.

**Preferred:**

```bash
forge snapshot CSC-3410
```

**Direct script invocation:**

```bash
/home/ceroc/InSPIRE/bin/scripts/snapshot.sh CSC-3410
```

Snapshot naming: `daily-snapshot-YYYYMMDDHHMM`. The timestamp is
captured once when the script starts, so every instance in a single run
shares the same snapshot name — useful for `lxc restore` later.

To restore a single instance to that snapshot:

```bash
lxc restore <instance> daily-snapshot-202604270830 --project CSC-3410
```

To delete a snapshot (snapshots take real disk; clean them up):

```bash
lxc delete <instance>/daily-snapshot-202604270830 --project CSC-3410
```

`snapshot.sh` does **not** clean up old snapshots. Operations team is
responsible for retention.

### Migrate every VM to another cluster member

`move_vms.sh` walks every instance in a project and migrates it to the
target cluster member you pass in. Use it to drain a node before
maintenance, or to rebalance after a host comes back online.

**Preferred (Forge wrapper):**

```bash
forge migrate CSC-3410 micro-06
```

The forge wrapper queries `lxc list` first, prints the affected
instances and their current node, and prompts before it does anything.
Pass `-yes` (or `-y`) to skip the prompt in scripted use.

**Direct script invocation:**

```bash
# Argument order: <project> <target-node>
/home/ceroc/InSPIRE/bin/scripts/move_vms.sh CSC-3410 micro-06
```

Important caveats:

- VMs are **stopped** during migration. Cold migration is what `lxc
  move` does for VMs by default. Plan a maintenance window.
- The script pauses 10 seconds between instances. A full team-65 project
  takes ~15 minutes baseline, plus the time to copy each VM's disk over
  the cluster network.
- Migrating a VM updates LXD state but does **not** update the OpenTofu
  state. The next `forge plan` will see drift and try to set
  `target = "@Cluster-C"` back. To make the migration permanent, also
  update the `target = …` line in `main.tf` and re-apply.

To migrate a single instance manually instead of running the script:

```bash
lxc project switch CSC-3410
lxc stop  CSC-3410-team5-Guac
lxc move  CSC-3410-team5-Guac --target micro-06
lxc start CSC-3410-team5-Guac
```

### Drain a single node (move only the VMs on one host)

`move_vms_nodes.sh` is the surgical version of `move_vms.sh`. Where
`move_vms.sh` migrates *every* instance in a project to one hard-coded
target, `move_vms_nodes.sh` only moves the instances that are *currently
running on a specific source node* and lets you choose the destination.

This is the right tool when one cluster member is having problems
(disk pressure, network flapping, hardware fault) and you need to evacuate
just that node's share of a project — leaving instances on healthy
members alone.

**Preferred (Forge wrapper):**

```bash
# Drain micro-05 to micro-01 for the CIG-Lab project
forge migrate CIG-Lab micro-01 --source micro-05
```

The same wrapper drives both modes: when `--source` is set, forge
delegates to `move_vms_nodes.sh`; when omitted, it delegates to
`move_vms.sh`.

**Direct script invocation:**

```bash
/home/ceroc/InSPIRE/bin/scripts/move_vms_nodes.sh CIG-Lab micro-01 micro-05

# Argument order:
#   $1 = project name
#   $2 = target node (where instances should land)
#   $3 = source node (where instances live now)
```

Behavior:

- Filters `lxc list --format json` by `.location == "<source>"`, so
  instances on every other cluster member are untouched.
- Stops, moves, then starts each matched instance, with a 10-second pause
  between iterations. Cold migration only — `lxc move` for VMs requires
  the instance to be stopped.
- Exits cleanly with "Nothing to do" if no instances on the source node
  match the project (so it's safe to script in monitoring loops).
- Refuses to run with an empty argument and prints a usage message.

Same OpenTofu drift caveat applies: the next `forge plan` will see
`target = "@<original>"` in `main.tf` and try to move the instances back.
If the migration is permanent, edit `target = …` in `main.tf` to match
the new node and re-apply. If you're just temporarily evacuating a
member during maintenance, plan to migrate back once the original host
is healthy.

When to reach for `move_vms_nodes.sh` vs `move_vms.sh`:

| Scenario | Use |
|----------|-----|
| Whole project being relocated to one new node           | `move_vms.sh`       |
| One cluster member is sick, drain just its VMs          | `move_vms_nodes.sh` |
| Rebalancing — only some teams need to move              | `move_vms_nodes.sh` |
| Maintenance window on a specific host                   | `move_vms_nodes.sh` |

### Delete networks by prefix

After a botched apply, after manual `lxc network create` experiments, or
after a `forge destroy` that was interrupted, you can end up with
orphaned OVN networks lying around. `remove_networks.sh` cleans them up
by name prefix — and so does `forge networks prune`, which is just a
thin wrapper around it.

**Preferred (Forge wrapper):**

```bash
# Dry run — list what would be deleted
forge networks prune CSC-3410- --project CSC-3410 --dry-run

# Delete with interactive confirmation (re-type the prefix)
forge networks prune CSC-3410- --project CSC-3410

# Delete without prompting (CI / scripted)
forge -yes networks prune CSC-3410- --project CSC-3410
```

**Direct script invocation:**

```bash
/home/ceroc/InSPIRE/bin/scripts/remove_networks.sh CSC-3410- --project CSC-3410 --dry-run
/home/ceroc/InSPIRE/bin/scripts/remove_networks.sh CSC-3410- --project CSC-3410
/home/ceroc/InSPIRE/bin/scripts/remove_networks.sh CSC-3410- --project CSC-3410 --yes
```

Flags:

| Flag | Default | Meaning |
|------|---------|---------|
| `<prefix>`     | required           | Networks whose name starts with this string are deleted. |
| `--project`    | current project    | Operate inside a specific LXD project. |
| `--dry-run`    | off                | Print matches, change nothing. |
| `--yes`        | off                | Skip the "type the prefix to confirm" prompt. |

Behavior worth knowing:

- The script lists every network in the project, filters by prefix, and
  prints them before doing anything destructive. It then asks you to
  re-type the prefix before it will delete.
- If a network is in use by an instance or profile, `lxc network
  delete` will fail. The script logs a warning and continues to the
  next network. **Stop and investigate** any failures — they usually
  mean a stale instance reference.
- The matching is *prefix*, not regex. Pick a specific prefix like
  `CSC-3410-` (with the trailing dash) to avoid accidentally deleting
  `CSC-3410-archive` or similar.

When **not** to use this script:

- During a live `forge apply` — let it finish or fail cleanly first.
- On `default` project networks. The cluster uplink networks
  (`CLASS_WAN`, `DCIG_WAN`, `GUAC_WAN`, `internal_link5`) live in
  `default` and must not be deleted.

---

## Deleting an entire project

The supported way to remove a project is **always**:

```bash
cd /home/ceroc/InSPIRE/<track>/<project>      # e.g. CIG/OCIG/Win-lin
forge destroy
```

`forge destroy` runs `tofu destroy`, which removes every resource
declared in `main.tf` and frees the guac subnet octet in
`subnets.json`. Once it returns successfully, the project is empty and
you can remove the project itself:

```bash
lxc project delete <project-name>
```

If `forge destroy` fails partway through (network failure, stuck
instance), fix the underlying issue and re-run `forge destroy`. It is
idempotent.

### Recovering from a stuck destroy

If `tofu destroy` is stuck on a single resource:

1. Identify the stuck resource from the tofu output.
2. Try the equivalent `lxc` command manually:

   ```bash
   lxc stop <instance> --force --project <name>
   lxc delete <instance> --project <name>
   ```

3. Tell tofu to forget the now-deleted resource:

   ```bash
   tofu state rm 'lxd_instance.guac[12]'
   ```

4. Re-run `forge destroy`.

If the *project* itself will not delete because LXD claims something is
still inside:

```bash
# What is left?
lxc list      --project <name>
lxc network list --project <name>
lxc storage volume list --all-projects | grep <name>
```

Clean up anything found, then `lxc project delete <name>`.

---

## Cluster member operations

These are platform-team tasks rather than range-engineer tasks, but they
come up often enough to document.

### List cluster members

```bash
lxc cluster list
```

Look for the `STATUS` column. Anything that is not `Online` will not
accept new workloads — `forge apply` against a target on that member
will hang.

### Drain a member for maintenance

1. Migrate every project that targets that member off it. The cleanest
   way is `forge migrate <project> <healthy-node> --source <member>`
   per affected project — this only touches instances actually living
   on the member you're draining.
2. Mark the member as evacuated:

   ```bash
   lxc cluster evacuate <member-name>
   ```

3. Do the maintenance.
4. Restore the member:

   ```bash
   lxc cluster restore <member-name>
   ```

5. Optionally migrate workloads back. Most ranges happily stay on
   whatever member they ended up on — only migrate back if there is a
   specific reason (capacity, OVN locality, etc.).

---

## Image management

Every range references pre-baked images by name (`openwrt-team-new`,
`guac-xfce4-v02`, `windows-10-base`, etc.). Images live in the `default`
LXD project and are available cluster-wide because every range project
sets `features.images = false`.

```bash
# List available images
lxc image list

# Show full info for one image
lxc image show <fingerprint-or-alias>
```

Building a new image is out of scope for this guide — see the InSPIRE
runbook. The relevant rule for range engineers: **do not delete or
rebuild an image while any project is mid-exercise**. New instances
created from a freshly rebuilt image will not match the existing
fleet's behavior.

---

## Troubleshooting

| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| `lxc list` shows nothing                       | Wrong active project              | `lxc project switch <name>` or pass `--project <name>` |
| `forge apply` hangs on instance create         | Cluster member offline or out of disk | `lxc cluster list`, check `lxc storage list` |
| Network delete fails: "in use by instance"     | Orphaned instance still attached  | `lxc list --project … --columns nN` to find it; delete it first |
| `forge destroy` deletes most resources, leaves networks | OVN logical-switch cleanup race | Re-run `forge destroy`. If it still fails, use `remove_networks.sh` |
| Snapshot fails: "instance is running"          | LXD requires --stateful for running VMs | Stop the instance first, or pass `--stateful` to `lxc snapshot` |
| Migration fails: "no such cluster member"      | Target name typo or member offline | `lxc cluster list` to confirm the member exists and is `Online` |
| `move_vms.sh` migrated everything, but `forge plan` shows drift | `target` in HCL still points at the old member | Update `target = "@<new-member>"` in `main.tf` and `forge apply` |

---

## See also

- [Forge CLI reference](./forge.md) — the wrapper that drives every
  project lifecycle
- [Building Infrastructure](./infrastructure.md) — when you should be
  reaching for `forge` instead of `lxc`
- [Setup Guide](./setup.md) — initial install of the Forge binary,
  config server, and per-OS clients

[← Back to docs index](./README.md)

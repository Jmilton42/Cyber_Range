# forge apply

Full deployment: allocates the project's subnet, runs `tofu apply`,
exports the LXD instance list, starts the HTTP config server, and kicks
off Windows VM bootstrap. The single command that turns "I edited
`main.tf`" into "the range is live."

## Synopsis

```
forge apply [tofu apply flags...]
```

Any `tofu apply` flag passes through (`-auto-approve`, `-parallelism=N`,
`-target=...`, `-var-file=...`).

## Why use it

`forge apply` is the way you **deploy** a range. Compared to running
`tofu apply` directly:

- Subnet allocation is automatic and persisted, so two operators don't
  collide.
- `instances.json` is generated for the config server so newly-booted
  Windows VMs can fetch hostnames/IPs.
- The config server is started in the background (idle-timeout aware) so
  you don't have to remember a separate `forge serve` call.
- Windows VMs are auto-started via the platform's `start_win.sh`.

## What it does

1. Reads `project_name` from `main.tf`.
2. Allocates the next available subnet octet (idempotent if already
   allocated for this project).
3. Prints a deployment banner.
4. Runs `tofu apply -var project_name=X -var guac_subnet_octet=Y` (plus
   any flags you passed).
5. Sleeps a few seconds for VMs to come up.
6. Switches to the LXD project and exports `instances.json`.
7. Detaches a `forge serve` child process to host the config server.
8. Invokes `/home/ceroc/InSPIRE/bin/scripts/start_win.sh` to boot
   Windows VMs.
9. Prints the deployment summary (server URL, idle timeout, endpoints).

## Examples

```bash
# Standard deploy with confirmation
forge apply

# Skip the "Do you want to perform these actions?" prompt
forge apply -auto-approve

# Faster apply on a beefy host
forge apply -parallelism=20

# Apply only one VM (useful when iterating on a single host)
forge apply -target=lxd_instance.dc01

# Deploy a project living elsewhere
forge -chdir=/home/ceroc/InSPIRE/CIG/OCIG/Win-lin apply -auto-approve
```

## Output

```
==========================================
  Cyber Range Deployment
  Project: ocig-win-lin
==========================================

[INFO] Project: ocig-win-lin
[INFO] Subnet:  10.0.4.0/24 (gateway: 10.0.4.1)

... tofu apply output ...

[INFO] Waiting for VMs to initialize...
[INFO] Switching to LXD project ocig-win-lin
[INFO] Exported 12 instances to instances.json
[INFO] Started config server (PID 23145)
[INFO] Starting Windows VMs...

Deployment complete!

Server running at: http://10.0.14.6:8080
Idle timeout:      5m
Subnet:            10.0.4.0/24
```

## Related

- [`forge plan`](./plan.md) — preview before applying.
- [`forge logs -f`](./logs.md) — tail the config server.
- [`forge status`](./status.md) — confirm allocation.
- [`forge destroy`](./destroy.md) — tear it all back down.

## Notes

- The config server runs as a detached `forge serve` child. It exits
  automatically after the configured idle timeout (default 5m); restart
  it with [`forge serve`](./serve.md) if you need it back.
- If `start_win.sh` isn't present (you're not on the canonical
  CEROC host), forge prints a warning but `tofu apply` still succeeds.
- A failing `tofu apply` exits non-zero **before** any post-apply step
  runs, so a botched deploy never leaves a stale config server.

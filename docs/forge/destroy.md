# forge destroy

Full teardown: stops the config server, runs `tofu destroy`, and
releases the subnet so somebody else can use it.

## Synopsis

```
forge destroy [tofu destroy flags...]
```

Any `tofu destroy` flag passes through (`-auto-approve`,
`-parallelism=N`, `-target=...`).

## Why use it

Compared to `tofu destroy`:

- The config server is stopped first, so it can't keep serving stale
  hostnames during the rollback.
- The project's subnet is released back into the pool — without this,
  octets accumulate over a semester until you run out.
- You get a single command instead of a 3-step manual ritual.

## What it does

1. Reads `project_name` from `main.tf`.
2. Looks up the project's subnet octet in `subnets.json` (errors if
   none).
3. Prints a teardown banner.
4. Stops the running config server (best-effort; ignored if not running).
5. Runs `tofu destroy -var project_name=X -var guac_subnet_octet=Y` plus
   any flags you passed.
6. Removes the project from `subnets.json`.
7. Prints a confirmation that the octet is now free.

## Examples

```bash
# Standard interactive destroy
forge destroy

# Skip the confirmation prompt
forge destroy -auto-approve

# Destroy a single resource (subnet stays allocated)
forge destroy -target=lxd_instance.kali

# From outside the project
forge -chdir=/home/ceroc/InSPIRE/Classes/CSC-3410-CS destroy -auto-approve
```

## Output

```
==========================================
  Cyber Range Destroy
  Project: ocig-win-lin
==========================================

[INFO] Project: ocig-win-lin
[INFO] Subnet:  10.0.4.0/24 (will be released after destroy)

... tofu destroy output ...

[INFO] Releasing subnet allocation...
[INFO] Released subnet 10.0.4.0/24

[INFO] Destroy complete!
[INFO] Subnet 10.0.4.0/24 has been released and is available for reuse.
```

## Related

- [`forge stop`](./stop.md) — stop VMs without tearing them down.
- [`forge subnets free <project>`](./subnets.md) — release a subnet
  manually if `tofu destroy` already happened (e.g. you nuked the
  project from the LXD side).
- [`forge apply`](./apply.md) — bring the project back.

## Notes

- If you used `-target` to destroy only part of the project, the subnet
  is **not** released — that's intentional.
- A failing `tofu destroy` does **not** release the subnet either.
  Re-run, or release manually with
  [`forge subnets free <project>`](./subnets.md) once the LXD side is
  clean.
- Releasing the subnet does not touch any LXD resources directly. It
  just removes a row from `subnets.json`.

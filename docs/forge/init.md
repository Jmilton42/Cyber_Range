# forge init

Prepare a working directory so other forge commands have everything they
need: a usable `subnets.json` and an initialized `.terraform` directory
for OpenTofu provider downloads.

## Synopsis

```
forge init [tofu init flags...]
```

Any flags after `init` are passed through to `tofu init`, so existing
muscle memory (e.g. `-upgrade`, `-reconfigure`, `-backend-config=...`)
works unchanged.

## Why use it

You run this **once** per project directory, before your first
`forge plan` or `forge apply`. Without it:

- Forge has nowhere to record subnet allocations — every `apply` would
  race or fail.
- OpenTofu hasn't downloaded the providers yet, so `plan`/`apply` will
  error on missing modules.

## What it does

1. Creates `/home/ceroc/InSPIRE/bin/guac_subnet/subnets.json` (and its
   parent dir) if it does not already exist. Existing allocations are
   never touched.
2. Runs `tofu init` in the current directory (or the directory passed via
   `-chdir=DIR`).

## Examples

```bash
# Standard first-time init in a project directory
cd /home/ceroc/InSPIRE/CIG/OCIG/Win-lin
forge init

# From anywhere, against a specific directory
forge -chdir=/home/ceroc/InSPIRE/Classes/CSC-3410-CS init

# Pass a tofu init flag through
forge init -upgrade
```

## Output

```
[INFO] Initializing subnets file...
[INFO] Subnets file ready: /home/ceroc/InSPIRE/bin/guac_subnet/subnets.json
[INFO] Running tofu init...
... tofu output ...
```

## Related

- [`forge new`](./new.md) — scaffold a fresh project, then run `forge init` inside it.
- [`forge plan`](./plan.md) — natural next step after `init`.
- [`forge doctor`](./doctor.md) — run this if `init` fails or you're not sure why.

## Notes

- Safe to re-run. It's a no-op if `subnets.json` already exists, and
  `tofu init` is itself idempotent.
- Requires write access to `/home/ceroc/InSPIRE/bin/guac_subnet/`. If you
  hit a permission error, see the
  [troubleshooting section](../forge.md#troubleshooting).

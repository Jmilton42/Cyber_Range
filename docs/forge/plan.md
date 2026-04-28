# forge plan

Preview the changes a `forge apply` would make. Allocates this project's
subnet (idempotently) so the plan reflects the real values that `apply`
will use.

## Synopsis

```
forge plan [tofu plan flags...]
```

Any `tofu plan` flag passes through (`-out=FILE`, `-detailed-exitcode`,
`-target=...`, etc.).

## Why use it

`plan` is the safety net between editing `main.tf` and pushing changes
to the cluster. Specifically with forge:

- The subnet octet is locked in **here** — running `plan` first means
  the eventual `apply` won't reserve a different octet from underneath
  another teammate doing the same thing.
- You see exactly which LXD instances, networks, snapshots, and OVN
  resources would be added/changed/destroyed before any of it happens.

## What it does

1. Reads `project_name` from `main.tf`.
2. Idempotently allocates a subnet octet for the project (writes
   `subnets.json` if it's a new project; reuses the existing octet
   otherwise).
3. Runs `tofu plan -var project_name=X -var guac_subnet_octet=Y` plus
   any flags you passed.

## Examples

```bash
# Preview deployment
forge plan

# Save the plan to a file for `apply` to consume
forge plan -out=tfplan

# Plan only one resource
forge plan -target=lxd_instance.kali

# From outside the project
forge -chdir=/home/ceroc/InSPIRE/CIG/OCIG/Win-lin plan
```

## Output

The header includes the resolved subnet so you can sanity-check it:

```
[INFO] Project: ocig-win-lin
[INFO] Subnet:  10.0.4.0/24 (gateway: 10.0.4.1)

... tofu plan output ...

Plan: 12 to add, 0 to change, 0 to destroy.
```

Exit codes match `tofu plan` (0 = no changes, 2 = changes pending when
`-detailed-exitcode` is set, 1 = error).

## Related

- [`forge validate`](./validate.md) — cheaper syntax-only check.
- [`forge apply`](./apply.md) — actually make the changes.
- [`forge status`](./status.md) — confirm which subnet got allocated.

## Notes

- Subnet allocation persists even if you don't run `apply`. Use
  [`forge subnets free <project>`](./subnets.md) to release it manually
  if you decide not to deploy.
- If `subnets.json` is missing or unreadable, `plan` aborts before
  invoking `tofu`.

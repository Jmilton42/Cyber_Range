# forge validate

Pure passthrough to `tofu validate` for the current project directory.
Use it as a fast syntax check before `plan` or `apply`.

## Synopsis

```
forge validate [tofu validate flags...]
```

## Why use it

`tofu validate` only checks HCL syntax and provider schema — it does not
hit the network or talk to LXD, so it is the cheapest check you can run
while editing `main.tf`. Catching a typo here saves a wasted `plan`.

## What it does

Runs `tofu validate` in the current directory (or `-chdir=DIR`) with no
forge-specific behavior layered on top. No subnets are allocated, no
config server is touched.

## Examples

```bash
# Inside a project
forge validate

# Against a project elsewhere
forge -chdir=/home/ceroc/InSPIRE/CIG/OCIG/Win-lin validate
```

## Output

```
Success! The configuration is valid.
```

or, on failure, a normal `tofu` error message. Exit code matches `tofu`.

## Related

- [`forge plan`](./plan.md) — natural next step.
- [`forge doctor`](./doctor.md) — sanity-check your environment if `validate` fails for non-syntax reasons.

## Notes

- Does **not** require a subnet allocation — safe to run before
  `forge init`.
- For HCL formatting issues, run `tofu fmt` directly; forge does not wrap it.

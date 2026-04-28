# forge new

Scaffold a brand-new project directory from a template stored at
`/home/ceroc/InSPIRE/templates/`. Copies the template, rewrites
`project_name`, and (optionally) reserves a subnet up front.

## Synopsis

```
forge new [--template <id>] [--name <name>] [--dir <path>] [--reserve | --no-reserve] [--yes]
```

Run with no flags inside a TTY to get an interactive picker.

## Why use it

Creating a new project by hand means:

1. Copying a working `main.tf` from another project.
2. Remembering to rename every `project_name` reference.
3. Picking a free subnet octet by reading `subnets.json`.
4. Hoping you didn't miss any of the above.

`forge new` does all of it in one step and refuses to overwrite an
existing directory, so you can't accidentally trash a deployed project.

## What it does

1. Validates the project name against `^[A-Za-z0-9][A-Za-z0-9_-]{1,40}$`
   (mixed case allowed, e.g. `CSC-4100-Test`).
2. Refuses if the target directory already exists.
3. Resolves the requested template via the manifest lookup chain (see
   below).
4. Copies the template directory into the target.
5. Rewrites the `default = "..."` value of `project_name` in `main.tf`.
6. If `--reserve` (default), allocates a subnet octet immediately so a
   teammate running `forge new` a minute later can't grab the same one.
7. Prints the recommended next steps (`cd ... && forge plan && forge apply`).

## Examples

```bash
# Interactive picker
forge new

# Scripted / CI-friendly (template id = the directory name)
forge new \
  --template CSC-3410-single \
  --name fall26-csc3410-section-a \
  --dir range/fall26-csc3410-section-a \
  --reserve --yes

# Minimal blank project (synthetic template, always available)
forge new --template blank --name my-experiment

# Skip the up-front reservation if you want to plan before committing
forge new --template win-lin-Muli --name my-test-lab --no-reserve
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--template <id>` | (prompted) | Template id (= directory name under the templates dir, or the synthetic `blank`). Case-insensitive. |
| `--name <name>` | (prompted) | New project name. Lowercase + dashes, 2–41 chars. |
| `--dir <path>` | `./<name>` | Target directory (relative to cwd). Must not exist. |
| `--reserve` | on | Allocate a subnet octet up front. |
| `--no-reserve` | off | Skip the up-front allocation; the next `forge apply` will allocate normally. |
| `--yes` / `-y` | off | Disables the interactive picker; required when stdin isn't a TTY. |

## Where templates come from

Forge discovers templates by **scanning a directory**. Every immediate
subdirectory that contains a `main.tf` becomes an available template;
its directory name is the template id you pass to `--template`. There
is **no manifest file** — drop a folder in, it shows up. Delete it, it's
gone.

```
/home/ceroc/InSPIRE/templates/
├── CSC-3410-single/
│   └── main.tf
├── win-lin-Muli/
│   └── main.tf
└── CPTC - Muli/
    └── main.tf
```

A built-in synthetic template called `blank` is always available — it
writes a minimal `main.tf` with just the variables forge needs.

### Skipped automatically

- Hidden directories (names starting with `.`, e.g. `.git`).
- Directories with no `main.tf` (so `docs/`, `assets/`, etc. don't
  show up).

### Lookup order for the templates dir

1. `$FORGE_TEMPLATES` if it points at an existing directory.
2. `/home/ceroc/InSPIRE/templates/` (the canonical production location).
3. A `templates/` or `range/` directory found by walking up from cwd
   (developer convenience inside the repo).
4. None — only the synthetic `blank` template is offered.

[`forge doctor`](./doctor.md) reports the resolved templates dir and
how many templates were discovered there.

## Output

```
[INFO] Created range/fall26-csc3410-section-a
[INFO] Set project_name = "fall26-csc3410-section-a"
[INFO] Reserved subnet octet 7 (10.0.7.0/24)

Next steps:
  cd range/fall26-csc3410-section-a
  forge plan
  forge apply
```

## Adding a new template

1. Create a new subdirectory under the templates dir.
2. Drop a working `main.tf` inside it (any other files will be copied
   too).
3. That's it. The directory name is the template id; `forge new` and
   tab-completion both pick it up immediately.

```bash
mkdir /home/ceroc/InSPIRE/templates/My-New-Template
cp existing-main.tf /home/ceroc/InSPIRE/templates/My-New-Template/main.tf
forge new --template My-New-Template --name try-it-out
```

## Related

- [`forge init`](./init.md) — run inside the new directory before your first plan.
- [`forge subnets`](./subnets.md) — manage the allocation reserved by `--reserve`.

## Notes

- `--reserve` and `--no-reserve` are mirror flags; passing `--no-reserve`
  takes precedence.
- Tab-completion for `--template=<TAB>` reads the manifest at runtime.

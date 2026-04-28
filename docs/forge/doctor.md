# forge doctor

Run a battery of preflight checks on the operator environment. Designed
to surface every "you forgot to install X" / "Y has the wrong perms"
problem before they cause a `forge apply` to half-deploy.

## Synopsis

```
forge doctor [-json]
```

## Why use it

Run this:

- The first time you set up forge on a new host.
- After a server reboot or LXD upgrade.
- Whenever something feels off (apply fails for non-HCL reasons,
  completion is silent, etc.).
- In CI, to gate deployments on a healthy environment.

It exits non-zero if any check **FAILs**, so it's safe to use in scripts:

```bash
forge doctor || { echo "fix doctor before deploying"; exit 1; }
```

## What it checks

| Check | Severity if missing | What it means |
|-------|---------------------|---------------|
| `tofu binary` | FAIL | OpenTofu installed and on `$PATH`. |
| `lxc binary` | FAIL | LXD client installed and on `$PATH`. |
| `jq binary` | WARN | Required by the bundled bash scripts. |
| `subnets.json` | WARN/FAIL | File exists and parses as valid JSON. |
| `config.yaml` | WARN | One of the candidate paths loaded successfully. |
| `start_win.sh` | WARN | Windows VM start script present. |
| `lxd_scripts` | WARN | Bash wrappers (snapshot.sh / start_vms.sh / etc.) reachable. |
| `lxc cluster` | WARN | `lxc cluster list` runs and parses. |
| `templates` | WARN | A templates dir was found and reports the count of template subdirectories discovered. |
| `plugins` | WARN | Every bundled `forge-*` plugin is on `$PATH`. |
| `config server` | WARN | `http://<server>:<port>/status` responds. |

`OK` means everything is fine. `WARN` means it's not blocking but you
should know. `FAIL` means a basic capability is missing and `apply`
will not work.

## Examples

```bash
# Human-friendly output
forge doctor

# CI pipeline
forge doctor -json | jq '.healthy' | grep -q true || exit 1
```

## Output

```
[OK]   tofu binary          /usr/local/bin/tofu
[OK]   lxc binary           /usr/bin/lxc
[OK]   jq binary            /usr/bin/jq
[OK]   subnets.json         /home/ceroc/InSPIRE/bin/guac_subnet/subnets.json (3 allocations)
[OK]   config.yaml          loaded successfully
[OK]   start_win.sh         /home/ceroc/InSPIRE/bin/scripts/start_win.sh
[OK]   lxd_scripts          all wrappers present in /home/ceroc/InSPIRE/bin/scripts
[OK]   lxc cluster          5 members
[OK]   templates            5 template(s) discovered in /home/ceroc/InSPIRE/templates
[WARN] plugins              missing on $PATH: [forge-snapshot] - run forge/scripts/build_all.sh
[OK]   config server        http://10.0.14.6:8080/status responded 200

Summary: 10 ok, 1 warn, 0 fail
```

## JSON shape

```json
{
  "checks": [
    {"name":"tofu binary","status":"OK","message":"/usr/local/bin/tofu"},
    ...
  ],
  "ok": 10,
  "warn": 1,
  "fail": 0,
  "healthy": true
}
```

`healthy` is `true` iff `fail == 0`. WARNs do not flip it.

## Exit codes

| Exit | Meaning |
|------|---------|
| `0` | No FAILs (warnings allowed). |
| `1` | At least one FAIL. |

## Related

- [`forge config`](./config.md) — see exactly what config values doctor checked against.
- [`forge plugins list`](./plugins.md) — see which `forge-*` binaries got found.

## Notes

- Read-only. Doctor never modifies anything.
- Doctor opens a 2-second HTTP request to the config server's `/status`
  endpoint. A WARN here often just means "the server isn't running yet,"
  which is expected before your first `forge apply`.

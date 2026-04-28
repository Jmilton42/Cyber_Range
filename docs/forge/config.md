# forge config

Print the resolved deploy configuration: every value forge would use,
plus where it came from (embedded default vs. on-disk `config.yaml` vs.
environment override).

## Synopsis

```
forge config [-json]
```

## Why use it

When `forge apply` does something unexpected — picks a different config
server IP, writes to the wrong subnets file, or invokes a script you
didn't expect — the answer is almost always "you have a `config.yaml`
overriding the default and didn't know about it." This command is the
fastest way to confirm what's actually loaded.

## What it does

1. Walks the candidate `config.yaml` paths (in order):
   - `$FORGE_CONFIG`
   - `./config.yaml`
   - `~/.config/forge/config.yaml`
   - `/etc/forge/config.yaml`
   - the canonical install path under `/home/ceroc/InSPIRE/`
2. Loads the first one that exists and merges it on top of the embedded
   defaults.
3. Applies any environment variable overrides
   (`FORGE_SERVER_IP`, `FORGE_SERVER_PORT`, etc.).
4. Prints the resulting effective config.

## Examples

```bash
# Human-friendly
forge config

# Diff effective vs. file
forge config -json | diff <(yq -o json . config.yaml) -

# Confirm an env override is applied
FORGE_SERVER_PORT=9000 forge config | grep "Server Port"
```

## Output

```
Loaded config: /home/ceroc/.config/forge/config.yaml

Server IP:        10.0.14.6
Server Port:      8080
Idle Timeout:     5m
Instances File:   instances.json
Start Win Script: /home/ceroc/InSPIRE/bin/scripts/start_win.sh
Subnets File:     /home/ceroc/InSPIRE/bin/guac_subnet/subnets.json
Templates Dir:    /home/ceroc/InSPIRE/templates
LXD Scripts Dir:  /home/ceroc/InSPIRE/bin/scripts
```

If no file is found:

```
Loaded config: <embedded defaults>
...
```

## JSON shape

```json
{
  "loaded_from": "/home/ceroc/.config/forge/config.yaml",
  "server_ip": "10.0.14.6",
  "server_port": 8080,
  "idle_timeout": "5m",
  "instances_file": "instances.json",
  "start_win_script": "/home/ceroc/InSPIRE/bin/scripts/start_win.sh",
  "subnets_file": "/home/ceroc/InSPIRE/bin/guac_subnet/subnets.json",
  "templates_dir": "/home/ceroc/InSPIRE/templates",
  "lxd_scripts_dir": "/home/ceroc/InSPIRE/bin/scripts"
}
```

## Related

- [`forge doctor`](./doctor.md) — same values, but it also tries to
  reach each one (`lxc cluster list`, `curl /status`, etc.).
- [`forge serve`](./serve.md) — the server reads this same config.

## Notes

- Read-only.
- `loaded_from` will report `<embedded defaults>` if no `config.yaml`
  is on disk. That's the supported way to run forge — the embedded
  values match the canonical CEROC layout.

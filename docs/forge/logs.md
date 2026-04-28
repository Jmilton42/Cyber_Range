# forge logs

Print or tail `server.log` for the current project — i.e. the output of
the [`forge serve`](./serve.md) instance that `forge apply` started.

## Synopsis

```
forge logs [-f] [-n <lines>]
```

## Why use it

When a Windows VM is failing to boot, the most useful clue is whether it
ever hit the config server. `forge logs -f` shows you incoming requests
in real time. Use this instead of remembering the absolute path to
`server.log`.

## What it does

1. Looks for `server.log` in the current directory.
2. With `-n N`, prints the last `N` lines and exits.
3. With `-f`, prints the last `N` lines and then follows the file
   (`tail -f`-style) until you Ctrl-C.
4. If the file doesn't exist, prints a hint that the server has not been
   started.

## Examples

```bash
# Last 50 lines (default)
forge logs

# Last 200 lines
forge logs -n 200

# Live tail while VMs are booting
forge logs -f

# From outside the project
forge -chdir=/home/ceroc/InSPIRE/CIG/OCIG/Win-lin logs -f
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-f` | off | Follow the file (continuously print new lines). |
| `-n <lines>` | `50` | Number of trailing lines to print before following. |

## Output

```
[INFO] Loaded 12 instances from instances.json
[INFO] Listening on 10.0.14.6:8080 (idle timeout 5m)
[INFO] GET / from 10.0.4.10  (kali-1)
[INFO] GET / from 10.0.4.11  (dc01)
[INFO] GET /status from 10.0.14.6
```

## Related

- [`forge serve`](./serve.md) — start (or restart) the server.
- [`forge reload`](./reload.md) — re-read `instances.json` without
  restarting.

## Notes

- Read-only. No log rotation is performed; the file just grows. If it
  gets unwieldy, truncate it manually:
  ```bash
  : > server.log
  ```
- `-f` won't recreate the file if you delete it; restart the server
  with `forge serve` to get a fresh log.

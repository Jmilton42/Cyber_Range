# forge serve

Run the HTTP configuration server. This is the small Go service that
range VMs hit during boot to fetch their hostname, IP, role, and
provisioning data — the data exported by [`forge apply`](./apply.md)
into `instances.json`.

## Synopsis

```
forge serve [-addr <ip>] [-port <port>] [-idle <duration>] [-instances <path>]
```

## Why use it

`forge apply` already starts a detached config server, so day-to-day you
don't run `forge serve` yourself. Run it explicitly when:

- Apply finished, the server's idle timeout fired, and a new batch of VMs
  is booting (or rebooting) and needs the API back.
- You're debugging the server in the foreground.
- You're hosting `instances.json` from a non-standard path.

## What it does

1. Loads the deploy config (see [`forge config`](./config.md)).
2. Auto-discovers `instances.json` in the current dir if no
   `-instances` is passed.
3. Binds to `<addr>:<port>` (default `10.0.14.6:8080`).
4. Exposes:
   - `GET /` — JSON list of every instance in `instances.json`.
   - `GET /status` — `{ "ok": true, "instances": N }`.
   - `POST /reload` — re-read `instances.json` from disk
     (used by [`forge reload`](./reload.md)).
5. Resets a 5-minute idle timer on every successful request. Once the
   timer fires with no traffic, the server logs and exits.
6. Writes everything to `server.log` in the cwd.

## Examples

```bash
# Standard restart — uses defaults from config.yaml
forge serve

# One-off on a different port for testing
forge serve -port 9090

# Long-lived server (24h idle window)
forge serve -idle 24h

# Custom instances file
forge serve -instances /tmp/test-instances.json
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-addr <ip>` | from config (`10.0.14.6`) | Listen address. |
| `-port <port>` | from config (`8080`) | Listen port. |
| `-idle <duration>` | from config (`5m`) | Auto-shutdown timer. Reset on every request. |
| `-instances <path>` | `instances.json` (cwd) | LXD instance export to serve. |

## Output

`forge serve` runs in the foreground and prints to `server.log`:

```
[INFO] Loaded 12 instances from instances.json
[INFO] Listening on 10.0.14.6:8080 (idle timeout 5m)
[INFO] GET / from 10.0.4.7
[INFO] GET / from 10.0.4.8
...
[INFO] Idle timeout reached, shutting down
```

`forge apply` invokes `forge serve` as a detached child process so you
don't see this output unless you tail with [`forge logs -f`](./logs.md).

## Endpoints

### `GET /`

```json
[
  {
    "name": "kali-1",
    "ipv4": "10.0.4.10",
    "role": "attacker",
    "project": "ocig-win-lin"
  },
  ...
]
```

### `GET /status`

```json
{ "ok": true, "instances": 12, "uptime": "47s" }
```

### `POST /reload`

Re-reads `instances.json` from disk; returns the new instance count.

```json
{ "ok": true, "instances": 13, "reloaded_at": "2026-04-28T10:30:00-05:00" }
```

## Related

- [`forge logs`](./logs.md) — read or tail `server.log`.
- [`forge reload`](./reload.md) — POST `/reload` after editing `instances.json`.
- [`forge apply`](./apply.md) — the normal way the server gets started.

## Notes

- The server is intentionally simple: no TLS, no auth. It binds to the
  internal management IP (`10.0.14.6` by default) and trusts the network.
- If the process exits unexpectedly, `forge apply` will start a new one
  on the next deploy. To keep one running long-term, use a systemd unit
  with `forge serve -idle 720h` (or similar).

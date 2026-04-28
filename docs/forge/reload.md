# forge reload

POST `/reload` to the running config server so it re-reads
`instances.json` from disk. Use this when you've edited `instances.json`
(for example after a `forge import`) and want VMs to see the new
mapping without restarting the server.

## Synopsis

```
forge reload
```

## Why use it

The config server caches `instances.json` in memory at startup so it can
serve `GET /` without hitting disk. If you edit the file and want the
server to pick up the new contents, you have two choices:

1. Wait for the idle timeout to fire, then run `forge serve` again.
2. Run `forge reload` and the server picks up the change immediately.

`forge reload` is the option you want when VMs are actively making
requests and you can't take a 30-second cache miss.

## What it does

1. Reads the deploy config to figure out the server URL.
2. Issues `POST http://<server>:<port>/reload`.
3. Prints the JSON response or an error if the server isn't reachable.

## Examples

```bash
# Standard reload after editing instances.json
forge reload

# After a forge import
forge import existing-project
forge reload
```

## Output

```
[INFO] POST http://10.0.14.6:8080/reload
[INFO] Server reloaded: 13 instances (was 12)
```

If the server isn't running:

```
[ERROR] connection refused: http://10.0.14.6:8080/reload
[INFO]  Run `forge serve` (or `forge apply`) to start it.
```

## Related

- [`forge serve`](./serve.md) — start the server in the first place.
- [`forge import`](./import.md) — the most common reason you'd be
  editing `instances.json` by hand.
- [`forge logs -f`](./logs.md) — confirm the reload landed.

## Notes

- The `/reload` endpoint resets the server's idle timer, so calling
  `forge reload` keeps the server alive for another idle window.
- No-op if `instances.json` hasn't changed; the server still rebuilds
  its in-memory cache.

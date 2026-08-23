# Contributing

Boop is intentionally small. Before adding a feature, check the non-goals in `prd.md`; anything that needs another infrastructure service is unlikely to be accepted.

## Layout

- `server/` — Go server. `cmd/boop` is the entry point; `internal/` holds one package per concern (api, apns, auth, config, database, delivery, devices, events, pairing, projects, settings, web). `migrations/` holds SQL applied in filename order.
- `server/web/` — Svelte + Vite frontend. `npm run build` writes into `server/internal/web/dist`, which `go:embed` picks up.
- `Design System/` — the design tokens and reference components the UI follows.

## Working locally

```bash
# terminal 1
cd server && go run ./cmd/boop
# terminal 2
cd server/web && npm install && npm run dev   # http://localhost:5173, proxies /api to :8080
```

Set `BOOP_DATABASE_PATH=./data/boop.db` to keep the database in the repo directory instead of `/data`.

## Tests

```bash
make test            # go vet + go test + svelte-check + vitest
```

Go tests spin up the real HTTP handler against a temporary SQLite file. Add a test for every new endpoint or behaviour.

## Conventions

- Log with `slog` and dotted event names (`event.created`, `push.failed`). Never log keys, credentials, or full payloads.
- Sentence case in the UI, no exclamation marks, no emoji in copy.
- Use the design tokens (`--up-*`) rather than hard-coded colours.

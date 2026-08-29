# Contributing

Boop is intentionally small: one Go binary, one SQLite file, direct APNs. Anything that needs another infrastructure service (a queue, a hosted relay, an external database) is unlikely to be accepted.

## Layout

- `server/` — Go server. `cmd/boop` is the entry point; `internal/` holds one package per concern (api, apns, auth, config, database, delivery, devices, events, pairing, projects, settings, web). `migrations/` holds SQL applied in filename order.
- `server/web/` — Svelte + Vite frontend. `npm run build` writes into `server/internal/web/dist`, which `go:embed` picks up.
- `ios/` — the SwiftUI app; `project.yml` generates `Boop.xcodeproj` with XcodeGen. Web colours and type live as CSS variables in `server/web/src/app.css`; the iOS equivalents are in `ios/Boop/Design/DS.swift`.

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

## Changelog

Add a line under **Unreleased** in [CHANGELOG.md](CHANGELOG.md) for anything a user would notice: new endpoints or fields, UI changes, configuration, migrations. Bump the version in `server/internal/api/api.go`, `server/Dockerfile` and `ios/project.yml` (then `xcodegen generate`) when cutting a release, and tag it `vX.Y.Z`.

<p align="center">
  <img src="https://github.com/chrisgreg/boop/raw/main/docs/boop.png" width="160" alt="Boop logo" />

  <h1 align="center">Boop</h1>

<p align="center">
  <img src="https://img.shields.io/github/actions/workflow/status/chrisgreg/boop/ci.yml?branch=main" alt="CI" />
  <img src="https://img.shields.io/github/go-mod/go-version/chrisgreg/boop?filename=server%2Fgo.mod" alt="Go version" />
  <img src="https://img.shields.io/github/license/chrisgreg/boop" alt="License" />
</p>

A tiny, self-hosted notification inbox for developers. Something happened in one of your apps; Boop tells you on your phone.

One Go binary, one SQLite file, one Docker container. Pushes go straight from your server to Apple's APNs. There is no hosted relay, account system, or telemetry.

</p>

```bash
curl https://boop.example.com/api/v1/events \
  -H "Authorization: Bearer $BOOP_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"title": "Backup complete", "level": "success"}'
```

## What is in the box

| Part | Status |
| --- | --- |
| Go server (API, SQLite, APNs, embedded web UI) | `server/` |
| Web UI (Svelte, built into the binary) | `server/web/` |
| iOS app | not yet |
| Elixir client and ErrorTracker integration | not yet |

## Quick start (Docker)

```bash
git clone https://github.com/chrisgreg/boop && cd boop
cp .env.example .env          # optional: BOOP_BASE_URL and APNS_* values
mkdir -p data && chown 1000:1000 data   # Linux hosts only; the container runs as uid 1000
docker compose up -d --build
open http://localhost:8080
```

The first visit opens a setup wizard: server check, APNs, pairing, first project, test notification. APNs credentials are optional; without them events are stored and shown in the UI but pushes are skipped, and the settings page says so.

Data lives in `./data/boop.db`. Back up by copying that file (use `sqlite3 data/boop.db ".backup backup.db"` for a consistent copy while running). Back up your `.p8` key separately.

## Send an event

Create a project in the web UI and copy its API key (shown once). Then:

```bash
# minimum
curl http://localhost:8080/api/v1/events \
  -H "Authorization: Bearer boop_proj_..." -H "Content-Type: application/json" \
  -d '{"title": "Deploy complete"}'

# rich
curl http://localhost:8080/api/v1/events \
  -H "Authorization: Bearer boop_proj_..." -H "Content-Type: application/json" \
  -d '{
    "title": "KeyError", "body": "key :can_palette? not found",
    "level": "error", "source": "error_tracker", "fingerprint": "uini-keyerror",
    "data": {
      "exception": {"type": "KeyError", "message": "key :can_palette? not found"},
      "stacktrace": [{"file": "lib/uini_web/live/widget_settings_live.ex", "line": 49, "function": "handle_event/3", "in_app": true}],
      "tags": {"environment": "production"},
      "context": {"user_id": "123"}
    }
  }'
```

Levels: `info`, `success`, `warning`, `error`, `critical`. Anything in `data` is kept as-is (recognised sections such as `exception`, `stacktrace`, `tags`, `context` and `breadcrumbs` get a nicer rendering) after sensitive keys are redacted.

From a shell script:

```bash
boop() { curl -fsS "$BOOP_URL/api/v1/events" -H "Authorization: Bearer $BOOP_API_KEY" \
  -H "Content-Type: application/json" -d "{\"title\": \"$1\", \"body\": \"${2:-}\", \"level\": \"${3:-info}\"}"; }
pg_dump mydb > backup.sql && boop "Backup complete" "" success || boop "Backup failed" "$(tail -1 backup.log)" error
```

From GitHub Actions:

```yaml
- name: Boop
  if: always()
  run: |
    curl -fsS "${{ secrets.BOOP_URL }}/api/v1/events" \
      -H "Authorization: Bearer ${{ secrets.BOOP_API_KEY }}" \
      -H "Content-Type: application/json" \
      -d '{"title": "${{ github.workflow }} ${{ job.status }}", "body": "${{ github.repository }}@${{ github.ref_name }}", "level": "${{ job.status == 'success' && 'success' || 'error' }}", "source": "github_actions"}'
```

## API

All endpoints are under `/api/v1`. Errors are JSON: `{"error": "code", "message": "..."}`.

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| GET | `/health` | none | `{"status":"ok"}` |
| POST | `/api/v1/events` | project key | Create event, returns `{id, created_at}` |
| GET | `/api/v1/events?project=&level=&source=&before=&limit=` | device or none | List, newest first; `next_cursor` feeds `before` |
| GET | `/api/v1/events/:id` | device or none | Full event |
| GET | `/api/v1/events/:id/deliveries` | device or none | Push attempts for an event |
| GET/POST | `/api/v1/projects` | admin | List / create (returns `api_key` once) |
| GET/PATCH/DELETE | `/api/v1/projects/:id` | admin | Manage |
| POST | `/api/v1/projects/:id/rotate-key` | admin | New key, old one stops working |
| POST | `/api/v1/pairing` | admin | One-time pairing token + QR payload (10 min, single use) |
| DELETE | `/api/v1/pairing/:id` | admin | Revoke |
| POST | `/api/v1/pairing/exchange` | none | `{token, name, platform}` → `{device, credential}` |
| POST | `/api/v1/devices` | device | Register APNs token `{device_token, name, app_bundle_id}` |
| PATCH/DELETE | `/api/v1/devices/:id` | device (self) or admin | Update / remove |
| GET | `/api/v1/devices` | admin | List paired devices |
| GET | `/api/v1/status` | admin | Health, APNs state, counts, last push |
| GET/PATCH | `/api/v1/settings` | admin | `retention_days`, `redact_keys`, `setup_completed` |
| POST | `/api/v1/test` | admin | Create a test event and push it |

Credentials: project keys (`boop_proj_...`) can only create events; device credentials (`boop_dev_...`) can only read events and manage their own device. Only SHA-256 hashes are stored.

**Admin auth.** Set `BOOP_ADMIN_USER` and `BOOP_ADMIN_PASSWORD` and the web UI shows a sign-in screen; admin endpoints then need the session cookie it sets (`POST /api/v1/auth/login`) or HTTP Basic credentials (`curl -u user:pass …`). Sessions last 30 days and live in memory, so a restart signs everyone out. Leave both unset and everything is open — only do that behind your own proxy, Tailscale, or VPN. Either way, project and device credentials are refused on admin endpoints, so a leaked client secret never grants admin rights.

## Configuration

| Variable | Default | Notes |
| --- | --- | --- |
| `BOOP_PORT` | `8080` | |
| `BOOP_BASE_URL` | request origin | Public URL your phone can reach; used in the pairing QR |
| `BOOP_DATABASE_PATH` | `/data/boop.db` | WAL mode, migrations applied on start |
| `BOOP_RETENTION_DAYS` | `30` | Initial value; changeable in the UI. `0` = keep forever |
| `BOOP_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `BOOP_ADMIN_USER` | | Web UI / admin API username; set together with the password |
| `BOOP_ADMIN_PASSWORD` | | 8+ characters. Unset = no login |
| `APNS_TEAM_ID` | | Apple Developer team id |
| `APNS_KEY_ID` | | Id of the APNs auth key |
| `APNS_BUNDLE_ID` | | Bundle id of your Boop iOS build |
| `APNS_PRIVATE_KEY_PATH` | | Path to the mounted `.p8` (preferred) |
| `APNS_PRIVATE_KEY` | | Alternative: the `.p8` contents, as PEM text or base64 (`base64 -i key.p8 \| tr -d '\n'`) |
| `APNS_ENVIRONMENT` | `production` | `sandbox` for Xcode debug builds |

## Apple setup

1. In the Apple Developer portal, create an App identifier for your Boop iOS build and enable Push Notifications.
2. Under Keys, create an APNs authentication key. Download the `.p8` (only possible once). Note the Key id.
3. Note your Team id (top right of the portal).
4. Put the `.p8` at `./secrets/apns.p8`, uncomment the secrets volume in `docker-compose.yml`, and fill the `APNS_*` values in `.env`.
5. Restart: `docker compose up -d`. Settings should show APNs as configured.
6. Build the iOS app with the same bundle id, install it on your phone, open Devices → Pair iPhone, and scan the QR.
7. Settings → Send test notification.

## Deploying with Dokploy (or any compose host)

1. New **Compose** application → your repo, compose path `docker-compose.yml`.
2. Change the data volume to a named one so it survives redeploys: replace `"./data:/data"` with `boop-data:/data` and add a top-level `volumes: { boop-data: {} }` (or set a persistent mount in Dokploy's **Mounts** tab pointing at `/data`).
3. **Environment** tab: `BOOP_BASE_URL`, `BOOP_ADMIN_USER`, `BOOP_ADMIN_PASSWORD`, `APNS_TEAM_ID`, `APNS_KEY_ID`, `APNS_BUNDLE_ID`, `APNS_ENVIRONMENT`.
4. The `.p8` key, either:
   - `APNS_PRIVATE_KEY` = `base64 -i AuthKey_XXXXXX.p8 | tr -d '\n'` (one line, easiest), or
   - Dokploy **Mounts → File mount**: paste the `.p8` contents, mount path `/run/secrets/apns.p8`, and set `APNS_PRIVATE_KEY_PATH=/run/secrets/apns.p8`.
5. Add a domain with HTTPS in Dokploy pointing at port `8080`; deploy. Settings in the web UI should show **APNs · Configured**.

## Pairing

The web UI generates a one-time token (10 minutes, single use, revocable) and shows it as a QR code containing:

```json
{"version": 1, "server": "https://boop.example.com", "token": "pair_..."}
```

The app posts the token to `/api/v1/pairing/exchange`, stores the returned device credential, registers for APNs, and posts its token to `/api/v1/devices`. Registering the same APNs token twice updates the existing device instead of creating a duplicate.

## Redaction

Values under these keys are replaced with `[REDACTED]` anywhere in `data` before storage: `password`, `password_confirmation`, `secret`, `token`, `access_token`, `refresh_token`, `api_key`, `authorization`, `cookie`, `set-cookie`, `private_key`. Matching is case-insensitive and treats `-` and `_` alike. Add your own keys in Settings.

## Development

```bash
cd server && BOOP_DATABASE_PATH=./data/boop.db go run ./cmd/boop   # API on :8080
cd server/web && npm install && npm run dev                         # UI on :5173, proxies /api
make test                                                           # Go + web tests
make build                                                          # bin/boop with the UI embedded
```

Requires Go 1.27 and Node 24 (see `.tool-versions`). The SQLite driver is pure Go, so `CGO_ENABLED=0` builds work everywhere.

## Licence

MIT.

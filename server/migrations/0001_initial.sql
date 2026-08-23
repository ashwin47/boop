CREATE TABLE projects (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    slug         TEXT NOT NULL UNIQUE,
    icon         TEXT NOT NULL DEFAULT '',
    api_key_hash TEXT NOT NULL UNIQUE,
    notify       INTEGER NOT NULL DEFAULT 1,
    min_level    TEXT NOT NULL DEFAULT 'info',
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);

CREATE TABLE devices (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL DEFAULT '',
    device_token    TEXT,
    platform        TEXT NOT NULL DEFAULT 'ios',
    app_bundle_id   TEXT NOT NULL DEFAULT '',
    credential_hash TEXT NOT NULL UNIQUE,
    last_seen_at    TEXT,
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL
);
CREATE UNIQUE INDEX devices_device_token ON devices(device_token) WHERE device_token IS NOT NULL;

CREATE TABLE pairing_tokens (
    id         TEXT PRIMARY KEY,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TEXT NOT NULL,
    used_at    TEXT,
    revoked_at TEXT,
    created_at TEXT NOT NULL
);

CREATE TABLE events (
    id           TEXT PRIMARY KEY,
    external_id  TEXT,
    project_id   TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    source       TEXT NOT NULL DEFAULT '',
    type         TEXT NOT NULL DEFAULT '',
    level        TEXT NOT NULL DEFAULT 'info',
    title        TEXT NOT NULL,
    body         TEXT NOT NULL DEFAULT '',
    fingerprint  TEXT NOT NULL DEFAULT '',
    payload_json TEXT NOT NULL DEFAULT '{}',
    occurred_at  TEXT NOT NULL,
    created_at   TEXT NOT NULL
);
CREATE INDEX events_project_id ON events(project_id);
CREATE INDEX events_occurred_at ON events(occurred_at);
CREATE INDEX events_created_at ON events(created_at, id);
CREATE INDEX events_level ON events(level);
CREATE INDEX events_fingerprint ON events(fingerprint);
CREATE INDEX events_source ON events(source);

CREATE TABLE deliveries (
    id           TEXT PRIMARY KEY,
    event_id     TEXT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    device_id    TEXT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    status       TEXT NOT NULL,
    apns_id      TEXT NOT NULL DEFAULT '',
    error        TEXT NOT NULL DEFAULT '',
    attempted_at TEXT NOT NULL,
    created_at   TEXT NOT NULL
);
CREATE INDEX deliveries_event_id ON deliveries(event_id);
CREATE INDEX deliveries_attempted_at ON deliveries(attempted_at);

CREATE TABLE settings (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

ALTER TABLE events ADD COLUMN actions_json TEXT NOT NULL DEFAULT '[]';
CREATE INDEX events_group ON events(project_id, fingerprint, created_at DESC, id DESC);

-- +goose Up
CREATE TABLE IF NOT EXISTS tracks (
  slug         TEXT PRIMARY KEY,
  object_key   TEXT NOT NULL UNIQUE,
  title        TEXT NOT NULL,
  status       TEXT NOT NULL DEFAULT 'pending',
  content_type TEXT,
  size_bytes   INTEGER,
  peaks_json   TEXT,
  duration_seconds REAL,
  expires_at   DATETIME,
  created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  ready_at     DATETIME
);

-- +goose Down
DROP TABLE IF EXISTS tracks;

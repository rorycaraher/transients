-- +goose Up
CREATE TABLE playlists (
  slug        TEXT PRIMARY KEY,
  title       TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  expires_at  DATETIME,
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ON UPDATE CASCADE keeps memberships intact when either side is Rotated
-- (slug is the primary key and Rotate rewrites it in place, ADR 0005);
-- ON DELETE CASCADE is how deleting a Track drops it from every Playlist.
CREATE TABLE playlist_entries (
  id            INTEGER PRIMARY KEY,
  playlist_slug TEXT NOT NULL REFERENCES playlists(slug) ON UPDATE CASCADE ON DELETE CASCADE,
  track_slug    TEXT NOT NULL REFERENCES tracks(slug) ON UPDATE CASCADE ON DELETE CASCADE,
  position      INTEGER NOT NULL,
  UNIQUE (playlist_slug, track_slug)
);

CREATE INDEX playlist_entries_track_slug ON playlist_entries (track_slug);

-- +goose Down
DROP TABLE IF EXISTS playlist_entries;
DROP TABLE IF EXISTS playlists;

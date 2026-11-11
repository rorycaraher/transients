package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrAlreadyInPlaylist = errors.New("track already in playlist")
	ErrInvalidOrder      = errors.New("order must list every entry exactly once")
)

type Playlist struct {
	Slug        string
	Title       string
	Description string
	ExpiresAt   sql.NullTime
	CreatedAt   time.Time
	EntryCount  int
}

func (p *Playlist) Expired() bool {
	return p.ExpiresAt.Valid && time.Now().After(p.ExpiresAt.Time)
}

// PlaylistEntry is one Track's place in a Playlist. ID, not the Track's
// slug, is how public Playlist routes address the Track (ADR 0006).
type PlaylistEntry struct {
	ID       int64
	Position int
	Track    *Track
}

const playlistColumns = `p.slug, p.title, p.description, p.expires_at, p.created_at,
	(SELECT COUNT(*) FROM playlist_entries e WHERE e.playlist_slug = p.slug)`

const entryColumns = `e.id, e.position,
	t.slug, t.object_key, t.title, t.status, t.content_type, t.size_bytes, t.downloadable, t.expires_at, t.created_at, t.ready_at, t.play_count, t.description`

func (s *Store) CreatePlaylist(slug, title, description string, expiresAt *time.Time) error {
	_, err := s.db.Exec(
		`INSERT INTO playlists (slug, title, description, expires_at) VALUES (?, ?, ?, ?)`,
		slug, title, description, nullTime(expiresAt),
	)
	return err
}

func (s *Store) GetPlaylist(slug string) (*Playlist, error) {
	p, err := scanPlaylist(s.db.QueryRow(`SELECT `+playlistColumns+` FROM playlists p WHERE p.slug = ?`, slug))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

func (s *Store) ListPlaylists() ([]*Playlist, error) {
	rows, err := s.db.Query(`SELECT ` + playlistColumns + ` FROM playlists p ORDER BY p.created_at DESC, p.slug`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Playlist
	for rows.Next() {
		p, err := scanPlaylist(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) UpdatePlaylist(slug, title, description string, expiresAt *time.Time) error {
	return expectOneRow(s.db.Exec(
		`UPDATE playlists SET title = ?, description = ?, expires_at = ? WHERE slug = ?`,
		title, description, nullTime(expiresAt), slug,
	))
}

// RotatePlaylistSlug is the Playlist counterpart of RotateTrackSlug; its
// entries follow via ON UPDATE CASCADE.
func (s *Store) RotatePlaylistSlug(oldSlug, newSlug string) error {
	return expectOneRow(s.db.Exec(`UPDATE playlists SET slug = ? WHERE slug = ?`, newSlug, oldSlug))
}

func (s *Store) DeletePlaylist(slug string) error {
	return expectOneRow(s.db.Exec(`DELETE FROM playlists WHERE slug = ?`, slug))
}

func (s *Store) ListPlaylistEntries(playlistSlug string) ([]*PlaylistEntry, error) {
	rows, err := s.db.Query(
		`SELECT `+entryColumns+`
		 FROM playlist_entries e JOIN tracks t ON t.slug = e.track_slug
		 WHERE e.playlist_slug = ?
		 ORDER BY e.position, e.id`, playlistSlug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*PlaylistEntry
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// GetPlaylistEntry is scoped to playlistSlug so an entry ID can never be
// used to reach a Track through some other Playlist.
func (s *Store) GetPlaylistEntry(playlistSlug string, entryID int64) (*PlaylistEntry, error) {
	e, err := scanEntry(s.db.QueryRow(
		`SELECT `+entryColumns+`
		 FROM playlist_entries e JOIN tracks t ON t.slug = e.track_slug
		 WHERE e.playlist_slug = ? AND e.id = ?`, playlistSlug, entryID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return e, err
}

func (s *Store) AddTrackToPlaylist(playlistSlug, trackSlug string) error {
	_, err := s.db.Exec(
		`INSERT INTO playlist_entries (playlist_slug, track_slug, position)
		 SELECT ?, ?, COALESCE(MAX(position) + 1, 0) FROM playlist_entries WHERE playlist_slug = ?`,
		playlistSlug, trackSlug, playlistSlug,
	)
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "UNIQUE constraint failed"):
		return ErrAlreadyInPlaylist
	case strings.Contains(msg, "FOREIGN KEY constraint failed"):
		return ErrNotFound
	}
	return err
}

func (s *Store) RemovePlaylistEntry(playlistSlug string, entryID int64) error {
	return expectOneRow(s.db.Exec(`DELETE FROM playlist_entries WHERE playlist_slug = ? AND id = ?`, playlistSlug, entryID))
}

// MovePlaylistEntry moves an entry delta places up (negative) or down
// (positive), clamped to the ends of the Playlist.
func (s *Store) MovePlaylistEntry(playlistSlug string, entryID int64, delta int) error {
	ids, err := s.entryIDs(playlistSlug)
	if err != nil {
		return err
	}
	from := -1
	for i, id := range ids {
		if id == entryID {
			from = i
			break
		}
	}
	if from < 0 {
		return ErrNotFound
	}
	to := min(max(from+delta, 0), len(ids)-1)
	if to == from {
		return nil
	}
	ids = append(ids[:from], ids[from+1:]...)
	ids = append(ids[:to], append([]int64{entryID}, ids[to:]...)...)
	return s.SetPlaylistOrder(playlistSlug, ids)
}

// SetPlaylistOrder rewrites positions to match entryIDs, which must be an
// exact permutation of the Playlist's current entries — a stale drag-and-drop
// from another tab is rejected rather than half-applied.
func (s *Store) SetPlaylistOrder(playlistSlug string, entryIDs []int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	current, err := entryIDsIn(tx, playlistSlug)
	if err != nil {
		return err
	}
	if len(current) != len(entryIDs) {
		return ErrInvalidOrder
	}
	want := make(map[int64]bool, len(current))
	for _, id := range current {
		want[id] = true
	}
	for _, id := range entryIDs {
		if !want[id] {
			return ErrInvalidOrder
		}
		delete(want, id)
	}

	for i, id := range entryIDs {
		if _, err := tx.Exec(`UPDATE playlist_entries SET position = ? WHERE id = ?`, i, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// PlaylistTitlesByTrack maps each Track slug to the titles of the Playlists
// containing it, for warning before a Track is deleted.
func (s *Store) PlaylistTitlesByTrack() (map[string][]string, error) {
	rows, err := s.db.Query(
		`SELECT e.track_slug, p.title FROM playlist_entries e JOIN playlists p ON p.slug = e.playlist_slug
		 ORDER BY p.title`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string][]string{}
	for rows.Next() {
		var slug, title string
		if err := rows.Scan(&slug, &title); err != nil {
			return nil, err
		}
		out[slug] = append(out[slug], title)
	}
	return out, rows.Err()
}

func (s *Store) entryIDs(playlistSlug string) ([]int64, error) {
	return entryIDsIn(s.db, playlistSlug)
}

type querier interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

func entryIDsIn(q querier, playlistSlug string) ([]int64, error) {
	rows, err := q.Query(`SELECT id FROM playlist_entries WHERE playlist_slug = ? ORDER BY position, id`, playlistSlug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func scanPlaylist(row rowScanner) (*Playlist, error) {
	var p Playlist
	if err := row.Scan(&p.Slug, &p.Title, &p.Description, &p.ExpiresAt, &p.CreatedAt, &p.EntryCount); err != nil {
		return nil, err
	}
	return &p, nil
}

func scanEntry(row rowScanner) (*PlaylistEntry, error) {
	e := PlaylistEntry{Track: &Track{}}
	if err := row.Scan(append([]any{&e.ID, &e.Position}, e.Track.scanDest()...)...); err != nil {
		return nil, err
	}
	return &e, nil
}

func expectOneRow(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	if n > 1 {
		return fmt.Errorf("expected 1 row affected, got %d", n)
	}
	return nil
}

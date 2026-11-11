// Package store is the SQLite persistence layer for Tracks and Playlists:
// the ingest poller and the web handlers both read and write through it, and
// nothing else touches the database directly.
package store

import (
	"database/sql"
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

type rowScanner interface {
	Scan(dest ...any) error
}

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

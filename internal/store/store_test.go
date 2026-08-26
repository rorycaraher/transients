package store

import (
	"path/filepath"
	"testing"

	"github.com/rorycaraher/transients/internal/db"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return New(conn)
}

func TestBeginReplaceUpdatesObjectKeyAndStatusOnly(t *testing.T) {
	s := newTestStore(t)

	if err := s.CreateFromDiscovery("track1", "track1.mp3", "My Track"); err != nil {
		t.Fatalf("CreateFromDiscovery: %v", err)
	}
	if err := s.MarkReady("track1", "audio/mpeg", 1234); err != nil {
		t.Fatalf("MarkReady: %v", err)
	}
	if err := s.IncrementPlayCount("track1"); err != nil {
		t.Fatalf("IncrementPlayCount: %v", err)
	}

	if err := s.BeginReplace("track1", "track1.wav"); err != nil {
		t.Fatalf("BeginReplace: %v", err)
	}

	track, err := s.GetBySlug("track1")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	if track.ObjectKey != "track1.wav" {
		t.Fatalf("expected object_key to be updated, got %q", track.ObjectKey)
	}
	if track.Status != StatusPending {
		t.Fatalf("expected status pending, got %q", track.Status)
	}
	if track.Title != "My Track" {
		t.Fatalf("expected title untouched, got %q", track.Title)
	}
	if track.PlayCount != 1 {
		t.Fatalf("expected play_count untouched, got %d", track.PlayCount)
	}
}

func TestBeginReplaceUnknownSlug(t *testing.T) {
	s := newTestStore(t)

	if err := s.BeginReplace("does-not-exist", "new-key.mp3"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

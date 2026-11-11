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

	if err := s.CreateTrackFromDiscovery("track1", "track1.mp3", "My Track"); err != nil {
		t.Fatalf("CreateTrackFromDiscovery: %v", err)
	}
	if err := s.MarkTrackReady("track1", "audio/mpeg", 1234); err != nil {
		t.Fatalf("MarkTrackReady: %v", err)
	}
	if err := s.IncrementTrackPlayCount("track1"); err != nil {
		t.Fatalf("IncrementTrackPlayCount: %v", err)
	}

	if err := s.BeginTrackReplace("track1", "track1.wav"); err != nil {
		t.Fatalf("BeginTrackReplace: %v", err)
	}

	track, err := s.GetTrack("track1")
	if err != nil {
		t.Fatalf("GetTrack: %v", err)
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

	if err := s.BeginTrackReplace("does-not-exist", "new-key.mp3"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestRotateSlugUpdatesSlugOnly(t *testing.T) {
	s := newTestStore(t)

	if err := s.CreateTrackFromDiscovery("track1", "track1.mp3", "My Track"); err != nil {
		t.Fatalf("CreateTrackFromDiscovery: %v", err)
	}
	if err := s.MarkTrackReady("track1", "audio/mpeg", 1234); err != nil {
		t.Fatalf("MarkTrackReady: %v", err)
	}
	if err := s.IncrementTrackPlayCount("track1"); err != nil {
		t.Fatalf("IncrementTrackPlayCount: %v", err)
	}

	if err := s.RotateTrackSlug("track1", "track2"); err != nil {
		t.Fatalf("RotateTrackSlug: %v", err)
	}

	if _, err := s.GetTrack("track1"); err != ErrNotFound {
		t.Fatalf("expected old slug to 404, got %v", err)
	}

	track, err := s.GetTrack("track2")
	if err != nil {
		t.Fatalf("GetTrack: %v", err)
	}
	if track.ObjectKey != "track1.mp3" {
		t.Fatalf("expected object_key untouched, got %q", track.ObjectKey)
	}
	if track.Title != "My Track" {
		t.Fatalf("expected title untouched, got %q", track.Title)
	}
	if track.PlayCount != 1 {
		t.Fatalf("expected play_count untouched, got %d", track.PlayCount)
	}
	if track.Status != StatusReady {
		t.Fatalf("expected status untouched, got %q", track.Status)
	}
}

func TestRotateSlugUnknownSlug(t *testing.T) {
	s := newTestStore(t)

	if err := s.RotateTrackSlug("does-not-exist", "new-slug"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

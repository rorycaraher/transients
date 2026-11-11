package store

import (
	"errors"
	"slices"
	"testing"
	"time"
)

func readyTrack(t *testing.T, s *Store, slug string) {
	t.Helper()
	if err := s.CreateTrackFromDiscovery(slug, slug+".mp3", "Title "+slug); err != nil {
		t.Fatalf("CreateTrackFromDiscovery: %v", err)
	}
	if err := s.MarkTrackReady(slug, "audio/mpeg", 1); err != nil {
		t.Fatalf("MarkTrackReady: %v", err)
	}
}

func entryTrackSlugs(t *testing.T, s *Store, playlistSlug string) []string {
	t.Helper()
	entries, err := s.ListPlaylistEntries(playlistSlug)
	if err != nil {
		t.Fatalf("ListPlaylistEntries: %v", err)
	}
	var out []string
	for _, e := range entries {
		out = append(out, e.Track.Slug)
	}
	return out
}

func entryIDByTrack(t *testing.T, s *Store, playlistSlug, trackSlug string) int64 {
	t.Helper()
	entries, err := s.ListPlaylistEntries(playlistSlug)
	if err != nil {
		t.Fatalf("ListPlaylistEntries: %v", err)
	}
	for _, e := range entries {
		if e.Track.Slug == trackSlug {
			return e.ID
		}
	}
	t.Fatalf("track %q not in playlist %q", trackSlug, playlistSlug)
	return 0
}

func newPlaylistWith(t *testing.T, s *Store, slug string, trackSlugs ...string) {
	t.Helper()
	if err := s.CreatePlaylist(slug, "Playlist "+slug, "", nil); err != nil {
		t.Fatalf("CreatePlaylist: %v", err)
	}
	for _, ts := range trackSlugs {
		if err := s.AddTrackToPlaylist(slug, ts); err != nil {
			t.Fatalf("AddTrackToPlaylist(%s): %v", ts, err)
		}
	}
}

func TestAddTrackToPlaylistAppendsAndRejectsDuplicates(t *testing.T) {
	s := newTestStore(t)
	readyTrack(t, s, "a")
	readyTrack(t, s, "b")
	newPlaylistWith(t, s, "pl", "b", "a")

	if got := entryTrackSlugs(t, s, "pl"); !slices.Equal(got, []string{"b", "a"}) {
		t.Fatalf("expected append order [b a], got %v", got)
	}
	if err := s.AddTrackToPlaylist("pl", "a"); !errors.Is(err, ErrAlreadyInPlaylist) {
		t.Fatalf("expected ErrAlreadyInPlaylist, got %v", err)
	}
	if err := s.AddTrackToPlaylist("pl", "nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for unknown track, got %v", err)
	}
	if err := s.AddTrackToPlaylist("nope", "a"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for unknown playlist, got %v", err)
	}

	p, err := s.GetPlaylist("pl")
	if err != nil {
		t.Fatalf("GetPlaylist: %v", err)
	}
	if p.EntryCount != 2 {
		t.Fatalf("expected EntryCount 2, got %d", p.EntryCount)
	}
}

func TestMoveAndSetPlaylistOrder(t *testing.T) {
	s := newTestStore(t)
	for _, slug := range []string{"a", "b", "c"} {
		readyTrack(t, s, slug)
	}
	newPlaylistWith(t, s, "pl", "a", "b", "c")

	if err := s.MovePlaylistEntry("pl", entryIDByTrack(t, s, "pl", "c"), -1); err != nil {
		t.Fatalf("MovePlaylistEntry: %v", err)
	}
	if got := entryTrackSlugs(t, s, "pl"); !slices.Equal(got, []string{"a", "c", "b"}) {
		t.Fatalf("after move up expected [a c b], got %v", got)
	}

	if err := s.MovePlaylistEntry("pl", entryIDByTrack(t, s, "pl", "a"), -1); err != nil {
		t.Fatalf("MovePlaylistEntry at top: %v", err)
	}
	if got := entryTrackSlugs(t, s, "pl"); !slices.Equal(got, []string{"a", "c", "b"}) {
		t.Fatalf("move past top should be a no-op, got %v", got)
	}

	ids := []int64{entryIDByTrack(t, s, "pl", "b"), entryIDByTrack(t, s, "pl", "a"), entryIDByTrack(t, s, "pl", "c")}
	if err := s.SetPlaylistOrder("pl", ids); err != nil {
		t.Fatalf("SetPlaylistOrder: %v", err)
	}
	if got := entryTrackSlugs(t, s, "pl"); !slices.Equal(got, []string{"b", "a", "c"}) {
		t.Fatalf("after SetPlaylistOrder expected [b a c], got %v", got)
	}

	if err := s.SetPlaylistOrder("pl", ids[:2]); !errors.Is(err, ErrInvalidOrder) {
		t.Fatalf("expected ErrInvalidOrder for missing id, got %v", err)
	}
	if err := s.SetPlaylistOrder("pl", []int64{ids[0], ids[0], ids[1]}); !errors.Is(err, ErrInvalidOrder) {
		t.Fatalf("expected ErrInvalidOrder for duplicate id, got %v", err)
	}
}

func TestRotateTrackSlugKeepsPlaylistMembership(t *testing.T) {
	s := newTestStore(t)
	readyTrack(t, s, "a")
	newPlaylistWith(t, s, "pl", "a")

	if err := s.RotateTrackSlug("a", "a2"); err != nil {
		t.Fatalf("RotateTrackSlug: %v", err)
	}
	if got := entryTrackSlugs(t, s, "pl"); !slices.Equal(got, []string{"a2"}) {
		t.Fatalf("expected membership to follow rotated slug, got %v", got)
	}
}

func TestRotatePlaylistSlugKeepsEntries(t *testing.T) {
	s := newTestStore(t)
	readyTrack(t, s, "a")
	newPlaylistWith(t, s, "pl", "a")

	if err := s.RotatePlaylistSlug("pl", "pl2"); err != nil {
		t.Fatalf("RotatePlaylistSlug: %v", err)
	}
	if _, err := s.GetPlaylist("pl"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected old playlist slug to be gone, got %v", err)
	}
	if got := entryTrackSlugs(t, s, "pl2"); !slices.Equal(got, []string{"a"}) {
		t.Fatalf("expected entries to follow rotated playlist, got %v", got)
	}
	if err := s.RotatePlaylistSlug("nope", "x"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteTrackRemovesItFromPlaylists(t *testing.T) {
	s := newTestStore(t)
	readyTrack(t, s, "a")
	readyTrack(t, s, "b")
	newPlaylistWith(t, s, "pl1", "a", "b")
	newPlaylistWith(t, s, "pl2", "a")

	titles, err := s.PlaylistTitlesByTrack()
	if err != nil {
		t.Fatalf("PlaylistTitlesByTrack: %v", err)
	}
	if got := titles["a"]; !slices.Equal(got, []string{"Playlist pl1", "Playlist pl2"}) {
		t.Fatalf("expected a in both playlists, got %v", got)
	}

	if err := s.DeleteTrack("a"); err != nil {
		t.Fatalf("DeleteTrack: %v", err)
	}
	if got := entryTrackSlugs(t, s, "pl1"); !slices.Equal(got, []string{"b"}) {
		t.Fatalf("expected [b] left in pl1, got %v", got)
	}
	if got := entryTrackSlugs(t, s, "pl2"); len(got) != 0 {
		t.Fatalf("expected pl2 empty, got %v", got)
	}
}

func TestDeletePlaylistLeavesTracks(t *testing.T) {
	s := newTestStore(t)
	readyTrack(t, s, "a")
	newPlaylistWith(t, s, "pl", "a")

	if err := s.DeletePlaylist("pl"); err != nil {
		t.Fatalf("DeletePlaylist: %v", err)
	}
	if _, err := s.GetTrack("a"); err != nil {
		t.Fatalf("expected track to survive playlist delete, got %v", err)
	}
	if _, err := s.GetPlaylistEntry("pl", 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected entries gone, got %v", err)
	}
}

func TestGetPlaylistEntryScopedToPlaylist(t *testing.T) {
	s := newTestStore(t)
	readyTrack(t, s, "a")
	newPlaylistWith(t, s, "pl1", "a")
	newPlaylistWith(t, s, "pl2")

	id := entryIDByTrack(t, s, "pl1", "a")
	if _, err := s.GetPlaylistEntry("pl2", id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected entry from another playlist to be ErrNotFound, got %v", err)
	}
	e, err := s.GetPlaylistEntry("pl1", id)
	if err != nil {
		t.Fatalf("GetPlaylistEntry: %v", err)
	}
	if e.Track.Slug != "a" || !e.Track.Available() {
		t.Fatalf("unexpected entry %+v", e.Track)
	}
}

func TestPlaylistExpiry(t *testing.T) {
	s := newTestStore(t)
	past := time.Now().Add(-time.Hour)
	if err := s.CreatePlaylist("pl", "T", "D", &past); err != nil {
		t.Fatalf("CreatePlaylist: %v", err)
	}
	p, err := s.GetPlaylist("pl")
	if err != nil {
		t.Fatalf("GetPlaylist: %v", err)
	}
	if !p.Expired() {
		t.Fatalf("expected playlist to be expired")
	}
	if err := s.UpdatePlaylist("pl", "T2", "D2", nil); err != nil {
		t.Fatalf("UpdatePlaylist: %v", err)
	}
	p, _ = s.GetPlaylist("pl")
	if p.Expired() || p.Title != "T2" || p.Description != "D2" {
		t.Fatalf("unexpected playlist after update: %+v", p)
	}
}

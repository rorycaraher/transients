package web

import (
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"strconv"

	"github.com/rorycaraher/transients/internal/store"
)

// handlePlaylistShare renders a Playlist's Share Page. Per ADR 0006 nothing
// on it — links, metadata, player data — may carry a Track's own slug:
// every per-Track action goes through /p/{slug}/entries/{id}/..., so access
// gained here ends when the Playlist expires or is Rotated.
func (s *Server) handlePlaylistShare(w http.ResponseWriter, r *http.Request) {
	playlist, ok := s.lookupPlaylist(w, r)
	if !ok {
		return
	}
	if playlist.Expired() {
		s.render(w, "share_expired.html", nil)
		return
	}

	entries, err := s.store.ListPlaylistEntries(playlist.Slug)
	if err != nil {
		s.log.Error("list playlist entries failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	base := "/p/" + playlist.Slug + "/entries/"
	var tracks []playerTrack
	for _, e := range entries {
		if !e.Track.Available() {
			continue
		}
		entryBase := base + strconv.FormatInt(e.ID, 10)
		pt := playerTrack{
			Title:       e.Track.Title,
			AudioURL:    entryBase + "/audio",
			PlayURL:     entryBase + "/play",
			Description: e.Track.Description,
		}
		if e.Track.Downloadable {
			pt.DownloadURL = entryBase + "/download"
		}
		tracks = append(tracks, pt)
	}

	dataJSON, err := json.Marshal(playerData{Playlist: true, Tracks: tracks})
	if err != nil {
		s.log.Error("marshal player data failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	s.render(w, "playlist_share.html", map[string]any{
		"Playlist":       playlist,
		"Tracks":         tracks,
		"PlayerDataJSON": template.JS(dataJSON),
		"ShareURL":       s.cfg.BaseURL + "/p/" + playlist.Slug,
		"OGImageURL":     s.cfg.BaseURL + staticURL("og-image.png"),
	})
}

// handlePlaylistEntryAudio redirects to a freshly presigned GET for the
// entry's Audio File. Minting per request, rather than once per page load
// like the Track Share Page, keeps a long-open Playlist page from ever
// holding a URL past PresignedGetTTL.
func (s *Server) handlePlaylistEntryAudio(w http.ResponseWriter, r *http.Request) {
	entry, ok := s.lookupAvailableEntry(w, r)
	if !ok {
		return
	}
	url, err := s.r2.PresignGet(r.Context(), entry.Track.ObjectKey, s.cfg.PresignedGetTTL)
	if err != nil {
		s.log.Error("presign get failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, url, http.StatusFound)
}

func (s *Server) handlePlaylistEntryDownload(w http.ResponseWriter, r *http.Request) {
	entry, ok := s.lookupAvailableEntry(w, r)
	if !ok {
		return
	}
	if !entry.Track.Downloadable {
		s.handleNotFound(w, r)
		return
	}
	url, err := s.r2.PresignGetAttachment(r.Context(), entry.Track.ObjectKey, s.cfg.PresignedGetTTL, downloadFilename(entry.Track))
	if err != nil {
		s.log.Error("presign get attachment failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, url, http.StatusFound)
}

// handlePlaylistEntryPlay records a Play against the entry's Track — Plays
// are only ever counted per Track, wherever it's heard.
func (s *Server) handlePlaylistEntryPlay(w http.ResponseWriter, r *http.Request) {
	entry, ok := s.lookupAvailableEntry(w, r)
	if !ok {
		return
	}
	if err := s.store.IncrementTrackPlayCount(entry.Track.Slug); err != nil {
		s.log.Error("increment play count failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// lookupAvailableEntry resolves /p/{slug}/entries/{id}: the Playlist must
// exist and not be expired, the entry must belong to it, and its Track must
// be Available. Every failure is the same plain 404.
func (s *Server) lookupAvailableEntry(w http.ResponseWriter, r *http.Request) (*store.PlaylistEntry, bool) {
	playlist, ok := s.lookupPlaylist(w, r)
	if !ok {
		return nil, false
	}
	if playlist.Expired() {
		s.handleNotFound(w, r)
		return nil, false
	}
	entryID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.handleNotFound(w, r)
		return nil, false
	}
	entry, err := s.store.GetPlaylistEntry(playlist.Slug, entryID)
	if errors.Is(err, store.ErrNotFound) {
		s.handleNotFound(w, r)
		return nil, false
	}
	if err != nil {
		s.log.Error("get playlist entry failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return nil, false
	}
	if !entry.Track.Available() {
		s.handleNotFound(w, r)
		return nil, false
	}
	return entry, true
}

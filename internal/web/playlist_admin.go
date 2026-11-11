package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/rorycaraher/transients/internal/idgen"
	"github.com/rorycaraher/transients/internal/store"
)

func (s *Server) handleAdminPlaylists(w http.ResponseWriter, r *http.Request) {
	playlists, err := s.store.ListPlaylists()
	if err != nil {
		s.log.Error("list playlists failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	s.render(w, "admin_playlists.html", map[string]any{"Playlists": playlists})
}

func (s *Server) handleNewPlaylistForm(w http.ResponseWriter, r *http.Request) {
	s.render(w, "admin_playlist_new.html", nil)
}

func (s *Server) handleCreatePlaylist(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	title := r.FormValue("title")
	if title == "" {
		http.Error(w, "title required", http.StatusBadRequest)
		return
	}
	expiresAt, err := parseExpiresAt(r.FormValue("expires_at"))
	if err != nil {
		http.Error(w, "invalid expires_at", http.StatusBadRequest)
		return
	}

	slug := idgen.New()
	if err := s.store.CreatePlaylist(slug, title, r.FormValue("description"), expiresAt); err != nil {
		s.log.Error("create playlist failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/playlists/"+slug+"/edit", http.StatusSeeOther)
}

func (s *Server) handlePlaylistEditForm(w http.ResponseWriter, r *http.Request) {
	playlist, ok := s.lookupPlaylist(w, r)
	if !ok {
		return
	}
	entries, err := s.store.ListPlaylistEntries(playlist.Slug)
	if err != nil {
		s.log.Error("list playlist entries failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	tracks, err := s.store.ListTracks()
	if err != nil {
		s.log.Error("list tracks failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	inPlaylist := make(map[string]bool, len(entries))
	for _, e := range entries {
		inPlaylist[e.Track.Slug] = true
	}
	var addable []*store.Track
	for _, t := range tracks {
		if !inPlaylist[t.Slug] {
			addable = append(addable, t)
		}
	}

	s.render(w, "admin_playlist_edit.html", map[string]any{
		"Playlist":       playlist,
		"Entries":        entries,
		"AddableTracks":  addable,
		"ExpiresAtValue": formatExpiresAt(playlist.ExpiresAt),
		"ShareURL":       s.cfg.BaseURL + "/p/" + playlist.Slug,
	})
}

func (s *Server) handlePlaylistEditSubmit(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	title := r.FormValue("title")
	if title == "" {
		http.Error(w, "title required", http.StatusBadRequest)
		return
	}
	expiresAt, err := parseExpiresAt(r.FormValue("expires_at"))
	if err != nil {
		http.Error(w, "invalid expires_at", http.StatusBadRequest)
		return
	}

	if err := s.store.UpdatePlaylist(slug, title, r.FormValue("description"), expiresAt); err != nil {
		s.playlistWriteError(w, r, "update playlist", err)
		return
	}
	http.Redirect(w, r, "/admin/playlists/"+slug+"/edit", http.StatusSeeOther)
}

func (s *Server) handlePlaylistAddTrack(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	err := s.store.AddTrackToPlaylist(slug, r.FormValue("track_slug"))
	if err != nil && !errors.Is(err, store.ErrAlreadyInPlaylist) {
		s.playlistWriteError(w, r, "add track to playlist", err)
		return
	}
	http.Redirect(w, r, "/admin/playlists/"+slug+"/edit", http.StatusSeeOther)
}

func (s *Server) handlePlaylistRemoveEntry(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	entryID, ok := entryIDParam(w, r)
	if !ok {
		return
	}
	if err := s.store.RemovePlaylistEntry(slug, entryID); err != nil {
		s.playlistWriteError(w, r, "remove playlist entry", err)
		return
	}
	http.Redirect(w, r, "/admin/playlists/"+slug+"/edit", http.StatusSeeOther)
}

func (s *Server) handlePlaylistMoveEntry(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	entryID, ok := entryIDParam(w, r)
	if !ok {
		return
	}
	var delta int
	switch r.FormValue("dir") {
	case "up":
		delta = -1
	case "down":
		delta = 1
	default:
		http.Error(w, "dir must be up or down", http.StatusBadRequest)
		return
	}
	if err := s.store.MovePlaylistEntry(slug, entryID, delta); err != nil {
		s.playlistWriteError(w, r, "move playlist entry", err)
		return
	}
	http.Redirect(w, r, "/admin/playlists/"+slug+"/edit", http.StatusSeeOther)
}

// handlePlaylistOrder applies a full new order from the edit page's
// drag-and-drop; the ↑/↓ forms are the no-JS path to the same result.
func (s *Server) handlePlaylistOrder(w http.ResponseWriter, r *http.Request) {
	var body struct {
		EntryIDs []int64 `json:"entry_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	err := s.store.SetPlaylistOrder(r.PathValue("slug"), body.EntryIDs)
	if errors.Is(err, store.ErrInvalidOrder) {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if err != nil {
		s.log.Error("set playlist order failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handlePlaylistRotate(w http.ResponseWriter, r *http.Request) {
	newSlug := idgen.New()
	if err := s.store.RotatePlaylistSlug(r.PathValue("slug"), newSlug); err != nil {
		s.playlistWriteError(w, r, "rotate playlist slug", err)
		return
	}
	http.Redirect(w, r, "/admin/playlists/"+newSlug+"/edit", http.StatusSeeOther)
}

func (s *Server) handlePlaylistDelete(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeletePlaylist(r.PathValue("slug")); err != nil {
		s.playlistWriteError(w, r, "delete playlist", err)
		return
	}
	http.Redirect(w, r, "/admin/playlists", http.StatusSeeOther)
}

func (s *Server) lookupPlaylist(w http.ResponseWriter, r *http.Request) (*store.Playlist, bool) {
	playlist, err := s.store.GetPlaylist(r.PathValue("slug"))
	if errors.Is(err, store.ErrNotFound) {
		s.handleNotFound(w, r)
		return nil, false
	}
	if err != nil {
		s.log.Error("get playlist failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return nil, false
	}
	return playlist, true
}

func (s *Server) playlistWriteError(w http.ResponseWriter, r *http.Request, op string, err error) {
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	s.log.Error(op+" failed", "err", err)
	http.Error(w, "internal error", http.StatusInternalServerError)
}

func entryIDParam(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return 0, false
	}
	return id, true
}

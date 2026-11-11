package web

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rorycaraher/transients/internal/idgen"
	"github.com/rorycaraher/transients/internal/store"
)

func (s *Server) handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	tracks, err := s.store.ListTracks()
	if err != nil {
		s.log.Error("list tracks failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	playlistTitles, err := s.store.PlaylistTitlesByTrack()
	if err != nil {
		s.log.Error("list playlist memberships failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	deleteConfirms := make(map[string]string, len(tracks))
	for _, t := range tracks {
		deleteConfirms[t.Slug] = deleteTrackConfirm(playlistTitles[t.Slug])
	}
	s.render(w, "admin_dashboard.html", map[string]any{
		"Tracks":         tracks,
		"DeleteConfirms": deleteConfirms,
	})
}

func (s *Server) handleEditForm(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	track, err := s.store.GetTrack(slug)
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.log.Error("get track failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	expiresAtValue := ""
	if track.ExpiresAt.Valid {
		expiresAtValue = track.ExpiresAt.Time.Format("2006-01-02")
	}

	s.render(w, "admin_edit.html", map[string]any{
		"Track":          track,
		"ExpiresAtValue": expiresAtValue,
	})
}

func (s *Server) handleEditSubmit(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}

	title := r.FormValue("title")
	expiresAt, err := parseExpiresAt(r.FormValue("expires_at"))
	if err != nil {
		http.Error(w, "invalid expires_at", http.StatusBadRequest)
		return
	}
	downloadable := r.FormValue("downloadable") != ""
	description := r.FormValue("description")

	if err := s.store.UpdateTrack(slug, title, expiresAt, downloadable, description); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		s.log.Error("update track failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// handleRotateSlug generates a fresh slug and repoints the track at it (see
// store.RotateTrackSlug), so the old /t/{slug} link 404s exactly like an unknown
// one from then on. Everything else about the track — file, expiry, stats —
// is untouched.
func (s *Server) handleRotateSlug(w http.ResponseWriter, r *http.Request) {
	oldSlug := r.PathValue("slug")
	newSlug := idgen.New()

	if err := s.store.RotateTrackSlug(oldSlug, newSlug); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		s.log.Error("rotate slug failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/tracks/"+newSlug+"/edit", http.StatusSeeOther)
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	track, err := s.store.GetTrack(slug)
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.log.Error("get track failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := s.r2.Delete(r.Context(), track.ObjectKey); err != nil {
		s.log.Error("delete r2 object failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := s.store.DeleteTrack(slug); err != nil {
		s.log.Error("delete track failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// parseExpiresAt reads an admin form's yyyy-mm-dd expiry field (blank =
// never), expiring at the end of the chosen day.
func parseExpiresAt(v string) (*time.Time, error) {
	if v == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", v)
	if err != nil {
		return nil, err
	}
	t = t.Add(24 * time.Hour)
	return &t, nil
}

func formatExpiresAt(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Add(-24 * time.Hour).Format("2006-01-02")
}

func deleteTrackConfirm(playlistTitles []string) string {
	msg := "Delete this track permanently?"
	switch len(playlistTitles) {
	case 0:
		return msg
	case 1:
		return msg + "\n\nIt will also be removed from 1 playlist: " + playlistTitles[0]
	default:
		return msg + fmt.Sprintf("\n\nIt will also be removed from %d playlists: %s", len(playlistTitles), strings.Join(playlistTitles, ", "))
	}
}

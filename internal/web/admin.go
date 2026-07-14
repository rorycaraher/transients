package web

import (
	"errors"
	"net/http"
	"time"

	"github.com/rorycaraher/transients/internal/store"
)

func (s *Server) handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	tracks, err := s.store.ListAll()
	if err != nil {
		s.log.Error("list tracks failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	s.render(w, "admin_dashboard.html", map[string]any{"Tracks": tracks})
}

func (s *Server) handleEditForm(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	track, err := s.store.GetBySlug(slug)
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
	var expiresAt *time.Time
	if v := r.FormValue("expires_at"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			http.Error(w, "invalid expires_at", http.StatusBadRequest)
			return
		}
		// Expire at the end of the chosen day.
		t = t.Add(24 * time.Hour)
		expiresAt = &t
	}
	downloadable := r.FormValue("downloadable") != ""

	if err := s.store.UpdateTrack(slug, title, expiresAt, downloadable); err != nil {
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

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	track, err := s.store.GetBySlug(slug)
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

	if err := s.store.Delete(slug); err != nil {
		s.log.Error("delete track failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

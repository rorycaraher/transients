package web

import (
	"encoding/json"
	"errors"
	"html/template"
	"net/http"

	"github.com/rorycaraher/transients/internal/store"
)

type playerData struct {
	Title    string `json:"title"`
	AudioURL string `json:"audioUrl"`
}

func (s *Server) handleShare(w http.ResponseWriter, r *http.Request) {
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

	if track.Status != store.StatusReady {
		http.NotFound(w, r)
		return
	}

	if track.Expired() {
		s.render(w, "share_expired.html", nil)
		return
	}

	audioURL, err := s.r2.PresignGet(r.Context(), track.ObjectKey, s.cfg.PresignedGetTTL)
	if err != nil {
		s.log.Error("presign get failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	data := playerData{
		Title:    track.Title,
		AudioURL: audioURL,
	}
	dataJSON, err := json.Marshal(data)
	if err != nil {
		s.log.Error("marshal player data failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	s.render(w, "share.html", map[string]any{
		"Track":          track,
		"PlayerDataJSON": template.JS(dataJSON),
	})
}

package web

import (
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"path"
	"strings"

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
		s.handleNotFound(w, r)
		return
	}
	if err != nil {
		s.log.Error("get track failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if track.Status != store.StatusReady {
		s.handleNotFound(w, r)
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

	templateData := map[string]any{
		"Track":          track,
		"PlayerDataJSON": template.JS(dataJSON),
	}

	if track.Downloadable {
		downloadURL, err := s.r2.PresignGetAttachment(r.Context(), track.ObjectKey, s.cfg.PresignedGetTTL, downloadFilename(track))
		if err != nil {
			s.log.Error("presign get attachment failed", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		templateData["DownloadURL"] = downloadURL
	}

	s.render(w, "share.html", templateData)
}

// downloadFilename derives the filename a downloaded track should be saved
// as: the track's title, plus the object key's extension if the title
// doesn't already end with it (rclone-discovered tracks have the extension
// baked into the title already, since it's just the R2 object's filename).
func downloadFilename(track *store.Track) string {
	ext := path.Ext(track.ObjectKey)
	if ext == "" || strings.HasSuffix(strings.ToLower(track.Title), strings.ToLower(ext)) {
		return track.Title
	}
	return track.Title + ext
}

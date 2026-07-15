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

// lookupReadyTrack fetches the track for slug and enforces the same
// visibility rules everywhere a share link is resolved (main share page,
// bare embed page, oEmbed JSON): unknown or not-yet-ready tracks 404, as do
// expired tracks reached via the embed/oEmbed routes (ok is false and the
// response has already been written in every failure case).
func (s *Server) lookupReadyTrack(w http.ResponseWriter, r *http.Request, slug string) (*store.Track, bool) {
	track, err := s.store.GetBySlug(slug)
	if errors.Is(err, store.ErrNotFound) {
		s.handleNotFound(w, r)
		return nil, false
	}
	if err != nil {
		s.log.Error("get track failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return nil, false
	}
	if track.Status != store.StatusReady {
		s.handleNotFound(w, r)
		return nil, false
	}
	return track, true
}

// playerTemplateData builds the PlayerDataJSON field shared by the share
// page and the bare embed page: a fresh presigned GET URL minted on every
// load, so link expiry is enforced by R2 itself.
func (s *Server) playerTemplateData(r *http.Request, track *store.Track) (map[string]any, error) {
	audioURL, err := s.r2.PresignGet(r.Context(), track.ObjectKey, s.cfg.PresignedGetTTL)
	if err != nil {
		return nil, err
	}
	dataJSON, err := json.Marshal(playerData{Title: track.Title, AudioURL: audioURL})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"Track":          track,
		"PlayerDataJSON": template.JS(dataJSON),
	}, nil
}

func (s *Server) handleShare(w http.ResponseWriter, r *http.Request) {
	track, ok := s.lookupReadyTrack(w, r, r.PathValue("slug"))
	if !ok {
		return
	}

	if track.Expired() {
		s.render(w, "share_expired.html", nil)
		return
	}

	templateData, err := s.playerTemplateData(r, track)
	if err != nil {
		s.log.Error("presign get failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	shareURL := s.cfg.BaseURL + "/t/" + track.Slug
	templateData["ShareURL"] = shareURL
	templateData["OGImageURL"] = s.cfg.BaseURL + staticURL("og-image.png")
	templateData["EmbedURL"] = shareURL + "/embed"
	templateData["OEmbedURL"] = shareURL + "/oembed.json"

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

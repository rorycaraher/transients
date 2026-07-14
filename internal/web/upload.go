package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"time"

	"github.com/rorycaraher/transients/internal/idgen"
	"github.com/rorycaraher/transients/internal/store"
)

const putPresignTTL = 15 * time.Minute

func (s *Server) handleUploadForm(w http.ResponseWriter, r *http.Request) {
	s.render(w, "admin_upload.html", nil)
}

type uploadRequestBody struct {
	Title         string `json:"title"`
	Filename      string `json:"filename"`
	ContentType   string `json:"content_type"`
	ExpiresInDays *int   `json:"expires_in_days"`
}

type uploadRequestResponse struct {
	Slug   string `json:"slug"`
	PutURL string `json:"put_url"`
}

func (s *Server) handleUploadRequest(w http.ResponseWriter, r *http.Request) {
	var body uploadRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	slug := idgen.New()
	objectKey := slug + filepath.Ext(body.Filename)

	title := body.Title
	if title == "" {
		title = body.Filename
	}
	if title == "" {
		title = slug
	}

	var expiresAt *time.Time
	if body.ExpiresInDays != nil && *body.ExpiresInDays > 0 {
		t := time.Now().Add(time.Duration(*body.ExpiresInDays) * 24 * time.Hour)
		expiresAt = &t
	}

	if err := s.store.CreatePending(slug, objectKey, title, expiresAt); err != nil {
		s.log.Error("create pending track failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	putURL, err := s.r2.PresignPut(r.Context(), objectKey, putPresignTTL)
	if err != nil {
		s.log.Error("presign put failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, uploadRequestResponse{Slug: slug, PutURL: putURL})
}

func (s *Server) handleUploadStatus(w http.ResponseWriter, r *http.Request) {
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

	resp := map[string]any{"status": track.Status}
	if track.Status == store.StatusReady {
		resp["share_url"] = s.cfg.BaseURL + "/t/" + track.Slug
	}
	writeJSON(w, resp)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

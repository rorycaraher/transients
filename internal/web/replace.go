package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"

	"github.com/rorycaraher/transients/internal/store"
)

type replaceRequestBody struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
}

// handleReplaceRequest swaps the audio behind an existing track: it deletes
// the current R2 object, then points the track at a fresh object key
// (mirroring CreatePending's role for a first-time upload, see
// store.BeginReplace) and presigns a PUT to it. The old object is deleted
// before the DB row changes, so a track only breaks once its file is
// actually gone — see ADR 0003 for why this isn't deferred until the
// replacement is confirmed ready.
func (s *Server) handleReplaceRequest(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	var body replaceRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

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
		s.log.Error("delete old r2 object failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	newObjectKey := slug + filepath.Ext(body.Filename)
	if err := s.store.BeginReplace(slug, newObjectKey); err != nil {
		s.log.Error("begin replace failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	putURL, err := s.r2.PresignPut(r.Context(), newObjectKey, putPresignTTL)
	if err != nil {
		s.log.Error("presign put failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, uploadRequestResponse{Slug: slug, PutURL: putURL})
}

package web

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// handleShareEmbed renders a bare, chrome-less player page meant to be
// loaded inside a third-party iframe (Twitter/X Player Card, and whatever
// html an oEmbed consumer like Discord renders from handleOEmbed's
// response). Expired tracks 404 here rather than showing share_expired.html
// — there's nothing useful to embed either way.
func (s *Server) handleShareEmbed(w http.ResponseWriter, r *http.Request) {
	track, ok := s.lookupReadyTrack(w, r, r.PathValue("slug"))
	if !ok {
		return
	}
	if track.Expired() {
		s.handleNotFound(w, r)
		return
	}

	templateData, err := s.playerTemplateData(r, track)
	if err != nil {
		s.log.Error("presign get failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	s.renderBare(w, "share_embed.html", templateData)
}

type oembedResponse struct {
	Version      string `json:"version"`
	Type         string `json:"type"`
	ProviderName string `json:"provider_name"`
	ProviderURL  string `json:"provider_url"`
	Title        string `json:"title"`
	HTML         string `json:"html"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
}

const (
	embedWidth  = 480
	embedHeight = 180
)

// handleOEmbed serves the JSON body discovered via share.html's
// application/json+oembed <link> tag. type is "rich" (not "video") since
// this is an arbitrary HTML widget, not a video file.
func (s *Server) handleOEmbed(w http.ResponseWriter, r *http.Request) {
	track, ok := s.lookupReadyTrack(w, r, r.PathValue("slug"))
	if !ok {
		return
	}
	if track.Expired() {
		s.handleNotFound(w, r)
		return
	}

	embedURL := s.cfg.BaseURL + "/t/" + track.Slug + "/embed"
	resp := oembedResponse{
		Version:      "1.0",
		Type:         "rich",
		ProviderName: "NLTL",
		ProviderURL:  s.cfg.BaseURL,
		Title:        track.Title,
		HTML:         fmt.Sprintf(`<iframe src=%q width="%d" height="%d" frameborder="0" allow="autoplay"></iframe>`, embedURL, embedWidth, embedHeight),
		Width:        embedWidth,
		Height:       embedHeight,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		s.log.Error("encode oembed response failed", "err", err)
	}
}

// Package web implements the HTTP surface: the single-admin login and
// dashboard, the upload flow, Playlist curation, and the public Share
// Pages for Tracks and Playlists.
package web

import (
	"html/template"
	"log/slog"
	"net/http"

	"github.com/rorycaraher/transients/internal/auth"
	"github.com/rorycaraher/transients/internal/config"
	"github.com/rorycaraher/transients/internal/r2"
	"github.com/rorycaraher/transients/internal/store"
)

type Server struct {
	cfg   *config.Config
	store *store.Store
	r2    *r2.Client
	tmpl  map[string]*template.Template
	log   *slog.Logger
}

func NewServer(cfg *config.Config, st *store.Store, r2c *r2.Client, log *slog.Logger) (*Server, error) {
	tmpl, err := loadTemplates()
	if err != nil {
		return nil, err
	}
	return &Server{cfg: cfg, store: st, r2: r2c, tmpl: tmpl, log: log}, nil
}

func (s *Server) Mux() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/static/", staticHandler())

	mux.HandleFunc("GET /healthz", s.handleHealthz)

	mux.HandleFunc("GET /{$}", s.handleHome)

	mux.HandleFunc("GET /login", s.handleLoginForm)
	mux.HandleFunc("POST /login", s.handleLoginSubmit)
	mux.HandleFunc("POST /logout", s.handleLogout)

	mux.HandleFunc("GET /t/{slug}", s.handleShare)
	mux.HandleFunc("GET /t/{slug}/embed", s.handleShareEmbed)
	mux.HandleFunc("GET /t/{slug}/oembed.json", s.handleOEmbed)
	mux.HandleFunc("POST /t/{slug}/play", s.handleTrackPlay)

	mux.HandleFunc("GET /p/{slug}", s.handlePlaylistShare)
	mux.HandleFunc("GET /p/{slug}/entries/{id}/audio", s.handlePlaylistEntryAudio)
	mux.HandleFunc("GET /p/{slug}/entries/{id}/download", s.handlePlaylistEntryDownload)
	mux.HandleFunc("POST /p/{slug}/entries/{id}/play", s.handlePlaylistEntryPlay)

	mux.Handle("GET /admin", s.requireAuth(s.handleAdminDashboard))
	mux.Handle("GET /admin/upload", s.requireAuth(s.handleUploadForm))
	mux.Handle("POST /admin/upload/request", s.requireAuth(s.handleUploadRequest))
	mux.Handle("GET /admin/upload/status/{slug}", s.requireAuth(s.handleUploadStatus))
	mux.Handle("GET /admin/tracks/{slug}/edit", s.requireAuth(s.handleEditForm))
	mux.Handle("POST /admin/tracks/{slug}/edit", s.requireAuth(s.handleEditSubmit))
	mux.Handle("POST /admin/tracks/{slug}/delete", s.requireAuth(s.handleDelete))
	mux.Handle("POST /admin/tracks/{slug}/rotate", s.requireAuth(s.handleRotateSlug))
	mux.Handle("POST /admin/tracks/{slug}/replace/request", s.requireAuth(s.handleReplaceRequest))

	mux.Handle("GET /admin/playlists", s.requireAuth(s.handleAdminPlaylists))
	mux.Handle("GET /admin/playlists/new", s.requireAuth(s.handleNewPlaylistForm))
	mux.Handle("POST /admin/playlists", s.requireAuth(s.handleCreatePlaylist))
	mux.Handle("GET /admin/playlists/{slug}/edit", s.requireAuth(s.handlePlaylistEditForm))
	mux.Handle("POST /admin/playlists/{slug}/edit", s.requireAuth(s.handlePlaylistEditSubmit))
	mux.Handle("POST /admin/playlists/{slug}/tracks", s.requireAuth(s.handlePlaylistAddTrack))
	mux.Handle("POST /admin/playlists/{slug}/entries/{id}/remove", s.requireAuth(s.handlePlaylistRemoveEntry))
	mux.Handle("POST /admin/playlists/{slug}/entries/{id}/move", s.requireAuth(s.handlePlaylistMoveEntry))
	mux.Handle("POST /admin/playlists/{slug}/order", s.requireAuth(s.handlePlaylistOrder))
	mux.Handle("POST /admin/playlists/{slug}/rotate", s.requireAuth(s.handlePlaylistRotate))
	mux.Handle("POST /admin/playlists/{slug}/delete", s.requireAuth(s.handlePlaylistDelete))

	// Catch-all fallback: more specific patterns above always win, so this
	// only fires for unmatched routes.
	mux.HandleFunc("/", s.handleNotFound)

	return mux
}

func (s *Server) requireAuth(h http.HandlerFunc) http.Handler {
	return auth.RequireAuth(s.cfg.SessionSecret)(h)
}

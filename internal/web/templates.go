package web

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

// staticVersion is a hash of every embedded static file's path and content,
// computed once at startup. Appending it as a query string on static asset
// URLs busts both the browser's cache and Cloudflare's edge cache the
// moment a deploy actually changes one of those files, and leaves both free
// to cache aggressively otherwise.
var staticVersion = computeStaticVersion()

func computeStaticVersion() string {
	h := sha256.New()
	_ = fs.WalkDir(staticFS, "static", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := staticFS.ReadFile(path)
		if err != nil {
			return err
		}
		fmt.Fprint(h, path)
		h.Write(data)
		return nil
	})
	return hex.EncodeToString(h.Sum(nil))[:12]
}

func staticURL(name string) string {
	return "/static/" + name + "?v=" + staticVersion
}

var pageNames = []string{
	"home.html",
	"login.html",
	"admin_dashboard.html",
	"admin_upload.html",
	"admin_edit.html",
	"share.html",
	"share_expired.html",
	"not_found.html",
}

func loadTemplates() (map[string]*template.Template, error) {
	out := make(map[string]*template.Template, len(pageNames))
	for _, name := range pageNames {
		t, err := template.New(name).Funcs(template.FuncMap{"static": staticURL}).
			ParseFS(templateFS, "templates/layout.html", "templates/"+name)
		if err != nil {
			return nil, fmt.Errorf("parse template %s: %w", name, err)
		}
		out[name] = t
	}
	return out, nil
}

func (s *Server) render(w http.ResponseWriter, page string, data any) {
	s.renderStatus(w, page, http.StatusOK, data)
}

func (s *Server) renderStatus(w http.ResponseWriter, page string, status int, data any) {
	t, ok := s.tmpl[page]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := t.ExecuteTemplate(w, "layout", data); err != nil {
		s.log.Error("template render failed", "page", page, "err", err)
	}
}

func staticHandler() http.Handler {
	return http.FileServer(http.FS(staticFS))
}

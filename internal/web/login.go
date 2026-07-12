package web

import (
	"net/http"

	"github.com/rorycaraher/transients/internal/auth"
)

func (s *Server) handleLoginForm(w http.ResponseWriter, r *http.Request) {
	s.render(w, "login.html", map[string]any{
		"Next": r.URL.Query().Get("next"),
	})
}

func (s *Server) handleLoginSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}

	password := r.FormValue("password")
	next := r.FormValue("next")

	if !auth.CheckPassword(s.cfg.AdminPasswordHash, password) {
		s.render(w, "login.html", map[string]any{
			"Error": "Incorrect password",
			"Next":  next,
		})
		return
	}

	auth.SetSessionCookie(w, s.cfg.SessionSecret)

	if next == "" {
		next = "/admin"
	}
	http.Redirect(w, r, next, http.StatusSeeOther)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	auth.ClearSessionCookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

package web

import (
	"net/http"

	"github.com/rorycaraher/transients/internal/auth"
)

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	if auth.IsAuthenticated(r, s.cfg.SessionSecret) {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	s.render(w, "home.html", nil)
}

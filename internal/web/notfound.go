package web

import "net/http"

func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	s.renderStatus(w, "not_found.html", http.StatusNotFound, nil)
}

package api

import (
	"net/http"
	"runtime"
)

// handleHealth returns a simple health check response.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":     "ok",
		"service":    "taskbridge-server",
		"go_version": runtime.Version(),
	})
}

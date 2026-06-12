package api

import (
	"log/slog"
	"net/http"
	"time"

	"taskbridge/internal/store"
)

// Server is the central HTTP server that wires together the store and routes.
type Server struct {
	store     store.Store
	logger    *slog.Logger
	authToken string
}

// NewServer constructs a Server with the required dependencies.
func NewServer(s store.Store, logger *slog.Logger, authToken string) *Server {
	return &Server{
		store:     s,
		logger:    logger,
		authToken: authToken,
	}
}

// Handler returns a fully configured http.Handler with all routes and middleware.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.registerRoutes(mux)

	// Apply middleware in order: recovery → auth → logging.
	var handler http.Handler = mux
	handler = AuthMiddleware(s.authToken, s.logger, handler)
	handler = LoggingMiddleware(s.logger, handler)
	handler = RecoveryMiddleware(s.logger, handler)

	return handler
}

// registerRoutes binds all API routes using Go 1.22 method-aware patterns.
func (s *Server) registerRoutes(mux *http.ServeMux) {
	// Health
	mux.HandleFunc("GET /health", s.handleHealth)

	// Jobs
	mux.HandleFunc("POST /jobs", s.handleCreateJob)
	mux.HandleFunc("GET /jobs", s.handleListJobs)
	mux.HandleFunc("GET /jobs/{jobId}", s.handleGetJob)
	mux.HandleFunc("POST /jobs/{jobId}/cancel", s.handleCancelJob)
	mux.HandleFunc("POST /jobs/{jobId}/result", s.handleJobResult)

	// Agents
	mux.HandleFunc("POST /agents/register", s.handleRegisterAgent)
	mux.HandleFunc("POST /agents/{agentId}/heartbeat", s.handleHeartbeat)
	mux.HandleFunc("POST /agents/{agentId}/next-job", s.handleNextJob)
	mux.HandleFunc("GET /agents", s.handleListAgents)
}

// StartAgentHealthChecker runs a background goroutine that periodically marks
// stale agents as offline. It respects context cancellation for clean shutdown.
func (s *Server) StartAgentHealthChecker(interval, threshold time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			count := s.store.MarkStaleAgentsOffline(threshold)
			if count > 0 {
				s.logger.Info("marked agents offline", "count", count)
			}
		}
	}()
}

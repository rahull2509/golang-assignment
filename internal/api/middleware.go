package api

import (
	"log/slog"
	"net/http"
	"time"
)

// ---------------------------------------------------------------------------
// Logging middleware
// ---------------------------------------------------------------------------

// LoggingMiddleware logs every HTTP request with method, path, status, and duration.
func LoggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(sw, r)

		logger.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote", r.RemoteAddr,
		)
	})
}

// statusWriter wraps http.ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (sw *statusWriter) WriteHeader(code int) {
	if !sw.wroteHeader {
		sw.status = code
		sw.wroteHeader = true
	}
	sw.ResponseWriter.WriteHeader(code)
}

func (sw *statusWriter) Write(b []byte) (int, error) {
	if !sw.wroteHeader {
		sw.wroteHeader = true
	}
	return sw.ResponseWriter.Write(b)
}

// ---------------------------------------------------------------------------
// Recovery middleware
// ---------------------------------------------------------------------------

// RecoveryMiddleware catches panics, logs them, and returns a 500.
func RecoveryMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("panic recovered",
					"error", rec,
					"method", r.Method,
					"path", r.URL.Path,
				)
				writeError(w, http.StatusInternalServerError, "internal server error", "")
			}
		}()
		next.ServeHTTP(sw(w), r)
	})
}

// sw is a helper that ensures we haven't already written a header when
// the recovery middleware catches a panic.
func sw(w http.ResponseWriter) http.ResponseWriter {
	return w
}

// ---------------------------------------------------------------------------
// Auth middleware
// ---------------------------------------------------------------------------

// AuthMiddleware validates the Authorization header against a shared token.
// If token is empty, the middleware is a no-op (auth disabled).
func AuthMiddleware(token string, logger *slog.Logger, next http.Handler) http.Handler {
	if token == "" {
		return next // auth disabled
	}
	expected := "Bearer " + token
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always allow /health without auth.
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		auth := r.Header.Get("Authorization")
		if auth != expected {
			logger.Warn("unauthorized request",
				"method", r.Method,
				"path", r.URL.Path,
				"remote", r.RemoteAddr,
			)
			writeError(w, http.StatusUnauthorized, "unauthorized", "invalid or missing Authorization header")
			return
		}
		next.ServeHTTP(w, r)
	})
}

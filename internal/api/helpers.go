package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"taskbridge/internal/model"
)

// maxBodySize limits request body size to prevent abuse (1 MB).
const maxBodySize = 1 << 20

// ---------------------------------------------------------------------------
// JSON helpers
// ---------------------------------------------------------------------------

// writeJSON serialises data as JSON and writes it with the given status code.
func writeJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

// writeError sends a structured error response.
func writeError(w http.ResponseWriter, code int, message string, details string) {
	writeJSON(w, code, model.ErrorResponse{
		Error:   message,
		Code:    code,
		Details: details,
	})
}

// readJSON decodes the request body into dst. It enforces a max body size
// and returns a user-friendly error message on failure.
func readJSON(r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, maxBodySize)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	// Ensure there's no trailing content.
	if dec.More() {
		return fmt.Errorf("request body must contain a single JSON object")
	}
	return nil
}

// pathParam extracts a named path parameter from the request.
// Go 1.22 patterns like "/jobs/{jobId}" make this available via r.PathValue.
func pathParam(r *http.Request, name string) string {
	return r.PathValue(name)
}

// ---------------------------------------------------------------------------
// Method guard
// ---------------------------------------------------------------------------

// requireMethod is a simple guard — returns false and writes a 405 if the
// request method doesn't match. This is mostly a safety net since Go 1.22
// routing handles methods, but some edge cases can slip through.
func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", "expected "+method)
		return false
	}
	return true
}

// readBodyAndClose ensures the body is fully read and closed to avoid leaking
// connections. Called in middleware.
func readBodyAndClose(body io.ReadCloser) {
	_, _ = io.Copy(io.Discard, body)
	_ = body.Close()
}

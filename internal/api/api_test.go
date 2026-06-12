package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"taskbridge/internal/model"
	"taskbridge/internal/store"
)

// newTestServer creates a Server backed by a fresh MemoryStore for testing.
func newTestServer(t *testing.T) (*Server, *store.MemoryStore) {
	t.Helper()
	ms := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	srv := NewServer(ms, logger, "")
	return srv, ms
}

func doRequest(handler http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

// ---------------------------------------------------------------------------
// Health
// ---------------------------------------------------------------------------

func TestHealth(t *testing.T) {
	srv, _ := newTestServer(t)
	rr := doRequest(srv.Handler(), "GET", "/health", nil)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var resp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %v", resp["status"])
	}
}

// ---------------------------------------------------------------------------
// POST /jobs
// ---------------------------------------------------------------------------

func TestCreateJob_Success(t *testing.T) {
	srv, _ := newTestServer(t)
	body := model.CreateJobRequest{
		Name:           "test-http",
		Type:           model.JobHTTPCheck,
		Payload:        map[string]any{"url": "http://example.com"},
		TimeoutSeconds: 10,
		MaxRetries:     2,
	}

	rr := doRequest(srv.Handler(), "POST", "/jobs", body)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var job model.Job
	_ = json.Unmarshal(rr.Body.Bytes(), &job)
	if job.Name != "test-http" {
		t.Errorf("expected name test-http, got %s", job.Name)
	}
	if job.Status != model.JobPending {
		t.Errorf("expected PENDING, got %s", job.Status)
	}
	if job.ID == "" {
		t.Error("expected generated job ID")
	}
}

func TestCreateJob_ValidationError(t *testing.T) {
	srv, _ := newTestServer(t)
	body := model.CreateJobRequest{
		Name: "", // missing
		Type: model.JobHTTPCheck,
		Payload: map[string]any{}, // missing url
	}

	rr := doRequest(srv.Handler(), "POST", "/jobs", body)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestCreateJob_InvalidJSON(t *testing.T) {
	srv, _ := newTestServer(t)
	req := httptest.NewRequest("POST", "/jobs", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// GET /jobs
// ---------------------------------------------------------------------------

func TestListJobs_Empty(t *testing.T) {
	srv, _ := newTestServer(t)
	rr := doRequest(srv.Handler(), "GET", "/jobs", nil)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp model.ListJobsResponse
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Total != 0 {
		t.Errorf("expected 0 jobs, got %d", resp.Total)
	}
}

func TestListJobs_WithJobs(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Handler()

	// Create two jobs.
	doRequest(handler, "POST", "/jobs", model.CreateJobRequest{
		Name: "j1", Type: model.JobHTTPCheck, Payload: map[string]any{"url": "http://a.com"},
	})
	doRequest(handler, "POST", "/jobs", model.CreateJobRequest{
		Name: "j2", Type: model.JobWait, Payload: map[string]any{"duration_seconds": 1.0},
	})

	rr := doRequest(handler, "GET", "/jobs", nil)
	var resp model.ListJobsResponse
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Total != 2 {
		t.Errorf("expected 2 jobs, got %d", resp.Total)
	}
}

// ---------------------------------------------------------------------------
// GET /jobs/{jobId}
// ---------------------------------------------------------------------------

func TestGetJob_NotFound(t *testing.T) {
	srv, _ := newTestServer(t)
	rr := doRequest(srv.Handler(), "GET", "/jobs/nonexistent", nil)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestGetJob_Found(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Handler()

	// Create a job first.
	createRR := doRequest(handler, "POST", "/jobs", model.CreateJobRequest{
		Name: "find-me", Type: model.JobHTTPCheck, Payload: map[string]any{"url": "http://a.com"},
	})
	var created model.Job
	_ = json.Unmarshal(createRR.Body.Bytes(), &created)

	// Fetch it.
	rr := doRequest(handler, "GET", "/jobs/"+created.ID, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var fetched model.Job
	_ = json.Unmarshal(rr.Body.Bytes(), &fetched)
	if fetched.ID != created.ID {
		t.Errorf("expected %s, got %s", created.ID, fetched.ID)
	}
}

// ---------------------------------------------------------------------------
// POST /jobs/{jobId}/cancel
// ---------------------------------------------------------------------------

func TestCancelJob_Success(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Handler()

	createRR := doRequest(handler, "POST", "/jobs", model.CreateJobRequest{
		Name: "cancel-me", Type: model.JobWait, Payload: map[string]any{"duration_seconds": 5.0},
	})
	var created model.Job
	_ = json.Unmarshal(createRR.Body.Bytes(), &created)

	rr := doRequest(handler, "POST", "/jobs/"+created.ID+"/cancel", nil)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCancelJob_NotFound(t *testing.T) {
	srv, _ := newTestServer(t)
	rr := doRequest(srv.Handler(), "POST", "/jobs/nonexistent/cancel", nil)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

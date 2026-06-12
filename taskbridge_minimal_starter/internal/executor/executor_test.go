package executor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"taskbridge/internal/model"
)

// ---------------------------------------------------------------------------
// WaitExecutor tests
// ---------------------------------------------------------------------------

func TestWaitExecutor_Success(t *testing.T) {
	ex := &WaitExecutor{}
	job := model.Job{
		ID:   "j1",
		Type: model.JobWait,
		Payload: map[string]any{
			"duration_seconds": 0.1,
		},
	}

	result := ex.Execute(context.Background(), job)
	if result.Status != model.JobSuccess {
		t.Errorf("expected SUCCESS, got %s: %s", result.Status, result.Error)
	}
}

func TestWaitExecutor_ContextCanceled(t *testing.T) {
	ex := &WaitExecutor{}
	job := model.Job{
		ID:   "j1",
		Type: model.JobWait,
		Payload: map[string]any{
			"duration_seconds": 10.0,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result := ex.Execute(ctx, job)
	if result.Status != model.JobFailed {
		t.Errorf("expected FAILED (timeout), got %s", result.Status)
	}
}

func TestWaitExecutor_InvalidDuration(t *testing.T) {
	ex := &WaitExecutor{}
	job := model.Job{
		ID:      "j1",
		Type:    model.JobWait,
		Payload: map[string]any{"duration_seconds": 0.0},
	}

	result := ex.Execute(context.Background(), job)
	if result.Status != model.JobFailed {
		t.Errorf("expected FAILED for zero duration, got %s", result.Status)
	}
}

// ---------------------------------------------------------------------------
// HTTPCheckExecutor tests
// ---------------------------------------------------------------------------

func TestHTTPCheckExecutor_Success(t *testing.T) {
	// Start a test HTTP server.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	ex := &HTTPCheckExecutor{}
	job := model.Job{
		ID:   "j1",
		Type: model.JobHTTPCheck,
		Payload: map[string]any{
			"url":             ts.URL,
			"expected_status": float64(200),
		},
	}

	result := ex.Execute(context.Background(), job)
	if result.Status != model.JobSuccess {
		t.Errorf("expected SUCCESS, got %s: %s", result.Status, result.Error)
	}
}

func TestHTTPCheckExecutor_StatusMismatch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	ex := &HTTPCheckExecutor{}
	job := model.Job{
		ID:   "j1",
		Type: model.JobHTTPCheck,
		Payload: map[string]any{
			"url":             ts.URL,
			"expected_status": float64(200),
		},
	}

	result := ex.Execute(context.Background(), job)
	if result.Status != model.JobFailed {
		t.Errorf("expected FAILED, got %s", result.Status)
	}
}

func TestHTTPCheckExecutor_Timeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer ts.Close()

	ex := &HTTPCheckExecutor{}
	job := model.Job{
		ID:   "j1",
		Type: model.JobHTTPCheck,
		Payload: map[string]any{
			"url":             ts.URL,
			"expected_status": float64(200),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result := ex.Execute(ctx, job)
	if result.Status != model.JobFailed {
		t.Errorf("expected FAILED (timeout), got %s", result.Status)
	}
}

// ---------------------------------------------------------------------------
// FileExistsExecutor tests
// ---------------------------------------------------------------------------

func TestFileExistsExecutor_Exists(t *testing.T) {
	// Create a temp file.
	tmpFile := filepath.Join(t.TempDir(), "testfile.txt")
	if err := os.WriteFile(tmpFile, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	ex := &FileExistsExecutor{}
	job := model.Job{
		ID:      "j1",
		Type:    model.JobFileExists,
		Payload: map[string]any{"path": tmpFile},
	}

	result := ex.Execute(context.Background(), job)
	if result.Status != model.JobSuccess {
		t.Errorf("expected SUCCESS, got %s: %s", result.Status, result.Error)
	}
	if result.Result["exists"] != true {
		t.Error("expected exists=true")
	}
}

func TestFileExistsExecutor_NotExists(t *testing.T) {
	ex := &FileExistsExecutor{}
	job := model.Job{
		ID:      "j1",
		Type:    model.JobFileExists,
		Payload: map[string]any{"path": "/nonexistent/file/path"},
	}

	result := ex.Execute(context.Background(), job)
	if result.Status != model.JobFailed {
		t.Errorf("expected FAILED, got %s", result.Status)
	}
}

// ---------------------------------------------------------------------------
// WriteFileExecutor tests
// ---------------------------------------------------------------------------

func TestWriteFileExecutor_Success(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "output.txt")

	ex := &WriteFileExecutor{}
	job := model.Job{
		ID:   "j1",
		Type: model.JobWriteFile,
		Payload: map[string]any{
			"path":    tmpFile,
			"content": "hello world",
		},
	}

	result := ex.Execute(context.Background(), job)
	if result.Status != model.JobSuccess {
		t.Errorf("expected SUCCESS, got %s: %s", result.Status, result.Error)
	}

	// Verify file content.
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello world" {
		t.Errorf("expected 'hello world', got %q", string(data))
	}
}

// ---------------------------------------------------------------------------
// ChecksumExecutor tests
// ---------------------------------------------------------------------------

func TestChecksumExecutor_Success(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "checksum-test.txt")
	if err := os.WriteFile(tmpFile, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	ex := &ChecksumExecutor{}
	job := model.Job{
		ID:      "j1",
		Type:    model.JobChecksum,
		Payload: map[string]any{"path": tmpFile},
	}

	result := ex.Execute(context.Background(), job)
	if result.Status != model.JobSuccess {
		t.Errorf("expected SUCCESS, got %s: %s", result.Status, result.Error)
	}
	if result.Result["checksum"] == nil {
		t.Error("expected checksum in result")
	}
	// SHA256("hello") = 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824
	expected := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if result.Result["checksum"] != expected {
		t.Errorf("expected checksum %s, got %s", expected, result.Result["checksum"])
	}
}

func TestChecksumExecutor_Mismatch(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "checksum-test.txt")
	if err := os.WriteFile(tmpFile, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	ex := &ChecksumExecutor{}
	job := model.Job{
		ID:   "j1",
		Type: model.JobChecksum,
		Payload: map[string]any{
			"path":              tmpFile,
			"expected_checksum": "wrong",
		},
	}

	result := ex.Execute(context.Background(), job)
	if result.Status != model.JobFailed {
		t.Errorf("expected FAILED, got %s", result.Status)
	}
}

// ---------------------------------------------------------------------------
// CopyFileExecutor tests
// ---------------------------------------------------------------------------

func TestCopyFileExecutor_Success(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.txt")
	dst := filepath.Join(dir, "dest.txt")

	if err := os.WriteFile(src, []byte("copy me"), 0644); err != nil {
		t.Fatal(err)
	}

	ex := &CopyFileExecutor{}
	job := model.Job{
		ID:   "j1",
		Type: model.JobCopyFile,
		Payload: map[string]any{
			"source":      src,
			"destination": dst,
		},
	}

	result := ex.Execute(context.Background(), job)
	if result.Status != model.JobSuccess {
		t.Errorf("expected SUCCESS, got %s: %s", result.Status, result.Error)
	}

	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "copy me" {
		t.Errorf("expected 'copy me', got %q", string(data))
	}
}

// ---------------------------------------------------------------------------
// Registry tests
// ---------------------------------------------------------------------------

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()
	r.Register(&WaitExecutor{})

	ex, ok := r.Get(model.JobWait)
	if !ok {
		t.Fatal("expected wait executor to be registered")
	}
	if ex.Type() != model.JobWait {
		t.Errorf("expected wait type, got %s", ex.Type())
	}
}

func TestRegistry_GetUnknown(t *testing.T) {
	r := NewRegistry()
	_, ok := r.Get("nonexistent")
	if ok {
		t.Error("expected not found")
	}
}

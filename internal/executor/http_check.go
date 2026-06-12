package executor

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"taskbridge/internal/model"
)

// HTTPCheckExecutor checks whether an HTTP endpoint returns the expected status code.
//
// Required payload keys:
//   - url (string): the URL to check
//   - expected_status (float64): the expected HTTP status code
type HTTPCheckExecutor struct{}

func (e *HTTPCheckExecutor) Type() model.JobType {
	return model.JobHTTPCheck
}

func (e *HTTPCheckExecutor) Execute(ctx context.Context, job model.Job) Result {
	logs := make([]string, 0, 4)

	// Extract payload.
	rawURL, _ := job.Payload["url"].(string)
	if rawURL == "" {
		return Result{
			Status: model.JobFailed,
			Logs:   []string{"payload.url is required"},
			Error:  "missing url in payload",
		}
	}

	expectedStatus := 200
	if v, ok := job.Payload["expected_status"]; ok {
		switch val := v.(type) {
		case float64:
			expectedStatus = int(val)
		case int:
			expectedStatus = val
		case string:
			if parsed, err := strconv.Atoi(val); err == nil {
				expectedStatus = parsed
			}
		}
	}

	logs = append(logs, fmt.Sprintf("checking %s (expected: %d)", rawURL, expectedStatus))

	// Build request with context for timeout support.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return Result{
			Status: model.JobFailed,
			Logs:   append(logs, "failed to create request: "+err.Error()),
			Error:  err.Error(),
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Result{
			Status: model.JobFailed,
			Logs:   append(logs, "request failed: "+err.Error()),
			Error:  err.Error(),
		}
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	logs = append(logs, fmt.Sprintf("received status: %d", resp.StatusCode))

	if resp.StatusCode != expectedStatus {
		return Result{
			Status: model.JobFailed,
			Logs:   logs,
			Error:  fmt.Sprintf("expected status %d, got %d", expectedStatus, resp.StatusCode),
			Result: map[string]any{"actual_status": resp.StatusCode},
		}
	}

	logs = append(logs, "http check passed")
	return Result{
		Status: model.JobSuccess,
		Logs:   logs,
		Result: map[string]any{"actual_status": resp.StatusCode},
	}
}

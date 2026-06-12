package executor

import (
	"context"
	"fmt"
	"os"

	"taskbridge/internal/model"
)

// WriteFileExecutor writes controlled content to a file.
//
// Required payload keys:
//   - path (string): the file path to write to
//   - content (string): the content to write
//
// Optional payload keys:
//   - mode (float64): file permission mode (default: 0644)
type WriteFileExecutor struct{}

func (e *WriteFileExecutor) Type() model.JobType {
	return model.JobWriteFile
}

func (e *WriteFileExecutor) Execute(ctx context.Context, job model.Job) Result {
	logs := make([]string, 0, 3)

	filePath, _ := job.Payload["path"].(string)
	content, _ := job.Payload["content"].(string)

	if filePath == "" {
		return Result{
			Status: model.JobFailed,
			Logs:   []string{"payload.path is required"},
			Error:  "missing path in payload",
		}
	}

	// Note: content can be empty string (valid: creating an empty file).
	if _, ok := job.Payload["content"]; !ok {
		return Result{
			Status: model.JobFailed,
			Logs:   []string{"payload.content is required"},
			Error:  "missing content in payload",
		}
	}

	// Default file permissions.
	mode := os.FileMode(0644)
	if v, ok := job.Payload["mode"].(float64); ok {
		mode = os.FileMode(int(v))
	}

	logs = append(logs, fmt.Sprintf("writing %d bytes to %s (mode: %o)", len(content), filePath, mode))

	// Check context.
	select {
	case <-ctx.Done():
		return Result{
			Status: model.JobFailed,
			Logs:   append(logs, "operation canceled"),
			Error:  ctx.Err().Error(),
		}
	default:
	}

	if err := os.WriteFile(filePath, []byte(content), mode); err != nil {
		return Result{
			Status: model.JobFailed,
			Logs:   append(logs, "write failed: "+err.Error()),
			Error:  err.Error(),
		}
	}

	logs = append(logs, "file written successfully")
	return Result{
		Status: model.JobSuccess,
		Logs:   logs,
		Result: map[string]any{"bytes_written": len(content), "path": filePath},
	}
}

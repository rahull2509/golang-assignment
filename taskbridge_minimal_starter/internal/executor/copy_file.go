package executor

import (
	"context"
	"fmt"
	"io"
	"os"

	"taskbridge/internal/model"
)

// CopyFileExecutor copies a file from source to destination.
//
// Required payload keys:
//   - source (string): source file path
//   - destination (string): destination file path
type CopyFileExecutor struct{}

func (e *CopyFileExecutor) Type() model.JobType {
	return model.JobCopyFile
}

func (e *CopyFileExecutor) Execute(ctx context.Context, job model.Job) Result {
	logs := make([]string, 0, 4)

	source, _ := job.Payload["source"].(string)
	destination, _ := job.Payload["destination"].(string)

	if source == "" {
		return Result{
			Status: model.JobFailed,
			Logs:   []string{"payload.source is required"},
			Error:  "missing source in payload",
		}
	}
	if destination == "" {
		return Result{
			Status: model.JobFailed,
			Logs:   []string{"payload.destination is required"},
			Error:  "missing destination in payload",
		}
	}

	logs = append(logs, fmt.Sprintf("copying %s → %s", source, destination))

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

	src, err := os.Open(source)
	if err != nil {
		return Result{
			Status: model.JobFailed,
			Logs:   append(logs, "failed to open source: "+err.Error()),
			Error:  err.Error(),
		}
	}
	defer src.Close()

	dst, err := os.Create(destination)
	if err != nil {
		return Result{
			Status: model.JobFailed,
			Logs:   append(logs, "failed to create destination: "+err.Error()),
			Error:  err.Error(),
		}
	}
	defer dst.Close()

	n, err := io.Copy(dst, src)
	if err != nil {
		return Result{
			Status: model.JobFailed,
			Logs:   append(logs, "copy failed: "+err.Error()),
			Error:  err.Error(),
		}
	}

	logs = append(logs, fmt.Sprintf("copied %d bytes successfully", n))
	return Result{
		Status: model.JobSuccess,
		Logs:   logs,
		Result: map[string]any{"bytes_copied": n},
	}
}

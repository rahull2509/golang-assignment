package executor

import (
	"context"
	"fmt"
	"os"

	"taskbridge/internal/model"
)

// FileExistsExecutor checks whether a file exists at a given path.
//
// Required payload keys:
//   - path (string): the file path to check
type FileExistsExecutor struct{}

func (e *FileExistsExecutor) Type() model.JobType {
	return model.JobFileExists
}

func (e *FileExistsExecutor) Execute(ctx context.Context, job model.Job) Result {
	logs := make([]string, 0, 3)

	filePath, _ := job.Payload["path"].(string)
	if filePath == "" {
		return Result{
			Status: model.JobFailed,
			Logs:   []string{"payload.path is required"},
			Error:  "missing path in payload",
		}
	}

	logs = append(logs, fmt.Sprintf("checking if file exists: %s", filePath))

	// Check context before doing I/O.
	select {
	case <-ctx.Done():
		return Result{
			Status: model.JobFailed,
			Logs:   append(logs, "operation canceled"),
			Error:  ctx.Err().Error(),
		}
	default:
	}

	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return Result{
				Status: model.JobFailed,
				Logs:   append(logs, "file does not exist"),
				Error:  "file not found",
				Result: map[string]any{"exists": false},
			}
		}
		return Result{
			Status: model.JobFailed,
			Logs:   append(logs, "stat error: "+err.Error()),
			Error:  err.Error(),
		}
	}

	logs = append(logs, fmt.Sprintf("file exists — size: %d bytes", info.Size()))
	return Result{
		Status: model.JobSuccess,
		Logs:   logs,
		Result: map[string]any{
			"exists":    true,
			"size":      info.Size(),
			"is_dir":    info.IsDir(),
			"mod_time":  info.ModTime().UTC().String(),
		},
	}
}

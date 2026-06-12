package executor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"taskbridge/internal/model"
)

// ChecksumExecutor calculates a SHA-256 checksum for a file.
//
// Required payload keys:
//   - path (string): the file path to checksum
//
// Optional payload keys:
//   - expected_checksum (string): if provided, the result is compared
type ChecksumExecutor struct{}

func (e *ChecksumExecutor) Type() model.JobType {
	return model.JobChecksum
}

func (e *ChecksumExecutor) Execute(ctx context.Context, job model.Job) Result {
	logs := make([]string, 0, 4)

	filePath, _ := job.Payload["path"].(string)
	if filePath == "" {
		return Result{
			Status: model.JobFailed,
			Logs:   []string{"payload.path is required"},
			Error:  "missing path in payload",
		}
	}

	logs = append(logs, fmt.Sprintf("computing SHA-256 checksum for: %s", filePath))

	// Check context before expensive I/O.
	select {
	case <-ctx.Done():
		return Result{
			Status: model.JobFailed,
			Logs:   append(logs, "operation canceled"),
			Error:  ctx.Err().Error(),
		}
	default:
	}

	f, err := os.Open(filePath)
	if err != nil {
		return Result{
			Status: model.JobFailed,
			Logs:   append(logs, "failed to open file: "+err.Error()),
			Error:  err.Error(),
		}
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return Result{
			Status: model.JobFailed,
			Logs:   append(logs, "failed to read file: "+err.Error()),
			Error:  err.Error(),
		}
	}

	checksum := hex.EncodeToString(h.Sum(nil))
	logs = append(logs, fmt.Sprintf("checksum: %s", checksum))

	// Optional comparison.
	if expected, ok := job.Payload["expected_checksum"].(string); ok && expected != "" {
		if checksum != expected {
			return Result{
				Status: model.JobFailed,
				Logs:   append(logs, fmt.Sprintf("mismatch: expected %s", expected)),
				Error:  "checksum mismatch",
				Result: map[string]any{"checksum": checksum, "expected": expected, "match": false},
			}
		}
		logs = append(logs, "checksum matches expected value")
	}

	return Result{
		Status: model.JobSuccess,
		Logs:   logs,
		Result: map[string]any{"checksum": checksum, "algorithm": "sha256"},
	}
}

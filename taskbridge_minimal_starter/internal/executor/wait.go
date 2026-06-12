package executor

import (
	"context"
	"fmt"
	"time"

	"taskbridge/internal/model"
)

// WaitExecutor waits for a fixed duration. Useful for testing retries and timeouts.
//
// Required payload keys:
//   - duration_seconds (float64): number of seconds to wait
type WaitExecutor struct{}

func (e *WaitExecutor) Type() model.JobType {
	return model.JobWait
}

func (e *WaitExecutor) Execute(ctx context.Context, job model.Job) Result {
	logs := make([]string, 0, 3)

	durationSec := 0.0
	if v, ok := job.Payload["duration_seconds"].(float64); ok {
		durationSec = v
	}
	if durationSec <= 0 {
		return Result{
			Status: model.JobFailed,
			Logs:   []string{"payload.duration_seconds must be > 0"},
			Error:  "invalid duration",
		}
	}

	dur := time.Duration(durationSec * float64(time.Second))
	logs = append(logs, fmt.Sprintf("waiting for %s", dur))

	start := time.Now()

	select {
	case <-ctx.Done():
		elapsed := time.Since(start)
		return Result{
			Status: model.JobFailed,
			Logs:   append(logs, fmt.Sprintf("canceled after %s", elapsed.Round(time.Millisecond))),
			Error:  ctx.Err().Error(),
			Result: map[string]any{"elapsed_seconds": elapsed.Seconds()},
		}
	case <-time.After(dur):
		elapsed := time.Since(start)
		logs = append(logs, fmt.Sprintf("wait completed after %s", elapsed.Round(time.Millisecond)))
		return Result{
			Status: model.JobSuccess,
			Logs:   logs,
			Result: map[string]any{"elapsed_seconds": elapsed.Seconds()},
		}
	}
}

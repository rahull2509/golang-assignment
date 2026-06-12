package executor

import (
	"context"
	"fmt"
	"net"
	"time"

	"taskbridge/internal/model"
)

// TCPCheckExecutor checks whether a TCP address is reachable.
//
// Required payload keys:
//   - address (string): host:port to check (e.g. "localhost:5432")
type TCPCheckExecutor struct{}

func (e *TCPCheckExecutor) Type() model.JobType {
	return model.JobTCPCheck
}

func (e *TCPCheckExecutor) Execute(ctx context.Context, job model.Job) Result {
	logs := make([]string, 0, 3)

	address, _ := job.Payload["address"].(string)
	if address == "" {
		return Result{
			Status: model.JobFailed,
			Logs:   []string{"payload.address is required"},
			Error:  "missing address in payload",
		}
	}

	logs = append(logs, fmt.Sprintf("checking TCP connectivity to %s", address))

	// Use a dialer that respects the context for cancellation/timeout.
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
	}

	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return Result{
			Status: model.JobFailed,
			Logs:   append(logs, "connection failed: "+err.Error()),
			Error:  err.Error(),
			Result: map[string]any{"reachable": false},
		}
	}
	_ = conn.Close()

	logs = append(logs, "tcp check passed — connection successful")
	return Result{
		Status: model.JobSuccess,
		Logs:   logs,
		Result: map[string]any{"reachable": true},
	}
}

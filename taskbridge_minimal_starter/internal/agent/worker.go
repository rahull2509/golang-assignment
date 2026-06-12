package agent

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"time"

	"taskbridge/internal/executor"
	"taskbridge/internal/model"
)

// Worker is the agent's main event loop. It registers with the server,
// sends periodic heartbeats, and polls for jobs to execute.
type Worker struct {
	client       *Client
	registry     *executor.Registry
	logger       *slog.Logger
	capabilities []string
	pollInterval time.Duration
}

// NewWorker creates a Worker with all required dependencies.
func NewWorker(
	client *Client,
	registry *executor.Registry,
	logger *slog.Logger,
	capabilities []string,
	pollInterval time.Duration,
) *Worker {
	return &Worker{
		client:       client,
		registry:     registry,
		logger:       logger,
		capabilities: capabilities,
		pollInterval: pollInterval,
	}
}

// Run starts the agent. It blocks until the context is canceled.
//
// Flow:
//  1. Register with the server.
//  2. Start a heartbeat goroutine.
//  3. Enter a poll loop: fetch job → execute → submit result.
func (w *Worker) Run(ctx context.Context) error {
	// -----------------------------------------------------------------------
	// 1. Register
	// -----------------------------------------------------------------------
	hostname, _ := os.Hostname()
	regReq := model.RegisterAgentRequest{
		ID:           w.client.agentID,
		Hostname:     hostname,
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		Version:      "1.0.0",
		Capabilities: w.capabilities,
	}

	agent, err := w.client.Register(regReq)
	if err != nil {
		return err
	}
	w.logger.Info("registered with server",
		"agent_id", agent.ID,
		"capabilities", agent.Capabilities,
	)

	// -----------------------------------------------------------------------
	// 2. Heartbeat goroutine
	// -----------------------------------------------------------------------
	go w.heartbeatLoop(ctx)

	// -----------------------------------------------------------------------
	// 3. Poll loop
	// -----------------------------------------------------------------------
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	w.logger.Info("starting poll loop", "interval", w.pollInterval)

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("agent shutting down")
			return nil
		case <-ticker.C:
			w.pollAndExecute(ctx)
		}
	}
}

// heartbeatLoop sends heartbeats every 10 seconds until the context is canceled.
func (w *Worker) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.client.Heartbeat(); err != nil {
				w.logger.Warn("heartbeat failed", "error", err)
			}
		}
	}
}

// pollAndExecute fetches the next job, executes it, and submits the result.
func (w *Worker) pollAndExecute(ctx context.Context) {
	job, err := w.client.PollNextJob(w.capabilities)
	if err != nil {
		w.logger.Warn("poll failed", "error", err)
		return
	}
	if job == nil {
		// No jobs available — normal idle state.
		return
	}

	w.logger.Info("job received",
		"job_id", job.ID,
		"type", job.Type,
		"name", job.Name,
		"attempt", job.AttemptCount,
	)

	// Look up the executor.
	ex, ok := w.registry.Get(job.Type)
	if !ok {
		w.logger.Error("no executor for job type", "type", job.Type)
		result := executor.Result{
			Status: model.JobFailed,
			Logs:   []string{"no executor registered for type: " + string(job.Type)},
			Error:  "unsupported job type",
		}
		if submitErr := w.client.SubmitResult(job.ID, result); submitErr != nil {
			w.logger.Error("failed to submit unsupported-type result", "error", submitErr)
		}
		return
	}

	// Execute with timeout.
	timeout := time.Duration(job.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	w.logger.Info("executing job",
		"job_id", job.ID,
		"timeout", timeout,
	)

	start := time.Now()
	result := ex.Execute(execCtx, *job)
	elapsed := time.Since(start)

	w.logger.Info("job executed",
		"job_id", job.ID,
		"status", result.Status,
		"duration_ms", elapsed.Milliseconds(),
	)

	// Submit result.
	if err := w.client.SubmitResult(job.ID, result); err != nil {
		w.logger.Error("failed to submit result",
			"job_id", job.ID,
			"error", err,
		)
	}
}

package store

import (
	"time"

	"taskbridge/internal/model"
)

// Store defines the required persistence operations.
// Two implementations are expected:
//   - MemoryStore (mandatory, in memory.go)
//   - SQLiteStore (stretch goal, in sqlite.go)
type Store interface {
	// Job operations
	CreateJob(job model.Job) (model.Job, error)
	ListJobs() ([]model.Job, error)
	GetJob(jobID string) (model.Job, bool, error)
	UpdateJob(job model.Job) error
	CancelJob(jobID string) error

	// Agent operations
	RegisterAgent(agent model.Agent) (model.Agent, error)
	Heartbeat(agentID string) error
	ListAgents() ([]model.Agent, error)

	// Assignment operations
	AssignNextJob(agentID string, capabilities []model.JobType) (model.Job, bool, error)
	CompleteJob(jobID string, status model.JobStatus, logs []string, result map[string]any, errMsg string) error

	// Health operations
	MarkStaleAgentsOffline(threshold time.Duration) int
}

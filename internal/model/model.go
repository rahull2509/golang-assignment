package model

import "time"

// ---------------------------------------------------------------------------
// Job Status
// ---------------------------------------------------------------------------

// JobStatus represents the lifecycle state of a job.
type JobStatus string

const (
	JobPending  JobStatus = "PENDING"
	JobRunning  JobStatus = "RUNNING"
	JobRetrying JobStatus = "RETRYING"
	JobSuccess  JobStatus = "SUCCESS"
	JobFailed   JobStatus = "FAILED"
	JobCanceled JobStatus = "CANCELED"
)

// IsTerminal returns true if the status represents a final state.
func (s JobStatus) IsTerminal() bool {
	return s == JobSuccess || s == JobFailed || s == JobCanceled
}

// ---------------------------------------------------------------------------
// Job Type
// ---------------------------------------------------------------------------

// JobType represents supported job execution types.
type JobType string

const (
	JobHTTPCheck  JobType = "http_check"
	JobTCPCheck   JobType = "tcp_check"
	JobFileExists JobType = "file_exists"
	JobChecksum   JobType = "checksum"
	JobCopyFile   JobType = "copy_file"
	JobWriteFile  JobType = "write_file"
	JobWait       JobType = "wait"
)

// AllJobTypes contains every recognised job type for validation purposes.
var AllJobTypes = map[JobType]bool{
	JobHTTPCheck:  true,
	JobTCPCheck:   true,
	JobFileExists: true,
	JobChecksum:   true,
	JobCopyFile:   true,
	JobWriteFile:  true,
	JobWait:       true,
}

// ---------------------------------------------------------------------------
// Agent Status
// ---------------------------------------------------------------------------

// AgentStatus represents the connectivity state of an agent.
type AgentStatus string

const (
	AgentOnline  AgentStatus = "online"
	AgentOffline AgentStatus = "offline"
)

// ---------------------------------------------------------------------------
// Job Entity
// ---------------------------------------------------------------------------

// Job is the main server-side job entity.
type Job struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Type            JobType        `json:"type"`
	Payload         map[string]any `json:"payload"`
	Status          JobStatus      `json:"status"`
	AssignedAgentID string         `json:"assigned_agent_id,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	StartedAt       *time.Time     `json:"started_at,omitempty"`
	FinishedAt      *time.Time     `json:"finished_at,omitempty"`
	AttemptCount    int            `json:"attempt_count"`
	MaxRetries      int            `json:"max_retries"`
	TimeoutSeconds  int            `json:"timeout_seconds"`
	Logs            []string       `json:"logs,omitempty"`
	Error           string         `json:"error,omitempty"`
	Result          map[string]any `json:"result,omitempty"`
}

// ---------------------------------------------------------------------------
// Agent Entity
// ---------------------------------------------------------------------------

// Agent is the server-side representation of a connected worker.
type Agent struct {
	ID           string      `json:"id"`
	Hostname     string      `json:"hostname"`
	OS           string      `json:"os"`
	Arch         string      `json:"arch"`
	Version      string      `json:"version"`
	Capabilities []JobType   `json:"capabilities"`
	LastSeen     time.Time   `json:"last_seen"`
	Status       AgentStatus `json:"status"`
}

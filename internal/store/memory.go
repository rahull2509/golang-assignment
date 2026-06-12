package store

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"taskbridge/internal/model"
)

// MemoryStore is a concurrency-safe in-memory implementation of the Store interface.
// It uses sync.RWMutex to allow concurrent reads while serialising writes.
type MemoryStore struct {
	mu     sync.RWMutex
	jobs   map[string]model.Job
	agents map[string]model.Agent

	// jobOrder preserves insertion order so AssignNextJob is deterministic (FIFO).
	jobOrder []string
}

// NewMemoryStore returns an initialised MemoryStore ready for use.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		jobs:     make(map[string]model.Job),
		agents:   make(map[string]model.Agent),
		jobOrder: make([]string, 0),
	}
}

// ---------------------------------------------------------------------------
// Job operations
// ---------------------------------------------------------------------------

// CreateJob stores a new job. The caller is expected to set the ID and CreatedAt.
func (m *MemoryStore) CreateJob(job model.Job) (model.Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.jobs[job.ID]; exists {
		return model.Job{}, fmt.Errorf("job %q already exists", job.ID)
	}

	m.jobs[job.ID] = job
	m.jobOrder = append(m.jobOrder, job.ID)
	return job, nil
}

// ListJobs returns all jobs sorted by creation time (oldest first).
func (m *MemoryStore) ListJobs() ([]model.Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]model.Job, 0, len(m.jobs))
	for _, id := range m.jobOrder {
		if job, ok := m.jobs[id]; ok {
			result = append(result, job)
		}
	}
	return result, nil
}

// GetJob returns a single job by ID. The boolean indicates whether the job was found.
func (m *MemoryStore) GetJob(jobID string) (model.Job, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	job, ok := m.jobs[jobID]
	return job, ok, nil
}

// UpdateJob replaces the job in the store. Returns an error if the job doesn't exist.
func (m *MemoryStore) UpdateJob(job model.Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.jobs[job.ID]; !ok {
		return fmt.Errorf("job %q not found", job.ID)
	}
	m.jobs[job.ID] = job
	return nil
}

// CancelJob transitions a job to CANCELED if it is in a cancellable state.
func (m *MemoryStore) CancelJob(jobID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[jobID]
	if !ok {
		return fmt.Errorf("job %q not found", jobID)
	}
	if job.Status.IsTerminal() {
		return fmt.Errorf("job %q is already in terminal state %s", jobID, job.Status)
	}

	now := time.Now().UTC()
	job.Status = model.JobCanceled
	job.FinishedAt = &now
	m.jobs[jobID] = job
	return nil
}

// ---------------------------------------------------------------------------
// Agent operations
// ---------------------------------------------------------------------------

// RegisterAgent stores or updates an agent. Re-registration is allowed (idempotent).
func (m *MemoryStore) RegisterAgent(agent model.Agent) (model.Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.agents[agent.ID] = agent
	return agent, nil
}

// Heartbeat updates the last_seen timestamp for the given agent.
func (m *MemoryStore) Heartbeat(agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, ok := m.agents[agentID]
	if !ok {
		return fmt.Errorf("agent %q not found", agentID)
	}

	agent.LastSeen = time.Now().UTC()
	agent.Status = model.AgentOnline
	m.agents[agentID] = agent
	return nil
}

// ListAgents returns all agents sorted alphabetically by ID.
func (m *MemoryStore) ListAgents() ([]model.Agent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]model.Agent, 0, len(m.agents))
	for _, a := range m.agents {
		result = append(result, a)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result, nil
}

// ---------------------------------------------------------------------------
// Job Assignment
// ---------------------------------------------------------------------------

// AssignNextJob finds the oldest PENDING job whose type matches the agent's
// capabilities, marks it RUNNING, and assigns it to the agent.
func (m *MemoryStore) AssignNextJob(agentID string, capabilities []model.JobType) (model.Job, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check agent exists.
	if _, ok := m.agents[agentID]; !ok {
		return model.Job{}, false, fmt.Errorf("agent %q not found", agentID)
	}

	// Build a set for O(1) capability lookup.
	capSet := make(map[model.JobType]bool, len(capabilities))
	for _, c := range capabilities {
		capSet[c] = true
	}

	// Walk jobs in insertion order (FIFO).
	for _, id := range m.jobOrder {
		job, ok := m.jobs[id]
		if !ok {
			continue
		}
		if job.Status != model.JobPending {
			continue
		}
		if !capSet[job.Type] {
			continue
		}

		// Assign the job.
		now := time.Now().UTC()
		job.Status = model.JobRunning
		job.AssignedAgentID = agentID
		job.StartedAt = &now
		job.AttemptCount++
		m.jobs[id] = job
		return job, true, nil
	}

	return model.Job{}, false, nil
}

// CompleteJob records the result of a job execution.
// If the job FAILED and retries remain, it is set back to PENDING for re-assignment.
func (m *MemoryStore) CompleteJob(jobID string, status model.JobStatus, logs []string, result map[string]any, errMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[jobID]
	if !ok {
		return fmt.Errorf("job %q not found", jobID)
	}

	now := time.Now().UTC()
	job.Logs = append(job.Logs, logs...)
	job.Result = result
	job.Error = errMsg

	// Retry logic: if the job failed and has retries remaining, queue it again.
	if status == model.JobFailed && job.AttemptCount < job.MaxRetries {
		job.Status = model.JobRetrying
		// Reset assignment so it can be picked up again.
		job.AssignedAgentID = ""
		job.StartedAt = nil
		// After a brief RETRYING state, we set it back to PENDING.
		// In a production system this could involve a delay, but for
		// the assignment we immediately re-queue.
		job.Status = model.JobPending
		m.jobs[jobID] = job
		return nil
	}

	// Terminal state.
	job.Status = status
	job.FinishedAt = &now
	m.jobs[jobID] = job
	return nil
}

// ---------------------------------------------------------------------------
// Agent Health
// ---------------------------------------------------------------------------

// MarkStaleAgentsOffline sets any agent whose LastSeen is older than the
// threshold to offline status. Returns the number of agents marked offline.
func (m *MemoryStore) MarkStaleAgentsOffline(threshold time.Duration) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().UTC().Add(-threshold)
	count := 0
	for id, agent := range m.agents {
		if agent.Status == model.AgentOnline && agent.LastSeen.Before(cutoff) {
			agent.Status = model.AgentOffline
			m.agents[id] = agent
			count++
		}
	}
	return count
}

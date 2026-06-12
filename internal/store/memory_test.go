package store

import (
	"sync"
	"testing"
	"time"

	"taskbridge/internal/model"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestJob(id string, jobType model.JobType) model.Job {
	return model.Job{
		ID:             id,
		Name:           "test-" + id,
		Type:           jobType,
		Payload:        map[string]any{"key": "value"},
		Status:         model.JobPending,
		CreatedAt:      time.Now().UTC(),
		MaxRetries:     2,
		TimeoutSeconds: 10,
	}
}

func newTestAgent(id string, caps ...model.JobType) model.Agent {
	return model.Agent{
		ID:           id,
		Hostname:     "test-host",
		OS:           "linux",
		Arch:         "amd64",
		Capabilities: caps,
		LastSeen:     time.Now().UTC(),
		Status:       model.AgentOnline,
	}
}

// ---------------------------------------------------------------------------
// Job CRUD tests
// ---------------------------------------------------------------------------

func TestMemoryStore_CreateJob(t *testing.T) {
	s := NewMemoryStore()
	job := newTestJob("j1", model.JobHTTPCheck)

	created, err := s.CreateJob(job)
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}
	if created.ID != "j1" {
		t.Errorf("expected ID j1, got %s", created.ID)
	}
}

func TestMemoryStore_CreateJob_Duplicate(t *testing.T) {
	s := NewMemoryStore()
	job := newTestJob("j1", model.JobHTTPCheck)
	_, _ = s.CreateJob(job)

	_, err := s.CreateJob(job)
	if err == nil {
		t.Fatal("expected error for duplicate job ID")
	}
}

func TestMemoryStore_ListJobs(t *testing.T) {
	s := NewMemoryStore()
	_, _ = s.CreateJob(newTestJob("j1", model.JobHTTPCheck))
	_, _ = s.CreateJob(newTestJob("j2", model.JobWait))

	jobs, err := s.ListJobs()
	if err != nil {
		t.Fatalf("ListJobs failed: %v", err)
	}
	if len(jobs) != 2 {
		t.Errorf("expected 2 jobs, got %d", len(jobs))
	}
	// Should be in insertion order.
	if jobs[0].ID != "j1" || jobs[1].ID != "j2" {
		t.Error("jobs not in insertion order")
	}
}

func TestMemoryStore_GetJob(t *testing.T) {
	s := NewMemoryStore()
	_, _ = s.CreateJob(newTestJob("j1", model.JobHTTPCheck))

	job, found, err := s.GetJob("j1")
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}
	if !found {
		t.Fatal("expected job to be found")
	}
	if job.ID != "j1" {
		t.Errorf("expected ID j1, got %s", job.ID)
	}
}

func TestMemoryStore_GetJob_NotFound(t *testing.T) {
	s := NewMemoryStore()
	_, found, err := s.GetJob("nonexistent")
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}
	if found {
		t.Error("expected not found")
	}
}

func TestMemoryStore_CancelJob(t *testing.T) {
	s := NewMemoryStore()
	_, _ = s.CreateJob(newTestJob("j1", model.JobHTTPCheck))

	if err := s.CancelJob("j1"); err != nil {
		t.Fatalf("CancelJob failed: %v", err)
	}

	job, _, _ := s.GetJob("j1")
	if job.Status != model.JobCanceled {
		t.Errorf("expected CANCELED, got %s", job.Status)
	}
	if job.FinishedAt == nil {
		t.Error("expected FinishedAt to be set")
	}
}

func TestMemoryStore_CancelJob_Terminal(t *testing.T) {
	s := NewMemoryStore()
	job := newTestJob("j1", model.JobHTTPCheck)
	job.Status = model.JobSuccess
	_, _ = s.CreateJob(job)

	if err := s.CancelJob("j1"); err == nil {
		t.Fatal("expected error when canceling terminal job")
	}
}

// ---------------------------------------------------------------------------
// Agent tests
// ---------------------------------------------------------------------------

func TestMemoryStore_RegisterAgent(t *testing.T) {
	s := NewMemoryStore()
	agent := newTestAgent("a1", model.JobHTTPCheck)

	registered, err := s.RegisterAgent(agent)
	if err != nil {
		t.Fatalf("RegisterAgent failed: %v", err)
	}
	if registered.ID != "a1" {
		t.Errorf("expected ID a1, got %s", registered.ID)
	}
}

func TestMemoryStore_RegisterAgent_Idempotent(t *testing.T) {
	s := NewMemoryStore()
	agent := newTestAgent("a1", model.JobHTTPCheck)

	_, _ = s.RegisterAgent(agent)
	_, err := s.RegisterAgent(agent) // Re-register
	if err != nil {
		t.Fatalf("re-registration should be idempotent, got: %v", err)
	}

	agents, _ := s.ListAgents()
	if len(agents) != 1 {
		t.Errorf("expected 1 agent, got %d", len(agents))
	}
}

func TestMemoryStore_Heartbeat(t *testing.T) {
	s := NewMemoryStore()
	agent := newTestAgent("a1", model.JobHTTPCheck)
	agent.LastSeen = time.Now().UTC().Add(-1 * time.Hour) // old timestamp
	_, _ = s.RegisterAgent(agent)

	if err := s.Heartbeat("a1"); err != nil {
		t.Fatalf("Heartbeat failed: %v", err)
	}

	agents, _ := s.ListAgents()
	if time.Since(agents[0].LastSeen) > time.Second {
		t.Error("expected LastSeen to be updated to now")
	}
}

func TestMemoryStore_Heartbeat_NotFound(t *testing.T) {
	s := NewMemoryStore()
	if err := s.Heartbeat("nonexistent"); err == nil {
		t.Fatal("expected error for unknown agent")
	}
}

func TestMemoryStore_ListAgents(t *testing.T) {
	s := NewMemoryStore()
	_, _ = s.RegisterAgent(newTestAgent("b1", model.JobHTTPCheck))
	_, _ = s.RegisterAgent(newTestAgent("a1", model.JobWait))

	agents, err := s.ListAgents()
	if err != nil {
		t.Fatalf("ListAgents failed: %v", err)
	}
	if len(agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(agents))
	}
	// Should be sorted alphabetically.
	if agents[0].ID != "a1" {
		t.Errorf("expected first agent a1, got %s", agents[0].ID)
	}
}

// ---------------------------------------------------------------------------
// Assignment tests
// ---------------------------------------------------------------------------

func TestMemoryStore_AssignNextJob(t *testing.T) {
	s := NewMemoryStore()
	_, _ = s.RegisterAgent(newTestAgent("a1", model.JobHTTPCheck, model.JobWait))
	_, _ = s.CreateJob(newTestJob("j1", model.JobHTTPCheck))

	job, found, err := s.AssignNextJob("a1", []model.JobType{model.JobHTTPCheck, model.JobWait})
	if err != nil {
		t.Fatalf("AssignNextJob failed: %v", err)
	}
	if !found {
		t.Fatal("expected a job to be assigned")
	}
	if job.ID != "j1" {
		t.Errorf("expected job j1, got %s", job.ID)
	}
	if job.Status != model.JobRunning {
		t.Errorf("expected RUNNING, got %s", job.Status)
	}
	if job.AssignedAgentID != "a1" {
		t.Errorf("expected assigned to a1, got %s", job.AssignedAgentID)
	}
	if job.AttemptCount != 1 {
		t.Errorf("expected attempt_count 1, got %d", job.AttemptCount)
	}
}

func TestMemoryStore_AssignNextJob_CapabilityMismatch(t *testing.T) {
	s := NewMemoryStore()
	_, _ = s.RegisterAgent(newTestAgent("a1", model.JobHTTPCheck))
	_, _ = s.CreateJob(newTestJob("j1", model.JobWait)) // agent can't handle wait

	_, found, err := s.AssignNextJob("a1", []model.JobType{model.JobHTTPCheck})
	if err != nil {
		t.Fatalf("AssignNextJob failed: %v", err)
	}
	if found {
		t.Error("expected no job assigned due to capability mismatch")
	}
}

func TestMemoryStore_AssignNextJob_FIFO(t *testing.T) {
	s := NewMemoryStore()
	_, _ = s.RegisterAgent(newTestAgent("a1", model.JobHTTPCheck))
	_, _ = s.CreateJob(newTestJob("j1", model.JobHTTPCheck))
	_, _ = s.CreateJob(newTestJob("j2", model.JobHTTPCheck))

	job, found, _ := s.AssignNextJob("a1", []model.JobType{model.JobHTTPCheck})
	if !found {
		t.Fatal("expected job")
	}
	if job.ID != "j1" {
		t.Errorf("expected first job j1 (FIFO), got %s", job.ID)
	}
}

func TestMemoryStore_AssignNextJob_SkipsNonPending(t *testing.T) {
	s := NewMemoryStore()
	_, _ = s.RegisterAgent(newTestAgent("a1", model.JobHTTPCheck))

	// Create two jobs, assign the first (making it RUNNING).
	_, _ = s.CreateJob(newTestJob("j1", model.JobHTTPCheck))
	_, _ = s.CreateJob(newTestJob("j2", model.JobHTTPCheck))
	_, _, _ = s.AssignNextJob("a1", []model.JobType{model.JobHTTPCheck})

	// Second assignment should get j2.
	job, found, _ := s.AssignNextJob("a1", []model.JobType{model.JobHTTPCheck})
	if !found {
		t.Fatal("expected j2 to be assigned")
	}
	if job.ID != "j2" {
		t.Errorf("expected j2, got %s", job.ID)
	}
}

// ---------------------------------------------------------------------------
// CompleteJob and retry tests
// ---------------------------------------------------------------------------

func TestMemoryStore_CompleteJob_Success(t *testing.T) {
	s := NewMemoryStore()
	_, _ = s.RegisterAgent(newTestAgent("a1", model.JobHTTPCheck))
	_, _ = s.CreateJob(newTestJob("j1", model.JobHTTPCheck))
	_, _, _ = s.AssignNextJob("a1", []model.JobType{model.JobHTTPCheck})

	err := s.CompleteJob("j1", model.JobSuccess, []string{"done"}, map[string]any{"ok": true}, "")
	if err != nil {
		t.Fatalf("CompleteJob failed: %v", err)
	}

	job, _, _ := s.GetJob("j1")
	if job.Status != model.JobSuccess {
		t.Errorf("expected SUCCESS, got %s", job.Status)
	}
	if job.FinishedAt == nil {
		t.Error("expected FinishedAt to be set")
	}
}

func TestMemoryStore_CompleteJob_RetryOnFailure(t *testing.T) {
	s := NewMemoryStore()
	_, _ = s.RegisterAgent(newTestAgent("a1", model.JobHTTPCheck))
	_, _ = s.CreateJob(newTestJob("j1", model.JobHTTPCheck)) // max_retries = 2
	_, _, _ = s.AssignNextJob("a1", []model.JobType{model.JobHTTPCheck})

	// First failure — should retry (attempt_count=1 < max_retries=2).
	err := s.CompleteJob("j1", model.JobFailed, []string{"error 1"}, nil, "fail")
	if err != nil {
		t.Fatalf("CompleteJob failed: %v", err)
	}

	job, _, _ := s.GetJob("j1")
	if job.Status != model.JobPending {
		t.Errorf("expected re-queued as PENDING, got %s", job.Status)
	}
	if job.AssignedAgentID != "" {
		t.Error("expected assigned_agent_id to be cleared for retry")
	}
}

func TestMemoryStore_CompleteJob_ExhaustedRetries(t *testing.T) {
	s := NewMemoryStore()
	_, _ = s.RegisterAgent(newTestAgent("a1", model.JobHTTPCheck))

	job := newTestJob("j1", model.JobHTTPCheck)
	job.MaxRetries = 1
	_, _ = s.CreateJob(job)

	// First attempt.
	_, _, _ = s.AssignNextJob("a1", []model.JobType{model.JobHTTPCheck})
	_ = s.CompleteJob("j1", model.JobFailed, nil, nil, "fail 1") // attempt 1 < max_retries 1 → retry

	// Second attempt (re-assigned).
	_, _, _ = s.AssignNextJob("a1", []model.JobType{model.JobHTTPCheck})
	_ = s.CompleteJob("j1", model.JobFailed, nil, nil, "fail 2") // attempt 2 >= max_retries 1 → fail

	result, _, _ := s.GetJob("j1")
	if result.Status != model.JobFailed {
		t.Errorf("expected FAILED after exhausting retries, got %s", result.Status)
	}
	if result.FinishedAt == nil {
		t.Error("expected FinishedAt to be set")
	}
}

// ---------------------------------------------------------------------------
// MarkStaleAgentsOffline tests
// ---------------------------------------------------------------------------

func TestMemoryStore_MarkStaleAgentsOffline(t *testing.T) {
	s := NewMemoryStore()
	agent := newTestAgent("a1", model.JobHTTPCheck)
	agent.LastSeen = time.Now().UTC().Add(-1 * time.Minute) // 1 minute ago
	_, _ = s.RegisterAgent(agent)

	count := s.MarkStaleAgentsOffline(30 * time.Second)
	if count != 1 {
		t.Errorf("expected 1 agent marked offline, got %d", count)
	}

	agents, _ := s.ListAgents()
	if agents[0].Status != model.AgentOffline {
		t.Errorf("expected offline, got %s", agents[0].Status)
	}
}

func TestMemoryStore_MarkStaleAgentsOffline_FreshAgent(t *testing.T) {
	s := NewMemoryStore()
	_, _ = s.RegisterAgent(newTestAgent("a1", model.JobHTTPCheck))

	count := s.MarkStaleAgentsOffline(30 * time.Second)
	if count != 0 {
		t.Errorf("expected 0 agents marked offline, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// Concurrent access test
// ---------------------------------------------------------------------------

func TestMemoryStore_ConcurrentAccess(t *testing.T) {
	s := NewMemoryStore()
	_, _ = s.RegisterAgent(newTestAgent("a1", model.JobHTTPCheck))

	const n = 100
	var wg sync.WaitGroup
	wg.Add(n * 3)

	// Concurrent writers.
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			j := newTestJob("conc-"+string(rune('A'+i%26))+"-"+time.Now().Format("150405.000"), model.JobHTTPCheck)
			_, _ = s.CreateJob(j)
		}(i)
	}

	// Concurrent readers.
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_, _ = s.ListJobs()
		}()
	}

	// Concurrent heartbeats.
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_ = s.Heartbeat("a1")
		}()
	}

	wg.Wait()
}

package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"taskbridge/internal/model"
)

// ---------------------------------------------------------------------------
// POST /agents/register
// ---------------------------------------------------------------------------

func TestRegisterAgent_Success(t *testing.T) {
	srv, _ := newTestServer(t)
	body := model.RegisterAgentRequest{
		ID:           "agent-1",
		Hostname:     "test-host",
		OS:           "linux",
		Arch:         "amd64",
		Version:      "1.0.0",
		Capabilities: []string{"http_check", "wait"},
	}

	rr := doRequest(srv.Handler(), "POST", "/agents/register", body)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var agent model.Agent
	_ = json.Unmarshal(rr.Body.Bytes(), &agent)
	if agent.ID != "agent-1" {
		t.Errorf("expected agent-1, got %s", agent.ID)
	}
	if agent.Status != model.AgentOnline {
		t.Errorf("expected online, got %s", agent.Status)
	}
}

func TestRegisterAgent_ValidationError(t *testing.T) {
	srv, _ := newTestServer(t)
	body := model.RegisterAgentRequest{
		ID:           "",
		Capabilities: []string{},
	}

	rr := doRequest(srv.Handler(), "POST", "/agents/register", body)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// POST /agents/{agentId}/heartbeat
// ---------------------------------------------------------------------------

func TestHeartbeat_Success(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Handler()

	// Register first.
	doRequest(handler, "POST", "/agents/register", model.RegisterAgentRequest{
		ID: "agent-1", Capabilities: []string{"http_check"},
	})

	rr := doRequest(handler, "POST", "/agents/agent-1/heartbeat", nil)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestHeartbeat_NotFound(t *testing.T) {
	srv, _ := newTestServer(t)
	rr := doRequest(srv.Handler(), "POST", "/agents/nonexistent/heartbeat", nil)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// GET /agents
// ---------------------------------------------------------------------------

func TestListAgents_Empty(t *testing.T) {
	srv, _ := newTestServer(t)
	rr := doRequest(srv.Handler(), "GET", "/agents", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp model.ListAgentsResponse
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Total != 0 {
		t.Errorf("expected 0, got %d", resp.Total)
	}
}

// ---------------------------------------------------------------------------
// POST /agents/{agentId}/next-job
// ---------------------------------------------------------------------------

func TestNextJob_NoJobs(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Handler()

	doRequest(handler, "POST", "/agents/register", model.RegisterAgentRequest{
		ID: "agent-1", Capabilities: []string{"http_check"},
	})

	rr := doRequest(handler, "POST", "/agents/agent-1/next-job", map[string]any{
		"capabilities": []string{"http_check"},
	})
	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204 (no jobs), got %d", rr.Code)
	}
}

func TestNextJob_AssignsJob(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Handler()

	// Register agent.
	doRequest(handler, "POST", "/agents/register", model.RegisterAgentRequest{
		ID: "agent-1", Capabilities: []string{"http_check"},
	})

	// Create job.
	doRequest(handler, "POST", "/jobs", model.CreateJobRequest{
		Name: "j1", Type: model.JobHTTPCheck, Payload: map[string]any{"url": "http://a.com"},
	})

	// Poll.
	rr := doRequest(handler, "POST", "/agents/agent-1/next-job", map[string]any{
		"capabilities": []string{"http_check"},
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var job model.Job
	_ = json.Unmarshal(rr.Body.Bytes(), &job)
	if job.Status != model.JobRunning {
		t.Errorf("expected RUNNING, got %s", job.Status)
	}
}

// ---------------------------------------------------------------------------
// POST /jobs/{jobId}/result
// ---------------------------------------------------------------------------

func TestJobResult_Success(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Handler()

	// Register + create + assign.
	doRequest(handler, "POST", "/agents/register", model.RegisterAgentRequest{
		ID: "agent-1", Capabilities: []string{"http_check"},
	})
	createRR := doRequest(handler, "POST", "/jobs", model.CreateJobRequest{
		Name: "j1", Type: model.JobHTTPCheck, Payload: map[string]any{"url": "http://a.com"},
	})
	var created model.Job
	_ = json.Unmarshal(createRR.Body.Bytes(), &created)

	doRequest(handler, "POST", "/agents/agent-1/next-job", map[string]any{
		"capabilities": []string{"http_check"},
	})

	// Submit result.
	rr := doRequest(handler, "POST", "/jobs/"+created.ID+"/result", model.JobResultRequest{
		Status: "SUCCESS",
		Logs:   []string{"check passed"},
		Result: map[string]any{"actual_status": 200},
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var job model.Job
	_ = json.Unmarshal(rr.Body.Bytes(), &job)
	if job.Status != model.JobSuccess {
		t.Errorf("expected SUCCESS, got %s", job.Status)
	}
}

func TestJobResult_InvalidStatus(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Handler()

	doRequest(handler, "POST", "/agents/register", model.RegisterAgentRequest{
		ID: "agent-1", Capabilities: []string{"http_check"},
	})
	createRR := doRequest(handler, "POST", "/jobs", model.CreateJobRequest{
		Name: "j1", Type: model.JobHTTPCheck, Payload: map[string]any{"url": "http://a.com"},
	})
	var created model.Job
	_ = json.Unmarshal(createRR.Body.Bytes(), &created)

	rr := doRequest(handler, "POST", "/jobs/"+created.ID+"/result", model.JobResultRequest{
		Status: "RUNNING", // not terminal
	})
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Full lifecycle test: create → assign → result
// ---------------------------------------------------------------------------

func TestFullJobLifecycle(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Handler()

	// 1. Register agent.
	doRequest(handler, "POST", "/agents/register", model.RegisterAgentRequest{
		ID: "agent-1", Hostname: "test", OS: "linux", Arch: "amd64",
		Version: "1.0", Capabilities: []string{"http_check", "wait"},
	})

	// 2. Create job.
	createRR := doRequest(handler, "POST", "/jobs", model.CreateJobRequest{
		Name: "lifecycle-test", Type: model.JobWait,
		Payload: map[string]any{"duration_seconds": 1.0},
		TimeoutSeconds: 10,
	})
	if createRR.Code != http.StatusCreated {
		t.Fatalf("create failed: %d", createRR.Code)
	}
	var created model.Job
	_ = json.Unmarshal(createRR.Body.Bytes(), &created)

	// 3. Verify PENDING.
	getRR := doRequest(handler, "GET", "/jobs/"+created.ID, nil)
	var pending model.Job
	_ = json.Unmarshal(getRR.Body.Bytes(), &pending)
	if pending.Status != model.JobPending {
		t.Errorf("expected PENDING, got %s", pending.Status)
	}

	// 4. Assign job.
	nextRR := doRequest(handler, "POST", "/agents/agent-1/next-job", map[string]any{
		"capabilities": []string{"http_check", "wait"},
	})
	if nextRR.Code != http.StatusOK {
		t.Fatalf("next-job failed: %d", nextRR.Code)
	}
	var assigned model.Job
	_ = json.Unmarshal(nextRR.Body.Bytes(), &assigned)
	if assigned.Status != model.JobRunning {
		t.Errorf("expected RUNNING, got %s", assigned.Status)
	}

	// 5. Submit SUCCESS result.
	resultRR := doRequest(handler, "POST", "/jobs/"+created.ID+"/result", model.JobResultRequest{
		Status: "SUCCESS", Logs: []string{"waited 1s"},
	})
	if resultRR.Code != http.StatusOK {
		t.Fatalf("result failed: %d", resultRR.Code)
	}
	var completed model.Job
	_ = json.Unmarshal(resultRR.Body.Bytes(), &completed)
	if completed.Status != model.JobSuccess {
		t.Errorf("expected SUCCESS, got %s", completed.Status)
	}
	if completed.FinishedAt == nil {
		t.Error("expected FinishedAt to be set")
	}
}

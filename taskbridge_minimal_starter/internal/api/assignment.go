package api

import (
	"net/http"

	"taskbridge/internal/model"
)

// handleNextJob handles POST /agents/{agentId}/next-job.
// The agent posts its capabilities and receives the next compatible PENDING job.
func (s *Server) handleNextJob(w http.ResponseWriter, r *http.Request) {
	agentID := pathParam(r, "agentId")
	if agentID == "" {
		writeError(w, http.StatusBadRequest, "missing agent ID", "")
		return
	}

	// Read capabilities from the request body (agent sends what it can do).
	var body struct {
		Capabilities []string `json:"capabilities"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	capabilities := make([]model.JobType, len(body.Capabilities))
	for i, c := range body.Capabilities {
		capabilities[i] = model.JobType(c)
	}

	job, found, err := s.store.AssignNextJob(agentID, capabilities)
	if err != nil {
		s.logger.Error("failed to assign job", "error", err, "agent_id", agentID)
		writeError(w, http.StatusInternalServerError, "failed to assign job", err.Error())
		return
	}

	if !found {
		// No pending job available — 204 No Content.
		w.WriteHeader(http.StatusNoContent)
		return
	}

	s.logger.Info("job assigned",
		"job_id", job.ID,
		"agent_id", agentID,
		"type", job.Type,
	)

	writeJSON(w, http.StatusOK, job)
}

// handleJobResult handles POST /jobs/{jobId}/result.
// The agent submits the execution outcome.
func (s *Server) handleJobResult(w http.ResponseWriter, r *http.Request) {
	jobID := pathParam(r, "jobId")
	if jobID == "" {
		writeError(w, http.StatusBadRequest, "missing job ID", "")
		return
	}

	var req model.JobResultRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	// Verify the job exists.
	_, found, err := s.store.GetJob(jobID)
	if err != nil {
		s.logger.Error("failed to get job for result", "error", err, "job_id", jobID)
		writeError(w, http.StatusInternalServerError, "failed to get job", err.Error())
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "job not found", jobID)
		return
	}

	status := model.JobStatus(req.Status)
	if err := s.store.CompleteJob(jobID, status, req.Logs, req.Result, req.Error); err != nil {
		s.logger.Error("failed to complete job", "error", err, "job_id", jobID)
		writeError(w, http.StatusInternalServerError, "failed to complete job", err.Error())
		return
	}

	// Re-fetch the job to return the updated state.
	updatedJob, _, _ := s.store.GetJob(jobID)

	s.logger.Info("job result received",
		"job_id", jobID,
		"status", updatedJob.Status,
		"attempt", updatedJob.AttemptCount,
	)

	writeJSON(w, http.StatusOK, updatedJob)
}

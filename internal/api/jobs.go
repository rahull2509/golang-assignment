package api

import (
	"net/http"
	"time"

	"taskbridge/internal/model"

	"crypto/rand"
	"encoding/hex"
)

// handleCreateJob handles POST /jobs.
func (s *Server) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	var req model.CreateJobRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	// Apply defaults.
	if req.TimeoutSeconds == 0 {
		req.TimeoutSeconds = 30 // sensible default
	}

	job := model.Job{
		ID:             generateID("job"),
		Name:           req.Name,
		Type:           req.Type,
		Payload:        req.Payload,
		Status:         model.JobPending,
		CreatedAt:      time.Now().UTC(),
		AttemptCount:   0,
		MaxRetries:     req.MaxRetries,
		TimeoutSeconds: req.TimeoutSeconds,
	}

	created, err := s.store.CreateJob(job)
	if err != nil {
		s.logger.Error("failed to create job", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create job", err.Error())
		return
	}

	s.logger.Info("job created",
		"job_id", created.ID,
		"type", created.Type,
		"name", created.Name,
	)

	writeJSON(w, http.StatusCreated, created)
}

// handleListJobs handles GET /jobs.
func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := s.store.ListJobs()
	if err != nil {
		s.logger.Error("failed to list jobs", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list jobs", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, model.ListJobsResponse{
		Jobs:  jobs,
		Total: len(jobs),
	})
}

// handleGetJob handles GET /jobs/{jobId}.
func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	jobID := pathParam(r, "jobId")
	if jobID == "" {
		writeError(w, http.StatusBadRequest, "missing job ID", "")
		return
	}

	job, found, err := s.store.GetJob(jobID)
	if err != nil {
		s.logger.Error("failed to get job", "error", err, "job_id", jobID)
		writeError(w, http.StatusInternalServerError, "failed to get job", err.Error())
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "job not found", jobID)
		return
	}

	writeJSON(w, http.StatusOK, job)
}

// handleCancelJob handles POST /jobs/{jobId}/cancel.
func (s *Server) handleCancelJob(w http.ResponseWriter, r *http.Request) {
	jobID := pathParam(r, "jobId")
	if jobID == "" {
		writeError(w, http.StatusBadRequest, "missing job ID", "")
		return
	}

	if err := s.store.CancelJob(jobID); err != nil {
		// Distinguish "not found" from "already terminal".
		if job, found, _ := s.store.GetJob(jobID); !found {
			writeError(w, http.StatusNotFound, "job not found", jobID)
		} else if job.Status.IsTerminal() {
			writeError(w, http.StatusConflict, "job already in terminal state", string(job.Status))
		} else {
			s.logger.Error("failed to cancel job", "error", err, "job_id", jobID)
			writeError(w, http.StatusInternalServerError, "failed to cancel job", err.Error())
		}
		return
	}

	s.logger.Info("job canceled", "job_id", jobID)
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "canceled",
		"job_id":  jobID,
	})
}

// ---------------------------------------------------------------------------
// ID generation
// ---------------------------------------------------------------------------

// generateID creates a unique ID with the given prefix (e.g. "job-a1b2c3d4").
func generateID(prefix string) string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails.
		return prefix + "-" + time.Now().Format("20060102150405")
	}
	return prefix + "-" + hex.EncodeToString(b)
}

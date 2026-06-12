package api

import (
	"net/http"
	"time"

	"taskbridge/internal/model"
)

// handleRegisterAgent handles POST /agents/register.
func (s *Server) handleRegisterAgent(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterAgentRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	// Convert string capabilities to typed JobType.
	capabilities := make([]model.JobType, len(req.Capabilities))
	for i, c := range req.Capabilities {
		capabilities[i] = model.JobType(c)
	}

	agent := model.Agent{
		ID:           req.ID,
		Hostname:     req.Hostname,
		OS:           req.OS,
		Arch:         req.Arch,
		Version:      req.Version,
		Capabilities: capabilities,
		LastSeen:     time.Now().UTC(),
		Status:       model.AgentOnline,
	}

	registered, err := s.store.RegisterAgent(agent)
	if err != nil {
		s.logger.Error("failed to register agent", "error", err, "agent_id", req.ID)
		writeError(w, http.StatusInternalServerError, "failed to register agent", err.Error())
		return
	}

	s.logger.Info("agent registered",
		"agent_id", registered.ID,
		"hostname", registered.Hostname,
		"capabilities", registered.Capabilities,
	)

	writeJSON(w, http.StatusCreated, registered)
}

// handleHeartbeat handles POST /agents/{agentId}/heartbeat.
func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	agentID := pathParam(r, "agentId")
	if agentID == "" {
		writeError(w, http.StatusBadRequest, "missing agent ID", "")
		return
	}

	if err := s.store.Heartbeat(agentID); err != nil {
		writeError(w, http.StatusNotFound, "agent not found", agentID)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "ok",
		"agent_id": agentID,
	})
}

// handleListAgents handles GET /agents.
func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := s.store.ListAgents()
	if err != nil {
		s.logger.Error("failed to list agents", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list agents", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, model.ListAgentsResponse{
		Agents: agents,
		Total:  len(agents),
	})
}

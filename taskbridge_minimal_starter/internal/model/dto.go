package model

// ---------------------------------------------------------------------------
// Request DTOs
// ---------------------------------------------------------------------------

// CreateJobRequest is the payload for POST /jobs.
type CreateJobRequest struct {
	Name           string         `json:"name"`
	Type           JobType        `json:"type"`
	Payload        map[string]any `json:"payload"`
	TimeoutSeconds int            `json:"timeout_seconds"`
	MaxRetries     int            `json:"max_retries"`
}

// RegisterAgentRequest is the payload for POST /agents/register.
type RegisterAgentRequest struct {
	ID           string   `json:"id"`
	Hostname     string   `json:"hostname"`
	OS           string   `json:"os"`
	Arch         string   `json:"arch"`
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities"`
}

// JobResultRequest is the payload for POST /jobs/{jobId}/result.
type JobResultRequest struct {
	Status string         `json:"status"`
	Logs   []string       `json:"logs"`
	Result map[string]any `json:"result,omitempty"`
	Error  string         `json:"error,omitempty"`
}

// ---------------------------------------------------------------------------
// Response DTOs
// ---------------------------------------------------------------------------

// ErrorResponse is a structured error returned by all API error paths.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Details string `json:"details,omitempty"`
}

// ListJobsResponse wraps a list of jobs with a count.
type ListJobsResponse struct {
	Jobs  []Job `json:"jobs"`
	Total int   `json:"total"`
}

// ListAgentsResponse wraps a list of agents with a count.
type ListAgentsResponse struct {
	Agents []Agent `json:"agents"`
	Total  int     `json:"total"`
}

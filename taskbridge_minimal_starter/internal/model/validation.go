package model

import (
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// Validation errors
// ---------------------------------------------------------------------------

// ValidationError holds one or more field-level errors.
type ValidationError struct {
	Fields map[string]string
}

func (v *ValidationError) Error() string {
	parts := make([]string, 0, len(v.Fields))
	for k, msg := range v.Fields {
		parts = append(parts, fmt.Sprintf("%s: %s", k, msg))
	}
	return "validation failed: " + strings.Join(parts, "; ")
}

// HasErrors returns true if any field error has been recorded.
func (v *ValidationError) HasErrors() bool {
	return len(v.Fields) > 0
}

// ---------------------------------------------------------------------------
// CreateJobRequest validation
// ---------------------------------------------------------------------------

// Validate checks that all required fields are present and valid.
func (r *CreateJobRequest) Validate() error {
	ve := &ValidationError{Fields: make(map[string]string)}

	if strings.TrimSpace(r.Name) == "" {
		ve.Fields["name"] = "must not be empty"
	}

	if r.Type == "" {
		ve.Fields["type"] = "must not be empty"
	} else if !AllJobTypes[r.Type] {
		ve.Fields["type"] = fmt.Sprintf("unsupported job type %q", r.Type)
	}

	if r.Payload == nil {
		ve.Fields["payload"] = "must not be null"
	}

	if r.TimeoutSeconds < 0 {
		ve.Fields["timeout_seconds"] = "must be >= 0"
	}
	if r.MaxRetries < 0 {
		ve.Fields["max_retries"] = "must be >= 0"
	}

	// Per-type payload validation
	if r.Payload != nil {
		switch r.Type {
		case JobHTTPCheck:
			validatePayloadKey(ve, r.Payload, "url")
		case JobTCPCheck:
			validatePayloadKey(ve, r.Payload, "address")
		case JobFileExists:
			validatePayloadKey(ve, r.Payload, "path")
		case JobChecksum:
			validatePayloadKey(ve, r.Payload, "path")
		case JobCopyFile:
			validatePayloadKey(ve, r.Payload, "source")
			validatePayloadKey(ve, r.Payload, "destination")
		case JobWriteFile:
			validatePayloadKey(ve, r.Payload, "path")
			validatePayloadKey(ve, r.Payload, "content")
		case JobWait:
			validatePayloadKey(ve, r.Payload, "duration_seconds")
		}
	}

	if ve.HasErrors() {
		return ve
	}
	return nil
}

// ---------------------------------------------------------------------------
// RegisterAgentRequest validation
// ---------------------------------------------------------------------------

// Validate checks that the agent registration request is well-formed.
func (r *RegisterAgentRequest) Validate() error {
	ve := &ValidationError{Fields: make(map[string]string)}

	if strings.TrimSpace(r.ID) == "" {
		ve.Fields["id"] = "must not be empty"
	}

	if len(r.Capabilities) == 0 {
		ve.Fields["capabilities"] = "must include at least one capability"
	}

	if ve.HasErrors() {
		return ve
	}
	return nil
}

// ---------------------------------------------------------------------------
// JobResultRequest validation
// ---------------------------------------------------------------------------

// Validate checks that the job result submission is well-formed.
func (r *JobResultRequest) Validate() error {
	ve := &ValidationError{Fields: make(map[string]string)}

	status := JobStatus(r.Status)
	if status != JobSuccess && status != JobFailed {
		ve.Fields["status"] = "must be SUCCESS or FAILED"
	}

	if ve.HasErrors() {
		return ve
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func validatePayloadKey(ve *ValidationError, payload map[string]any, key string) {
	if _, ok := payload[key]; !ok {
		ve.Fields["payload."+key] = "required field missing"
	}
}

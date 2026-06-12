package model

import (
	"testing"
)

// ---------------------------------------------------------------------------
// CreateJobRequest validation tests
// ---------------------------------------------------------------------------

func TestCreateJobRequest_Validate_Valid(t *testing.T) {
	req := CreateJobRequest{
		Name:           "test-job",
		Type:           JobHTTPCheck,
		Payload:        map[string]any{"url": "http://example.com"},
		TimeoutSeconds: 10,
		MaxRetries:     2,
	}
	if err := req.Validate(); err != nil {
		t.Errorf("expected valid request, got error: %v", err)
	}
}

func TestCreateJobRequest_Validate_MissingName(t *testing.T) {
	req := CreateJobRequest{
		Name:    "",
		Type:    JobHTTPCheck,
		Payload: map[string]any{"url": "http://example.com"},
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected validation error for empty name")
	}
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if _, exists := ve.Fields["name"]; !exists {
		t.Error("expected error on field 'name'")
	}
}

func TestCreateJobRequest_Validate_InvalidType(t *testing.T) {
	req := CreateJobRequest{
		Name:    "test",
		Type:    "unknown_type",
		Payload: map[string]any{},
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected validation error for invalid type")
	}
}

func TestCreateJobRequest_Validate_MissingPayloadKeys(t *testing.T) {
	tests := []struct {
		name    string
		jobType JobType
		payload map[string]any
		wantKey string
	}{
		{"http_check missing url", JobHTTPCheck, map[string]any{}, "payload.url"},
		{"tcp_check missing address", JobTCPCheck, map[string]any{}, "payload.address"},
		{"file_exists missing path", JobFileExists, map[string]any{}, "payload.path"},
		{"checksum missing path", JobChecksum, map[string]any{}, "payload.path"},
		{"copy_file missing source", JobCopyFile, map[string]any{"destination": "/tmp/out"}, "payload.source"},
		{"write_file missing path", JobWriteFile, map[string]any{"content": "hello"}, "payload.path"},
		{"wait missing duration", JobWait, map[string]any{}, "payload.duration_seconds"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := CreateJobRequest{
				Name:    "test",
				Type:    tt.jobType,
				Payload: tt.payload,
			}
			err := req.Validate()
			if err == nil {
				t.Fatal("expected validation error")
			}
			ve, ok := err.(*ValidationError)
			if !ok {
				t.Fatalf("expected *ValidationError, got %T", err)
			}
			if _, exists := ve.Fields[tt.wantKey]; !exists {
				t.Errorf("expected error on field %q, got fields: %v", tt.wantKey, ve.Fields)
			}
		})
	}
}

func TestCreateJobRequest_Validate_NegativeTimeout(t *testing.T) {
	req := CreateJobRequest{
		Name:           "test",
		Type:           JobWait,
		Payload:        map[string]any{"duration_seconds": 5.0},
		TimeoutSeconds: -1,
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected validation error for negative timeout")
	}
}

// ---------------------------------------------------------------------------
// RegisterAgentRequest validation tests
// ---------------------------------------------------------------------------

func TestRegisterAgentRequest_Validate_Valid(t *testing.T) {
	req := RegisterAgentRequest{
		ID:           "agent-1",
		Capabilities: []string{"http_check"},
	}
	if err := req.Validate(); err != nil {
		t.Errorf("expected valid request, got error: %v", err)
	}
}

func TestRegisterAgentRequest_Validate_MissingID(t *testing.T) {
	req := RegisterAgentRequest{
		ID:           "",
		Capabilities: []string{"http_check"},
	}
	if err := req.Validate(); err == nil {
		t.Fatal("expected validation error for empty ID")
	}
}

func TestRegisterAgentRequest_Validate_NoCapabilities(t *testing.T) {
	req := RegisterAgentRequest{
		ID:           "agent-1",
		Capabilities: []string{},
	}
	if err := req.Validate(); err == nil {
		t.Fatal("expected validation error for empty capabilities")
	}
}

// ---------------------------------------------------------------------------
// JobResultRequest validation tests
// ---------------------------------------------------------------------------

func TestJobResultRequest_Validate_Valid(t *testing.T) {
	tests := []struct {
		status string
	}{
		{"SUCCESS"},
		{"FAILED"},
	}
	for _, tt := range tests {
		req := JobResultRequest{Status: tt.status}
		if err := req.Validate(); err != nil {
			t.Errorf("expected valid for status %q, got: %v", tt.status, err)
		}
	}
}

func TestJobResultRequest_Validate_InvalidStatus(t *testing.T) {
	req := JobResultRequest{Status: "RUNNING"}
	if err := req.Validate(); err == nil {
		t.Fatal("expected validation error for non-terminal status")
	}
}

// ---------------------------------------------------------------------------
// JobStatus helpers
// ---------------------------------------------------------------------------

func TestJobStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status   JobStatus
		terminal bool
	}{
		{JobPending, false},
		{JobRunning, false},
		{JobRetrying, false},
		{JobSuccess, true},
		{JobFailed, true},
		{JobCanceled, true},
	}
	for _, tt := range tests {
		if got := tt.status.IsTerminal(); got != tt.terminal {
			t.Errorf("IsTerminal(%s) = %v, want %v", tt.status, got, tt.terminal)
		}
	}
}

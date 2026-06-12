package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"taskbridge/internal/executor"
	"taskbridge/internal/model"
)

// Client handles HTTP communication between the agent and the server.
type Client struct {
	serverURL  string
	agentID    string
	authToken  string
	httpClient *http.Client
}

// NewClient creates a new agent HTTP client.
func NewClient(serverURL, agentID, authToken string) *Client {
	return &Client{
		serverURL: serverURL,
		agentID:   agentID,
		authToken: authToken,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Register registers the agent with the server.
func (c *Client) Register(req model.RegisterAgentRequest) (*model.Agent, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal register request: %w", err)
	}

	resp, err := c.doRequest("POST", "/agents/register", body)
	if err != nil {
		return nil, fmt.Errorf("register request failed: %w", err)
	}
	defer closeBody(resp)

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("register failed: status %d", resp.StatusCode)
	}

	var agent model.Agent
	if err := json.NewDecoder(resp.Body).Decode(&agent); err != nil {
		return nil, fmt.Errorf("decode register response: %w", err)
	}
	return &agent, nil
}

// Heartbeat sends a heartbeat to the server.
func (c *Client) Heartbeat() error {
	url := fmt.Sprintf("/agents/%s/heartbeat", c.agentID)
	resp, err := c.doRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("heartbeat request failed: %w", err)
	}
	defer closeBody(resp)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("heartbeat failed: status %d", resp.StatusCode)
	}
	return nil
}

// PollNextJob requests the next available job from the server.
func (c *Client) PollNextJob(capabilities []string) (*model.Job, error) {
	body, err := json.Marshal(map[string]any{
		"capabilities": capabilities,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal poll request: %w", err)
	}

	url := fmt.Sprintf("/agents/%s/next-job", c.agentID)
	resp, err := c.doRequest("POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("poll request failed: %w", err)
	}
	defer closeBody(resp)

	// 204 = no jobs available.
	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("poll failed: status %d", resp.StatusCode)
	}

	var job model.Job
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return nil, fmt.Errorf("decode poll response: %w", err)
	}
	return &job, nil
}

// SubmitResult sends the job execution result to the server.
func (c *Client) SubmitResult(jobID string, result executor.Result) error {
	req := model.JobResultRequest{
		Status: string(result.Status),
		Logs:   result.Logs,
		Result: result.Result,
		Error:  result.Error,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal result request: %w", err)
	}

	url := fmt.Sprintf("/jobs/%s/result", jobID)
	resp, err := c.doRequest("POST", url, body)
	if err != nil {
		return fmt.Errorf("submit result request failed: %w", err)
	}
	defer closeBody(resp)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("submit result failed: status %d", resp.StatusCode)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func (c *Client) doRequest(method, path string, body []byte) (*http.Response, error) {
	url := c.serverURL + path

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	return c.httpClient.Do(req)
}

func closeBody(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}
}

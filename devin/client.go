// Package devin provides an HTTP client for the Devin AI API.
package devin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	baseURL = "https://api.devin.ai/v1"

	// Polling configuration
	defaultPollInterval = 15 * time.Second
	defaultPollTimeout  = 60 * time.Minute

	// maxPollErrors is the number of consecutive transient errors tolerated
	// during polling before giving up.
	maxPollErrors = 5
)

// Client communicates with the Devin AI API.
type Client struct {
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new Devin API client.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// CreateSessionRequest is the payload for creating a new Devin session.
type CreateSessionRequest struct {
	Prompt     string `json:"prompt"`
	Idempotent bool   `json:"idempotent,omitempty"`
}

// CreateSessionResponse is returned when a session is created.
type CreateSessionResponse struct {
	SessionID    string `json:"session_id"`
	URL          string `json:"url"`
	IsNewSession bool   `json:"is_new_session"`
}

// SessionStatus represents the current state of a Devin session.
type SessionStatus struct {
	SessionID        string          `json:"session_id"`
	Status           string          `json:"status"`
	StatusEnum       string          `json:"status_enum"`
	Title            string          `json:"title"`
	StructuredOutput json.RawMessage `json:"structured_output,omitempty"`
	PullRequest      *PullRequest    `json:"pull_request,omitempty"`
}

// PullRequest contains PR information from a completed session.
type PullRequest struct {
	URL string `json:"url"`
}

// CreateSession creates a new Devin session with the given prompt.
func (c *Client) CreateSession(ctx context.Context, req CreateSessionRequest) (*CreateSessionResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/sessions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("devin API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result CreateSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// GetSession retrieves the current status of a Devin session.
func (c *Client) GetSession(ctx context.Context, sessionID string) (*SessionStatus, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/session/"+sessionID, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("devin API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result SessionStatus
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// PollUntilDone polls the session status until it reaches a terminal state
// or the context is cancelled. It returns the final session status.
//
// Terminal states are "finished", "stopped", and "blocked". Devin enters the
// "blocked" state when it has completed its task and is waiting for a follow-up
// prompt, so the plugin treats it as successful completion.
func (c *Client) PollUntilDone(ctx context.Context, sessionID string, pollInterval, pollTimeout time.Duration) (*SessionStatus, error) {
	if pollInterval == 0 {
		pollInterval = defaultPollInterval
	}
	if pollTimeout == 0 {
		pollTimeout = defaultPollTimeout
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	timeout := time.After(pollTimeout)
	consecutiveErrors := 0

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout:
			return nil, fmt.Errorf("polling timed out after %v for session %s", pollTimeout, sessionID)
		case <-ticker.C:
			status, err := c.GetSession(ctx, sessionID)
			if err != nil {
				consecutiveErrors++
				if consecutiveErrors >= maxPollErrors {
					return nil, fmt.Errorf("poll session %s: %d consecutive errors, last: %w", sessionID, consecutiveErrors, err)
				}
				// transient error, will retry on next tick
				continue
			}
			consecutiveErrors = 0
			switch status.StatusEnum {
			case "finished", "stopped", "blocked":
				return status, nil
			}
			// still working, continue polling
		}
	}
}

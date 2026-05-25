// Package tui provides the terminal user interface for Gastown Viewer Intent.
package tui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/intent-solutions-io/gastown-viewer-intent/internal/model"
)

// Client fetches data from the gvid daemon API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new API client.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// HealthResponse matches the API health response.
type HealthResponse struct {
	Status           string `json:"status"`
	BeadsInitialized bool   `json:"beads_initialized"`
	Version          string `json:"version"`
	BDVersion        string `json:"bd_version,omitempty"`
	Error            string `json:"error,omitempty"`
}

// Health checks the daemon health.
func (c *Client) Health() (*HealthResponse, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/v1/health")
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var health HealthResponse
	if err := json.Unmarshal(body, &health); err != nil {
		return nil, err
	}

	return &health, nil
}

// BoardResponse matches the API board response.
type BoardResponse struct {
	Columns []model.Column `json:"columns"`
	Total   int            `json:"total"`
}

// Board fetches the board view.
func (c *Client) Board() (*BoardResponse, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/v1/board")
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var board BoardResponse
	if err := json.Unmarshal(body, &board); err != nil {
		return nil, err
	}

	return &board, nil
}

// Issue fetches a single issue by ID.
func (c *Client) Issue(id string) (*model.Issue, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/v1/issues/" + id)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("issue not found: %s", id)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var issue model.Issue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, err
	}

	return &issue, nil
}

// Memories fetches the full memory list from the daemon. The daemon
// applies the 005-PP-POLICY redaction layer before serialization, so
// the TUI always renders pre-redacted content — there is no reveal
// path in the TUI (Council Q2 read-only-forever invariant; reveal is
// only available via the bd CLI directly).
func (c *Client) Memories() (*model.MemoriesResponse, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/v1/memories")
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var out model.MemoriesResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SearchMemories proxies to the daemon's substring search endpoint.
// Empty q falls back to the full list (matches daemon + bd semantics).
func (c *Client) SearchMemories(q string) (*model.MemoriesResponse, error) {
	if q == "" {
		return c.Memories()
	}
	u := c.baseURL + "/api/v1/memories/search?q=" + url.QueryEscape(q)
	resp, err := c.httpClient.Get(u)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var out model.MemoriesResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// HumanFlags fetches the read-only human-decision triage queue. The
// daemon returns an empty Flags slice (not nil) when nothing is flagged.
func (c *Client) HumanFlags() (*model.HumanFlagsResponse, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/v1/human")
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var out model.HumanFlagsResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

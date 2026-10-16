package docuhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client is a minimal DocuHook API client.
type Client struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
}

// New returns a client pointed at https://docuhook.com.
func New(apiKey string) *Client {
	return &Client{
		APIKey:  strings.TrimSpace(apiKey),
		BaseURL: "https://docuhook.com",
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
}

// RenderOptions controls sync/async render.
type RenderOptions struct {
	Mode     string // "sync" (default) or "async"
	Filename string
}

// Result is a normalized API response.
type Result struct {
	OK         bool
	HTML       string
	Filename   string
	JobID      string
	Status     string
	StatusURL  string
	ResultURL  string
	StatusCode int
	Raw        map[string]any
}

// Render merges data into html via POST /api/v1/render.
func (c *Client) Render(data map[string]any, html string, opts *RenderOptions) (*Result, error) {
	mode := "sync"
	filename := "document.pdf"
	if opts != nil {
		if opts.Mode == "async" {
			mode = "async"
		}
		if opts.Filename != "" {
			filename = opts.Filename
		}
	}
	return c.post("/api/v1/render", map[string]any{
		"mode":     mode,
		"data":     data,
		"html":     html,
		"filename": filename,
	})
}

// HookRender is the webhook alias (defaults to async).
func (c *Client) HookRender(data map[string]any, html string, opts *RenderOptions) (*Result, error) {
	mode := "async"
	filename := "document.pdf"
	if opts != nil {
		if opts.Mode == "sync" {
			mode = "sync"
		}
		if opts.Filename != "" {
			filename = opts.Filename
		}
	}
	return c.post("/api/v1/hooks/render", map[string]any{
		"mode":     mode,
		"data":     data,
		"html":     html,
		"filename": filename,
	})
}

// JobStatus fetches GET /api/v1/jobs/{id}.
func (c *Client) JobStatus(jobID string) (*Result, error) {
	return c.get("/api/v1/jobs/" + url.PathEscape(jobID))
}

// JobResult fetches GET /api/v1/jobs/{id}/result.
func (c *Client) JobResult(jobID string) (*Result, error) {
	return c.get("/api/v1/jobs/" + url.PathEscape(jobID) + "/result")
}

func (c *Client) post(path string, payload map[string]any) (*Result, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, c.base()+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

func (c *Client) get(path string) (*Result, error) {
	req, err := http.NewRequest(http.MethodGet, c.base()+path, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

func (c *Client) do(req *http.Request) (*Result, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return nil, fmt.Errorf("docuhook: API key is required")
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("X-DocuHook-Key", c.APIKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	rawBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var raw map[string]any
	if len(rawBytes) > 0 {
		if err := json.Unmarshal(rawBytes, &raw); err != nil {
			return nil, fmt.Errorf("docuhook: invalid JSON (%d): %w", res.StatusCode, err)
		}
	}
	if raw == nil {
		raw = map[string]any{}
	}

	if res.StatusCode >= 400 {
		msg, _ := raw["error"].(string)
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", res.StatusCode)
		}
		return &Result{OK: false, StatusCode: res.StatusCode, Raw: raw}, fmt.Errorf("docuhook: %s", msg)
	}

	out := &Result{OK: true, StatusCode: res.StatusCode, Raw: raw}
	if v, ok := raw["html"].(string); ok {
		out.HTML = v
	}
	if v, ok := raw["filename"].(string); ok {
		out.Filename = v
	}
	if v, ok := raw["job_id"].(string); ok {
		out.JobID = v
	}
	if v, ok := raw["status"].(string); ok {
		out.Status = v
	}
	if v, ok := raw["status_url"].(string); ok {
		out.StatusURL = v
	}
	if v, ok := raw["result_url"].(string); ok {
		out.ResultURL = v
	}
	return out, nil
}

func (c *Client) base() string {
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		return "https://docuhook.com"
	}
	return base
}

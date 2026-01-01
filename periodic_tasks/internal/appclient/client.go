package appclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is an HTTP client for calling app API endpoints
type Client struct {
	baseURL string
	token   string
	hc      *http.Client
}

// New creates a new app API client
func New(baseURL, token string) *Client {
	// Normalize baseURL - add http:// if missing
	normalizedURL := baseURL
	if !strings.HasPrefix(normalizedURL, "http://") && !strings.HasPrefix(normalizedURL, "https://") {
		if strings.HasPrefix(normalizedURL, ":") {
			normalizedURL = "http://app" + normalizedURL
		} else {
			normalizedURL = "http://" + normalizedURL
		}
	}

	return &Client{
		baseURL: strings.TrimRight(normalizedURL, "/"),
		token:   token,
		hc: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// doJSON performs a request and decodes JSON response
func (c *Client) doJSON(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("X-Internal-Token", c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("app api error: %s, body: %s", resp.Status, string(bodyBytes))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

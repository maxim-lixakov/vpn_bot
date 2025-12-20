package outline

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	hc      HttpClient
}

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type OutlineClientInterface interface {
	CreateAccessKey(ctx context.Context, name string) (AccessKey, error)
	DeleteAccessKey(ctx context.Context, id string) error
	MetricsTransfer(ctx context.Context) (map[string]int64, error)
	RemoveAccessKeyDataLimit(ctx context.Context, id string) error
	SetAccessKeyDataLimit(ctx context.Context, id string, bytesLimit int64) error
}

func NewClient(baseURL string, tlsInsecure bool) OutlineClientInterface {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: tlsInsecure}, // MVP only
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		hc: &http.Client{
			Timeout:   10 * time.Second,
			Transport: tr,
		},
	}
}

func (c *Client) doJSON(ctx context.Context, method, path string, in any, out any) error {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("outline api error: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}

	if out == nil {
		io.Copy(io.Discard, resp.Body)
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

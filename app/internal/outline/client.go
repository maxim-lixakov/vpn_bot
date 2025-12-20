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
	hc      *http.Client
}

func NewClient(baseURL string, tlsInsecure bool) *Client {
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

type AccessKey struct {
	ID        string `json:"id"`
	Name      string `json:"name,omitempty"`
	Password  string `json:"password,omitempty"`
	Port      int    `json:"port,omitempty"`
	Method    string `json:"method,omitempty"`
	AccessURL string `json:"accessUrl,omitempty"`
	// dataLimit optional in list, see api.yml :contentReference[oaicite:6]{index=6}
	DataLimit *struct {
		Bytes int64 `json:"bytes"`
	} `json:"dataLimit,omitempty"`
}

type createAccessKeyReq struct {
	Name string `json:"name,omitempty"`
	// Можно задать limit на создание (пример в api.yml) :contentReference[oaicite:7]{index=7}
	Limit *struct {
		Bytes int64 `json:"bytes"`
	} `json:"limit,omitempty"`
}

func (c *Client) CreateAccessKey(ctx context.Context, name string) (AccessKey, error) {
	reqBody := createAccessKeyReq{Name: name}

	var out AccessKey
	if err := c.doJSON(ctx, http.MethodPost, "/access-keys", reqBody, &out); err != nil {
		return AccessKey{}, err
	}
	return out, nil
}

func (c *Client) DeleteAccessKey(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, "/access-keys/"+id, nil, nil)
}

type dataLimitReq struct {
	// В примерах request body выглядит как {"limit":{"bytes":10000}} :contentReference[oaicite:8]{index=8}
	Limit struct {
		Bytes int64 `json:"bytes"`
	} `json:"limit"`
}

func (c *Client) SetAccessKeyDataLimit(ctx context.Context, id string, bytesLimit int64) error {
	var req dataLimitReq
	req.Limit.Bytes = bytesLimit
	return c.doJSON(ctx, http.MethodPut, "/access-keys/"+id+"/data-limit", req, nil)
}

func (c *Client) RemoveAccessKeyDataLimit(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, "/access-keys/"+id+"/data-limit", nil, nil)
}

type metricsTransferResp struct {
	BytesTransferredByUserID map[string]int64 `json:"bytesTransferredByUserId"`
}

func (c *Client) MetricsTransfer(ctx context.Context) (map[string]int64, error) {
	var out metricsTransferResp
	if err := c.doJSON(ctx, http.MethodGet, "/metrics/transfer", nil, &out); err != nil {
		return nil, err
	}
	return out.BytesTransferredByUserID, nil
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

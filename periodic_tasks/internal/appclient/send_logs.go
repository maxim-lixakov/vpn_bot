package appclient

import (
	"context"
	"net/http"
)

type SendLogsResp struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// SendLogs collects and sends logs for the past week to admin
func (c *Client) SendLogs(ctx context.Context) (SendLogsResp, error) {
	var out SendLogsResp
	err := c.doJSON(ctx, http.MethodPost, "/v1/send-logs", nil, &out)
	return out, err
}

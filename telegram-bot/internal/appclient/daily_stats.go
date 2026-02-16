package appclient

import (
	"context"
	"net/http"
)

type DailyStatsResp struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

func (c *Client) DailyStats(ctx context.Context) (*DailyStatsResp, error) {
	var resp DailyStatsResp
	err := c.do(ctx, http.MethodPost, "/v1/daily-stats", nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

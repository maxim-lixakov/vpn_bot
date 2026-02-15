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

// DailyStats triggers daily statistics collection and sends it to admin via Telegram
func (c *Client) DailyStats(ctx context.Context) (DailyStatsResp, error) {
	var out DailyStatsResp
	err := c.doJSON(ctx, http.MethodPost, "/v1/daily-stats", nil, &out)
	return out, err
}

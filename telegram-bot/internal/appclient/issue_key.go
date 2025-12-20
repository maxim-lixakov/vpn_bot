package appclient

import (
	"context"
	"net/http"
)

type IssueKeyResp struct {
	ServerName  string `json:"server_name"`
	Country     string `json:"country"`
	AccessKeyID string `json:"access_key_id"`
	AccessURL   string `json:"access_url"`
}

func (c *Client) IssueKey(ctx context.Context, tgUserID int64, country string) (IssueKeyResp, error) {
	var out IssueKeyResp
	req := map[string]any{"tg_user_id": tgUserID, "country": country}
	err := c.do(ctx, http.MethodPost, "/v1/issue-key", req, &out)
	return out, err
}

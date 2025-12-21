package appclient

import (
	"context"
	"net/http"
)

type IssueKeyReq struct {
	TgUserID int64  `json:"tg_user_id"`
	Country  string `json:"country"`
}

type IssueKeyResp struct {
	Status string `json:"status"` // "ok" | "payment_required"

	Country    string `json:"country"`
	ServerName string `json:"server_name"`

	AccessKeyID string `json:"access_key_id"`
	AccessURL   string `json:"access_url"`

	Payment *IssueKeyPayment `json:"payment"`
}

type IssueKeyPayment struct {
	Kind        string `json:"kind"`
	CountryCode string `json:"country_code"`
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

func (c *Client) IssueKey(ctx context.Context, tgUserID int64, country string) (IssueKeyResp, error) {
	req := IssueKeyReq{TgUserID: tgUserID, Country: country}
	var out IssueKeyResp
	err := c.do(ctx, http.MethodPost, "/v1/issue-key", req, &out)
	return out, err
}

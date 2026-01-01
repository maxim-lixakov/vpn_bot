package appclient

import (
	"context"
	"net/http"
)

type RevokedSubscriptionInfo struct {
	SubscriptionID int64  `json:"subscription_id"`
	TgUserID       int64  `json:"tg_user_id"`
	CountryCode    string `json:"country_code"`
}

type RevokeExpiredKeysResp struct {
	RevokedCount int                       `json:"revoked_count"`
	Revoked      []RevokedSubscriptionInfo `json:"revoked,omitempty"`
	Errors       []string                  `json:"errors,omitempty"`
}

// RevokeExpiredKeys revokes expired access keys and sends notifications
func (c *Client) RevokeExpiredKeys(ctx context.Context) (RevokeExpiredKeysResp, error) {
	var out RevokeExpiredKeysResp
	err := c.doJSON(ctx, http.MethodPost, "/v1/revoke-expired-keys", nil, &out)
	return out, err
}

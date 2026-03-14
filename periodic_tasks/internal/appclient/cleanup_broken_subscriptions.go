package appclient

import (
	"context"
	"net/http"
	"time"
)

// CleanupBrokenSubscriptionsResponse represents the response from cleanup endpoint
type CleanupBrokenSubscriptionsResponse struct {
	TotalFound    int                      `json:"total_found"`
	Cleaned       int                      `json:"cleaned"`
	Failed        int                      `json:"failed"`
	Subscriptions []BrokenSubscriptionInfo `json:"subscriptions"`
}

// BrokenSubscriptionInfo contains details about a broken subscription
type BrokenSubscriptionInfo struct {
	SubscriptionID int64     `json:"subscription_id"`
	UserID         int64     `json:"user_id"`
	CountryCode    string    `json:"country_code,omitempty"`
	PaidAt         time.Time `json:"paid_at"`
	ActiveUntil    time.Time `json:"active_until"`
	Action         string    `json:"action"`
	Error          string    `json:"error,omitempty"`
	IsPromocode    bool      `json:"is_promocode"`
}

// CleanupBrokenSubscriptions calls the cleanup-broken-subscriptions endpoint
func (c *Client) CleanupBrokenSubscriptions(ctx context.Context) (*CleanupBrokenSubscriptionsResponse, error) {
	var result CleanupBrokenSubscriptionsResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/cleanup-broken-subscriptions", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

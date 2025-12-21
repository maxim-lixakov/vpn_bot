package appclient

import (
	"context"
	"net/http"
	"time"

	"vpn-bot/internal/utils"
)

type SubscriptionDTO struct {
	Kind        string     `json:"kind"`
	CountryCode *string    `json:"country_code"`
	PaidAt      time.Time  `json:"paid_at"`
	ActiveUntil *time.Time `json:"active_until"`
	IsActive    bool       `json:"is_active"`
}

type TelegramSubscriptionsResp struct {
	Items []SubscriptionDTO `json:"items"`
}

func (c *Client) TelegramSubscriptions(ctx context.Context, tgUserID int64) (TelegramSubscriptionsResp, error) {
	var out TelegramSubscriptionsResp
	path := "/v1/telegram/subscriptions?tg_user_id=" + utils.Itoa64(tgUserID)
	err := c.do(ctx, http.MethodGet, path, nil, &out)
	return out, err
}

package appclient

import (
	"context"
	"net/http"
)

func (c *Client) TelegramUpdatePromocodeSubscription(ctx context.Context, tgUserID int64, countryCode string) error {
	req := map[string]any{
		"tg_user_id":   tgUserID,
		"country_code": countryCode,
	}
	return c.do(ctx, http.MethodPost, "/v1/telegram/update-promocode-subscription", req, nil)
}

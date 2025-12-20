package appclient

import (
	"context"
	"net/http"
	"time"
)

type TelegramUpsertReq struct {
	TgUserID     int64   `json:"tg_user_id"`
	Username     *string `json:"username"`
	FirstName    *string `json:"first_name"`
	LastName     *string `json:"last_name"`
	LanguageCode *string `json:"language_code"`
	Phone        *string `json:"phone"`
}

type TelegramUpsertResp struct {
	UserID          int64      `json:"user_id"`
	State           string     `json:"state"`
	SelectedCountry *string    `json:"selected_country"`
	SubscriptionOK  bool       `json:"subscription_ok"`
	ActiveUntil     *time.Time `json:"active_until"`
}

func (c *Client) TelegramUpsert(ctx context.Context, req TelegramUpsertReq) (TelegramUpsertResp, error) {
	var out TelegramUpsertResp
	err := c.do(ctx, http.MethodPost, "/v1/telegram/upsert", req, &out)
	return out, err
}

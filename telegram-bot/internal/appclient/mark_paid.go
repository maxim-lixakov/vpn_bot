package appclient

import (
	"context"
	"net/http"
	"time"
)

type TelegramMarkPaidReq struct {
	TgUserID                int64  `json:"tg_user_id"`
	AmountMinor             int64  `json:"amount_minor"`
	Currency                string `json:"currency"`
	TelegramPaymentChargeID string `json:"telegram_payment_charge_id"`
	ProviderPaymentChargeID string `json:"provider_payment_charge_id"`
}

type TelegramMarkPaidResp struct {
	ActiveUntil time.Time `json:"active_until"`
}

func (c *Client) TelegramMarkPaid(ctx context.Context, req TelegramMarkPaidReq) (TelegramMarkPaidResp, error) {
	var out TelegramMarkPaidResp
	err := c.do(ctx, http.MethodPost, "/v1/telegram/mark-paid", req, &out)
	return out, err
}

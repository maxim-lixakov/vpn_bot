package appclient

import (
	"context"
	"net/http"
)

type TelegramReferralCodeReq struct {
	TgUserID int64 `json:"tg_user_id"`
}

type TelegramReferralCodeResp struct {
	Promocode string `json:"promocode"`
	Error     string `json:"error,omitempty"`
}

func (c *Client) TelegramReferralCode(ctx context.Context, tgUserID int64) (*TelegramReferralCodeResp, error) {
	req := TelegramReferralCodeReq{TgUserID: tgUserID}
	var resp TelegramReferralCodeResp
	err := c.do(ctx, http.MethodPost, "/v1/telegram/referral-code", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

package appclient

import (
	"context"
	"net/http"
)

type TelegramPromocodeUseReq struct {
	TgUserID int64  `json:"tg_user_id"`
	Code     string `json:"code"`
}

type TelegramPromocodeUseResp struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message"`
	Months  int    `json:"months,omitempty"`
}

func (c *Client) TelegramPromocodeUse(ctx context.Context, tgUserID int64, code string) (*TelegramPromocodeUseResp, error) {
	req := TelegramPromocodeUseReq{TgUserID: tgUserID, Code: code}
	var resp TelegramPromocodeUseResp
	err := c.do(ctx, http.MethodPost, "/v1/telegram/promocode-use", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

type TelegramPromocodeRollbackReq struct {
	TgUserID int64  `json:"tg_user_id"`
	Code     string `json:"code,omitempty"` // Опционально: если не указан, откатываем последний использованный
}

func (c *Client) TelegramPromocodeRollback(ctx context.Context, tgUserID int64, code string) error {
	req := TelegramPromocodeRollbackReq{TgUserID: tgUserID}
	if code != "" {
		req.Code = code
	}
	return c.do(ctx, http.MethodPost, "/v1/telegram/promocode-rollback", req, nil)
}

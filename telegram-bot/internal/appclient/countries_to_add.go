package appclient

import (
	"context"
	"net/http"
)

type TelegramCreateCountryToAddReq struct {
	TgUserID int64  `json:"tg_user_id"`
	Text     string `json:"text"`
}

func (c *Client) TelegramCreateCountryToAdd(ctx context.Context, tgUserID int64, text string) error {
	req := TelegramCreateCountryToAddReq{TgUserID: tgUserID, Text: text}
	return c.do(ctx, http.MethodPost, "/v1/telegram/countries-to-add", req, nil)
}

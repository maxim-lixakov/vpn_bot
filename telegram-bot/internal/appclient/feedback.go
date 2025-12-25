package appclient

import (
	"context"
	"net/http"
)

type TelegramFeedbackReq struct {
	TgUserID int64  `json:"tg_user_id"`
	Text     string `json:"text"`
}

func (c *Client) TelegramFeedback(ctx context.Context, tgUserID int64, text string) error {
	req := TelegramFeedbackReq{TgUserID: tgUserID, Text: text}
	return c.do(ctx, http.MethodPost, "/v1/telegram/feedback", req, nil)
}

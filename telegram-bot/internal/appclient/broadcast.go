package appclient

import (
	"context"
	"net/http"
)

type TelegramBroadcastReq struct {
	AdminTgUserID int64  `json:"admin_tg_user_id"`
	Message       string `json:"message"`
	Target        string `json:"target"` // "all", "with_subscription", "without_subscription"
}

type TelegramBroadcastResp struct {
	SentCount  int      `json:"sent_count"`
	Recipients []string `json:"recipients"`
}

func (c *Client) TelegramBroadcast(ctx context.Context, req TelegramBroadcastReq) (*TelegramBroadcastResp, error) {
	var resp TelegramBroadcastResp
	err := c.do(ctx, http.MethodPost, "/v1/telegram/broadcast", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

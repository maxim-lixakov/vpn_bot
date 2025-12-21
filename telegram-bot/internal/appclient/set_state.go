package appclient

import (
	"context"
	"net/http"
)

func (c *Client) TelegramSetState(ctx context.Context, tgUserID int64, state string, selectedCountry *string) error {
	req := map[string]any{
		"tg_user_id": tgUserID,
		"state":      state,
	}
	if selectedCountry != nil {
		req["selected_country"] = *selectedCountry
	}
	return c.do(ctx, http.MethodPost, "/v1/telegram/set-state", req, nil)
}

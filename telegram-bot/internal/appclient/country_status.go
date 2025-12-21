package appclient

import (
	"context"
	"net/http"
	"time"

	"vpn-bot/internal/utils"
)

type TelegramCountryStatusResp struct {
	Active      bool      `json:"active"`
	ActiveUntil time.Time `json:"active_until"`
}

func (c *Client) TelegramCountryStatus(ctx context.Context, tgUserID int64, country string) (TelegramCountryStatusResp, error) {
	var out TelegramCountryStatusResp
	path := "/v1/telegram/country-status?tg_user_id=" + utils.Itoa64(tgUserID) + "&country=" + country
	err := c.do(ctx, http.MethodGet, path, nil, &out)
	return out, err
}

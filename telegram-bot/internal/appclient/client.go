package appclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL string
	token   string
	hc      *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		hc:      &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) do(ctx context.Context, method, path string, in any, out any) error {
	var body *bytes.Reader
	if in != nil {
		b, _ := json.Marshal(in)
		body = bytes.NewReader(b)
	} else {
		body = bytes.NewReader(nil)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("X-Internal-Token", c.token)
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("app error: %s", resp.Status)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
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

type TelegramUpsertReq struct {
	TgUserID     int64   `json:"tg_user_id"`
	Username     *string `json:"username"`
	FirstName    *string `json:"first_name"`
	LastName     *string `json:"last_name"`
	LanguageCode *string `json:"language_code"`
	Phone        *string `json:"phone"`
}

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

func (c *Client) TelegramMarkPaid(ctx context.Context, req TelegramMarkPaidReq) (TelegramMarkPaidResp, error) {
	var out TelegramMarkPaidResp
	err := c.do(ctx, http.MethodPost, "/v1/telegram/mark-paid", req, &out)
	return out, err
}

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

type IssueKeyResp struct {
	ServerName  string `json:"server_name"`
	Country     string `json:"country"`
	AccessKeyID string `json:"access_key_id"`
	AccessURL   string `json:"access_url"`
}

func (c *Client) IssueKey(ctx context.Context, tgUserID int64, country string) (IssueKeyResp, error) {
	var out IssueKeyResp
	req := map[string]any{"tg_user_id": tgUserID, "country": country}
	err := c.do(ctx, http.MethodPost, "/v1/issue-key", req, &out)
	return out, err
}

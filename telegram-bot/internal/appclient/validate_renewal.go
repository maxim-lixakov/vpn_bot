package appclient

import "context"

type ValidateRenewalReq struct {
	SubscriptionID int64 `json:"subscription_id"`
}

type ValidateRenewalResp struct {
	Valid        bool   `json:"valid"`
	ErrorMessage string `json:"error_message,omitempty"`
}

func (c *Client) ValidateRenewal(ctx context.Context, subscriptionID int64) (*ValidateRenewalResp, error) {
	var resp ValidateRenewalResp
	err := c.do(ctx, "POST", "/v1/telegram/validate-renewal", ValidateRenewalReq{SubscriptionID: subscriptionID}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

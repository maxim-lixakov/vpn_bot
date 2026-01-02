package appclient

import (
	"context"
	"net/http"
)

type SubscriptionRenewalReminderResp struct {
	NotifiedCount int      `json:"notified_count"`
	Errors        []string `json:"errors,omitempty"`
}

// SubscriptionRenewalReminder sends renewal reminders and invoices to users with subscriptions expiring tomorrow
func (c *Client) SubscriptionRenewalReminder(ctx context.Context) (SubscriptionRenewalReminderResp, error) {
	var out SubscriptionRenewalReminderResp
	err := c.doJSON(ctx, http.MethodPost, "/v1/subscription-renewal-reminder", nil, &out)
	return out, err
}

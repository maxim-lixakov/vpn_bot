package cleanup_broken_subscriptions

import (
	"context"
	"fmt"
	"log"

	"vpn-periodic-tasks/internal/appclient"
	"vpn-periodic-tasks/internal/config"
)

// Task implements the scheduler.Task interface for cleaning up broken subscriptions
type Task struct {
	client *appclient.Client
}

// New creates a new cleanup broken subscriptions task
func New(client *appclient.Client) *Task {
	return &Task{
		client: client,
	}
}

// Name returns the task name
func (t *Task) Name() string {
	return "cleanup_broken_subscriptions"
}

// Run executes the task
func (t *Task) Run(ctx context.Context, cfg config.Config) error {
	result, err := t.client.CleanupBrokenSubscriptions(ctx)
	if err != nil {
		return fmt.Errorf("call cleanup-broken-subscriptions endpoint: %w", err)
	}

	if result.TotalFound == 0 {
		log.Printf("cleanup: no broken subscriptions found")
		return nil
	}

	log.Printf("cleanup: found %d broken subscriptions, cleaned %d, failed %d",
		result.TotalFound, result.Cleaned, result.Failed)

	if result.Cleaned > 0 {
		log.Printf("successfully cleaned subscriptions:")
		for _, sub := range result.Subscriptions {
			if sub.Action == "deleted" {
				log.Printf("  - subscription %d (user %d, country %s, paid_at=%s, promocode=%v)",
					sub.SubscriptionID, sub.UserID, sub.CountryCode,
					sub.PaidAt.Format("2006-01-02 15:04"), sub.IsPromocode)
			}
		}
	}

	if result.Failed > 0 {
		log.Printf("failed to clean %d subscriptions:", result.Failed)
		for _, sub := range result.Subscriptions {
			if sub.Action == "failed" {
				log.Printf("  - subscription %d: %s", sub.SubscriptionID, sub.Error)
			}
		}
	}

	return nil
}

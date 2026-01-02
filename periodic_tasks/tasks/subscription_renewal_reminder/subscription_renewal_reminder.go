package subscription_renewal_reminder

import (
	"context"
	"fmt"
	"log"

	"vpn-periodic-tasks/internal/appclient"
	"vpn-periodic-tasks/internal/config"
)

// Task implements the scheduler.Task interface for subscription renewal reminders
type Task struct {
	appClient *appclient.Client
}

// New creates a new subscription renewal reminder task
func New(appClient *appclient.Client) *Task {
	return &Task{appClient: appClient}
}

// Name returns the task name
func (t *Task) Name() string {
	return "subscription_renewal_reminder"
}

// Run executes the task
func (t *Task) Run(ctx context.Context, cfg config.Config) error {
	log.Println("starting subscription renewal reminder task...")

	result, err := t.appClient.SubscriptionRenewalReminder(ctx)
	if err != nil {
		log.Printf("Error calling app API for subscription renewal reminder: %v", err)
		return fmt.Errorf("app API call failed: %w", err)
	}

	log.Printf("Subscription renewal reminder task completed: notified %d users", result.NotifiedCount)
	if len(result.Errors) > 0 {
		log.Printf("Encountered %d errors during notification:", len(result.Errors))
		for _, errMsg := range result.Errors {
			log.Printf("  - %s", errMsg)
		}
	}

	return nil
}

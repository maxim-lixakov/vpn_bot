package revoke_expired_keys

import (
	"context"
	"fmt"
	"log"

	"vpn-periodic-tasks/internal/appclient"
	"vpn-periodic-tasks/internal/config"
)

// Task implements the scheduler.Task interface for revoking expired access keys
type Task struct {
	client *appclient.Client
}

// New creates a new revoke expired keys task
func New(client *appclient.Client) *Task {
	return &Task{
		client: client,
	}
}

// Name returns the task name
func (t *Task) Name() string {
	return "revoke_expired_keys"
}

// Run executes the task
func (t *Task) Run(ctx context.Context, cfg config.Config) error {
	result, err := t.client.RevokeExpiredKeys(ctx)
	if err != nil {
		return fmt.Errorf("call revoke-expired-keys endpoint: %w", err)
	}

	log.Printf("revoked %d expired access keys", result.RevokedCount)
	if len(result.Errors) > 0 {
		log.Printf("encountered %d errors during revocation:", len(result.Errors))
		for _, errMsg := range result.Errors {
			log.Printf("  - %s", errMsg)
		}
	}

	return nil
}
